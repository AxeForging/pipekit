package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMatrixFromDirs(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "api"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "web"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, ".hidden"), 0755)

	result, err := MatrixFromDirs(tmpDir, "service")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string][]string
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	services := parsed["service"]
	if len(services) != 2 {
		t.Fatalf("expected 2 dirs (hidden excluded), got %d", len(services))
	}
	if services[0] != "api" || services[1] != "web" {
		t.Errorf("expected [api, web], got %v", services)
	}
}

func TestMatrixFromFiles(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test1.yaml"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "test2.yaml"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "other.json"), []byte(""), 0644)

	result, err := MatrixFromFiles(filepath.Join(tmpDir, "*.yaml"), "config")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string][]string
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(parsed["config"]) != 2 {
		t.Errorf("expected 2 files, got %d", len(parsed["config"]))
	}
}

func TestMatrixFromJSON(t *testing.T) {
	input := `[{"name": "api", "deploy": true}, {"name": "web", "deploy": false}]`
	result, err := MatrixFromJSON(strings.NewReader(input), "service", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, `"service"`) {
		t.Error("expected service key in output")
	}
}

func TestMatrixFromJSON_Filtered(t *testing.T) {
	input := `[{"name": "api", "deploy": "true"}, {"name": "web", "deploy": "false"}]`
	result, err := MatrixFromJSON(strings.NewReader(input), "service", "deploy", "true")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "web") {
		t.Error("expected web to be filtered out")
	}
}

func TestMatrixCombine(t *testing.T) {
	arrays := map[string][]string{
		"os":   {"linux", "darwin"},
		"arch": {"amd64", "arm64"},
	}
	result, err := MatrixCombine(arrays)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string][]map[string]string
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	includes := parsed["include"]
	if len(includes) != 4 {
		t.Errorf("expected 4 combinations (2x2), got %d", len(includes))
	}
}
