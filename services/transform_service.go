package services

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"hash"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"
	"text/template"
	"unicode"
)

// Base64Encode encodes data to base64.
func Base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// Base64Decode decodes base64 data.
func Base64Decode(data string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(strings.TrimSpace(data))
}

// URLEncode URL-encodes a string.
func URLEncode(s string) string {
	return url.QueryEscape(s)
}

// URLDecode URL-decodes a string.
func URLDecode(s string) (string, error) {
	return url.QueryUnescape(s)
}

// ConvertCase converts a string to the specified case format.
func ConvertCase(s, toCase string) string {
	words := splitWords(s)
	switch strings.ToLower(toCase) {
	case "camel", "camelcase":
		return toCamelCase(words)
	case "pascal", "pascalcase":
		return toPascalCase(words)
	case "snake", "snake_case":
		return toSnakeCase(words)
	case "upper-snake", "upper_snake", "upper_snake_case":
		return strings.ToUpper(toSnakeCase(words))
	case "kebab", "kebab-case":
		return toKebabCase(words)
	case "upper", "uppercase":
		return strings.ToUpper(s)
	case "lower", "lowercase":
		return strings.ToLower(s)
	default:
		return s
	}
}

// RegexReplace applies a regex find/replace on the input.
func RegexReplace(input, pattern, replace string) (string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex %q: %w", pattern, err)
	}
	return re.ReplaceAllString(input, replace), nil
}

// RenderTemplate renders a Go template with env vars and optional JSON data.
func RenderTemplate(tmplStr string, data map[string]interface{}) (string, error) {
	// Merge env vars into data
	if data == nil {
		data = make(map[string]interface{})
	}
	envMap := make(map[string]string)
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	data["Env"] = envMap

	tmpl, err := template.New("pipekit").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}
	return buf.String(), nil
}

// HashData computes a hash of the given data.
func HashData(r io.Reader, algorithm string) (string, error) {
	var h hash.Hash
	switch strings.ToLower(algorithm) {
	case "sha256":
		h = sha256.New()
	case "sha1":
		h = sha1.New()
	case "md5":
		h = md5.New()
	default:
		return "", fmt.Errorf("unsupported hash algorithm: %s", algorithm)
	}
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func splitWords(s string) []string {
	// Split on non-alphanumeric, and also on camelCase boundaries
	var words []string
	var current strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
			continue
		}
		if i > 0 && unicode.IsUpper(r) && unicode.IsLower(runes[i-1]) {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		words = append(words, current.String())
	}
	return words
}

func toCamelCase(words []string) string {
	if len(words) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(strings.ToLower(words[0]))
	for _, w := range words[1:] {
		if len(w) > 0 {
			b.WriteString(strings.ToUpper(w[:1]))
			b.WriteString(strings.ToLower(w[1:]))
		}
	}
	return b.String()
}

func toPascalCase(words []string) string {
	var b strings.Builder
	for _, w := range words {
		if len(w) > 0 {
			b.WriteString(strings.ToUpper(w[:1]))
			b.WriteString(strings.ToLower(w[1:]))
		}
	}
	return b.String()
}

func toSnakeCase(words []string) string {
	lower := make([]string, len(words))
	for i, w := range words {
		lower[i] = strings.ToLower(w)
	}
	return strings.Join(lower, "_")
}

func toKebabCase(words []string) string {
	lower := make([]string, len(words))
	for i, w := range words {
		lower[i] = strings.ToLower(w)
	}
	return strings.Join(lower, "-")
}

// slugNonAlphaNum matches any character that isn't a-z, 0-9, or hyphen.
var slugNonAlphaNum = regexp.MustCompile(`[^a-z0-9-]+`)

// slugMultiHyphen collapses multiple consecutive hyphens.
var slugMultiHyphen = regexp.MustCompile(`-{2,}`)

// Slugify converts a string (typically a branch name) into a URL-safe slug.
// It lowercases, replaces non-alphanumeric chars with hyphens, trims,
// and optionally truncates to maxLen.
func Slugify(input string, maxLen int) string {
	s := strings.ToLower(strings.TrimSpace(input))

	// Remove refs/heads/ prefix common in CI
	s = strings.TrimPrefix(s, "refs/heads/")

	// Replace slashes and underscores with hyphens
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "_", "-")

	// Remove all other non-alphanumeric characters (except hyphen)
	s = slugNonAlphaNum.ReplaceAllString(s, "")

	// Collapse multiple hyphens
	s = slugMultiHyphen.ReplaceAllString(s, "-")

	// Trim leading/trailing hyphens
	s = strings.Trim(s, "-")

	if maxLen > 0 && len(s) > maxLen {
		s = s[:maxLen]
		s = strings.TrimRight(s, "-") // don't end on a hyphen after truncation
	}

	return s
}
