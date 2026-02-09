package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAssertEnvExists_Success(t *testing.T) {
	os.Setenv("PIPEKIT_TEST_VAR", "value")
	defer os.Unsetenv("PIPEKIT_TEST_VAR")

	if err := AssertEnvExists([]string{"PIPEKIT_TEST_VAR"}); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestAssertEnvExists_Missing(t *testing.T) {
	os.Unsetenv("PIPEKIT_NONEXISTENT")
	if err := AssertEnvExists([]string{"PIPEKIT_NONEXISTENT"}); err == nil {
		t.Error("expected error for missing env var")
	}
}

func TestAssertFileExists_Success(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(f, []byte("test"), 0644)

	if err := AssertFileExists([]string{f}); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestAssertFileExists_Missing(t *testing.T) {
	if err := AssertFileExists([]string{"/nonexistent/file.txt"}); err == nil {
		t.Error("expected error for missing file")
	}
}

func TestAssertJSONPath_Success(t *testing.T) {
	data := []byte(`{"name": "pipekit", "version": "1.0.0"}`)
	if err := AssertJSONPath(data, ".name", "pipekit"); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestAssertJSONPath_Mismatch(t *testing.T) {
	data := []byte(`{"name": "pipekit"}`)
	if err := AssertJSONPath(data, ".name", "wrong"); err == nil {
		t.Error("expected error for value mismatch")
	}
}

func TestAssertSemver_Valid(t *testing.T) {
	if err := AssertSemver("1.2.3"); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if err := AssertSemver("1.0.0-alpha.1"); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestAssertSemver_Invalid(t *testing.T) {
	if err := AssertSemver("not-a-version"); err == nil {
		t.Error("expected error for invalid semver")
	}
}

func TestAssertSemverCompare(t *testing.T) {
	tests := []struct {
		v1       string
		op       string
		v2       string
		wantErr  bool
	}{
		{"2.0.0", "gt", "1.0.0", false},
		{"1.0.0", "lt", "2.0.0", false},
		{"1.0.0", "eq", "1.0.0", false},
		{"1.0.0", "gt", "2.0.0", true},
		{"2.0.0", "lt", "1.0.0", true},
		{"1.0.0", "eq", "2.0.0", true},
		{"1.0.0", "gte", "1.0.0", false},
		{"2.0.0", "gte", "1.0.0", false},
		{"0.9.0", "gte", "1.0.0", true},
		{"1.0.0", "lte", "1.0.0", false},
		{"0.9.0", "lte", "1.0.0", false},
		{"2.0.0", "lte", "1.0.0", true},
	}
	for _, tc := range tests {
		err := AssertSemverCompare(tc.v1, tc.op, tc.v2)
		if tc.wantErr && err == nil {
			t.Errorf("AssertSemverCompare(%s, %s, %s) expected error", tc.v1, tc.op, tc.v2)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("AssertSemverCompare(%s, %s, %s) unexpected error: %v", tc.v1, tc.op, tc.v2, err)
		}
	}
}
