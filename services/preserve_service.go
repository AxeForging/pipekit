package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Preserving in-place edits.
//
// The default Decode→mutate→Encode round-trip normalizes the whole document:
// it drops comments, reorders keys, rewrites quoting, and re-indents. That is
// fine for generated data but wrong when editing a hand-maintained file (e.g. a
// Helm values.yaml or a chart's image tag) where the intent is "change ONLY the
// targeted value and touch nothing else".
//
// These functions perform surgical, byte-level edits so EVERYTHING outside the
// target stays byte-for-byte identical — comments (including their column
// alignment), key order, quoting style, and blank lines:
//
//   - YAML: locate the target node via the yaml.Node tree (which carries source
//     line/column), then splice the new value into the original bytes. Every
//     splice is validated by re-parsing the result and confirming the target now
//     holds the intended value; if anything looks off we fall back to the safe
//     node re-encode path, so a surgical edit can never corrupt the file.
//   - JSON: splice the exact value span (or insert a new key, formatting-matched
//     to existing siblings), leaving all surrounding bytes untouched.
//
// TOML/CSV are not supported in preserving mode; callers get a clear error.

// SetPreserving updates the value at path while preserving the document's
// original formatting. Returns the rewritten bytes.
func SetPreserving(data []byte, format DataFormat, path string, value interface{}) ([]byte, error) {
	switch format {
	case FormatYAML:
		return setYAMLPreserving(data, path, value)
	case FormatJSON:
		return setJSONPreserving(data, path, value)
	default:
		return nil, fmt.Errorf("--preserve is only supported for yaml and json (got %s)", format)
	}
}

// DelPreserving removes the value at path while preserving formatting.
func DelPreserving(data []byte, format DataFormat, path string) ([]byte, error) {
	switch format {
	case FormatYAML:
		return delYAMLPreserving(data, path)
	case FormatJSON:
		return delJSONPreserving(data, path)
	default:
		return nil, fmt.Errorf("--preserve is only supported for yaml and json (got %s)", format)
	}
}

// ============================ YAML ============================

func setYAMLPreserving(data []byte, path string, value interface{}) ([]byte, error) {
	parts, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	// Fast path: existing single-line scalar → byte-splice, preserving everything.
	if out, ok := tryYAMLScalarSplice(data, parts, value, path); ok {
		return out, nil
	}
	// Fallback (new keys, nested creation, complex values, multiline scalars):
	// node re-encode. This normalizes formatting but always produces valid YAML.
	return setYAMLNodeEncode(data, parts, value)
}

func delYAMLPreserving(data []byte, path string) ([]byte, error) {
	parts, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	// Fast path: deleting a single-line mapping entry → drop just that line.
	if out, ok := tryYAMLLeafDelete(data, parts, path); ok {
		return out, nil
	}
	return delYAMLNodeEncode(data, parts)
}

// ---- YAML byte-splice (full preservation) ----

func tryYAMLScalarSplice(data []byte, parts []string, value interface{}, path string) ([]byte, bool) {
	if !isScalarValue(value) {
		return nil, false
	}
	top, ok := yamlTopNode(data)
	if !ok {
		return nil, false
	}
	target, ok := findYAMLValueNode(top, parts)
	if !ok || target.Kind != yaml.ScalarNode {
		return nil, false
	}
	if target.Style == yaml.LiteralStyle || target.Style == yaml.FoldedStyle {
		return nil, false // block scalars span multiple lines
	}
	starts := lineStarts(data)
	start, ok := offsetAt(starts, target.Line, target.Column)
	if !ok {
		return nil, false
	}
	end, ok := scalarEnd(data, start, target.Style, target.Value)
	if !ok {
		return nil, false
	}
	repl, ok := buildScalarText(target.Style, value)
	if !ok {
		return nil, false
	}
	out := splice(data, start, end, []byte(repl))
	// Validate: result must parse and the target path must now hold value.
	// A wrong-location splice would leave the real target unchanged and fail here.
	if !yamlPathHasValue(out, path, value) {
		return nil, false
	}
	return out, true
}

func tryYAMLLeafDelete(data []byte, parts []string, path string) ([]byte, bool) {
	top, ok := yamlTopNode(data)
	if !ok {
		return nil, false
	}
	leaf, ok := findYAMLLeaf(top, parts)
	if !ok || leaf.inSeq {
		return nil, false // sequence element removal handled by fallback
	}
	if leaf.val.Kind != yaml.ScalarNode {
		return nil, false // nested block: multiple lines, hard to bound safely
	}
	if leaf.val.Style == yaml.LiteralStyle || leaf.val.Style == yaml.FoldedStyle {
		return nil, false
	}
	if leaf.val.Line != leaf.key.Line {
		return nil, false // value continues on another line
	}
	starts := lineStarts(data)
	line := leaf.key.Line
	if line < 1 || line > len(starts) {
		return nil, false
	}
	lineStart := starts[line-1]
	lineEnd := len(data)
	if line < len(starts) {
		lineEnd = starts[line]
	}
	out := splice(data, lineStart, lineEnd, nil)
	// Validate: must still parse and the path must be gone.
	var chk interface{}
	if yaml.Unmarshal(out, &chk) != nil {
		return nil, false
	}
	if pathPresent(normalizeMaps(chk), parts) {
		return nil, false
	}
	return out, true
}

func yamlTopNode(data []byte) (*yaml.Node, bool) {
	var root yaml.Node
	if yaml.Unmarshal(data, &root) != nil {
		return nil, false
	}
	if root.Kind == yaml.DocumentNode {
		if len(root.Content) == 0 {
			return nil, false
		}
		return root.Content[0], true
	}
	if root.Kind == 0 {
		return nil, false
	}
	return &root, true
}

func findYAMLValueNode(node *yaml.Node, parts []string) (*yaml.Node, bool) {
	if len(parts) == 0 {
		return node, true
	}
	head, rest := parts[0], parts[1:]
	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			if node.Content[i].Value == head {
				return findYAMLValueNode(node.Content[i+1], rest)
			}
		}
	case yaml.SequenceNode:
		if idx, err := parseIndex(head, len(node.Content)); err == nil {
			return findYAMLValueNode(node.Content[idx], rest)
		}
	}
	return nil, false
}

type yamlLeaf struct {
	key   *yaml.Node // nil for sequence elements
	val   *yaml.Node
	inSeq bool
}

func findYAMLLeaf(top *yaml.Node, parts []string) (*yamlLeaf, bool) {
	if len(parts) == 0 {
		return nil, false
	}
	container, ok := findYAMLValueNode(top, parts[:len(parts)-1])
	if !ok {
		return nil, false
	}
	last := parts[len(parts)-1]
	switch container.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(container.Content); i += 2 {
			if container.Content[i].Value == last {
				return &yamlLeaf{key: container.Content[i], val: container.Content[i+1]}, true
			}
		}
	case yaml.SequenceNode:
		if idx, err := parseIndex(last, len(container.Content)); err == nil {
			return &yamlLeaf{val: container.Content[idx], inSeq: true}, true
		}
	}
	return nil, false
}

// scalarEnd returns the byte offset just past the scalar token that begins at
// start, or false if the token is malformed / spans multiple lines.
func scalarEnd(data []byte, start int, style yaml.Style, oldVal string) (int, bool) {
	if start < 0 || start >= len(data) {
		return 0, false
	}
	switch style {
	case yaml.DoubleQuotedStyle:
		if data[start] != '"' {
			return 0, false
		}
		for i := start + 1; i < len(data); i++ {
			switch data[i] {
			case '\\':
				i++ // skip escaped char
			case '\n':
				return 0, false
			case '"':
				return i + 1, true
			}
		}
		return 0, false
	case yaml.SingleQuotedStyle:
		if data[start] != '\'' {
			return 0, false
		}
		for i := start + 1; i < len(data); i++ {
			if data[i] == '\n' {
				return 0, false
			}
			if data[i] == '\'' {
				if i+1 < len(data) && data[i+1] == '\'' {
					i++ // escaped ''
					continue
				}
				return i + 1, true
			}
		}
		return 0, false
	default: // plain
		lineEnd := start
		for lineEnd < len(data) && data[lineEnd] != '\n' {
			lineEnd++
		}
		seg := data[start:lineEnd]
		cut := len(seg)
		for j := 1; j < len(seg); j++ {
			if seg[j] == '#' && (seg[j-1] == ' ' || seg[j-1] == '\t') {
				cut = j
				break
			}
		}
		valEnd := start + cut
		for valEnd > start && (data[valEnd-1] == ' ' || data[valEnd-1] == '\t') {
			valEnd--
		}
		// Strong location check: the bytes must equal the decoded value.
		if string(data[start:valEnd]) != oldVal {
			return 0, false
		}
		return valEnd, true
	}
}

// buildScalarText renders value as a scalar token in the given style.
func buildScalarText(style yaml.Style, value interface{}) (string, bool) {
	s := scalarToString(value)
	if strings.ContainsAny(s, "\n\r") {
		return "", false // would break a single-line splice
	}
	switch style {
	case yaml.DoubleQuotedStyle:
		str, ok := value.(string)
		if !ok {
			return "", false
		}
		b, err := json.Marshal(str) // JSON string escapes are valid YAML double-quoted
		if err != nil {
			return "", false
		}
		return string(b), true
	case yaml.SingleQuotedStyle:
		if _, ok := value.(string); !ok {
			return "", false
		}
		return "'" + strings.ReplaceAll(s, "'", "''") + "'", true
	default: // plain — validation backstops anything that isn't plain-safe
		return s, true
	}
}

func yamlPathHasValue(data []byte, path string, want interface{}) bool {
	var chk interface{}
	if yaml.Unmarshal(data, &chk) != nil {
		return false
	}
	got, err := JSONGet(normalizeMaps(chk), path)
	if err != nil {
		return false
	}
	return scalarToString(got) == scalarToString(want)
}

// ---- YAML node re-encode (fallback; normalizes formatting) ----

func setYAMLNodeEncode(data []byte, parts []string, value interface{}) ([]byte, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("decoding YAML: %w", err)
	}
	target := yamlRootContent(&root)
	if err := setNodeAt(target, parts, value); err != nil {
		return nil, err
	}
	return encodeYAMLNode(&root)
}

func delYAMLNodeEncode(data []byte, parts []string) ([]byte, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("decoding YAML: %w", err)
	}
	if len(root.Content) == 0 {
		return data, nil
	}
	if err := delNodeAt(root.Content[0], parts); err != nil {
		return nil, err
	}
	return encodeYAMLNode(&root)
}

func yamlRootContent(root *yaml.Node) *yaml.Node {
	if root.Kind == 0 || len(root.Content) == 0 {
		root.Kind = yaml.DocumentNode
		content := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		root.Content = []*yaml.Node{content}
		return content
	}
	return root.Content[0]
}

func encodeYAMLNode(root *yaml.Node) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return nil, fmt.Errorf("encoding YAML: %w", err)
	}
	_ = enc.Close()
	return buf.Bytes(), nil
}

func setNodeAt(node *yaml.Node, parts []string, value interface{}) error {
	head, rest := parts[0], parts[1:]
	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			if node.Content[i].Value == head {
				if len(rest) == 0 {
					return assignNodeValue(node, i+1, value)
				}
				return setNodeAt(node.Content[i+1], rest, value)
			}
		}
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: head}
		valNode := &yaml.Node{}
		if len(rest) == 0 {
			if err := valNode.Encode(value); err != nil {
				return err
			}
		} else {
			valNode.Kind = yaml.MappingNode
			valNode.Tag = "!!map"
			if err := setNodeAt(valNode, rest, value); err != nil {
				return err
			}
		}
		node.Content = append(node.Content, keyNode, valNode)
		return nil
	case yaml.SequenceNode:
		idx, err := parseIndex(head, len(node.Content))
		if err != nil {
			return err
		}
		if len(rest) == 0 {
			return assignNodeValue(node, idx, value)
		}
		return setNodeAt(node.Content[idx], rest, value)
	case yaml.ScalarNode:
		if node.Tag == "!!null" || node.Value == "" {
			node.Kind = yaml.MappingNode
			node.Tag = "!!map"
			node.Value = ""
			node.Style = 0
			return setNodeAt(node, parts, value)
		}
		return fmt.Errorf("cannot index into scalar value at %q", head)
	case 0:
		node.Kind = yaml.MappingNode
		node.Tag = "!!map"
		return setNodeAt(node, parts, value)
	}
	return fmt.Errorf("cannot index into node at %q", head)
}

func assignNodeValue(parent *yaml.Node, idx int, value interface{}) error {
	old := parent.Content[idx]
	newNode := &yaml.Node{}
	if err := newNode.Encode(value); err != nil {
		return err
	}
	newNode.HeadComment = old.HeadComment
	newNode.LineComment = old.LineComment
	newNode.FootComment = old.FootComment
	if old.Kind == yaml.ScalarNode && newNode.Kind == yaml.ScalarNode && old.Tag == newNode.Tag {
		newNode.Style = old.Style
	}
	parent.Content[idx] = newNode
	return nil
}

func delNodeAt(node *yaml.Node, parts []string) error {
	head, rest := parts[0], parts[1:]
	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			if node.Content[i].Value == head {
				if len(rest) == 0 {
					node.Content = append(node.Content[:i], node.Content[i+2:]...)
					return nil
				}
				return delNodeAt(node.Content[i+1], rest)
			}
		}
		return nil
	case yaml.SequenceNode:
		idx, err := parseIndex(head, len(node.Content))
		if err != nil {
			return err
		}
		if len(rest) == 0 {
			node.Content = append(node.Content[:idx], node.Content[idx+1:]...)
			return nil
		}
		return delNodeAt(node.Content[idx], rest)
	}
	return nil
}

// ============================ JSON ============================

func setJSONPreserving(data []byte, path string, value interface{}) ([]byte, error) {
	parts, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	start, end, locErr := locateJSONValue(data, parts)
	if locErr == nil {
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("encoding value: %w", err)
		}
		return splice(data, start, end, encoded), nil
	}
	// Path not found: insert into the (existing) parent object.
	out, err := insertJSONPreserving(data, parts, value)
	if err != nil {
		return nil, fmt.Errorf("%v (set --preserve updates existing paths or inserts a key into an existing object)", err)
	}
	return out, nil
}

func delJSONPreserving(data []byte, path string) ([]byte, error) {
	parts, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	start, end, err := locateJSONEntry(data, parts)
	if err != nil {
		return nil, err
	}
	return splice(data, start, end, nil), nil
}

func insertJSONPreserving(data []byte, parts []string, value interface{}) ([]byte, error) {
	parentParts, key := parts[:len(parts)-1], parts[len(parts)-1]
	var cStart, cEnd int
	if len(parentParts) == 0 {
		cStart, cEnd = 0, len(data)
	} else {
		var err error
		cStart, cEnd, err = locateJSONValue(data, parentParts)
		if err != nil {
			return nil, err
		}
	}
	container := bytes.TrimRight(data[cStart:cEnd], " \t\r\n")
	cEnd = cStart + len(container)
	newContainer, err := insertIntoObject(container, key, value)
	if err != nil {
		return nil, err
	}
	out := splice(data, cStart, cEnd, newContainer)
	// Safety: result must parse and contain the inserted value.
	var chk interface{}
	if json.Unmarshal(out, &chk) != nil {
		return nil, fmt.Errorf("insert would produce invalid JSON")
	}
	got, err := JSONGet(chk, "."+jqJoin(parts))
	if err != nil || !jsonEqual(got, value) {
		return nil, fmt.Errorf("insert validation failed")
	}
	return out, nil
}

// insertIntoObject returns container with key:value inserted, mirroring the
// separator and colon spacing of existing siblings.
func insertIntoObject(container []byte, key string, value interface{}) ([]byte, error) {
	keyJSON, _ := json.Marshal(key)
	valJSON, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encoding value: %w", err)
	}

	dec := json.NewDecoder(bytes.NewReader(container))
	open, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := open.(json.Delim); !ok || d != '{' {
		return nil, fmt.Errorf("cannot insert key into a non-object")
	}

	type child struct{ start, end int }
	var children []child
	for dec.More() {
		ks := skipBackToToken(container, int(dec.InputOffset()))
		keyTok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		if k, _ := keyTok.(string); k == key {
			return nil, fmt.Errorf("key %q already exists", key)
		}
		if err := skipJSONValue(dec); err != nil {
			return nil, err
		}
		children = append(children, child{ks, int(dec.InputOffset())})
	}
	openEnd := bytes.IndexByte(container, '{') + 1
	closeStart := bytes.LastIndexByte(container, '}')
	if openEnd <= 0 || closeStart < 0 {
		return nil, fmt.Errorf("malformed object")
	}

	if len(children) == 0 {
		entry := string(keyJSON) + ": " + string(valJSON)
		return splice(container, openEnd, openEnd, []byte(entry)), nil
	}

	last := children[len(children)-1]
	var sep string
	if len(children) >= 2 {
		sep = string(container[children[len(children)-2].end:last.start])
	} else {
		sep = "," + string(container[openEnd:children[0].start])
	}
	colon := detectColon(container[last.start:last.end])
	entry := sep + string(keyJSON) + colon + string(valJSON)
	return splice(container, last.end, last.end, []byte(entry)), nil
}

// detectColon returns the ":"+spacing used between a key and its value, read
// from one existing entry's bytes (e.g. `"a": 1` → ": ").
func detectColon(entry []byte) string {
	depth := 0
	for i := 0; i < len(entry); i++ {
		switch entry[i] {
		case '"':
			i++
			for i < len(entry) && entry[i] != '"' {
				if entry[i] == '\\' {
					i++
				}
				i++
			}
		case ':':
			if depth == 0 {
				j := i + 1
				for j < len(entry) && (entry[j] == ' ' || entry[j] == '\t') {
					j++
				}
				return ":" + string(entry[i+1:j])
			}
		case '{', '[':
			depth++
		case '}', ']':
			depth--
		}
	}
	return ":"
}

func locateJSONValue(data []byte, parts []string) (int, int, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	return descendJSON(dec, parts)
}

func descendJSON(dec *json.Decoder, parts []string) (int, int, error) {
	head, rest := parts[0], parts[1:]
	tok, err := dec.Token()
	if err != nil {
		return 0, 0, fmt.Errorf("path %q: %w", head, err)
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return 0, 0, fmt.Errorf("path %q: value is not a container", head)
	}
	switch delim {
	case '{':
		for dec.More() {
			keyTok, err := dec.Token()
			if err != nil {
				return 0, 0, err
			}
			if key, _ := keyTok.(string); key == head {
				if len(rest) == 0 {
					return valueSpan(dec)
				}
				return descendJSON(dec, rest)
			}
			if err := skipJSONValue(dec); err != nil {
				return 0, 0, err
			}
		}
		return 0, 0, fmt.Errorf("path not found: %q", head)
	case '[':
		idx, err := strconv.Atoi(head)
		if err != nil {
			return 0, 0, fmt.Errorf("not an array index: %q", head)
		}
		for i := 0; dec.More(); i++ {
			if i == idx {
				if len(rest) == 0 {
					return valueSpan(dec)
				}
				return descendJSON(dec, rest)
			}
			if err := skipJSONValue(dec); err != nil {
				return 0, 0, err
			}
		}
		return 0, 0, fmt.Errorf("array index out of bounds: %d", idx)
	}
	return 0, 0, fmt.Errorf("path %q: unexpected %q", head, delim)
}

func valueSpan(dec *json.Decoder) (int, int, error) {
	var raw json.RawMessage
	if err := dec.Decode(&raw); err != nil {
		return 0, 0, err
	}
	end := int(dec.InputOffset())
	return end - len(raw), end, nil
}

func skipJSONValue(dec *json.Decoder) error {
	var raw json.RawMessage
	return dec.Decode(&raw)
}

func locateJSONEntry(data []byte, parts []string) (int, int, error) {
	parent, last := parts[:len(parts)-1], parts[len(parts)-1]
	var cStart, cEnd int
	var err error
	if len(parent) == 0 {
		cStart, cEnd = 0, len(data)
	} else {
		cStart, cEnd, err = locateJSONValue(data, parent)
		if err != nil {
			return 0, 0, err
		}
	}
	relStart, relEnd, err := entrySpanInContainer(data[cStart:cEnd], last)
	if err != nil {
		return 0, 0, err
	}
	return cStart + relStart, cStart + relEnd, nil
}

func entrySpanInContainer(container []byte, key string) (int, int, error) {
	dec := json.NewDecoder(bytes.NewReader(container))
	open, err := dec.Token()
	if err != nil {
		return 0, 0, err
	}
	delim, ok := open.(json.Delim)
	if !ok {
		return 0, 0, fmt.Errorf("not a container")
	}

	type child struct{ start, end int }
	var children []child
	matched := -1

	switch delim {
	case '{':
		for dec.More() {
			keyStart := skipBackToToken(container, int(dec.InputOffset()))
			keyTok, err := dec.Token()
			if err != nil {
				return 0, 0, err
			}
			k, _ := keyTok.(string)
			if err := skipJSONValue(dec); err != nil {
				return 0, 0, err
			}
			if k == key {
				matched = len(children)
			}
			children = append(children, child{keyStart, int(dec.InputOffset())})
		}
	case '[':
		idx, err := strconv.Atoi(key)
		if err != nil {
			return 0, 0, fmt.Errorf("not an array index: %q", key)
		}
		for i := 0; dec.More(); i++ {
			start := skipBackToToken(container, int(dec.InputOffset()))
			if err := skipJSONValue(dec); err != nil {
				return 0, 0, err
			}
			if i == idx {
				matched = len(children)
			}
			children = append(children, child{start, int(dec.InputOffset())})
		}
	default:
		return 0, 0, fmt.Errorf("not an object or array")
	}

	if matched < 0 {
		return 0, 0, fmt.Errorf("path not found: %q", key)
	}
	c := children[matched]
	start, end := c.start, c.end
	if len(children) == 1 {
		return start, end, nil
	}
	if matched == len(children)-1 {
		start = children[matched-1].end // also drop the preceding comma
	} else {
		end = children[matched+1].start // also drop the trailing comma
	}
	return start, end, nil
}

func skipBackToToken(b []byte, off int) int {
	for off < len(b) {
		switch b[off] {
		case ' ', '\t', '\r', '\n', ',':
			off++
		default:
			return off
		}
	}
	return off
}

// ============================ shared helpers ============================

func splice(data []byte, start, end int, repl []byte) []byte {
	out := make([]byte, 0, len(data)-(end-start)+len(repl))
	out = append(out, data[:start]...)
	out = append(out, repl...)
	out = append(out, data[end:]...)
	return out
}

func lineStarts(data []byte) []int {
	starts := []int{0}
	for i, b := range data {
		if b == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

// offsetAt converts a 1-based (line, column) to a byte offset.
func offsetAt(starts []int, line, col int) (int, bool) {
	if line < 1 || line > len(starts) || col < 1 {
		return 0, false
	}
	return starts[line-1] + (col - 1), true
}

func isScalarValue(v interface{}) bool {
	switch v.(type) {
	case map[string]interface{}, []interface{}, map[interface{}]interface{}:
		return false
	}
	return true
}

func scalarToString(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return "null"
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case float64:
		return strconv.FormatFloat(t, 'g', -1, 64)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	default:
		return fmt.Sprintf("%v", t)
	}
}

func pathPresent(doc interface{}, parts []string) bool {
	cur := doc
	for _, p := range parts {
		switch c := cur.(type) {
		case map[string]interface{}:
			v, ok := c[p]
			if !ok {
				return false
			}
			cur = v
		case []interface{}:
			idx, err := strconv.Atoi(p)
			if err != nil || idx < 0 || idx >= len(c) {
				return false
			}
			cur = c[idx]
		default:
			return false
		}
	}
	return true
}

func jsonEqual(a, b interface{}) bool {
	ab, err1 := json.Marshal(a)
	bb, err2 := json.Marshal(b)
	return err1 == nil && err2 == nil && bytes.Equal(ab, bb)
}

func jqJoin(parts []string) string {
	var b strings.Builder
	for i, p := range parts {
		if _, err := strconv.Atoi(p); err == nil {
			b.WriteString("[" + p + "]")
			continue
		}
		if i > 0 {
			b.WriteByte('.')
		}
		b.WriteString(p)
	}
	return b.String()
}
