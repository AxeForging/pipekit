package services

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// AppendToGitHubSummary appends markdown content to the GITHUB_STEP_SUMMARY file.
func AppendToGitHubSummary(content string) error {
	path := os.Getenv("GITHUB_STEP_SUMMARY")
	if path == "" {
		return fmt.Errorf("$GITHUB_STEP_SUMMARY is not set")
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("opening summary file: %w", err)
	}
	defer f.Close()
	_, err = fmt.Fprintln(f, content)
	return err
}

// GenerateTable creates a Markdown table from JSON or CSV data.
func GenerateTable(r io.Reader, title string, inputFormat string) (string, error) {
	var headers []string
	var rows [][]string

	switch strings.ToLower(inputFormat) {
	case "json":
		var data []map[string]interface{}
		if err := json.NewDecoder(r).Decode(&data); err != nil {
			return "", fmt.Errorf("parsing JSON: %w", err)
		}
		if len(data) == 0 {
			return "", fmt.Errorf("empty JSON array")
		}
		// Collect headers from first item
		for k := range data[0] {
			headers = append(headers, k)
		}
		for _, row := range data {
			var vals []string
			for _, h := range headers {
				vals = append(vals, fmt.Sprintf("%v", row[h]))
			}
			rows = append(rows, vals)
		}
	case "csv":
		reader := csv.NewReader(r)
		records, err := reader.ReadAll()
		if err != nil {
			return "", fmt.Errorf("parsing CSV: %w", err)
		}
		if len(records) == 0 {
			return "", fmt.Errorf("empty CSV")
		}
		headers = records[0]
		rows = records[1:]
	default:
		return "", fmt.Errorf("unsupported format: %s (use json or csv)", inputFormat)
	}

	var b strings.Builder
	if title != "" {
		fmt.Fprintf(&b, "## %s\n\n", title)
	}
	// Header row
	fmt.Fprintf(&b, "| %s |\n", strings.Join(headers, " | "))
	// Separator
	seps := make([]string, len(headers))
	for i := range seps {
		seps[i] = "---"
	}
	fmt.Fprintf(&b, "| %s |\n", strings.Join(seps, " | "))
	// Data rows
	for _, row := range rows {
		fmt.Fprintf(&b, "| %s |\n", strings.Join(row, " | "))
	}
	return b.String(), nil
}

// GenerateBadge creates a status badge markdown line.
func GenerateBadge(label, status string) string {
	var emoji string
	switch strings.ToLower(status) {
	case "success", "pass", "passed":
		emoji = ":white_check_mark:"
	case "failure", "fail", "failed":
		emoji = ":x:"
	case "warning", "warn":
		emoji = ":warning:"
	default:
		emoji = ":information_source:"
	}
	return fmt.Sprintf("%s **%s**: %s", emoji, label, status)
}

// GenerateSection creates a collapsible details section.
func GenerateSection(title, body string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<details>\n<summary>%s</summary>\n\n", title)
	fmt.Fprintln(&b, body)
	fmt.Fprintln(&b, "\n</details>")
	return b.String()
}
