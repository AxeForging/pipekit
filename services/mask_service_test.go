package services

import (
	"bytes"
	"strings"
	"testing"
)

func TestMaskValues_Basic(t *testing.T) {
	input := "the password is secret123 and token is tk-abc"
	var buf bytes.Buffer
	err := MaskValues(strings.NewReader(input), &buf, []string{"secret123", "tk-abc"}, "***", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if strings.Contains(output, "secret123") {
		t.Errorf("expected secret123 to be masked, got: %s", output)
	}
	if strings.Contains(output, "tk-abc") {
		t.Errorf("expected tk-abc to be masked, got: %s", output)
	}
	if !strings.Contains(output, "***") {
		t.Errorf("expected *** in output, got: %s", output)
	}
}

func TestMaskValues_Partial(t *testing.T) {
	input := "token is sk-1234567890xf"
	var buf bytes.Buffer
	err := MaskValues(strings.NewReader(input), &buf, []string{"sk-1234567890xf"}, "***", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "sk-***0xf") {
		t.Errorf("expected partial masking sk-***0xf, got: %s", output)
	}
}

func TestMaskGitHub(t *testing.T) {
	var buf bytes.Buffer
	err := MaskGitHub(&buf, []string{"mysecret", "anothertoken"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "::add-mask::mysecret") {
		t.Errorf("expected ::add-mask::mysecret, got: %s", output)
	}
	if !strings.Contains(output, "::add-mask::anothertoken") {
		t.Errorf("expected ::add-mask::anothertoken, got: %s", output)
	}
}

func TestMaskGitHub_SkipsEmpty(t *testing.T) {
	var buf bytes.Buffer
	err := MaskGitHub(&buf, []string{"value", "", "other"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines (empty should be skipped), got %d", len(lines))
	}
}

func TestMaskString(t *testing.T) {
	tests := []struct {
		input       string
		replacement string
		partial     int
		expected    string
	}{
		{"secret", "***", 0, "***"},
		{"sk-1234567890xf", "***", 3, "sk-***0xf"},
		{"ab", "***", 3, "***"}, // too short for partial
	}
	for _, tc := range tests {
		result := maskString(tc.input, tc.replacement, tc.partial)
		if result != tc.expected {
			t.Errorf("maskString(%q, %q, %d) = %q, expected %q", tc.input, tc.replacement, tc.partial, result, tc.expected)
		}
	}
}
