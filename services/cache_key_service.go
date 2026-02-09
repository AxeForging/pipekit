package services

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CacheKeyFromFiles computes a SHA256 hash from one or more files.
func CacheKeyFromFiles(files []string, prefix string) (string, error) {
	h := sha256.New()
	for _, f := range files {
		file, err := os.Open(f)
		if err != nil {
			return "", fmt.Errorf("opening %s: %w", f, err)
		}
		if _, err := io.Copy(h, file); err != nil {
			file.Close()
			return "", fmt.Errorf("hashing %s: %w", f, err)
		}
		file.Close()
	}
	hash := fmt.Sprintf("%x", h.Sum(nil))
	return prefix + hash, nil
}

// CacheKeyFromGlob computes a SHA256 hash from all files matching a glob pattern.
func CacheKeyFromGlob(pattern string, prefix string) (string, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no files matched pattern %q", pattern)
	}
	sort.Strings(matches)
	return CacheKeyFromFiles(matches, prefix)
}

// CacheKeyComposite combines multiple parts into a single cache key.
func CacheKeyComposite(parts []string, prefix, separator string) string {
	if separator == "" {
		separator = "-"
	}
	key := strings.Join(parts, separator)
	return prefix + key
}
