package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChecksumFilesAndVerify(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "artifact.txt")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	sums, err := ChecksumFiles([]string{path}, "sha256")
	if err != nil {
		t.Fatal(err)
	}
	if len(sums) != 1 {
		t.Fatalf("expected one checksum, got %d", len(sums))
	}
	if sums[0].Checksum != "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Fatalf("unexpected checksum: %s", sums[0].Checksum)
	}

	manifest := filepath.Join(dir, "checksums.txt")
	relLine := strings.ReplaceAll(FormatChecksums([]FileChecksum{{
		Path:     "artifact.txt",
		Checksum: sums[0].Checksum,
	}}), "\\", "/")
	if err := os.WriteFile(manifest, []byte(relLine), 0644); err != nil {
		t.Fatal(err)
	}
	if err := VerifyChecksums(manifest, "sha256"); err != nil {
		t.Fatal(err)
	}
}

func TestChecksumVerifyMismatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "artifact.txt")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(dir, "checksums.txt")
	if err := os.WriteFile(manifest, []byte("deadbeef  artifact.txt\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := VerifyChecksums(manifest, "sha256"); err == nil {
		t.Fatal("expected mismatch error")
	}
}
