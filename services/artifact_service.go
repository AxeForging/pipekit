package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/bmatcuk/doublestar/v4"
)

// ArtifactEntry describes one file selected for CI artifact collection.
type ArtifactEntry struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

// ArtifactManifest expands glob patterns and returns deterministic file metadata.
func ArtifactManifest(patterns []string) ([]ArtifactEntry, error) {
	files, err := ExpandArtifactPatterns(patterns)
	if err != nil {
		return nil, err
	}
	var entries []ArtifactEntry
	for _, path := range files {
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", path, err)
		}
		if info.IsDir() {
			continue
		}
		sum, err := ChecksumFile(path, "sha256")
		if err != nil {
			return nil, err
		}
		entries = append(entries, ArtifactEntry{
			Path:   filepath.ToSlash(path),
			Size:   info.Size(),
			SHA256: sum,
		})
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no artifact files matched")
	}
	return entries, nil
}

// ExpandArtifactPatterns expands doublestar patterns and returns sorted files.
func ExpandArtifactPatterns(patterns []string) ([]string, error) {
	if len(patterns) == 0 {
		return nil, fmt.Errorf("at least one artifact path or glob required")
	}
	seen := make(map[string]bool)
	for _, pattern := range patterns {
		matches, err := doublestar.FilepathGlob(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid glob %q: %w", pattern, err)
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("no files matched %q", pattern)
		}
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				return nil, fmt.Errorf("stat %s: %w", match, err)
			}
			if !info.IsDir() {
				seen[match] = true
			}
		}
	}
	files := make([]string, 0, len(seen))
	for path := range seen {
		files = append(files, path)
	}
	sort.Strings(files)
	return files, nil
}

// AssertArtifacts verifies each path or glob resolves to at least one file.
func AssertArtifacts(patterns []string) error {
	_, err := ExpandArtifactPatterns(patterns)
	return err
}

// FormatArtifactManifestJSON renders a manifest as pretty JSON.
func FormatArtifactManifestJSON(entries []ArtifactEntry, pretty bool) (string, error) {
	var data []byte
	var err error
	if pretty {
		data, err = json.MarshalIndent(entries, "", "  ")
	} else {
		data, err = json.Marshal(entries)
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}
