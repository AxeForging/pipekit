package services

import (
	"strings"
	"testing"
)

func TestParseTOML(t *testing.T) {
	input := `[package]
name = "myapp"
version = "1.2.3"

[deps]
react = "18.0.0"
`
	kvs, err := ParseTOML(strings.NewReader(input), true, 0, "")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	got := map[string]string{}
	for _, kv := range kvs {
		got[kv.Key] = kv.Value
	}
	if got["package_name"] != "myapp" {
		t.Errorf("package.name not parsed: %v", got)
	}
	if got["deps_react"] != "18.0.0" {
		t.Errorf("deps.react not parsed: %v", got)
	}
}
