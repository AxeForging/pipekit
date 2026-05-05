package services

import "testing"

func TestMatchGlob_DoubleStar(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		match   bool
	}{
		{"api/**", "api/foo.go", true},
		{"api/**", "api/foo/bar.go", true},
		// doublestar treats `api/**` as zero-or-more segments under api,
		// so the bare directory entry also matches. Keep this documented.
		{"api/**", "api", true},
		{"api/**", "web/foo.go", false},
		{"**/*.go", "a/b/c/d.go", true},
		{"**/*.go", "main.go", true},
		{"**/*.go", "main.py", false},
		{"shared/**/*.proto", "shared/v1/foo.proto", true},
		{"*.go", "main.go", true},
		{"*.go", "sub/main.go", false},
	}
	for _, tc := range tests {
		got, err := matchGlob(tc.pattern, tc.path)
		if err != nil {
			t.Fatalf("matchGlob(%q,%q) error: %v", tc.pattern, tc.path, err)
		}
		if got != tc.match {
			t.Errorf("matchGlob(%q,%q) = %v, want %v", tc.pattern, tc.path, got, tc.match)
		}
	}
}
