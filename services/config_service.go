package services

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/AxeForging/pipekit/domain"
	"gopkg.in/yaml.v3"
)

// DefaultAliases maps common environment name variations to canonical names.
var DefaultAliases = map[string]string{
	"dev":         "dev",
	"develop":     "dev",
	"development": "dev",
	"test":        "dev",
	"testing":     "dev",
	"stage":       "staging",
	"staging":     "staging",
	"prod":        "production",
	"production":  "production",
}

// NormalizeEnvName resolves an environment name through aliases.
// It checks custom aliases first, then falls back to defaults.
// Returns the canonical name and an error if unresolved.
func NormalizeEnvName(raw string, customAliases map[string]string) (string, error) {
	lower := strings.ToLower(strings.TrimSpace(raw))
	if lower == "" {
		return "", fmt.Errorf("environment name is empty")
	}

	// Custom aliases take priority
	if customAliases != nil {
		if canonical, ok := customAliases[lower]; ok {
			return canonical, nil
		}
	}

	// Fall back to defaults
	if canonical, ok := DefaultAliases[lower]; ok {
		return canonical, nil
	}

	// If the raw name itself is used as a key in the config, pass it through
	return lower, nil
}

// ResolveConfig reads a JSON or YAML config map, normalizes the environment key,
// and returns the resolved key-value pairs for that environment.
func ResolveConfig(r io.Reader, envName string, format string, customAliases map[string]string) ([]domain.KeyValue, string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, "", fmt.Errorf("reading config: %w", err)
	}

	var configMap map[string]map[string]interface{}

	switch strings.ToLower(format) {
	case "yaml", "yml":
		if err := yaml.Unmarshal(data, &configMap); err != nil {
			return nil, "", fmt.Errorf("parsing YAML config: %w", err)
		}
	default:
		if err := json.Unmarshal(data, &configMap); err != nil {
			return nil, "", fmt.Errorf("parsing JSON config: %w", err)
		}
	}

	// Normalize the env name
	normalized, err := NormalizeEnvName(envName, customAliases)
	if err != nil {
		return nil, "", err
	}

	// Try normalized name first, then the raw lowered name
	envConfig, ok := configMap[normalized]
	if !ok {
		// Also try the original lowered value
		envConfig, ok = configMap[strings.ToLower(envName)]
		if !ok {
			available := make([]string, 0, len(configMap))
			for k := range configMap {
				available = append(available, k)
			}
			sort.Strings(available)
			return nil, normalized, fmt.Errorf("environment %q (normalized from %q) not found in config; available: %s",
				normalized, envName, strings.Join(available, ", "))
		}
	}

	kvs := make([]domain.KeyValue, 0, len(envConfig))
	for k, v := range envConfig {
		kvs = append(kvs, domain.KeyValue{Key: k, Value: fmt.Sprintf("%v", v)})
	}
	sort.Slice(kvs, func(i, j int) bool { return kvs[i].Key < kvs[j].Key })

	return kvs, normalized, nil
}

// ResolveConfigJSON returns the resolved environment config as a compact JSON string.
func ResolveConfigJSON(r io.Reader, envName string, format string, customAliases map[string]string) (string, string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", "", fmt.Errorf("reading config: %w", err)
	}

	var configMap map[string]map[string]interface{}

	switch strings.ToLower(format) {
	case "yaml", "yml":
		if err := yaml.Unmarshal(data, &configMap); err != nil {
			return "", "", fmt.Errorf("parsing YAML config: %w", err)
		}
	default:
		if err := json.Unmarshal(data, &configMap); err != nil {
			return "", "", fmt.Errorf("parsing JSON config: %w", err)
		}
	}

	normalized, err := NormalizeEnvName(envName, customAliases)
	if err != nil {
		return "", "", err
	}

	envConfig, ok := configMap[normalized]
	if !ok {
		envConfig, ok = configMap[strings.ToLower(envName)]
		if !ok {
			available := make([]string, 0, len(configMap))
			for k := range configMap {
				available = append(available, k)
			}
			sort.Strings(available)
			return "", normalized, fmt.Errorf("environment %q (normalized from %q) not found in config; available: %s",
				normalized, envName, strings.Join(available, ", "))
		}
	}

	jsonBytes, err := json.Marshal(envConfig)
	if err != nil {
		return "", normalized, fmt.Errorf("marshaling config JSON: %w", err)
	}

	return string(jsonBytes), normalized, nil
}

// BranchToEnv maps a git branch name to an environment using a mapping config.
// The mapping is a JSON object like {"main": "production", "develop": "dev", "release/*": "staging"}.
// Supports exact matches and prefix globs (e.g., "release/*").
func BranchToEnv(branch string, mappingJSON string) (string, error) {
	var mapping map[string]string
	if err := json.Unmarshal([]byte(mappingJSON), &mapping); err != nil {
		return "", fmt.Errorf("parsing branch mapping: %w", err)
	}

	// Clean branch name (remove refs/heads/ prefix)
	branch = strings.TrimPrefix(branch, "refs/heads/")

	// Exact match first
	if env, ok := mapping[branch]; ok {
		return env, nil
	}

	// Glob/prefix match
	for pattern, env := range mapping {
		if strings.HasSuffix(pattern, "/*") {
			prefix := strings.TrimSuffix(pattern, "/*")
			if strings.HasPrefix(branch, prefix+"/") {
				return env, nil
			}
		}
		if strings.HasSuffix(pattern, "*") {
			prefix := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(branch, prefix) {
				return env, nil
			}
		}
	}

	return "", fmt.Errorf("branch %q does not match any environment mapping", branch)
}
