package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestArtifactManifest(t *testing.T) {
	dir := t.TempDir()
	dist := filepath.Join(dir, "dist")
	if err := os.MkdirAll(dist, 0755); err != nil {
		t.Fatal(err)
	}
	a := filepath.Join(dist, "app-linux-amd64")
	b := filepath.Join(dist, "app-darwin-arm64")
	if err := os.WriteFile(a, []byte("linux"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("darwin"), 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := ArtifactManifest([]string{filepath.Join(dist, "app-*")})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Path > entries[1].Path {
		t.Fatalf("entries not sorted: %#v", entries)
	}
	if entries[0].SHA256 == "" || entries[0].Size == 0 {
		t.Fatalf("missing metadata: %#v", entries[0])
	}
}

func TestAssertArtifactsNoMatch(t *testing.T) {
	if err := AssertArtifacts([]string{filepath.Join(t.TempDir(), "*.missing")}); err == nil {
		t.Fatal("expected no-match error")
	}
}
