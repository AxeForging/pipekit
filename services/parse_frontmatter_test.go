package services

import (
	"strings"
	"testing"
)

func TestExtractFrontmatter_YAML(t *testing.T) {
	input := `---
title: My post
draft: true
---

Body here.`
	body, format, err := ExtractFrontmatter(strings.NewReader(input))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if format != "yaml" {
		t.Errorf("expected yaml, got %s", format)
	}
	if !strings.Contains(string(body), "title: My post") {
		t.Errorf("body missing: %s", body)
	}
}

func TestExtractFrontmatter_TOML(t *testing.T) {
	input := `+++
title = "My post"
draft = true
+++

Body here.`
	body, format, err := ExtractFrontmatter(strings.NewReader(input))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if format != "toml" {
		t.Errorf("expected toml, got %s", format)
	}
	if !strings.Contains(string(body), `title = "My post"`) {
		t.Errorf("body missing: %s", body)
	}
}

func TestExtractFrontmatter_None(t *testing.T) {
	body, _, err := ExtractFrontmatter(strings.NewReader("plain content\nno frontmatter"))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if body != nil {
		t.Errorf("expected nil for no frontmatter, got %s", body)
	}
}
