package services

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// MaskValues replaces occurrences of patterns in the input stream.
func MaskValues(r io.Reader, w io.Writer, patterns []string, replacement string, partial int) error {
	compiledPatterns, err := compilePatterns(patterns)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		for _, re := range compiledPatterns {
			line = re.ReplaceAllStringFunc(line, func(match string) string {
				return maskString(match, replacement, partial)
			})
		}
		fmt.Fprintln(w, line)
	}
	return scanner.Err()
}

// MaskFile reads a file, applies masks, and writes to w.
func MaskFile(path string, w io.Writer, patterns []string, replacement string, partial int) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()
	return MaskValues(f, w, patterns, replacement, partial)
}

// MaskGitHub emits ::add-mask:: commands for GitHub Actions.
func MaskGitHub(w io.Writer, values []string) error {
	for _, v := range values {
		if v == "" {
			continue
		}
		if _, err := fmt.Fprintf(w, "::add-mask::%s\n", v); err != nil {
			return err
		}
	}
	return nil
}

// MaskEnvVars finds env vars matching glob patterns and emits GitHub masks.
func MaskEnvVars(w io.Writer, envPatterns []string, github bool) error {
	var values []string
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		name, value := parts[0], parts[1]
		if value == "" {
			continue
		}
		for _, pattern := range envPatterns {
			matched, err := filepath.Match(pattern, name)
			if err != nil {
				return fmt.Errorf("invalid pattern %q: %w", pattern, err)
			}
			if matched {
				values = append(values, value)
				break
			}
		}
	}

	if github {
		return MaskGitHub(w, values)
	}

	for _, v := range values {
		fmt.Fprintln(w, "***")
		_ = v // values are masked, not printed
	}
	return nil
}

func compilePatterns(patterns []string) ([]*regexp.Regexp, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %q: %w", p, err)
		}
		compiled = append(compiled, re)
	}
	return compiled, nil
}

func maskString(s, replacement string, partial int) string {
	if partial > 0 && len(s) > partial*2 {
		return s[:partial] + replacement + s[len(s)-partial:]
	}
	return replacement
}
