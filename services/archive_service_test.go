package services

import (
	"archive/tar"
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchiveRoundTripFormats(t *testing.T) {
	formats := map[string]string{
		"zip":     "bundle.zip",
		"tar":     "bundle.tar",
		"tar.gz":  "bundle.tar.gz",
		"tar.xz":  "bundle.tar.xz",
		"tar.zst": "bundle.tar.zst",
	}
	for format, name := range formats {
		t.Run(format, func(t *testing.T) {
			dir := t.TempDir()
			src := filepath.Join(dir, "src")
			if err := os.MkdirAll(filepath.Join(src, "nested"), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(src, "nested", "file.txt"), []byte("hello"), 0644); err != nil {
				t.Fatal(err)
			}

			archive := filepath.Join(dir, name)
			if err := PackArchive(archive, []string{src}, format); err != nil {
				t.Fatalf("pack: %v", err)
			}
			entries, err := ListArchive(archive, "")
			if err != nil {
				t.Fatalf("list: %v", err)
			}
			if !archiveHasEntry(entries, "src/nested/file.txt") {
				t.Fatalf("entries missing file: %#v", entries)
			}

			dest := filepath.Join(dir, "out")
			if err := UnpackArchive(archive, dest, "", 1); err != nil {
				t.Fatalf("unpack: %v", err)
			}
			got, err := os.ReadFile(filepath.Join(dest, "nested", "file.txt"))
			if err != nil {
				t.Fatalf("read unpacked: %v", err)
			}
			if string(got) != "hello" {
				t.Fatalf("content = %q", got)
			}
		})
	}
}

func TestArchiveRejectsZipTraversal(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "bad.zip")
	out, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(out)
	w, err := zw.Create("../escape.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("bad"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}

	err = UnpackArchive(archive, filepath.Join(dir, "out"), "", 0)
	if err == nil || !strings.Contains(err.Error(), "escapes destination") {
		t.Fatalf("expected traversal error, got %v", err)
	}
}

func TestArchiveRejectsTarTraversal(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "bad.tar")
	out, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	tw := tar.NewWriter(out)
	if err := tw.WriteHeader(&tar.Header{Name: "../escape.txt", Mode: 0644, Size: 3}); err != nil {
		t.Fatal(err)
	}
	_, _ = tw.Write([]byte("bad"))
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}

	err = UnpackArchive(archive, filepath.Join(dir, "out"), "", 0)
	if err == nil || !strings.Contains(err.Error(), "escapes destination") {
		t.Fatalf("expected traversal error, got %v", err)
	}
}

func archiveHasEntry(entries []ArchiveEntry, name string) bool {
	for _, entry := range entries {
		if entry.Name == name {
			return true
		}
	}
	return false
}
