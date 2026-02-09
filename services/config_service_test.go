package services

import (
	"strings"
	"testing"
)

func TestNormalizeEnvName_Defaults(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"dev", "dev"},
		{"develop", "dev"},
		{"development", "dev"},
		{"test", "dev"},
		{"testing", "dev"},
		{"stage", "staging"},
		{"staging", "staging"},
		{"prod", "production"},
		{"production", "production"},
		{"PROD", "production"},
		{"Dev", "dev"},
	}
	for _, tc := range tests {
		result, err := NormalizeEnvName(tc.input, nil)
		if err != nil {
			t.Errorf("NormalizeEnvName(%q) unexpected error: %v", tc.input, err)
			continue
		}
		if result != tc.expected {
			t.Errorf("NormalizeEnvName(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

func TestNormalizeEnvName_CustomAliases(t *testing.T) {
	custom := map[string]string{
		"preview": "staging",
		"canary":  "production",
	}
	result, err := NormalizeEnvName("preview", custom)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "staging" {
		t.Errorf("expected staging, got %s", result)
	}

	result, err = NormalizeEnvName("canary", custom)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "production" {
		t.Errorf("expected production, got %s", result)
	}
}

func TestNormalizeEnvName_Empty(t *testing.T) {
	_, err := NormalizeEnvName("", nil)
	if err == nil {
		t.Error("expected error for empty env name")
	}
}

func TestNormalizeEnvName_Passthrough(t *testing.T) {
	result, err := NormalizeEnvName("custom-env", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "custom-env" {
		t.Errorf("expected custom-env passthrough, got %s", result)
	}
}

func TestResolveConfig_JSON(t *testing.T) {
	config := `{
		"dev": {"project_id": "my-dev-project", "region": "us-east1"},
		"staging": {"project_id": "my-staging-project", "region": "us-west1"},
		"production": {"project_id": "my-prod-project", "region": "eu-west1"}
	}`

	kvs, normalized, err := ResolveConfig(strings.NewReader(config), "develop", "json", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if normalized != "dev" {
		t.Errorf("expected normalized name 'dev', got %q", normalized)
	}
	if len(kvs) != 2 {
		t.Fatalf("expected 2 key-value pairs, got %d", len(kvs))
	}

	found := make(map[string]string)
	for _, kv := range kvs {
		found[kv.Key] = kv.Value
	}
	if found["project_id"] != "my-dev-project" {
		t.Errorf("expected project_id=my-dev-project, got %s", found["project_id"])
	}
	if found["region"] != "us-east1" {
		t.Errorf("expected region=us-east1, got %s", found["region"])
	}
}

func TestResolveConfig_YAML(t *testing.T) {
	config := `
dev:
  project_id: my-dev-project
  region: us-east1
production:
  project_id: my-prod-project
  region: eu-west1
`

	kvs, normalized, err := ResolveConfig(strings.NewReader(config), "prod", "yaml", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if normalized != "production" {
		t.Errorf("expected normalized name 'production', got %q", normalized)
	}
	if len(kvs) != 2 {
		t.Fatalf("expected 2 key-value pairs, got %d", len(kvs))
	}
}

func TestResolveConfig_NotFound(t *testing.T) {
	config := `{"dev": {"key": "val"}}`
	_, _, err := ResolveConfig(strings.NewReader(config), "nonexistent", "json", nil)
	if err == nil {
		t.Error("expected error for unknown environment")
	}
	if !strings.Contains(err.Error(), "not found in config") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestResolveConfigJSON(t *testing.T) {
	config := `{
		"dev": {"project_id": "my-dev", "num": 42},
		"production": {"project_id": "my-prod", "num": 99}
	}`

	jsonStr, normalized, err := ResolveConfigJSON(strings.NewReader(config), "prod", "json", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if normalized != "production" {
		t.Errorf("expected normalized name 'production', got %q", normalized)
	}
	if !strings.Contains(jsonStr, "my-prod") {
		t.Errorf("expected JSON to contain 'my-prod', got %s", jsonStr)
	}
}

func TestBranchToEnv_ExactMatch(t *testing.T) {
	mapping := `{"main": "production", "develop": "dev", "staging": "staging"}`

	env, err := BranchToEnv("main", mapping)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env != "production" {
		t.Errorf("expected production, got %s", env)
	}
}

func TestBranchToEnv_RefsHeadsPrefix(t *testing.T) {
	mapping := `{"main": "production"}`

	env, err := BranchToEnv("refs/heads/main", mapping)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env != "production" {
		t.Errorf("expected production, got %s", env)
	}
}

func TestBranchToEnv_GlobMatch(t *testing.T) {
	mapping := `{"main": "production", "develop": "dev", "release/*": "staging"}`

	env, err := BranchToEnv("release/v1.2.0", mapping)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env != "staging" {
		t.Errorf("expected staging, got %s", env)
	}
}

func TestBranchToEnv_PrefixGlob(t *testing.T) {
	mapping := `{"feature*": "dev", "main": "production"}`

	env, err := BranchToEnv("feature/my-thing", mapping)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env != "dev" {
		t.Errorf("expected dev, got %s", env)
	}
}

func TestBranchToEnv_NoMatch(t *testing.T) {
	mapping := `{"main": "production"}`

	_, err := BranchToEnv("unknown-branch", mapping)
	if err == nil {
		t.Error("expected error for unmatched branch")
	}
}

func TestBranchToEnv_InvalidJSON(t *testing.T) {
	_, err := BranchToEnv("main", "not json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
