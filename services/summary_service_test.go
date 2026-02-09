package services

import (
	"strings"
	"testing"
)

func TestGenerateTable_JSON(t *testing.T) {
	input := `[{"name": "api", "version": "1.0"}, {"name": "web", "version": "2.0"}]`
	table, err := GenerateTable(strings.NewReader(input), "Services", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(table, "## Services") {
		t.Error("expected title in table")
	}
	if !strings.Contains(table, "| ---") {
		t.Error("expected separator row in table")
	}
	if !strings.Contains(table, "api") || !strings.Contains(table, "web") {
		t.Error("expected data rows in table")
	}
}

func TestGenerateTable_CSV(t *testing.T) {
	input := "name,version\napi,1.0\nweb,2.0\n"
	table, err := GenerateTable(strings.NewReader(input), "", "csv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(table, "| name") {
		t.Error("expected header row in table")
	}
	if !strings.Contains(table, "| api") {
		t.Error("expected data rows in table")
	}
}

func TestGenerateTable_Empty(t *testing.T) {
	_, err := GenerateTable(strings.NewReader("[]"), "", "json")
	if err == nil {
		t.Error("expected error for empty JSON array")
	}
}

func TestGenerateBadge(t *testing.T) {
	tests := []struct {
		label    string
		status   string
		contains string
	}{
		{"Build", "success", ":white_check_mark:"},
		{"Test", "failure", ":x:"},
		{"Lint", "warning", ":warning:"},
		{"Info", "info", ":information_source:"},
	}
	for _, tc := range tests {
		badge := GenerateBadge(tc.label, tc.status)
		if !strings.Contains(badge, tc.contains) {
			t.Errorf("GenerateBadge(%q, %q) = %q, expected to contain %q", tc.label, tc.status, badge, tc.contains)
		}
		if !strings.Contains(badge, tc.label) {
			t.Errorf("expected badge to contain label %q", tc.label)
		}
	}
}

func TestGenerateSection(t *testing.T) {
	section := GenerateSection("Logs", "some log content")
	if !strings.Contains(section, "<details>") {
		t.Error("expected <details> tag")
	}
	if !strings.Contains(section, "<summary>Logs</summary>") {
		t.Error("expected <summary> with title")
	}
	if !strings.Contains(section, "some log content") {
		t.Error("expected body content")
	}
	if !strings.Contains(section, "</details>") {
		t.Error("expected closing </details> tag")
	}
}
