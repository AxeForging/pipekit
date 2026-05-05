package services

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/AxeForging/pipekit/domain"
	"github.com/itchyny/gojq"
	"gopkg.in/yaml.v3"
)

// ParseJSON reads JSON from r and returns flattened key-value pairs.
func ParseJSON(r io.Reader, flatten bool, depth int, filter string) ([]domain.KeyValue, error) {
	var raw interface{}
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	if filter != "" {
		filtered, err := applyJQFilter(raw, filter)
		if err != nil {
			return nil, err
		}
		raw = filtered
	}

	m, ok := raw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expected JSON object at top level")
	}

	if flatten || depth > 0 {
		maxDepth := -1
		if depth > 0 {
			maxDepth = depth
		}
		m = flattenMap("", m, maxDepth)
	}

	return mapToKVs(m), nil
}

// ParseYAML reads YAML from r and returns flattened key-value pairs.
func ParseYAML(r io.Reader, flatten bool, depth int, filter string) ([]domain.KeyValue, error) {
	var raw interface{}
	if err := yaml.NewDecoder(r).Decode(&raw); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	// yaml.v3 decodes map keys as string already in most cases,
	// but nested maps come as map[string]interface{}
	m := normalizeYAMLMap(raw)
	if m == nil {
		return nil, fmt.Errorf("expected YAML mapping at top level")
	}

	var asInterface interface{} = m
	if filter != "" {
		filtered, err := applyJQFilter(asInterface, filter)
		if err != nil {
			return nil, err
		}
		if fm, ok := filtered.(map[string]interface{}); ok {
			m = fm
		} else {
			return nil, fmt.Errorf("filter result is not a mapping")
		}
	}

	if flatten || depth > 0 {
		maxDepth := -1
		if depth > 0 {
			maxDepth = depth
		}
		m = flattenMap("", m, maxDepth)
	}

	return mapToKVs(m), nil
}

// ParseTOML reads TOML from r and returns flattened key-value pairs.
// Same flatten/depth/filter semantics as ParseJSON / ParseYAML.
func ParseTOML(r io.Reader, flatten bool, depth int, filter string) ([]domain.KeyValue, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading TOML: %w", err)
	}
	v, err := Decode(data, FormatTOML)
	if err != nil {
		return nil, err
	}

	if filter != "" {
		filtered, err := applyJQFilter(v, filter)
		if err != nil {
			return nil, err
		}
		v = filtered
	}

	m, ok := v.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expected TOML mapping at top level")
	}
	if flatten || depth > 0 {
		maxDepth := -1
		if depth > 0 {
			maxDepth = depth
		}
		m = flattenMap("", m, maxDepth)
	}
	return mapToKVs(m), nil
}

// ParseDotenv reads a .env file and returns key-value pairs.
func ParseDotenv(r io.Reader) ([]domain.KeyValue, error) {
	var kvs []domain.KeyValue
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Remove export prefix if present
		line = strings.TrimPrefix(line, "export ")
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		// Strip surrounding quotes
		value = stripQuotes(value)
		kvs = append(kvs, domain.KeyValue{Key: key, Value: value})
	}
	return kvs, scanner.Err()
}

// TransformKeys applies prefix, uppercase, and strip-quotes transformations.
func TransformKeys(kvs []domain.KeyValue, uppercase bool, prefix string, doStripQuotes bool) []domain.KeyValue {
	result := make([]domain.KeyValue, len(kvs))
	for i, kv := range kvs {
		key := kv.Key
		if uppercase {
			key = toUpperSnakeCase(key)
		}
		if prefix != "" {
			key = prefix + key
		}
		value := kv.Value
		if doStripQuotes {
			value = stripQuotes(value)
		}
		result[i] = domain.KeyValue{Key: key, Value: value}
	}
	return result
}

// WriteToShell writes export statements to w.
func WriteToShell(w io.Writer, kvs []domain.KeyValue) error {
	for _, kv := range kvs {
		if _, err := fmt.Fprintf(w, "export %s=%q\n", kv.Key, kv.Value); err != nil {
			return err
		}
	}
	return nil
}

// WriteToGitHubEnv appends key=value pairs to the GITHUB_ENV file using multiline syntax.
func WriteToGitHubEnv(kvs []domain.KeyValue) error {
	return writeToGitHubFile("GITHUB_ENV", kvs)
}

// WriteToGitHubOutput appends key=value pairs to the GITHUB_OUTPUT file using multiline syntax.
func WriteToGitHubOutput(kvs []domain.KeyValue) error {
	return writeToGitHubFile("GITHUB_OUTPUT", kvs)
}

// WriteToGitLab writes export statements suitable for GitLab CI.
func WriteToGitLab(w io.Writer, kvs []domain.KeyValue) error {
	for _, kv := range kvs {
		if _, err := fmt.Fprintf(w, "export %s=%q\n", kv.Key, kv.Value); err != nil {
			return err
		}
	}
	return nil
}

func writeToGitHubFile(envVar string, kvs []domain.KeyValue) error {
	path := os.Getenv(envVar)
	if path == "" {
		return fmt.Errorf("$%s is not set", envVar)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	for _, kv := range kvs {
		if strings.Contains(kv.Value, "\n") {
			delimiter, err := uniqueHeredocDelimiter(kv.Value)
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintf(f, "%s<<%s\n%s\n%s\n", kv.Key, delimiter, kv.Value, delimiter); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintf(f, "%s=%s\n", kv.Key, kv.Value); err != nil {
				return err
			}
		}
	}
	return nil
}

// uniqueHeredocDelimiter returns a delimiter guaranteed not to appear in value.
// We use a random suffix; if it collides (astronomically unlikely), we retry.
func uniqueHeredocDelimiter(value string) (string, error) {
	for i := 0; i < 8; i++ {
		buf := make([]byte, 12)
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("generating delimiter: %w", err)
		}
		d := "PIPEKIT_EOF_" + hex.EncodeToString(buf)
		if !strings.Contains(value, d) {
			return d, nil
		}
	}
	return "", fmt.Errorf("could not generate non-colliding heredoc delimiter")
}

func applyJQFilter(data interface{}, filter string) (interface{}, error) {
	query, err := gojq.Parse(filter)
	if err != nil {
		return nil, fmt.Errorf("parsing filter %q: %w", filter, err)
	}
	iter := query.Run(data)
	v, ok := iter.Next()
	if !ok {
		return nil, fmt.Errorf("filter %q produced no results", filter)
	}
	if err, ok := v.(error); ok {
		return nil, fmt.Errorf("filter error: %w", err)
	}
	return v, nil
}

func flattenMap(prefix string, m map[string]interface{}, maxDepth int) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "_" + k
		}
		if sub, ok := v.(map[string]interface{}); ok && maxDepth != 0 {
			for sk, sv := range flattenMap(key, sub, maxDepth-1) {
				result[sk] = sv
			}
		} else {
			result[key] = v
		}
	}
	return result
}

func mapToKVs(m map[string]interface{}) []domain.KeyValue {
	var kvs []domain.KeyValue
	for k, v := range m {
		kvs = append(kvs, domain.KeyValue{Key: k, Value: scalarOrJSON(v)})
	}
	sort.Slice(kvs, func(i, j int) bool { return kvs[i].Key < kvs[j].Key })
	return kvs
}

// scalarOrJSON renders scalars (string, bool, numbers, nil) as their natural
// string form and any compound type ([]any, map[string]any) as compact JSON.
// This avoids leaking Go's default rendering — e.g. "[a b c]" or "map[k:v]"
// — into env-var values.
func scalarOrJSON(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64, float32, int, int32, int64, uint, uint32, uint64:
		return fmt.Sprintf("%v", t)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	}
}

var nonAlphaNum = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func toUpperSnakeCase(s string) string {
	// Replace dots and dashes with underscores
	s = strings.ReplaceAll(s, ".", "_")
	s = strings.ReplaceAll(s, "-", "_")
	s = nonAlphaNum.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	return strings.ToUpper(s)
}

func stripQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func normalizeYAMLMap(v interface{}) map[string]interface{} {
	switch m := v.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, val := range m {
			if sub := normalizeYAMLMap(val); sub != nil {
				result[k] = sub
			} else {
				result[k] = val
			}
		}
		return result
	case map[interface{}]interface{}:
		result := make(map[string]interface{})
		for k, val := range m {
			key := fmt.Sprintf("%v", k)
			if sub := normalizeYAMLMap(val); sub != nil {
				result[key] = sub
			} else {
				result[key] = val
			}
		}
		return result
	default:
		return nil
	}
}
