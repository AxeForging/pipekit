package services

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// CodeBlock represents an extracted fenced code block.
type CodeBlock struct {
	Language string `json:"language,omitempty"`
	Content  string `json:"content"`
	Index    int    `json:"index"`
}

// fencedBlockRegex matches ``` or ~~~ fenced code blocks with optional language identifier.
var fencedBlockRegex = regexp.MustCompile("(?m)^(?:```|~~~)(\\S*)\\s*\\n((?:.|\\n)*?)^(?:```|~~~)\\s*$")

// ExtractCodeBlocks extracts all fenced code blocks from markdown/text input.
// If language is non-empty, only blocks matching that language are returned.
// Language matching is case-insensitive.
func ExtractCodeBlocks(r io.Reader, language string) ([]CodeBlock, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}

	matches := fencedBlockRegex.FindAllSubmatch(data, -1)
	if len(matches) == 0 {
		return nil, nil
	}

	var blocks []CodeBlock
	idx := 0
	for _, match := range matches {
		lang := strings.TrimSpace(string(match[1]))
		content := string(match[2])

		// Remove trailing newline from content
		content = strings.TrimRight(content, "\n")

		if language != "" && !strings.EqualFold(lang, language) {
			continue
		}

		blocks = append(blocks, CodeBlock{
			Language: lang,
			Content:  content,
			Index:    idx,
		})
		idx++
	}

	return blocks, nil
}

// ExtractAndParseYAML extracts YAML code blocks from markdown and parses them into JSON.
// If no language filter is given, it looks for blocks tagged as yaml or yml.
func ExtractAndParseYAML(r io.Reader) ([]map[string]interface{}, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}

	matches := fencedBlockRegex.FindAllSubmatch(data, -1)
	if len(matches) == 0 {
		return nil, nil
	}

	var results []map[string]interface{}
	for _, match := range matches {
		lang := strings.ToLower(strings.TrimSpace(string(match[1])))
		content := string(match[2])

		// Only process yaml/yml blocks (or untagged blocks)
		if lang != "" && lang != "yaml" && lang != "yml" {
			continue
		}

		var parsed map[string]interface{}
		if err := yaml.Unmarshal([]byte(content), &parsed); err != nil {
			continue // skip blocks that don't parse as YAML
		}
		if parsed != nil {
			results = append(results, parsed)
		}
	}

	return results, nil
}

// frontmatterRegex matches a YAML or TOML frontmatter block at the very
// start of the input. Supports `---` (YAML) and `+++` (TOML) delimiters.
var frontmatterRegex = regexp.MustCompile(`(?s)\A(?:---|\+\+\+)\n(.+?)\n(?:---|\+\+\+)\s*\n?`)

// ExtractFrontmatter pulls the leading frontmatter block out of input and
// returns the raw bytes plus the format detected ("yaml" or "toml").
// Returns nil if no frontmatter is present.
func ExtractFrontmatter(r io.Reader) ([]byte, string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, "", err
	}
	loc := frontmatterRegex.FindSubmatchIndex(data)
	if loc == nil {
		return nil, "", nil
	}
	body := data[loc[2]:loc[3]]
	delim := string(data[loc[0]:loc[0]+3])
	if delim == "+++" {
		return body, "toml", nil
	}
	return body, "yaml", nil
}

// FormatCodeBlocksJSON returns extracted code blocks as a JSON array.
func FormatCodeBlocksJSON(blocks []CodeBlock) (string, error) {
	data, err := json.Marshal(blocks)
	if err != nil {
		return "", fmt.Errorf("marshaling blocks: %w", err)
	}
	return string(data), nil
}

// FormatParsedYAMLJSON returns parsed YAML blocks as a JSON array.
func FormatParsedYAMLJSON(results []map[string]interface{}) (string, error) {
	data, err := json.Marshal(results)
	if err != nil {
		return "", fmt.Errorf("marshaling results: %w", err)
	}
	return string(data), nil
}
