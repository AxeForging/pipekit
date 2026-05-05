package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVersionGet_PackageJSON(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "package.json")
	os.WriteFile(f, []byte(`{"name": "myapp", "version": "1.2.3"}`), 0644)

	version, err := VersionGet(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version != "1.2.3" {
		t.Errorf("expected 1.2.3, got %s", version)
	}
}

func TestVersionGet_CargoToml(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "Cargo.toml")
	os.WriteFile(f, []byte("[package]\nname = \"myapp\"\nversion = \"0.5.1\"\n"), 0644)

	version, err := VersionGet(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version != "0.5.1" {
		t.Errorf("expected 0.5.1, got %s", version)
	}
}

func TestVersionGet_VersionFile(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "VERSION")
	os.WriteFile(f, []byte("3.0.0\n"), 0644)

	version, err := VersionGet(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version != "3.0.0" {
		t.Errorf("expected 3.0.0, got %s", version)
	}
}

func TestVersionBump_Patch(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "package.json")
	os.WriteFile(f, []byte(`{"name": "myapp", "version": "1.2.3"}`), 0644)

	newVersion, err := VersionBump(f, "patch", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newVersion != "1.2.4" {
		t.Errorf("expected 1.2.4, got %s", newVersion)
	}

	// Verify file was updated
	data, _ := os.ReadFile(f)
	if !contains(string(data), "1.2.4") {
		t.Error("expected file to contain updated version")
	}
}

func TestVersionBump_Minor(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "package.json")
	os.WriteFile(f, []byte(`{"name": "myapp", "version": "1.2.3"}`), 0644)

	newVersion, err := VersionBump(f, "minor", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newVersion != "1.3.0" {
		t.Errorf("expected 1.3.0, got %s", newVersion)
	}
}

func TestVersionBump_Major(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "package.json")
	os.WriteFile(f, []byte(`{"name": "myapp", "version": "1.2.3"}`), 0644)

	newVersion, err := VersionBump(f, "major", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newVersion != "2.0.0" {
		t.Errorf("expected 2.0.0, got %s", newVersion)
	}
}

func TestVersionBump_WithPreRelease(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "package.json")
	os.WriteFile(f, []byte(`{"name": "myapp", "version": "1.2.3"}`), 0644)

	newVersion, err := VersionBump(f, "patch", "alpha.1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newVersion != "1.2.4-alpha.1" {
		t.Errorf("expected 1.2.4-alpha.1, got %s", newVersion)
	}
}

func TestVersionCompare(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected int
	}{
		{"1.0.0", "1.0.0", 0},
		{"2.0.0", "1.0.0", 1},
		{"1.0.0", "2.0.0", -1},
		{"1.1.0", "1.0.0", 1},
		{"1.0.1", "1.0.0", 1},
	}
	for _, tc := range tests {
		result, err := VersionCompare(tc.v1, tc.v2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != tc.expected {
			t.Errorf("VersionCompare(%s, %s) = %d, expected %d", tc.v1, tc.v2, result, tc.expected)
		}
	}
}

func TestFormatVersion(t *testing.T) {
	tests := []struct {
		version  string
		format   string
		expected string
	}{
		{"1.0.0", "plain", "1.0.0"},
		{"1.0.0", "v-prefixed", "v1.0.0"},
		{"v1.0.0", "v-prefixed", "v1.0.0"},
	}
	for _, tc := range tests {
		result := FormatVersion(tc.version, tc.format)
		if result != tc.expected {
			t.Errorf("FormatVersion(%q, %q) = %q, expected %q", tc.version, tc.format, result, tc.expected)
		}
	}
}

// Regression: ensure VersionBump rewrites only the package's own version,
// not a dep that happens to pin the same literal earlier in the file.
func TestVersionBump_DoesNotRewriteDependencyPin(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "package.json")
	original := `{
  "name": "myapp",
  "dependencies": { "react": "1.2.3" },
  "version": "1.2.3"
}`
	os.WriteFile(f, []byte(original), 0644)

	newVersion, err := VersionBump(f, "patch", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newVersion != "1.2.4" {
		t.Errorf("expected 1.2.4, got %s", newVersion)
	}
	out, _ := os.ReadFile(f)
	got := string(out)
	if !contains(got, `"react": "1.2.3"`) {
		t.Errorf("dep pin was rewritten: %s", got)
	}
	if !contains(got, `"version": "1.2.4"`) {
		t.Errorf("version not bumped: %s", got)
	}
}

func TestVersionSet(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "Cargo.toml")
	os.WriteFile(f, []byte("[package]\nname = \"myapp\"\nversion = \"0.5.1\"\n"), 0644)

	if err := VersionSet(f, "1.0.0"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out, _ := os.ReadFile(f)
	if !contains(string(out), `version = "1.0.0"`) {
		t.Errorf("version not set: %s", string(out))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
