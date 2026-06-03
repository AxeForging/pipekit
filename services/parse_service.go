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

// ExtractCodeBlocks extracts all fenced code blocks from markdown/text input.
// If language is non-empty, only blocks matching that language are returned.
// Language matching is case-insensitive.
func ExtractCodeBlocks(r io.Reader, language string) ([]CodeBlock, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}

	allBlocks := scanCodeBlocks(string(data))
	var blocks []CodeBlock
	for _, block := range allBlocks {
		if language != "" && !strings.EqualFold(block.Language, language) {
			continue
		}
		block.Index = len(blocks)
		blocks = append(blocks, block)
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

	var results []map[string]interface{}
	for _, block := range scanCodeBlocks(string(data)) {
		lang := strings.ToLower(strings.TrimSpace(block.Language))
		// Only process yaml/yml blocks (or untagged blocks)
		if lang != "" && lang != "yaml" && lang != "yml" {
			continue
		}

		var parsed map[string]interface{}
		if err := yaml.Unmarshal([]byte(block.Content), &parsed); err != nil {
			continue // skip blocks that don't parse as YAML
		}
		if parsed != nil {
			results = append(results, parsed)
		}
	}

	return results, nil
}

func scanCodeBlocks(input string) []CodeBlock {
	lines := strings.Split(input, "\n")
	var blocks []CodeBlock
	for i := 0; i < len(lines); i++ {
		fenceChar, fenceLen, lang, ok := parseOpeningFence(lines[i])
		if !ok {
			continue
		}

		contentStart := i + 1
		for j := contentStart; j < len(lines); j++ {
			if isClosingFence(lines[j], fenceChar, fenceLen) {
				content := strings.Join(lines[contentStart:j], "\n")
				blocks = append(blocks, CodeBlock{
					Language: lang,
					Content:  strings.TrimRight(content, "\n"),
					Index:    len(blocks),
				})
				i = j
				break
			}
		}
	}
	return blocks
}

func parseOpeningFence(line string) (rune, int, string, bool) {
	trimmed := strings.TrimLeft(line, " \t")
	if trimmed == "" {
		return 0, 0, "", false
	}
	fenceChar := rune(trimmed[0])
	if fenceChar != '`' && fenceChar != '~' {
		return 0, 0, "", false
	}
	fenceLen := countFenceRun(trimmed, fenceChar)
	if fenceLen < 3 {
		return 0, 0, "", false
	}
	rest := strings.TrimSpace(trimmed[fenceLen:])
	if strings.ContainsRune(rest, fenceChar) {
		return 0, 0, "", false
	}
	return fenceChar, fenceLen, rest, true
}

func isClosingFence(line string, fenceChar rune, fenceLen int) bool {
	trimmed := strings.TrimLeft(line, " \t")
	run := countFenceRun(trimmed, fenceChar)
	if run < fenceLen {
		return false
	}
	return strings.TrimSpace(trimmed[run:]) == ""
}

func countFenceRun(s string, fenceChar rune) int {
	count := 0
	for _, r := range s {
		if r != fenceChar {
			break
		}
		count++
	}
	return count
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
	delim := string(data[loc[0] : loc[0]+3])
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
