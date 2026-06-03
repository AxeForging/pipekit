package services

import (
	"strings"
	"testing"
)

func TestExtractCodeBlocks_All(t *testing.T) {
	input := "Some text\n```yaml\nname: test\nversion: 1.0\n```\nMore text\n```json\n{\"key\": \"value\"}\n```\n"

	blocks, err := ExtractCodeBlocks(strings.NewReader(input), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
	if blocks[0].Language != "yaml" {
		t.Errorf("expected language yaml, got %q", blocks[0].Language)
	}
	if !strings.Contains(blocks[0].Content, "name: test") {
		t.Errorf("expected yaml content, got %q", blocks[0].Content)
	}
	if blocks[1].Language != "json" {
		t.Errorf("expected language json, got %q", blocks[1].Language)
	}
}

func TestExtractCodeBlocks_FilterByLanguage(t *testing.T) {
	input := "```yaml\nfoo: bar\n```\n```json\n{\"a\": 1}\n```\n```yaml\nbaz: qux\n```\n"

	blocks, err := ExtractCodeBlocks(strings.NewReader(input), "yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("expected 2 yaml blocks, got %d", len(blocks))
	}
	for _, b := range blocks {
		if b.Language != "yaml" {
			t.Errorf("expected only yaml blocks, got %q", b.Language)
		}
	}
}

func TestExtractCodeBlocks_CaseInsensitive(t *testing.T) {
	input := "```YAML\nfoo: bar\n```\n"

	blocks, err := ExtractCodeBlocks(strings.NewReader(input), "yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
}

func TestExtractCodeBlocks_TildeFence(t *testing.T) {
	input := "~~~python\nprint('hello')\n~~~\n"

	blocks, err := ExtractCodeBlocks(strings.NewReader(input), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].Language != "python" {
		t.Errorf("expected python, got %q", blocks[0].Language)
	}
}

func TestExtractCodeBlocks_LongFenceWithNestedBackticks(t *testing.T) {
	input := "````md\nbefore\n```yaml\nname: test\n```\nafter\n````\n"

	blocks, err := ExtractCodeBlocks(strings.NewReader(input), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].Language != "md" {
		t.Errorf("expected md, got %q", blocks[0].Language)
	}
	if !strings.Contains(blocks[0].Content, "```yaml\nname: test\n```") {
		t.Errorf("expected nested fence to be preserved, got %q", blocks[0].Content)
	}
}

func TestExtractCodeBlocks_NoLanguage(t *testing.T) {
	input := "```\nplain text\n```\n"

	blocks, err := ExtractCodeBlocks(strings.NewReader(input), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].Language != "" {
		t.Errorf("expected empty language, got %q", blocks[0].Language)
	}
	if blocks[0].Content != "plain text" {
		t.Errorf("expected 'plain text', got %q", blocks[0].Content)
	}
}

func TestExtractCodeBlocks_NoBlocks(t *testing.T) {
	input := "Just regular text, no code blocks here."

	blocks, err := ExtractCodeBlocks(strings.NewReader(input), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 0 {
		t.Errorf("expected 0 blocks, got %d", len(blocks))
	}
}

func TestExtractAndParseYAML(t *testing.T) {
	input := "Issue body:\n```yaml\nenv: production\nreplicas: 3\n```\nSome text\n```json\n{\"ignored\": true}\n```\n```yml\nregion: us-east1\n```\n"

	results, err := ExtractAndParseYAML(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 parsed YAML blocks, got %d", len(results))
	}

	if results[0]["env"] != "production" {
		t.Errorf("expected env=production, got %v", results[0]["env"])
	}
	if results[0]["replicas"] != 3 {
		t.Errorf("expected replicas=3, got %v", results[0]["replicas"])
	}
	if results[1]["region"] != "us-east1" {
		t.Errorf("expected region=us-east1, got %v", results[1]["region"])
	}
}

func TestExtractAndParseYAML_SkipsInvalidYAML(t *testing.T) {
	input := "```yaml\n: : invalid : yaml [[\n```\n```yaml\nvalid: true\n```\n"

	results, err := ExtractAndParseYAML(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 valid YAML block (skipping invalid), got %d", len(results))
	}
	if results[0]["valid"] != true {
		t.Errorf("expected valid=true, got %v", results[0]["valid"])
	}
}

func TestExtractAndParseYAML_IncludesUntagged(t *testing.T) {
	input := "```\nname: test\n```\n"

	results, err := ExtractAndParseYAML(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 block (untagged treated as potential YAML), got %d", len(results))
	}
}

func TestFormatCodeBlocksJSON(t *testing.T) {
	blocks := []CodeBlock{
		{Language: "yaml", Content: "foo: bar", Index: 0},
		{Language: "json", Content: `{"a": 1}`, Index: 1},
	}

	jsonStr, err := FormatCodeBlocksJSON(blocks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(jsonStr, `"language":"yaml"`) {
		t.Errorf("expected JSON with language yaml, got %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"language":"json"`) {
		t.Errorf("expected JSON with language json, got %s", jsonStr)
	}
}

func TestFormatParsedYAMLJSON(t *testing.T) {
	results := []map[string]interface{}{
		{"env": "production", "replicas": 3},
	}

	jsonStr, err := FormatParsedYAMLJSON(results)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(jsonStr, `"env":"production"`) {
		t.Errorf("expected JSON with env=production, got %s", jsonStr)
	}
}
