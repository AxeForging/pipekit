package services

import (
	"fmt"
	"os/exec"
	"strings"
)

// ChangelogOptions configures changelog generation.
type ChangelogOptions struct {
	From         string
	To           string
	Conventional bool
}

// ChangelogEntry is one git commit line used for release notes.
type ChangelogEntry struct {
	SHA     string `json:"sha"`
	Subject string `json:"subject"`
	Group   string `json:"group"`
}

// GenerateChangelog returns markdown release notes for a git range.
func GenerateChangelog(opts ChangelogOptions) (string, []ChangelogEntry, error) {
	if opts.To == "" {
		opts.To = "HEAD"
	}
	rangeArg := opts.To
	if opts.From != "" {
		rangeArg = opts.From + ".." + opts.To
	}
	out, err := exec.Command("git", "log", "--pretty=format:%h%x09%s", rangeArg).Output()
	if err != nil {
		return "", nil, fmt.Errorf("git log failed: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var entries []ChangelogEntry
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		entry := ChangelogEntry{SHA: parts[0], Subject: parts[1], Group: "Other"}
		if opts.Conventional {
			entry.Group = conventionalGroup(parts[1])
		}
		entries = append(entries, entry)
	}
	return FormatChangelogMarkdown(entries, opts.Conventional), entries, nil
}

// FormatChangelogMarkdown renders entries as release-note markdown.
func FormatChangelogMarkdown(entries []ChangelogEntry, grouped bool) string {
	if len(entries) == 0 {
		return "No changes.\n"
	}
	if !grouped {
		var b strings.Builder
		for _, e := range entries {
			fmt.Fprintf(&b, "- %s (%s)\n", e.Subject, e.SHA)
		}
		return b.String()
	}
	order := []string{"Breaking Changes", "Features", "Fixes", "Documentation", "Maintenance", "Other"}
	byGroup := make(map[string][]ChangelogEntry)
	for _, e := range entries {
		byGroup[e.Group] = append(byGroup[e.Group], e)
	}
	var b strings.Builder
	for _, group := range order {
		groupEntries := byGroup[group]
		if len(groupEntries) == 0 {
			continue
		}
		fmt.Fprintf(&b, "### %s\n\n", group)
		for _, e := range groupEntries {
			fmt.Fprintf(&b, "- %s (%s)\n", e.Subject, e.SHA)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func conventionalGroup(subject string) string {
	lower := strings.ToLower(subject)
	if strings.Contains(lower, "!:") || strings.Contains(lower, "breaking change") {
		return "Breaking Changes"
	}
	switch {
	case strings.HasPrefix(lower, "feat:") || strings.HasPrefix(lower, "feat("):
		return "Features"
	case strings.HasPrefix(lower, "fix:") || strings.HasPrefix(lower, "fix("):
		return "Fixes"
	case strings.HasPrefix(lower, "docs:") || strings.HasPrefix(lower, "docs("):
		return "Documentation"
	case strings.HasPrefix(lower, "chore:") || strings.HasPrefix(lower, "ci:") || strings.HasPrefix(lower, "test:") || strings.HasPrefix(lower, "refactor:"):
		return "Maintenance"
	default:
		return "Other"
	}
}
