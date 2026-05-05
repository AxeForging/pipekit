package services

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/AxeForging/pipekit/domain"
)

func TestParseJSON_Flat(t *testing.T) {
	input := `{"name": "pipekit", "version": "1.0.0"}`
	kvs, err := ParseJSON(strings.NewReader(input), false, 0, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(kvs) != 2 {
		t.Fatalf("expected 2 key-value pairs, got %d", len(kvs))
	}
	// Sorted by key
	if kvs[0].Key != "name" || kvs[0].Value != "pipekit" {
		t.Errorf("expected name=pipekit, got %s=%s", kvs[0].Key, kvs[0].Value)
	}
	if kvs[1].Key != "version" || kvs[1].Value != "1.0.0" {
		t.Errorf("expected version=1.0.0, got %s=%s", kvs[1].Key, kvs[1].Value)
	}
}

func TestParseJSON_Flatten(t *testing.T) {
	input := `{"db": {"host": "localhost", "port": 5432}}`
	kvs, err := ParseJSON(strings.NewReader(input), true, 0, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(kvs) != 2 {
		t.Fatalf("expected 2 key-value pairs, got %d", len(kvs))
	}
	found := make(map[string]string)
	for _, kv := range kvs {
		found[kv.Key] = kv.Value
	}
	if found["db_host"] != "localhost" {
		t.Errorf("expected db_host=localhost, got %s", found["db_host"])
	}
	if found["db_port"] != "5432" {
		t.Errorf("expected db_port=5432, got %s", found["db_port"])
	}
}

func TestParseJSON_Filter(t *testing.T) {
	input := `{"name": "pipekit", "version": "1.0.0", "private": true}`
	kvs, err := ParseJSON(strings.NewReader(input), false, 0, "{name, version}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(kvs) != 2 {
		t.Fatalf("expected 2 key-value pairs, got %d", len(kvs))
	}
}

func TestParseYAML(t *testing.T) {
	input := "name: pipekit\nversion: 1.0.0\n"
	kvs, err := ParseYAML(strings.NewReader(input), false, 0, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(kvs) != 2 {
		t.Fatalf("expected 2 key-value pairs, got %d", len(kvs))
	}
}

func TestParseDotenv(t *testing.T) {
	input := "# comment\nFOO=bar\nexport BAZ=\"quoted\"\nEMPTY=\n"
	kvs, err := ParseDotenv(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(kvs) != 3 {
		t.Fatalf("expected 3 key-value pairs, got %d", len(kvs))
	}
	if kvs[0].Key != "FOO" || kvs[0].Value != "bar" {
		t.Errorf("expected FOO=bar, got %s=%s", kvs[0].Key, kvs[0].Value)
	}
	if kvs[1].Key != "BAZ" || kvs[1].Value != "quoted" {
		t.Errorf("expected BAZ=quoted, got %s=%s", kvs[1].Key, kvs[1].Value)
	}
}

func TestTransformKeys(t *testing.T) {
	kvs := []domain.KeyValue{
		{Key: "my.key", Value: "val1"},
		{Key: "another-key", Value: "val2"},
	}

	result := TransformKeys(kvs, true, "APP_", false)
	if result[0].Key != "APP_MY_KEY" {
		t.Errorf("expected APP_MY_KEY, got %s", result[0].Key)
	}
	if result[1].Key != "APP_ANOTHER_KEY" {
		t.Errorf("expected APP_ANOTHER_KEY, got %s", result[1].Key)
	}
}

func TestWriteToShell(t *testing.T) {
	kvs := []domain.KeyValue{
		{Key: "FOO", Value: "bar"},
		{Key: "BAZ", Value: "hello world"},
	}

	var buf bytes.Buffer
	if err := WriteToShell(&buf, kvs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `export FOO="bar"`) {
		t.Errorf("expected export FOO=\"bar\", got %s", output)
	}
	if !strings.Contains(output, `export BAZ="hello world"`) {
		t.Errorf("expected export BAZ=\"hello world\", got %s", output)
	}
}

func TestToUpperSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"db.host", "DB_HOST"},
		{"my-key", "MY_KEY"},
		{"camelCase", "CAMELCASE"},
		{"simple", "SIMPLE"},
	}
	for _, tc := range tests {
		result := toUpperSnakeCase(tc.input)
		if result != tc.expected {
			t.Errorf("toUpperSnakeCase(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

// Regression: arrays/objects should be JSON-encoded, not rendered with
// Go's default %v formatting (which would leak "[a b c]" or "map[k:v]").
func TestParseJSON_NonScalarValuesAreJSONEncoded(t *testing.T) {
	input := `{"tags": ["a", "b", "c"], "meta": {"k": 1}, "scalar": "x", "n": 42, "b": true}`
	kvs, err := ParseJSON(strings.NewReader(input), false, 0, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := map[string]string{}
	for _, kv := range kvs {
		got[kv.Key] = kv.Value
	}
	if got["tags"] != `["a","b","c"]` {
		t.Errorf("tags: expected JSON array, got %q", got["tags"])
	}
	if got["meta"] != `{"k":1}` {
		t.Errorf("meta: expected JSON object, got %q", got["meta"])
	}
	if got["scalar"] != "x" {
		t.Errorf("scalar: expected raw, got %q", got["scalar"])
	}
	if got["n"] != "42" {
		t.Errorf("n: expected 42, got %q", got["n"])
	}
	if got["b"] != "true" {
		t.Errorf("b: expected true, got %q", got["b"])
	}
}

// Regression: GitHub heredoc delimiter must not collide with the value.
// Hardcoded "EOF_PIPEKIT" used to corrupt output for any value containing
// that literal.
func TestUniqueHeredocDelimiter_NoCollision(t *testing.T) {
	// Even if the value pretends to be the legacy delimiter, a unique one
	// must come back.
	d, err := uniqueHeredocDelimiter("PIPEKIT_EOF_BADGUESS\nEOF_PIPEKIT\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains("PIPEKIT_EOF_BADGUESS\nEOF_PIPEKIT\n", d) {
		t.Fatalf("delimiter %q collides with value", d)
	}
}

func TestWriteToGitHubEnv_UniqueDelimiterPerCall(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := tmpDir + "/env"
	t.Setenv("GITHUB_ENV", envFile)

	kvs := []domain.KeyValue{{Key: "MULTI", Value: "line1\nline2\nline3"}}
	if err := WriteToGitHubEnv(kvs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out, _ := readFile(envFile)
	if !strings.Contains(out, "MULTI<<PIPEKIT_EOF_") {
		t.Errorf("expected unique heredoc delimiter, got: %s", out)
	}
}

func readFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
