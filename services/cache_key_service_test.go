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

func TestCacheKeyFromFilesWithOpts_Length(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "lock")
	os.WriteFile(f, []byte("contents"), 0644)

	full, err := CacheKeyFromFilesWithOpts([]string{f}, CacheKeyOptions{})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(full) != 64 {
		t.Errorf("expected 64-char hex, got %d (%s)", len(full), full)
	}

	short, err := CacheKeyFromFilesWithOpts([]string{f}, CacheKeyOptions{Length: 16})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(short) != 16 {
		t.Errorf("expected 16-char truncated, got %d (%s)", len(short), short)
	}
	if full[:16] != short {
		t.Errorf("truncated key should be prefix of full: %s vs %s", short, full)
	}
}

func TestCacheKeyFromFilesWithOpts_WithEnvChangesHash(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "lock")
	os.WriteFile(f, []byte("same content"), 0644)

	t.Setenv("PIPEKIT_TEST_VAR", "v1")
	a, err := CacheKeyFromFilesWithOpts([]string{f}, CacheKeyOptions{
		EnvVar: []string{"PIPEKIT_TEST_VAR"},
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	t.Setenv("PIPEKIT_TEST_VAR", "v2")
	b, err := CacheKeyFromFilesWithOpts([]string{f}, CacheKeyOptions{
		EnvVar: []string{"PIPEKIT_TEST_VAR"},
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if a == b {
		t.Errorf("hash did not change when env var changed: %s", a)
	}

	// Same env, same hash.
	t.Setenv("PIPEKIT_TEST_VAR", "v2")
	bAgain, _ := CacheKeyFromFilesWithOpts([]string{f}, CacheKeyOptions{
		EnvVar: []string{"PIPEKIT_TEST_VAR"},
	})
	if b != bAgain {
		t.Errorf("hash should be stable with same env: %s vs %s", b, bAgain)
	}
}

func TestCacheKeyFromFilesWithOpts_PrefixApplied(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "lock")
	os.WriteFile(f, []byte("x"), 0644)

	got, err := CacheKeyFromFilesWithOpts([]string{f}, CacheKeyOptions{
		Prefix: "go-mod-",
		Length: 8,
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(got) != len("go-mod-")+8 {
		t.Errorf("unexpected length: %s", got)
	}
	if got[:7] != "go-mod-" {
		t.Errorf("prefix missing: %s", got)
	}
}
