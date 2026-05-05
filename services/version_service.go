package services

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// VersionGet extracts a version string from a file.
func VersionGet(source string) (string, error) {
	if source == "auto" {
		source = autoDetectVersionFile()
		if source == "" {
			return "", fmt.Errorf("could not auto-detect version file")
		}
	}

	data, err := os.ReadFile(source)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", source, err)
	}

	return extractVersion(source, string(data))
}

// VersionBump bumps a version in the given file.
func VersionBump(source, bumpType, preRelease, buildMeta string) (string, error) {
	if source == "auto" {
		source = autoDetectVersionFile()
		if source == "" {
			return "", fmt.Errorf("could not auto-detect version file")
		}
	}

	data, err := os.ReadFile(source)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", source, err)
	}

	currentStr, err := extractVersion(source, string(data))
	if err != nil {
		return "", err
	}

	current, err := semver.NewVersion(currentStr)
	if err != nil {
		return "", fmt.Errorf("parsing version %q: %w", currentStr, err)
	}

	var next semver.Version
	switch strings.ToLower(bumpType) {
	case "major":
		next = current.IncMajor()
	case "minor":
		next = current.IncMinor()
	case "patch":
		next = current.IncPatch()
	default:
		return "", fmt.Errorf("unknown bump type: %s (use major, minor, patch)", bumpType)
	}

	newVersion := next.String()
	if preRelease != "" {
		newVersion = newVersion + "-" + preRelease
	}
	if buildMeta != "" {
		newVersion = newVersion + "+" + buildMeta
	}

	newContent, err := replaceVersionInFile(source, string(data), currentStr, newVersion)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(source, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("writing %s: %w", source, err)
	}

	return newVersion, nil
}

// VersionSet writes an explicit version into the file, replacing whatever's
// currently there (according to the format's version regex).
func VersionSet(source, newVersion string) error {
	if source == "auto" {
		source = autoDetectVersionFile()
		if source == "" {
			return fmt.Errorf("could not auto-detect version file")
		}
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("reading %s: %w", source, err)
	}
	currentStr, err := extractVersion(source, string(data))
	if err != nil {
		return err
	}
	newContent, err := replaceVersionInFile(source, string(data), currentStr, newVersion)
	if err != nil {
		return err
	}
	return os.WriteFile(source, []byte(newContent), 0644)
}

// replaceVersionInFile rewrites only the version captured by the file's
// matching regex — it does NOT use a naive strings.Replace, which can rewrite
// the wrong line if the literal version appears elsewhere (e.g. as a dep pin).
func replaceVersionInFile(filename, content, oldVer, newVer string) (string, error) {
	re := matchingVersionRegex(filename)
	if re == nil {
		// Fall back to targeted replace via the first matching submatch.
		for _, candidate := range versionPatterns {
			if loc := candidate.FindStringSubmatchIndex(content); loc != nil && len(loc) >= 4 {
				if content[loc[2]:loc[3]] == oldVer {
					re = candidate
					break
				}
			}
		}
	}
	if re == nil {
		return "", fmt.Errorf("could not locate version in %s", filename)
	}
	loc := re.FindStringSubmatchIndex(content)
	if loc == nil || len(loc) < 4 {
		return "", fmt.Errorf("could not locate version capture in %s", filename)
	}
	start, end := loc[2], loc[3]
	if content[start:end] != oldVer {
		return "", fmt.Errorf("captured version %q does not match expected %q in %s",
			content[start:end], oldVer, filename)
	}
	return content[:start] + newVer + content[end:], nil
}

func matchingVersionRegex(filename string) *regexp.Regexp {
	for name, re := range versionPatterns {
		if strings.HasSuffix(filename, name) || strings.Contains(filename, name) {
			return re
		}
	}
	return nil
}

// VersionCompare compares two semver strings.
// Returns 0 if equal, 1 if v1 > v2, -1 if v1 < v2.
func VersionCompare(v1Str, v2Str string) (int, error) {
	v1, err := semver.NewVersion(v1Str)
	if err != nil {
		return 0, fmt.Errorf("parsing %q: %w", v1Str, err)
	}
	v2, err := semver.NewVersion(v2Str)
	if err != nil {
		return 0, fmt.Errorf("parsing %q: %w", v2Str, err)
	}
	return v1.Compare(v2), nil
}

// VersionNext determines the next version from conventional commits.
func VersionNext(source string) (string, error) {
	currentStr, err := VersionGet(source)
	if err != nil {
		return "", err
	}

	current, err := semver.NewVersion(currentStr)
	if err != nil {
		return "", fmt.Errorf("parsing version %q: %w", currentStr, err)
	}

	// Get commits since last tag
	out, err := exec.Command("git", "log", "--oneline", "--no-decorate", fmt.Sprintf("v%s..HEAD", currentStr)).Output()
	if err != nil {
		// Try without v prefix
		out, _ = exec.Command("git", "log", "--oneline", "--no-decorate", fmt.Sprintf("%s..HEAD", currentStr)).Output()
	}

	commits := strings.TrimSpace(string(out))
	if commits == "" {
		return currentStr, nil
	}

	// Analyze conventional commits
	hasBreaking := strings.Contains(commits, "BREAKING CHANGE") || strings.Contains(commits, "!:")
	hasFeat := false
	for _, line := range strings.Split(commits, "\n") {
		if strings.Contains(line, "feat:") || strings.Contains(line, "feat(") {
			hasFeat = true
		}
	}

	var next semver.Version
	if hasBreaking {
		next = current.IncMajor()
	} else if hasFeat {
		next = current.IncMinor()
	} else {
		next = current.IncPatch()
	}

	return next.String(), nil
}

// FormatVersion formats a version string.
func FormatVersion(version, format string) string {
	switch strings.ToLower(format) {
	case "json":
		v, err := semver.NewVersion(version)
		if err != nil {
			return version
		}
		data, _ := json.Marshal(map[string]interface{}{
			"version":    v.String(),
			"major":      v.Major(),
			"minor":      v.Minor(),
			"patch":      v.Patch(),
			"prerelease": v.Prerelease(),
			"metadata":   v.Metadata(),
		})
		return string(data)
	case "v-prefixed", "v":
		if !strings.HasPrefix(version, "v") {
			return "v" + version
		}
		return version
	default:
		return version
	}
}

var versionPatterns = map[string]*regexp.Regexp{
	"package.json":    regexp.MustCompile(`"version"\s*:\s*"([^"]+)"`),
	"Cargo.toml":      regexp.MustCompile(`(?m)^version\s*=\s*"([^"]+)"`),
	"pyproject.toml":  regexp.MustCompile(`(?m)^version\s*=\s*"([^"]+)"`),
	"go.mod":          regexp.MustCompile(`(?m)^// version:\s*(.+)$`),
	"VERSION":         regexp.MustCompile(`(?m)^(.+)$`),
	"version.txt":     regexp.MustCompile(`(?m)^(.+)$`),
	"Chart.yaml":      regexp.MustCompile(`(?m)^version:\s*(.+)$`),
	"setup.py":        regexp.MustCompile(`version\s*=\s*['"]([^'"]+)['"]`),
	"build.gradle":    regexp.MustCompile(`(?m)^version\s*=\s*['"]([^'"]+)['"]`),
	"pom.xml":         regexp.MustCompile(`<version>([^<]+)</version>`),
}

func extractVersion(filename, content string) (string, error) {
	// Try matching by filename
	for name, re := range versionPatterns {
		if strings.HasSuffix(filename, name) || strings.Contains(filename, name) {
			matches := re.FindStringSubmatch(content)
			if len(matches) >= 2 {
				return strings.TrimSpace(matches[1]), nil
			}
		}
	}

	// Generic fallback: try common patterns
	for _, re := range versionPatterns {
		matches := re.FindStringSubmatch(content)
		if len(matches) >= 2 {
			v := strings.TrimSpace(matches[1])
			if _, err := semver.NewVersion(v); err == nil {
				return v, nil
			}
		}
	}

	return "", fmt.Errorf("could not extract version from %s", filename)
}

func autoDetectVersionFile() string {
	candidates := []string{
		"package.json",
		"Cargo.toml",
		"pyproject.toml",
		"Chart.yaml",
		"VERSION",
		"version.txt",
		"setup.py",
		"build.gradle",
		"pom.xml",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}
