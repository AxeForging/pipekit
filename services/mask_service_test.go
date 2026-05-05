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

func TestMaskValuesMultiline_PEMKey(t *testing.T) {
	input := `before
-----BEGIN PRIVATE KEY-----
abcdef1234567890
ghijkl1234567890
-----END PRIVATE KEY-----
after`
	patterns, _ := PresetPatterns([]string{"gcp"})
	var buf bytes.Buffer
	if err := MaskValuesMultiline(strings.NewReader(input), &buf, patterns, "***", 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "abcdef1234567890") {
		t.Errorf("PEM body not masked: %s", out)
	}
	if !strings.Contains(out, "before") || !strings.Contains(out, "after") {
		t.Errorf("non-secret content lost: %s", out)
	}
}

func TestMaskValues_LongLineDoesNotSilentlyTruncate(t *testing.T) {
	// A single line longer than the legacy 1MB cap. Without the buffer fix
	// scanner.Scan() would return false silently and skip the secret.
	bigPad := strings.Repeat("x", 2*1024*1024)
	input := bigPad + " token=ghp_AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" + bigPad
	patterns, _ := PresetPatterns([]string{"github"})
	var buf bytes.Buffer
	if err := MaskValues(strings.NewReader(input), &buf, patterns, "***", 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "ghp_AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA") {
		t.Error("token leaked through long-line input")
	}
}

func TestPresetPatterns(t *testing.T) {
	pats, unknown := PresetPatterns([]string{"aws", "github", "nope"})
	if len(pats) == 0 {
		t.Error("expected aws+github patterns")
	}
	if len(unknown) != 1 || unknown[0] != "nope" {
		t.Errorf("expected unknown=[nope], got %v", unknown)
	}
}
