package services

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/AxeForging/pipekit/domain"
	"github.com/bmatcuk/doublestar/v4"
)

// DiffFiles returns files changed between two git refs.
func DiffFiles(base, head string, includes, excludes []string) ([]string, error) {
	args := []string{"diff", "--name-only", base + "..." + head}
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		// Fallback to two-dot diff
		args = []string{"diff", "--name-only", base, head}
		out, err = exec.Command("git", args...).Output()
		if err != nil {
			return nil, fmt.Errorf("git diff failed: %w", err)
		}
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil, nil
	}

	return filterPaths(lines, includes, excludes), nil
}

// DiffDirs returns unique top-level directories with changes.
func DiffDirs(base, head string, includes, excludes []string) ([]string, error) {
	files, err := DiffFiles(base, head, includes, excludes)
	if err != nil {
		return nil, err
	}

	dirSet := make(map[string]bool)
	for _, f := range files {
		parts := strings.SplitN(f, "/", 2)
		if len(parts) > 0 {
			dirSet[parts[0]] = true
		}
	}

	var dirs []string
	for d := range dirSet {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	return dirs, nil
}

// DiffMatch checks if any changed files match the given glob patterns.
// Patterns support doublestar globs: `**` matches across path segments,
// e.g. `api/**` matches `api/foo/bar.go`.
func DiffMatch(base, head string, patterns []string) (bool, error) {
	files, err := DiffFiles(base, head, nil, nil)
	if err != nil {
		return false, err
	}

	for _, f := range files {
		for _, pattern := range patterns {
			matched, err := matchGlob(pattern, f)
			if err != nil {
				return false, fmt.Errorf("invalid pattern %q: %w", pattern, err)
			}
			if matched {
				return true, nil
			}
		}
	}
	return false, nil
}

// matchGlob returns true if path matches pattern. Supports `**` (cross-segment)
// via doublestar and falls back to filepath.Match for simple patterns.
func matchGlob(pattern, path string) (bool, error) {
	if strings.Contains(pattern, "**") {
		return doublestar.Match(pattern, path)
	}
	return filepath.Match(pattern, path)
}

// DiffAffected maps changed paths to service names via a config.
func DiffAffected(base, head string, config domain.DiffConfig) ([]string, error) {
	files, err := DiffFiles(base, head, nil, nil)
	if err != nil {
		return nil, err
	}

	serviceSet := make(map[string]bool)
	for _, f := range files {
		for svc, prefixes := range config.Services {
			for _, prefix := range prefixes {
				if strings.HasPrefix(f, prefix) {
					serviceSet[svc] = true
				}
			}
		}
	}

	var services []string
	for s := range serviceSet {
		services = append(services, s)
	}
	sort.Strings(services)
	return services, nil
}

// FormatDiffOutput formats a list of strings in the specified format.
func FormatDiffOutput(items []string, format string) (string, error) {
	switch strings.ToLower(format) {
	case "json":
		data, err := json.Marshal(items)
		if err != nil {
			return "", err
		}
		return string(data), nil
	case "csv":
		return strings.Join(items, ","), nil
	case "list", "":
		return strings.Join(items, "\n"), nil
	default:
		return "", fmt.Errorf("unknown format: %s", format)
	}
}

func filterPaths(paths []string, includes, excludes []string) []string {
	if len(includes) == 0 && len(excludes) == 0 {
		return paths
	}

	var result []string
	for _, p := range paths {
		if len(includes) > 0 {
			matched := false
			for _, inc := range includes {
				if m, _ := matchGlob(inc, p); m {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		excluded := false
		for _, exc := range excludes {
			if m, _ := matchGlob(exc, p); m {
				excluded = true
				break
			}
		}
		if !excluded {
			result = append(result, p)
		}
	}
	return result
}
