package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateChangelogConventional(t *testing.T) {
	dir := initGitRepo(t)
	oldWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	if err := os.WriteFile(filepath.Join(dir, "fix.txt"), []byte("fix"), 0644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, dir, "add", ".")
	mustGit(t, dir, "commit", "-q", "-m", "fix: correct release path")

	markdown, entries, err := GenerateChangelog(ChangelogOptions{
		From:         "v0.1.0",
		To:           "HEAD",
		Conventional: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one entry, got %d", len(entries))
	}
	if !strings.Contains(markdown, "### Fixes") || !strings.Contains(markdown, "fix: correct release path") {
		t.Fatalf("unexpected changelog:\n%s", markdown)
	}
}
