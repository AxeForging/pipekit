package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/itchyny/gojq"
)

// AssertEnvExists checks that all named env vars exist and are non-empty.
func AssertEnvExists(names []string) error {
	var missing []string
	for _, name := range names {
		if os.Getenv(name) == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing or empty environment variables: %s", strings.Join(missing, ", "))
	}
	return nil
}

// AssertFileExists checks that all named files exist.
func AssertFileExists(paths []string) error {
	var missing []string
	for _, p := range paths {
		if _, err := os.Stat(p); os.IsNotExist(err) {
			missing = append(missing, p)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("files not found: %s", strings.Join(missing, ", "))
	}
	return nil
}

// AssertJSONPath checks that a value at a JSON path matches the expected value.
func AssertJSONPath(jsonData []byte, path, expected string) error {
	var raw interface{}
	if err := json.Unmarshal(jsonData, &raw); err != nil {
		return fmt.Errorf("parsing JSON: %w", err)
	}
	query, err := gojq.Parse(path)
	if err != nil {
		return fmt.Errorf("parsing path %q: %w", path, err)
	}
	iter := query.Run(raw)
	v, ok := iter.Next()
	if !ok {
		return fmt.Errorf("path %q not found", path)
	}
	if errVal, ok := v.(error); ok {
		return fmt.Errorf("query error: %w", errVal)
	}
	actual := fmt.Sprintf("%v", v)
	if actual != expected {
		return fmt.Errorf("assertion failed: %s = %q, expected %q", path, actual, expected)
	}
	return nil
}

// AssertSemver checks that a string is valid semver.
func AssertSemver(version string) error {
	_, err := semver.NewVersion(version)
	if err != nil {
		return fmt.Errorf("%q is not valid semver: %w", version, err)
	}
	return nil
}

// AssertSemverCompare compares two semver strings.
// Returns nil if the comparison holds, error otherwise.
func AssertSemverCompare(v1Str, operator, v2Str string) error {
	v1, err := semver.NewVersion(v1Str)
	if err != nil {
		return fmt.Errorf("parsing %q: %w", v1Str, err)
	}
	v2, err := semver.NewVersion(v2Str)
	if err != nil {
		return fmt.Errorf("parsing %q: %w", v2Str, err)
	}

	cmp := v1.Compare(v2)
	switch operator {
	case "gt", ">":
		if cmp <= 0 {
			return fmt.Errorf("%s is not greater than %s", v1Str, v2Str)
		}
	case "lt", "<":
		if cmp >= 0 {
			return fmt.Errorf("%s is not less than %s", v1Str, v2Str)
		}
	case "eq", "==":
		if cmp != 0 {
			return fmt.Errorf("%s is not equal to %s", v1Str, v2Str)
		}
	case "gte", ">=":
		if cmp < 0 {
			return fmt.Errorf("%s is not greater than or equal to %s", v1Str, v2Str)
		}
	case "lte", "<=":
		if cmp > 0 {
			return fmt.Errorf("%s is not less than or equal to %s", v1Str, v2Str)
		}
	default:
		return fmt.Errorf("unknown operator: %s (use gt, lt, eq, gte, lte)", operator)
	}
	return nil
}

// AssertURL checks that a URL returns one of the expected status codes.
func AssertURL(urlStr string, expectedCodes []int, timeout time.Duration) error {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(urlStr)
	if err != nil {
		return fmt.Errorf("requesting %s: %w", urlStr, err)
	}
	defer resp.Body.Close()

	for _, code := range expectedCodes {
		if resp.StatusCode == code {
			return nil
		}
	}
	return fmt.Errorf("URL %s returned status %d, expected one of %v", urlStr, resp.StatusCode, expectedCodes)
}
