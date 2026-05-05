package services

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

// RenderFuncs returns the function map exposed to pipekit templates. It's
// a focused subset of "sprig-style" helpers chosen for pipeline use cases —
// no external sprig dep, no surprise behaviors.
func RenderFuncs() template.FuncMap {
	return template.FuncMap{
		// Defaults / env
		"default": func(def, val interface{}) interface{} {
			if isEmpty(val) {
				return def
			}
			return val
		},
		"env": func(name string) string { return os.Getenv(name) },
		"envOr": func(def, name string) string {
			if v := os.Getenv(name); v != "" {
				return v
			}
			return def
		},

		// Strings
		"upper":      strings.ToUpper,
		"lower":      strings.ToLower,
		"trim":       strings.TrimSpace,
		"trimSpace":  strings.TrimSpace,
		"trimPrefix": strings.TrimPrefix,
		"trimSuffix": strings.TrimSuffix,
		"contains":   strings.Contains,
		"hasPrefix":  strings.HasPrefix,
		"hasSuffix":  strings.HasSuffix,
		"replace":    func(old, new, s string) string { return strings.ReplaceAll(s, old, new) },
		"split":      func(sep, s string) []string { return strings.Split(s, sep) },
		"join":       func(sep string, items []interface{}) string { return joinAny(sep, items) },
		"quote":      func(s string) string { return strconv.Quote(s) },
		"squote":     func(s string) string { return "'" + s + "'" },
		"indent":     indent,
		"nindent":    func(n int, s string) string { return "\n" + indent(n, s) },

		// Encoding
		"b64enc": func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) },
		"b64dec": func(s string) (string, error) {
			b, err := base64.StdEncoding.DecodeString(s)
			if err != nil {
				return "", err
			}
			return string(b), nil
		},

		// Hashing
		"sha256sum": func(s string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(s))) },
		"sha1sum":   func(s string) string { return fmt.Sprintf("%x", sha1.Sum([]byte(s))) },
		"md5sum":    func(s string) string { return fmt.Sprintf("%x", md5.Sum([]byte(s))) },

		// Regex
		"regexReplace": func(pattern, replacement, s string) (string, error) {
			re, err := regexp.Compile(pattern)
			if err != nil {
				return "", err
			}
			return re.ReplaceAllString(s, replacement), nil
		},
		"regexMatch": func(pattern, s string) (bool, error) {
			return regexp.MatchString(pattern, s)
		},

		// Conversions
		"toJson": func(v interface{}) (string, error) {
			b, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			return string(b), nil
		},
		"toYaml": func(v interface{}) (string, error) {
			b, err := yaml.Marshal(v)
			if err != nil {
				return "", err
			}
			return strings.TrimRight(string(b), "\n"), nil
		},
		"fromJson": func(s string) (interface{}, error) {
			var v interface{}
			err := json.Unmarshal([]byte(s), &v)
			return v, err
		},
		"fromYaml": func(s string) (interface{}, error) {
			var v interface{}
			err := yaml.Unmarshal([]byte(s), &v)
			return normalizeMaps(v), err
		},

		// Branching
		"ternary": func(trueVal, falseVal interface{}, cond bool) interface{} {
			if cond {
				return trueVal
			}
			return falseVal
		},

		// Time
		"now": func() time.Time { return time.Now().UTC() },
		"date": func(layout string, t time.Time) string {
			return t.Format(layout)
		},

		// Builders
		"list": func(items ...interface{}) []interface{} { return items },
		"dict": func(pairs ...interface{}) (map[string]interface{}, error) {
			if len(pairs)%2 != 0 {
				return nil, fmt.Errorf("dict requires an even number of args")
			}
			out := make(map[string]interface{}, len(pairs)/2)
			for i := 0; i < len(pairs); i += 2 {
				k, ok := pairs[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings")
				}
				out[k] = pairs[i+1]
			}
			return out, nil
		},
	}
}

func indent(n int, s string) string {
	pad := strings.Repeat(" ", n)
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = pad + l
	}
	return strings.Join(lines, "\n")
}

func joinAny(sep string, items []interface{}) string {
	parts := make([]string, len(items))
	for i, v := range items {
		parts[i] = fmt.Sprintf("%v", v)
	}
	return strings.Join(parts, sep)
}

func isEmpty(v interface{}) bool {
	if v == nil {
		return true
	}
	switch t := v.(type) {
	case string:
		return t == ""
	case bool:
		return !t
	case int, int64, float64:
		return false
	case []interface{}:
		return len(t) == 0
	case map[string]interface{}:
		return len(t) == 0
	}
	return false
}

// RenderTemplateString renders a template string with the given values map,
// merged under .Values, plus .Env auto-populated from os.Environ().
func RenderTemplateString(tmplStr string, values map[string]interface{}) (string, error) {
	tmpl, err := template.New("pipekit").Funcs(RenderFuncs()).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}
	data := buildTemplateData(values)
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}
	return buf.String(), nil
}

// RenderTemplateFile reads a template from path and renders it with values.
func RenderTemplateFile(path string, values map[string]interface{}) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading template %s: %w", path, err)
	}
	return RenderTemplateString(string(data), values)
}

// LoadValues reads and decodes a values file (JSON/YAML/TOML by extension).
func LoadValues(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading values file: %w", err)
	}
	format := DetectFormat(path)
	if format == "" {
		format = FormatYAML
	}
	v, err := Decode(data, format)
	if err != nil {
		return nil, err
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("values file must be an object/map at top level")
	}
	return m, nil
}

// ApplySetOverrides merges --set key=value entries into the values tree.
// Keys may use dotted notation: "image.tag=v1" sets values.image.tag = "v1".
func ApplySetOverrides(values map[string]interface{}, sets []string) error {
	for _, s := range sets {
		idx := strings.Index(s, "=")
		if idx < 0 {
			return fmt.Errorf("invalid --set %q (expected key=value)", s)
		}
		key, val := s[:idx], s[idx+1:]
		updated, err := JSONSet(values, "."+key, val)
		if err != nil {
			return fmt.Errorf("--set %s: %w", s, err)
		}
		// JSONSet returns interface{}; assert back.
		m, ok := updated.(map[string]interface{})
		if !ok {
			return fmt.Errorf("--set %s: produced non-map root", s)
		}
		// Replace map contents in place.
		for k := range values {
			delete(values, k)
		}
		for k, v := range m {
			values[k] = v
		}
	}
	return nil
}

func buildTemplateData(values map[string]interface{}) map[string]interface{} {
	envMap := make(map[string]string)
	for _, e := range os.Environ() {
		if i := strings.Index(e, "="); i > 0 {
			envMap[e[:i]] = e[i+1:]
		}
	}
	return map[string]interface{}{
		"Values": values,
		"Env":    envMap,
	}
}
