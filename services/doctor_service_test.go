package services

import (
	"strings"
	"testing"
)

func TestDetectCI_GithubActions(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("GITLAB_CI", "")
	t.Setenv("JENKINS_URL", "")
	t.Setenv("CI", "")
	if got := DetectCI(); got.Name != "github-actions" {
		t.Errorf("got %v", got)
	}
}

func TestDetectCI_None(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("GITLAB_CI", "")
	t.Setenv("JENKINS_URL", "")
	t.Setenv("BUILDKITE", "")
	t.Setenv("CIRCLECI", "")
	t.Setenv("CI", "")
	if got := DetectCI(); got.Name != "none" {
		t.Errorf("got %v", got)
	}
}

func TestRunDoctorChecks_HasSections(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("CI", "")
	results := RunDoctorChecks()
	if len(results) == 0 {
		t.Fatal("no results")
	}
	sections := map[string]bool{}
	for _, r := range results {
		sections[r.Section] = true
	}
	for _, want := range []string{"platform", "tools", "webhooks"} {
		if !sections[want] {
			t.Errorf("missing section %s", want)
		}
	}
}

func TestFormatDoctorText(t *testing.T) {
	results := []CheckResult{
		{Name: "go runtime", Status: "info", Detail: "go1.24", Section: "platform"},
		{Name: "git on PATH", Status: "ok", Section: "tools"},
	}
	out := FormatDoctorText(results)
	if !strings.Contains(out, "[platform]") || !strings.Contains(out, "[tools]") {
		t.Errorf("missing section headers: %s", out)
	}
	if !strings.Contains(out, "go runtime") {
		t.Errorf("missing entry: %s", out)
	}
}
