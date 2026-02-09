package services

import (
	"strings"
	"testing"
)

func TestBase64EncodeDecode(t *testing.T) {
	original := "hello world"
	encoded := Base64Encode([]byte(original))
	if encoded != "aGVsbG8gd29ybGQ=" {
		t.Errorf("expected aGVsbG8gd29ybGQ=, got %s", encoded)
	}
	decoded, err := Base64Decode(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(decoded) != original {
		t.Errorf("expected %q, got %q", original, string(decoded))
	}
}

func TestURLEncodeDecode(t *testing.T) {
	original := "hello world&foo=bar"
	encoded := URLEncode(original)
	if !strings.Contains(encoded, "%26") {
		t.Errorf("expected URL-encoded &, got %s", encoded)
	}
	decoded, err := URLDecode(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decoded != original {
		t.Errorf("expected %q, got %q", original, decoded)
	}
}

func TestConvertCase(t *testing.T) {
	tests := []struct {
		input    string
		toCase   string
		expected string
	}{
		{"myServiceName", "snake", "my_service_name"},
		{"myServiceName", "upper-snake", "MY_SERVICE_NAME"},
		{"myServiceName", "kebab", "my-service-name"},
		{"myServiceName", "pascal", "MyServiceName"},
		{"my_service_name", "camel", "myServiceName"},
		{"hello", "upper", "HELLO"},
		{"HELLO", "lower", "hello"},
	}
	for _, tc := range tests {
		result := ConvertCase(tc.input, tc.toCase)
		if result != tc.expected {
			t.Errorf("ConvertCase(%q, %q) = %q, expected %q", tc.input, tc.toCase, result, tc.expected)
		}
	}
}

func TestRegexReplace(t *testing.T) {
	result, err := RegexReplace("foo-123-bar", `\d+`, "***")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "foo-***-bar" {
		t.Errorf("expected foo-***-bar, got %s", result)
	}
}

func TestRegexReplace_Invalid(t *testing.T) {
	_, err := RegexReplace("test", "[invalid", "x")
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}

func TestRenderTemplate(t *testing.T) {
	tmpl := "Hello {{.Name}}, you have {{.Count}} items"
	data := map[string]interface{}{
		"Name":  "World",
		"Count": 42,
	}
	result, err := RenderTemplate(tmpl, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Hello World, you have 42 items" {
		t.Errorf("expected 'Hello World, you have 42 items', got %q", result)
	}
}

func TestHashData(t *testing.T) {
	tests := []struct {
		input     string
		algorithm string
		expected  string
	}{
		{"hello", "sha256", "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"},
		{"hello", "md5", "5d41402abc4b2a76b9719d911017c592"},
		{"hello", "sha1", "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d"},
	}
	for _, tc := range tests {
		result, err := HashData(strings.NewReader(tc.input), tc.algorithm)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != tc.expected {
			t.Errorf("HashData(%q, %q) = %q, expected %q", tc.input, tc.algorithm, result, tc.expected)
		}
	}
}

func TestHashData_Unsupported(t *testing.T) {
	_, err := HashData(strings.NewReader("test"), "blake2b")
	if err == nil {
		t.Error("expected error for unsupported algorithm")
	}
}

func TestSplitWords(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"camelCase", []string{"camel", "Case"}},
		{"snake_case", []string{"snake", "case"}},
		{"kebab-case", []string{"kebab", "case"}},
		{"PascalCase", []string{"Pascal", "Case"}},
		{"simple", []string{"simple"}},
	}
	for _, tc := range tests {
		result := splitWords(tc.input)
		if len(result) != len(tc.expected) {
			t.Errorf("splitWords(%q) = %v, expected %v", tc.input, result, tc.expected)
			continue
		}
		for i := range result {
			if result[i] != tc.expected[i] {
				t.Errorf("splitWords(%q)[%d] = %q, expected %q", tc.input, i, result[i], tc.expected[i])
			}
		}
	}
}
