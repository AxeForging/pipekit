package services

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// HiddenAnchor represents a non-rendered HTML marker in markdown.
type HiddenAnchor struct {
	Name   string `json:"name"`
	Marker string `json:"marker"`
	Start  int    `json:"start"`
	End    int    `json:"end"`
}

// CommentInspection is a structured view of a markdown or GitHub comment body.
type CommentInspection struct {
	ID      string         `json:"id,omitempty"`
	URL     string         `json:"url,omitempty"`
	Author  string         `json:"author,omitempty"`
	Body    string         `json:"body"`
	Anchors []HiddenAnchor `json:"anchors,omitempty"`
	Blocks  []CodeBlock    `json:"blocks,omitempty"`
}

var hiddenAnchorRegex = regexp.MustCompile(`<!--\s*pipekit:([A-Za-z0-9_.:/@-]+)\s*-->`)

// AnchorMarker returns the hidden markdown marker for an anchor name.
func AnchorMarker(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("anchor name required")
	}
	if strings.Contains(name, "--") || strings.ContainsAny(name, "<>\r\n\t ") {
		return "", fmt.Errorf("invalid anchor name %q", name)
	}
	return fmt.Sprintf("<!-- pipekit:%s -->", name), nil
}

// FindHiddenAnchors extracts pipekit hidden anchors from a markdown body.
func FindHiddenAnchors(body string) []HiddenAnchor {
	matches := hiddenAnchorRegex.FindAllStringSubmatchIndex(body, -1)
	anchors := make([]HiddenAnchor, 0, len(matches))
	for _, match := range matches {
		anchors = append(anchors, HiddenAnchor{
			Name:   body[match[2]:match[3]],
			Marker: body[match[0]:match[1]],
			Start:  match[0],
			End:    match[1],
		})
	}
	return anchors
}

// RenderAnchoredComment creates a markdown comment body with a hidden anchor.
func RenderAnchoredComment(anchor, body string) (string, error) {
	marker, err := AnchorMarker(anchor)
	if err != nil {
		return "", err
	}
	return marker + "\n\n" + strings.TrimRight(body, "\n") + "\n", nil
}

// GitHubCommentPayload returns a JSON object suitable for the issue comments API.
func GitHubCommentPayload(body string) (string, error) {
	data, err := json.Marshal(map[string]string{"body": body})
	if err != nil {
		return "", fmt.Errorf("marshaling comment payload: %w", err)
	}
	return string(data), nil
}

// AmendAnchoredComment replaces everything after a hidden anchor with body.
// If the anchor is not present, it returns a new anchored comment.
func AmendAnchoredComment(existing, anchor, body string) (string, error) {
	marker, err := AnchorMarker(anchor)
	if err != nil {
		return "", err
	}
	for _, found := range FindHiddenAnchors(existing) {
		if found.Name == anchor {
			prefix := strings.TrimRight(existing[:found.End], "\n")
			return prefix + "\n\n" + strings.TrimRight(body, "\n") + "\n", nil
		}
	}
	return marker + "\n\n" + strings.TrimRight(body, "\n") + "\n", nil
}

// RenderCodeFence renders markdown code using a fence long enough for content.
func RenderCodeFence(language, content string) string {
	fence := strings.Repeat("`", maxBacktickRun(content)+1)
	if len(fence) < 3 {
		fence = "```"
	}
	var b strings.Builder
	b.WriteString(fence)
	b.WriteString(strings.TrimSpace(language))
	b.WriteByte('\n')
	b.WriteString(strings.TrimRight(content, "\n"))
	b.WriteByte('\n')
	b.WriteString(fence)
	b.WriteByte('\n')
	return b.String()
}

func maxBacktickRun(s string) int {
	maxRun, run := 0, 0
	for _, r := range s {
		if r == '`' {
			run++
			if run > maxRun {
				maxRun = run
			}
			continue
		}
		run = 0
	}
	return maxRun
}

// InspectMarkdownComment returns anchors and fenced code blocks from markdown.
func InspectMarkdownComment(body string) (CommentInspection, error) {
	blocks, err := ExtractCodeBlocks(strings.NewReader(body), "")
	if err != nil {
		return CommentInspection{}, err
	}
	return CommentInspection{
		Body:    body,
		Anchors: FindHiddenAnchors(body),
		Blocks:  blocks,
	}, nil
}

// InspectComments reads either a GitHub comments JSON array/object or raw markdown.
func InspectComments(r io.Reader) ([]CommentInspection, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "[") {
		return inspectGitHubCommentArray(data)
	}
	if strings.HasPrefix(trimmed, "{") {
		item, err := inspectGitHubCommentObject(data)
		if err == nil && item.Body != "" {
			return []CommentInspection{item}, nil
		}
	}
	item, err := InspectMarkdownComment(string(data))
	if err != nil {
		return nil, err
	}
	return []CommentInspection{item}, nil
}

// SelectAnchoredComment returns the first inspected comment with anchor.
func SelectAnchoredComment(comments []CommentInspection, anchor string) (CommentInspection, bool) {
	for _, comment := range comments {
		for _, found := range comment.Anchors {
			if found.Name == anchor {
				return comment, true
			}
		}
	}
	return CommentInspection{}, false
}

func inspectGitHubCommentArray(data []byte) ([]CommentInspection, error) {
	var raw []map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing comments JSON: %w", err)
	}
	out := make([]CommentInspection, 0, len(raw))
	for _, item := range raw {
		inspected, err := inspectGitHubCommentMap(item)
		if err != nil {
			return nil, err
		}
		out = append(out, inspected)
	}
	return out, nil
}

func inspectGitHubCommentObject(data []byte) (CommentInspection, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return CommentInspection{}, err
	}
	return inspectGitHubCommentMap(raw)
}

func inspectGitHubCommentMap(item map[string]interface{}) (CommentInspection, error) {
	body, _ := item["body"].(string)
	inspected, err := InspectMarkdownComment(body)
	if err != nil {
		return CommentInspection{}, err
	}
	inspected.ID = stringifyJSONValue(item["id"])
	inspected.URL = firstString(item, "html_url", "url")
	if user, ok := item["user"].(map[string]interface{}); ok {
		inspected.Author, _ = user["login"].(string)
	}
	return inspected, nil
}

func stringifyJSONValue(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatInt(int64(t), 10)
	case json.Number:
		return t.String()
	default:
		return ""
	}
}

func firstString(item map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if v, ok := item[key].(string); ok && v != "" {
			return v
		}
	}
	return ""
}
