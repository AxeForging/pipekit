package services

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// GitSHA returns the current commit SHA.
func GitSHA(short bool) (string, error) {
	args := []string{"rev-parse", "HEAD"}
	if short {
		args = []string{"rev-parse", "--short", "HEAD"}
	}
	return gitOutput(args...)
}

// GitRef returns a useful branch or tag ref name, honoring GitHub Actions env first.
func GitRef(slug bool, maxLen int) (string, error) {
	ref := os.Getenv("GITHUB_HEAD_REF")
	if ref == "" {
		ref = os.Getenv("GITHUB_REF_NAME")
	}
	if ref == "" {
		if out, err := gitOutput("branch", "--show-current"); err == nil && out != "" {
			ref = out
		}
	}
	if ref == "" {
		if out, err := gitOutput("describe", "--tags", "--exact-match"); err == nil && out != "" {
			ref = out
		}
	}
	if ref == "" {
		return "", fmt.Errorf("could not determine git ref")
	}
	if slug {
		ref = Slugify(ref, maxLen)
	}
	return ref, nil
}

// GitCurrentTag returns the tag pointing at HEAD.
func GitCurrentTag() (string, error) {
	return gitOutput("describe", "--tags", "--exact-match")
}

// GitPreviousTag returns the previous reachable tag.
func GitPreviousTag() (string, error) {
	return gitOutput("describe", "--tags", "--abbrev=0")
}

// GitIsDirty reports whether tracked or untracked files are present.
func GitIsDirty() (bool, error) {
	out, err := gitOutput("status", "--porcelain")
	if err != nil {
		return false, err
	}
	return out != "", nil
}

func gitOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}
