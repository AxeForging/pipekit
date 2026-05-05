package services

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
)

// CheckResult is one line of `pipekit doctor` output.
type CheckResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // ok, warn, fail, info
	Detail  string `json:"detail,omitempty"`
	Section string `json:"section"`
}

// CIPlatform holds the detected CI environment, if any.
type CIPlatform struct {
	Name string `json:"name,omitempty"`
	Hint string `json:"hint,omitempty"`
}

// DetectCI returns the CI platform pipekit is running on, based on env vars.
func DetectCI() CIPlatform {
	switch {
	case os.Getenv("GITHUB_ACTIONS") == "true":
		return CIPlatform{Name: "github-actions"}
	case os.Getenv("GITLAB_CI") == "true":
		return CIPlatform{Name: "gitlab-ci"}
	case os.Getenv("BUILDKITE") == "true":
		return CIPlatform{Name: "buildkite"}
	case os.Getenv("CIRCLECI") == "true":
		return CIPlatform{Name: "circleci"}
	case os.Getenv("JENKINS_URL") != "":
		return CIPlatform{Name: "jenkins"}
	case os.Getenv("CI") == "true":
		return CIPlatform{Name: "ci-generic"}
	}
	return CIPlatform{Name: "none", Hint: "running locally"}
}

// RunDoctorChecks runs all environment checks and returns the results.
func RunDoctorChecks() []CheckResult {
	var out []CheckResult

	// Platform
	out = append(out, CheckResult{
		Name:    "go runtime",
		Status:  "info",
		Detail:  fmt.Sprintf("%s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH),
		Section: "platform",
	})

	// CI detection
	ci := DetectCI()
	out = append(out, CheckResult{
		Name:    "ci platform",
		Status:  "info",
		Detail:  ci.Name,
		Section: "platform",
	})

	// Git availability
	if _, err := exec.LookPath("git"); err == nil {
		out = append(out, CheckResult{Name: "git on PATH", Status: "ok", Section: "tools"})
	} else {
		out = append(out, CheckResult{
			Name: "git on PATH", Status: "warn",
			Detail:  "diff and version next will fail without git",
			Section: "tools",
		})
	}

	// CI variable contract per platform
	switch ci.Name {
	case "github-actions":
		out = append(out, checkGithubVars()...)
	case "gitlab-ci":
		out = append(out, checkGitlabVars()...)
	}

	// Webhook env vars (informational; not required)
	for _, v := range []string{"SLACK_WEBHOOK_URL", "DISCORD_WEBHOOK_URL", "TEAMS_WEBHOOK_URL"} {
		status := "info"
		detail := "not set"
		if os.Getenv(v) != "" {
			detail = "set"
		}
		out = append(out, CheckResult{Name: v, Status: status, Detail: detail, Section: "webhooks"})
	}

	// Stable order within sections.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Section != out[j].Section {
			return sectionRank(out[i].Section) < sectionRank(out[j].Section)
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func checkGithubVars() []CheckResult {
	vars := []string{"GITHUB_ENV", "GITHUB_OUTPUT", "GITHUB_STEP_SUMMARY", "GITHUB_REF", "GITHUB_REPOSITORY"}
	var out []CheckResult
	for _, v := range vars {
		val := os.Getenv(v)
		status := "ok"
		detail := "set"
		if val == "" {
			status = "warn"
			detail = "not set"
		}
		out = append(out, CheckResult{Name: v, Status: status, Detail: detail, Section: "ci-vars"})
	}
	return out
}

func checkGitlabVars() []CheckResult {
	vars := []string{"CI_PROJECT_ID", "CI_COMMIT_REF_NAME", "CI_PIPELINE_ID"}
	var out []CheckResult
	for _, v := range vars {
		val := os.Getenv(v)
		status := "ok"
		detail := "set"
		if val == "" {
			status = "warn"
			detail = "not set"
		}
		out = append(out, CheckResult{Name: v, Status: status, Detail: detail, Section: "ci-vars"})
	}
	return out
}

func sectionRank(s string) int {
	switch s {
	case "platform":
		return 0
	case "tools":
		return 1
	case "ci-vars":
		return 2
	case "webhooks":
		return 3
	default:
		return 99
	}
}

// FormatDoctorText renders results as a human-readable table grouped by section.
func FormatDoctorText(results []CheckResult) string {
	var b strings.Builder
	currentSection := ""
	for _, r := range results {
		if r.Section != currentSection {
			if currentSection != "" {
				b.WriteString("\n")
			}
			b.WriteString(fmt.Sprintf("[%s]\n", r.Section))
			currentSection = r.Section
		}
		marker := "  "
		switch r.Status {
		case "ok":
			marker = "✓ "
		case "warn":
			marker = "⚠ "
		case "fail":
			marker = "✗ "
		case "info":
			marker = "· "
		}
		if r.Detail != "" {
			b.WriteString(fmt.Sprintf("%s%-22s %s\n", marker, r.Name, r.Detail))
		} else {
			b.WriteString(fmt.Sprintf("%s%s\n", marker, r.Name))
		}
	}
	return b.String()
}
