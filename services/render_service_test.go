package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderTemplateString_Basics(t *testing.T) {
	tests := []struct {
		name     string
		tmpl     string
		values   map[string]interface{}
		envKey   string
		envVal   string
		expected string
	}{
		{
			name:     "values lookup",
			tmpl:     `Hello {{ .Values.name }}`,
			values:   map[string]interface{}{"name": "world"},
			expected: "Hello world",
		},
		{
			name:     "default with empty",
			tmpl:     `Tag: {{ .Values.tag | default "latest" }}`,
			values:   map[string]interface{}{"tag": ""},
			expected: "Tag: latest",
		},
		{
			name:     "default skipped when set",
			tmpl:     `Tag: {{ .Values.tag | default "latest" }}`,
			values:   map[string]interface{}{"tag": "v1"},
			expected: "Tag: v1",
		},
		{
			name:     "env",
			tmpl:     `User: {{ env "RENDER_TEST_USER" }}`,
			envKey:   "RENDER_TEST_USER",
			envVal:   "alice",
			expected: "User: alice",
		},
		{
			name:     "envOr fallback",
			tmpl:     `Z: {{ envOr "fallback" "RENDER_TEST_NOPE" }}`,
			expected: "Z: fallback",
		},
		{
			name:     "b64enc/dec",
			tmpl:     `{{ b64enc "hello" }}-{{ b64dec "aGVsbG8=" }}`,
			expected: "aGVsbG8=-hello",
		},
		{
			name:     "indent / nindent",
			tmpl:     `{{ "a\nb" | indent 2 }}`,
			expected: "  a\n  b",
		},
		{
			name:     "ternary",
			tmpl:     `{{ ternary "yes" "no" true }}`,
			expected: "yes",
		},
		{
			name:     "regexReplace",
			tmpl:     `{{ regexReplace "[0-9]+" "X" "abc123def" }}`,
			expected: "abcXdef",
		},
		{
			name:     "toJson",
			tmpl:     `{{ .Values.list | toJson }}`,
			values:   map[string]interface{}{"list": []interface{}{"a", "b"}},
			expected: `["a","b"]`,
		},
		{
			name:     "sha256sum",
			tmpl:     `{{ sha256sum "hello" }}`,
			expected: "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envKey != "" {
				t.Setenv(tc.envKey, tc.envVal)
			}
			got, err := RenderTemplateString(tc.tmpl, tc.values)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if got != tc.expected {
				t.Errorf("got %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestRenderTemplateFile(t *testing.T) {
	dir := t.TempDir()
	tmpl := filepath.Join(dir, "values.yaml.tpl")
	os.WriteFile(tmpl, []byte("image:\n  tag: {{ .Values.tag }}\n"), 0644)

	got, err := RenderTemplateFile(tmpl, map[string]interface{}{"tag": "v1.2.3"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !strings.Contains(got, "tag: v1.2.3") {
		t.Errorf("got: %s", got)
	}
}

func TestApplySetOverrides(t *testing.T) {
	values := map[string]interface{}{
		"image": map[string]interface{}{"tag": "v1.0.0"},
	}
	if err := ApplySetOverrides(values, []string{"image.tag=v2.0.0", "replicas=3"}); err != nil {
		t.Fatalf("error: %v", err)
	}
	if values["image"].(map[string]interface{})["tag"] != "v2.0.0" {
		t.Errorf("tag not overridden: %v", values)
	}
	if values["replicas"] != "3" {
		t.Errorf("replicas not added: %v", values)
	}
}

func TestLoadValues(t *testing.T) {
	dir := t.TempDir()
	yamlFile := filepath.Join(dir, "v.yaml")
	os.WriteFile(yamlFile, []byte("image:\n  tag: v1.0.0\n"), 0644)
	got, err := LoadValues(yamlFile)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if got["image"].(map[string]interface{})["tag"] != "v1.0.0" {
		t.Errorf("got: %v", got)
	}
}
