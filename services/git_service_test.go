package services

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitMetadata(t *testing.T) {
	dir := initGitRepo(t)
	oldWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	sha, err := GitSHA(true)
	if err != nil {
		t.Fatal(err)
	}
	if len(sha) < 7 {
		t.Fatalf("short sha too short: %q", sha)
	}

	ref, err := GitRef(true, 63)
	if err != nil {
		t.Fatal(err)
	}
	if ref == "" {
		t.Fatal("empty ref")
	}

	dirty, err := GitIsDirty()
	if err != nil {
		t.Fatal(err)
	}
	if dirty {
		t.Fatal("new repo should be clean")
	}

	if err := os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	dirty, err = GitIsDirty()
	if err != nil {
		t.Fatal(err)
	}
	if !dirty {
		t.Fatal("expected dirty repo")
	}
}

func TestGitRefUsesGitHubEnv(t *testing.T) {
	t.Setenv("GITHUB_HEAD_REF", "Feature/My_Branch")
	ref, err := GitRef(true, 20)
	if err != nil {
		t.Fatal(err)
	}
	if ref != "feature-my-branch" {
		t.Fatalf("unexpected ref slug: %q", ref)
	}
}

func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mustGit(t, dir, "init", "-q")
	mustGit(t, dir, "config", "user.email", "test@example.com")
	mustGit(t, dir, "config", "user.name", "test")
	mustGit(t, dir, "config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hi"), 0644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, dir, "add", ".")
	mustGit(t, dir, "commit", "-q", "-m", "feat: initial")
	mustGit(t, dir, "tag", "v0.1.0")
	return dir
}

func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t",
		"GIT_AUTHOR_EMAIL=t@t",
		"GIT_COMMITTER_NAME=t",
		"GIT_COMMITTER_EMAIL=t@t",
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, stderr.String())
	}
}
