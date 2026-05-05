package services

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/itchyny/gojq"
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// DataFormat identifies a structured-data serialization.
type DataFormat string

const (
	FormatJSON DataFormat = "json"
	FormatYAML DataFormat = "yaml"
	FormatTOML DataFormat = "toml"
	FormatCSV  DataFormat = "csv"
)

// DetectFormat picks a format from a filename extension. Returns "" if
// no known extension is present.
func DetectFormat(filename string) DataFormat {
	switch {
	case strings.HasSuffix(filename, ".json"), strings.HasSuffix(filename, ".jsonl"):
		return FormatJSON
	case strings.HasSuffix(filename, ".yaml"), strings.HasSuffix(filename, ".yml"):
		return FormatYAML
	case strings.HasSuffix(filename, ".toml"):
		return FormatTOML
	case strings.HasSuffix(filename, ".csv"):
		return FormatCSV
	}
	return ""
}

// Decode reads bytes in the given format into a generic map/slice tree.
func Decode(data []byte, format DataFormat) (interface{}, error) {
	switch format {
	case FormatJSON:
		var v interface{}
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, fmt.Errorf("decoding JSON: %w", err)
		}
		return v, nil
	case FormatYAML:
		var v interface{}
		if err := yaml.Unmarshal(data, &v); err != nil {
			return nil, fmt.Errorf("decoding YAML: %w", err)
		}
		return normalizeMaps(v), nil
	case FormatTOML:
		var v map[string]interface{}
		if err := toml.Unmarshal(data, &v); err != nil {
			return nil, fmt.Errorf("decoding TOML: %w", err)
		}
		return v, nil
	case FormatCSV:
		return decodeCSV(data)
	}
	return nil, fmt.Errorf("unknown format: %s", format)
}

// Encode serializes the given value into the given format.
func Encode(v interface{}, format DataFormat, pretty bool) ([]byte, error) {
	switch format {
	case FormatJSON:
		if pretty {
			return json.MarshalIndent(v, "", "  ")
		}
		return json.Marshal(v)
	case FormatYAML:
		var buf strings.Builder
		enc := yaml.NewEncoder(stringWriter{&buf})
		enc.SetIndent(2)
		if err := enc.Encode(v); err != nil {
			return nil, fmt.Errorf("encoding YAML: %w", err)
		}
		_ = enc.Close()
		return []byte(buf.String()), nil
	case FormatTOML:
		m, ok := v.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("TOML encoding requires a top-level object, got %T", v)
		}
		return toml.Marshal(m)
	case FormatCSV:
		return encodeCSV(v)
	}
	return nil, fmt.Errorf("unknown format: %s", format)
}

type stringWriter struct{ b *strings.Builder }

func (s stringWriter) Write(p []byte) (int, error) { return s.b.Write(p) }

// JSONGet returns the value at the given gojq path expression.
// The expression must start with "." (jq syntax). When the path is empty,
// the whole document is returned.
func JSONGet(doc interface{}, path string) (interface{}, error) {
	if path == "" || path == "." {
		return doc, nil
	}
	q, err := gojq.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("parsing path %q: %w", path, err)
	}
	iter := q.Run(doc)
	v, ok := iter.Next()
	if !ok {
		return nil, fmt.Errorf("path %q produced no result", path)
	}
	if e, ok := v.(error); ok {
		return nil, fmt.Errorf("path %q: %w", path, e)
	}
	return v, nil
}

// JSONSet returns a new document with the value at path replaced. Only
// supports simple dotted/bracketed paths like ".image.tag" or ".items[0].name".
func JSONSet(doc interface{}, path string, value interface{}) (interface{}, error) {
	parts, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	return setAt(doc, parts, value)
}

// JSONDel returns a new document with the entry at path removed.
func JSONDel(doc interface{}, path string) (interface{}, error) {
	parts, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	return delAt(doc, parts)
}

// DeepMerge recursively merges src into dst — maps merge key-by-key, scalars
// and slices in src override dst. Returns the merged value (does not mutate).
func DeepMerge(dst, src interface{}) interface{} {
	dstMap, dstOK := dst.(map[string]interface{})
	srcMap, srcOK := src.(map[string]interface{})
	if !dstOK || !srcOK {
		return src
	}
	out := make(map[string]interface{}, len(dstMap)+len(srcMap))
	for k, v := range dstMap {
		out[k] = v
	}
	for k, v := range srcMap {
		if existing, ok := out[k]; ok {
			out[k] = DeepMerge(existing, v)
		} else {
			out[k] = v
		}
	}
	return out
}

// RenderTable renders an array of objects as an aligned-column ASCII table.
// columns specifies which fields to include (in order); empty means use all
// fields from the first row, sorted.
func RenderTable(records []map[string]interface{}, columns []string) string {
	if len(records) == 0 {
		return ""
	}
	if len(columns) == 0 {
		seen := make(map[string]bool)
		for _, r := range records {
			for k := range r {
				if !seen[k] {
					seen[k] = true
					columns = append(columns, k)
				}
			}
		}
		sort.Strings(columns)
	}

	widths := make([]int, len(columns))
	for i, c := range columns {
		widths[i] = len(c)
	}
	rows := make([][]string, 0, len(records))
	for _, r := range records {
		row := make([]string, len(columns))
		for i, c := range columns {
			val := scalarOrJSON(r[c])
			row[i] = val
			if len(val) > widths[i] {
				widths[i] = len(val)
			}
		}
		rows = append(rows, row)
	}

	var b strings.Builder
	writeTableRow(&b, columns, widths)
	sep := make([]string, len(columns))
	for i, w := range widths {
		sep[i] = strings.Repeat("-", w)
	}
	writeTableRow(&b, sep, widths)
	for _, row := range rows {
		writeTableRow(&b, row, widths)
	}
	return b.String()
}

func writeTableRow(b *strings.Builder, cells []string, widths []int) {
	for i, c := range cells {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(c)
		if i < len(cells)-1 {
			b.WriteString(strings.Repeat(" ", widths[i]-len(c)))
		}
	}
	b.WriteString("\n")
}

// parsePath turns ".a.b[0].c" into ["a", "b", "0", "c"]. The leading dot is
// optional. Returns an error for empty/whitespace paths.
func parsePath(path string) ([]string, error) {
	path = strings.TrimSpace(path)
	if path == "" || path == "." {
		return nil, fmt.Errorf("empty path")
	}
	path = strings.TrimPrefix(path, ".")
	// Convert [N] to .N
	path = strings.ReplaceAll(path, "[", ".")
	path = strings.ReplaceAll(path, "]", "")
	parts := strings.Split(path, ".")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty path")
	}
	return out, nil
}

func setAt(doc interface{}, parts []string, value interface{}) (interface{}, error) {
	if len(parts) == 0 {
		return value, nil
	}
	head, rest := parts[0], parts[1:]
	switch d := doc.(type) {
	case map[string]interface{}:
		out := cloneMap(d)
		child, exists := out[head]
		if !exists {
			child = nil
		}
		updated, err := setAt(child, rest, value)
		if err != nil {
			return nil, err
		}
		out[head] = updated
		return out, nil
	case []interface{}:
		idx, err := parseIndex(head, len(d))
		if err != nil {
			return nil, err
		}
		out := make([]interface{}, len(d))
		copy(out, d)
		updated, err := setAt(out[idx], rest, value)
		if err != nil {
			return nil, err
		}
		out[idx] = updated
		return out, nil
	case nil:
		// Auto-create a map; if the head looks like an integer, create slice.
		if _, err := parseIndex(head, 0); err == nil {
			return nil, fmt.Errorf("cannot create slice at non-existent path")
		}
		out := map[string]interface{}{}
		updated, err := setAt(nil, rest, value)
		if err != nil {
			return nil, err
		}
		out[head] = updated
		return out, nil
	}
	return nil, fmt.Errorf("cannot index into %T at %q", doc, head)
}

func delAt(doc interface{}, parts []string) (interface{}, error) {
	if len(parts) == 0 {
		return nil, nil
	}
	head, rest := parts[0], parts[1:]
	switch d := doc.(type) {
	case map[string]interface{}:
		out := cloneMap(d)
		if len(rest) == 0 {
			delete(out, head)
			return out, nil
		}
		if child, ok := out[head]; ok {
			updated, err := delAt(child, rest)
			if err != nil {
				return nil, err
			}
			out[head] = updated
		}
		return out, nil
	case []interface{}:
		idx, err := parseIndex(head, len(d))
		if err != nil {
			return nil, err
		}
		if len(rest) == 0 {
			out := make([]interface{}, 0, len(d)-1)
			out = append(out, d[:idx]...)
			out = append(out, d[idx+1:]...)
			return out, nil
		}
		out := make([]interface{}, len(d))
		copy(out, d)
		updated, err := delAt(out[idx], rest)
		if err != nil {
			return nil, err
		}
		out[idx] = updated
		return out, nil
	}
	return doc, nil
}

func parseIndex(s string, length int) (int, error) {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return 0, fmt.Errorf("not an array index: %q", s)
	}
	if n < 0 || (length > 0 && n >= length) {
		return 0, fmt.Errorf("array index out of bounds: %d (len=%d)", n, length)
	}
	return n, nil
}

func cloneMap(m map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// normalizeMaps walks a yaml-decoded tree and converts map[interface{}]interface{}
// values (yaml.v3 sometimes produces these for non-string keys) into
// map[string]interface{} so json.Marshal works.
func normalizeMaps(v interface{}) interface{} {
	switch t := v.(type) {
	case map[interface{}]interface{}:
		out := make(map[string]interface{}, len(t))
		for k, val := range t {
			out[fmt.Sprintf("%v", k)] = normalizeMaps(val)
		}
		return out
	case map[string]interface{}:
		out := make(map[string]interface{}, len(t))
		for k, val := range t {
			out[k] = normalizeMaps(val)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(t))
		for i, item := range t {
			out[i] = normalizeMaps(item)
		}
		return out
	}
	return v
}

func decodeCSV(data []byte) (interface{}, error) {
	r := csv.NewReader(strings.NewReader(string(data)))
	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("decoding CSV: %w", err)
	}
	if len(rows) < 1 {
		return []interface{}{}, nil
	}
	headers := rows[0]
	out := make([]interface{}, 0, len(rows)-1)
	for _, row := range rows[1:] {
		obj := make(map[string]interface{}, len(headers))
		for i, h := range headers {
			if i < len(row) {
				obj[h] = row[i]
			}
		}
		out = append(out, obj)
	}
	return out, nil
}

func encodeCSV(v interface{}) ([]byte, error) {
	arr, ok := v.([]interface{})
	if !ok {
		return nil, fmt.Errorf("CSV encoding requires a top-level array, got %T", v)
	}
	if len(arr) == 0 {
		return []byte{}, nil
	}
	first, ok := arr[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("CSV encoding requires array of objects")
	}
	headers := make([]string, 0, len(first))
	for k := range first {
		headers = append(headers, k)
	}
	sort.Strings(headers)

	var buf strings.Builder
	w := csv.NewWriter(stringWriter{&buf})
	if err := w.Write(headers); err != nil {
		return nil, err
	}
	for _, item := range arr {
		obj, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("non-object in array")
		}
		row := make([]string, len(headers))
		for i, h := range headers {
			row[i] = scalarOrJSON(obj[h])
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return []byte(buf.String()), w.Error()
}

// FormatString returns a DataFormat from a user-supplied flag value, accepting
// the same set of aliases the CLI does.
func FormatString(s string) DataFormat {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "yaml", "yml":
		return FormatYAML
	case "toml":
		return FormatTOML
	case "csv":
		return FormatCSV
	default:
		return FormatJSON
	}
}

// PrettyJSON pretty-prints JSON-shaped bytes with the given indent width.
func PrettyJSON(data []byte, indent int) ([]byte, error) {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if indent < 0 {
		indent = 2
	}
	return json.MarshalIndent(v, "", strings.Repeat(" ", indent))
}

// AsRecords coerces a value to []map[string]interface{} when possible — used
// by RenderTable. Returns nil if the value isn't an array of objects.
func AsRecords(v interface{}) []map[string]interface{} {
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]map[string]interface{}, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]interface{}); ok {
			out = append(out, m)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ReadAndDecode reads everything from r and decodes it according to format.
func ReadAndDecode(r io.Reader, format DataFormat) (interface{}, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	return Decode(data, format)
}
