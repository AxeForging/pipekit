package services

import (
	"strings"
	"testing"
)

func TestAnchorMarker(t *testing.T) {
	got, err := AnchorMarker("preview/deploy")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "<!-- pipekit:preview/deploy -->" {
		t.Fatalf("unexpected marker: %q", got)
	}
}

func TestAnchorMarkerRejectsUnsafeNames(t *testing.T) {
	for _, name := range []string{"", "bad name", "bad--name", "bad<name", "bad\nname"} {
		if _, err := AnchorMarker(name); err == nil {
			t.Fatalf("expected error for %q", name)
		}
	}
}

func TestFindHiddenAnchors(t *testing.T) {
	body := "visible\n<!-- pipekit:preview -->\n<!--pipekit:test.case-->\n"
	anchors := FindHiddenAnchors(body)
	if len(anchors) != 2 {
		t.Fatalf("expected 2 anchors, got %d", len(anchors))
	}
	if anchors[0].Name != "preview" || anchors[1].Name != "test.case" {
		t.Fatalf("unexpected anchors: %#v", anchors)
	}
}

func TestFindHiddenAnchorsIgnoresRegularHTMLComments(t *testing.T) {
	body := "<!-- preview -->\n<!-- pipekit:preview -->\n<!-- pipekit:bad name -->\n"
	anchors := FindHiddenAnchors(body)
	if len(anchors) != 1 {
		t.Fatalf("expected 1 valid pipekit anchor, got %d", len(anchors))
	}
	if anchors[0].Name != "preview" {
		t.Fatalf("unexpected anchor: %#v", anchors[0])
	}
}

func TestRenderAnchoredComment(t *testing.T) {
	got, err := RenderAnchoredComment("ci/status", "## Status\n\nReady\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "<!-- pipekit:ci/status -->\n\n## Status\n\nReady\n"
	if got != want {
		t.Fatalf("unexpected body:\n%s", got)
	}
}

func TestGitHubCommentPayloadEscapesMarkdownBody(t *testing.T) {
	got, err := GitHubCommentPayload("line 1\n```yaml\nx: y\n```\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"body":"line 1\n` + "```yaml\\nx: y\\n```" + `\n"}`
	if got != want {
		t.Fatalf("unexpected payload: %s", got)
	}
}

func TestAmendAnchoredCommentReplacesBodyAfterAnchor(t *testing.T) {
	existing := "prefix\n<!-- pipekit:preview -->\n\nold body\n```yaml\nold: true\n```\n"
	got, err := AmendAnchoredComment(existing, "preview", "new body\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "prefix\n<!-- pipekit:preview -->\n\nnew body\n"
	if got != want {
		t.Fatalf("unexpected amended body:\n%s", got)
	}
}

func TestAmendAnchoredCommentUsesExactAnchor(t *testing.T) {
	existing := "<!-- pipekit:preview-db -->\n\nkeep db\n"
	got, err := AmendAnchoredComment(existing, "preview", "new preview")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "<!-- pipekit:preview -->\n\nnew preview\n"
	if got != want {
		t.Fatalf("expected new exact-anchor body, got:\n%s", got)
	}
}

func TestAmendAnchoredCommentCreatesWhenMissing(t *testing.T) {
	got, err := AmendAnchoredComment("old", "preview", "new")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "<!-- pipekit:preview -->\n\nnew\n" {
		t.Fatalf("unexpected body: %q", got)
	}
}

func TestRenderCodeFenceUsesMinimumFenceForPlainContent(t *testing.T) {
	got := RenderCodeFence("yaml", "url: https://example.com\n")
	want := "```yaml\nurl: https://example.com\n```\n"
	if got != want {
		t.Fatalf("unexpected fence:\n%s", got)
	}
}

func TestRenderCodeFenceExtendsFenceForNestedBackticks(t *testing.T) {
	got := RenderCodeFence("md", "before\n```yaml\nx: y\n```\nafter")
	if !strings.HasPrefix(got, "````md\n") {
		t.Fatalf("expected four-backtick opening fence, got:\n%s", got)
	}
	if !strings.HasSuffix(got, "\n````\n") {
		t.Fatalf("expected four-backtick closing fence, got:\n%s", got)
	}
}

func TestInspectCommentsRawMarkdownFallback(t *testing.T) {
	input := "<!-- pipekit:preview -->\n\n```yaml\nstatus: ready\n```\n"
	comments, err := InspectComments(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("expected one markdown comment, got %d", len(comments))
	}
	if comments[0].ID != "" || comments[0].URL != "" {
		t.Fatalf("raw markdown should not have GitHub metadata: %#v", comments[0])
	}
	if len(comments[0].Anchors) != 1 || len(comments[0].Blocks) != 1 {
		t.Fatalf("unexpected inspection: %#v", comments[0])
	}
}

func TestInspectMarkdownComment(t *testing.T) {
	body := "<!-- pipekit:preview -->\n\n```yaml\nurl: https://example.com\n```\n"
	got, err := InspectMarkdownComment(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Anchors) != 1 || got.Anchors[0].Name != "preview" {
		t.Fatalf("unexpected anchors: %#v", got.Anchors)
	}
	if len(got.Blocks) != 1 || got.Blocks[0].Language != "yaml" {
		t.Fatalf("unexpected blocks: %#v", got.Blocks)
	}
}

func TestInspectCommentsMalformedGitHubJSONReturnsError(t *testing.T) {
	input := `[{"id": 1, "body": "<!-- pipekit:preview -->"}`
	if _, err := InspectComments(strings.NewReader(input)); err == nil {
		t.Fatal("expected malformed JSON error")
	}
}

func TestInspectCommentsGitHubArrayAndSelect(t *testing.T) {
	input := "[\n" +
		`{"id": 101, "html_url": "https://github.test/1", "user": {"login": "bot"}, "body": "plain"},` + "\n" +
		`{"id": 102, "html_url": "https://github.test/2", "user": {"login": "bot"}, "body": "<!-- pipekit:preview -->\n\n` + "```js\\nconsole.log(1)\\n```" + `"}`
	input += "\n]"
	comments, err := InspectComments(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(comments))
	}
	selected, ok := SelectAnchoredComment(comments, "preview")
	if !ok {
		t.Fatal("expected selected comment")
	}
	if selected.ID != "102" || selected.URL != "https://github.test/2" || selected.Author != "bot" {
		t.Fatalf("unexpected selected metadata: %#v", selected)
	}
	if len(selected.Blocks) != 1 || selected.Blocks[0].Language != "js" {
		t.Fatalf("unexpected selected blocks: %#v", selected.Blocks)
	}
}

func TestSelectAnchoredCommentReturnsFirstExactMatch(t *testing.T) {
	comments := []CommentInspection{
		{
			ID:      "1",
			Anchors: []HiddenAnchor{{Name: "preview-db"}},
		},
		{
			ID:      "2",
			Anchors: []HiddenAnchor{{Name: "preview"}},
		},
		{
			ID:      "3",
			Anchors: []HiddenAnchor{{Name: "preview"}},
		},
	}
	selected, ok := SelectAnchoredComment(comments, "preview")
	if !ok {
		t.Fatal("expected exact match")
	}
	if selected.ID != "2" {
		t.Fatalf("expected first exact match id 2, got %q", selected.ID)
	}
}

func TestSelectAnchoredCommentNoMatch(t *testing.T) {
	comments := []CommentInspection{{ID: "1", Anchors: []HiddenAnchor{{Name: "other"}}}}
	if selected, ok := SelectAnchoredComment(comments, "preview"); ok {
		t.Fatalf("expected no match, got %#v", selected)
	}
}
