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

// maskScannerBufferBytes is the maximum line size pipekit's masker handles
// in line-by-line mode. The default bufio.Scanner cap is 64KB; we raise it
// to 64MB so a single oversize log line doesn't silently halt processing.
const maskScannerBufferBytes = 64 * 1024 * 1024

// SecretPresets is a curated set of regex patterns for well-known secret
// formats. Use via MaskValuesPreset / MaskValuesMultilinePreset.
var SecretPresets = map[string][]string{
	"aws":    {`AKIA[0-9A-Z]{16}`, `(?i)aws[_-]?secret[_-]?access[_-]?key["'\s:=]+[A-Za-z0-9/+=]{40}`},
	"github": {`gh[pousr]_[A-Za-z0-9]{36,}`, `github_pat_[A-Za-z0-9_]{82}`},
	"gcp":    {`-----BEGIN PRIVATE KEY-----[\s\S]*?-----END PRIVATE KEY-----`},
	"jwt":    {`eyJ[A-Za-z0-9_=-]+\.eyJ[A-Za-z0-9_=-]+\.[A-Za-z0-9_.+/=-]+`},
	"slack":  {`xox[baprs]-[A-Za-z0-9-]{10,}`},
	"stripe": {`sk_(test|live)_[A-Za-z0-9]{24,}`},
	"pem":    {`-----BEGIN [A-Z ]+PRIVATE KEY-----[\s\S]*?-----END [A-Z ]+PRIVATE KEY-----`},
}

// PresetPatterns returns the patterns for the given preset names, joined.
// Unknown preset names are returned in the second result for the caller to
// surface as an error if desired.
func PresetPatterns(names []string) (patterns []string, unknown []string) {
	for _, n := range names {
		key := strings.ToLower(strings.TrimSpace(n))
		if pats, ok := SecretPresets[key]; ok {
			patterns = append(patterns, pats...)
		} else if key != "" {
			unknown = append(unknown, key)
		}
	}
	return
}

// MaskValues replaces occurrences of patterns in the input stream, line by
// line. Use MaskValuesMultiline for patterns that span newlines.
func MaskValues(r io.Reader, w io.Writer, patterns []string, replacement string, partial int) error {
	compiledPatterns, err := compilePatterns(patterns)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), maskScannerBufferBytes)
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

// MaskValuesMultiline reads the entire input and applies patterns over the
// whole stream — required for secrets that span multiple lines (PEM keys,
// multi-line JWTs split across log lines, etc.).
func MaskValuesMultiline(r io.Reader, w io.Writer, patterns []string, replacement string, partial int) error {
	compiledPatterns, err := compilePatterns(patterns)
	if err != nil {
		return err
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	out := string(data)
	for _, re := range compiledPatterns {
		out = re.ReplaceAllStringFunc(out, func(match string) string {
			return maskString(match, replacement, partial)
		})
	}
	_, err = io.WriteString(w, out)
	return err
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
