package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCacheKeyFromFiles(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "go.sum")
	os.WriteFile(f, []byte("some lockfile content"), 0644)

	key, err := CacheKeyFromFiles([]string{f}, "go-mod-")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(key, "go-mod-") {
		t.Errorf("expected prefix 'go-mod-', got %s", key)
	}
	if len(key) != 7+64 { // prefix + sha256 hex
		t.Errorf("expected key length %d, got %d", 7+64, len(key))
	}
}

func TestCacheKeyFromFiles_Deterministic(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "lockfile")
	os.WriteFile(f, []byte("content"), 0644)

	key1, _ := CacheKeyFromFiles([]string{f}, "")
	key2, _ := CacheKeyFromFiles([]string{f}, "")

	if key1 != key2 {
		t.Errorf("cache keys should be deterministic: %s != %s", key1, key2)
	}
}

func TestCacheKeyFromGlob(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "a.lock"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "b.lock"), []byte("b"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "c.txt"), []byte("c"), 0644)

	key, err := CacheKeyFromGlob(filepath.Join(tmpDir, "*.lock"), "cache-")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(key, "cache-") {
		t.Errorf("expected prefix 'cache-', got %s", key)
	}
}

func TestCacheKeyFromGlob_NoMatch(t *testing.T) {
	_, err := CacheKeyFromGlob("/nonexistent/*.xyz", "")
	if err == nil {
		t.Error("expected error for no matching files")
	}
}

func TestCacheKeyComposite(t *testing.T) {
	key := CacheKeyComposite([]string{"linux", "amd64", "abc123"}, "go-", "-")
	if key != "go-linux-amd64-abc123" {
		t.Errorf("expected go-linux-amd64-abc123, got %s", key)
	}
}

func TestCacheKeyComposite_DefaultSeparator(t *testing.T) {
	key := CacheKeyComposite([]string{"a", "b", "c"}, "", "")
	if key != "a-b-c" {
		t.Errorf("expected a-b-c, got %s", key)
	}
}
