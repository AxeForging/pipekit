package services

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MatrixFromDirs generates a matrix JSON from directory names in the given path.
func MatrixFromDirs(dirPath, key string) (string, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return "", fmt.Errorf("reading directory %s: %w", dirPath, err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	matrix := map[string][]string{key: names}
	data, err := json.Marshal(matrix)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// MatrixFromFiles generates a matrix JSON from files matching a glob pattern.
func MatrixFromFiles(pattern, key string) (string, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
	}

	var names []string
	for _, m := range matches {
		names = append(names, filepath.Base(m))
	}
	sort.Strings(names)

	matrix := map[string][]string{key: names}
	data, err := json.Marshal(matrix)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// MatrixFromJSON transforms a JSON array into matrix format with filtering.
func MatrixFromJSON(r io.Reader, key string, filterField, filterValue string) (string, error) {
	var raw []interface{}
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return "", fmt.Errorf("parsing JSON array: %w", err)
	}

	if filterField != "" {
		var filtered []interface{}
		for _, item := range raw {
			if m, ok := item.(map[string]interface{}); ok {
				if v, exists := m[filterField]; exists && fmt.Sprintf("%v", v) == filterValue {
					filtered = append(filtered, item)
				}
			}
		}
		raw = filtered
	}

	matrix := map[string]interface{}{key: raw}
	data, err := json.Marshal(matrix)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// MatrixCombine generates a Cartesian product of multiple arrays.
func MatrixCombine(arrays map[string][]string) (string, error) {
	// Get sorted keys for deterministic output
	var keys []string
	for k := range arrays {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build include array (Cartesian product)
	var includes []map[string]string
	includes = append(includes, map[string]string{})

	for _, key := range keys {
		values := arrays[key]
		var newIncludes []map[string]string
		for _, existing := range includes {
			for _, val := range values {
				combo := make(map[string]string)
				for k, v := range existing {
					combo[k] = v
				}
				combo[key] = val
				newIncludes = append(newIncludes, combo)
			}
		}
		includes = newIncludes
	}

	result := map[string]interface{}{"include": includes}
	data, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteToGitHubOutputValue writes a single key=value to GITHUB_OUTPUT.
func WriteToGitHubOutputValue(key, value string) error {
	path := os.Getenv("GITHUB_OUTPUT")
	if path == "" {
		return fmt.Errorf("$GITHUB_OUTPUT is not set")
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("opening GITHUB_OUTPUT: %w", err)
	}
	defer f.Close()

	if strings.Contains(value, "\n") {
		delimiter := "EOF_PIPEKIT"
		_, err = fmt.Fprintf(f, "%s<<%s\n%s\n%s\n", key, delimiter, value, delimiter)
	} else {
		_, err = fmt.Fprintf(f, "%s=%s\n", key, value)
	}
	return err
}
