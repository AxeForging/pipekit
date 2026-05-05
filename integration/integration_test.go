// Package integration runs end-to-end tests against the built pipekit
// binary. Tests are skipped if the binary isn't available (so `go test ./...`
// alone doesn't fail on a fresh checkout). Run `make build` first, then:
//
//	go test ./integration/... -v
package integration

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// binaryPath finds dist/pipekit relative to the project root.
func binaryPath(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Dir(filepath.Dir(thisFile))
	bin := filepath.Join(root, "dist", "pipekit")
	if _, err := os.Stat(bin); err != nil {
		t.Skipf("dist/pipekit not built (run `make build` first): %v", err)
	}
	return bin
}

// runPipekit runs the built binary with given args (and optional stdin).
// Returns stdout, stderr, exit code.
func runPipekit(t *testing.T, args []string, stdin string, env ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(binaryPath(t), args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		exitCode = -1
	}
	return stdout.String(), stderr.String(), exitCode
}

func TestE2E_EnvFromJSON(t *testing.T) {
	stdout, _, code := runPipekit(t,
		[]string{"env", "from-json", "--uppercase-keys"},
		`{"name":"pipekit","version":"1.0.0"}`)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stdout, `export NAME="pipekit"`) {
		t.Errorf("stdout: %q", stdout)
	}
}

// Regression for env_service: a value containing the legacy EOF_PIPEKIT
// delimiter literal must not corrupt $GITHUB_ENV output.
func TestE2E_EnvHeredocCollisionRegression(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, "env")

	stdin := `{"X":"line1\nEOF_PIPEKIT\nline3"}`
	_, stderr, code := runPipekit(t,
		[]string{"env", "from-json", "--to-github"},
		stdin,
		"GITHUB_ENV="+envFile)
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr)
	}
	out, _ := os.ReadFile(envFile)
	got := string(out)
	if !strings.Contains(got, "PIPEKIT_EOF_") {
		t.Errorf("expected unique heredoc, got: %s", got)
	}
	// The body must contain all three lines intact.
	for _, want := range []string{"line1", "EOF_PIPEKIT", "line3"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in: %s", want, got)
		}
	}
}

func TestE2E_AssertSemver(t *testing.T) {
	_, _, ok := runPipekit(t, []string{"assert", "semver", "1.2.3"}, "")
	if ok != 0 {
		t.Errorf("valid semver should exit 0, got %d", ok)
	}
	_, _, bad := runPipekit(t, []string{"assert", "semver", "not-a-version"}, "")
	if bad == 0 {
		t.Error("invalid semver should exit non-zero")
	}
}

func TestE2E_JSONGetSetMerge(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "base.json")
	overlay := filepath.Join(dir, "overlay.json")
	os.WriteFile(base, []byte(`{"image":{"tag":"v1.0.0","repo":"old"},"keep":"yes"}`), 0644)
	os.WriteFile(overlay, []byte(`{"image":{"tag":"v2.0.0"},"new":"added"}`), 0644)

	// get
	stdout, _, code := runPipekit(t,
		[]string{"json", "get", base, "--path", ".image.tag", "--raw"}, "")
	if code != 0 || strings.TrimSpace(stdout) != "v1.0.0" {
		t.Errorf("get: exit %d stdout %q", code, stdout)
	}

	// merge
	stdout, _, code = runPipekit(t,
		[]string{"json", "merge", base, overlay, "--pretty"}, "")
	if code != 0 || !strings.Contains(stdout, `"v2.0.0"`) || !strings.Contains(stdout, `"keep"`) {
		t.Errorf("merge: %q", stdout)
	}

	// set --in-place
	_, _, code = runPipekit(t,
		[]string{"json", "set", base, "--path", ".image.tag", "--value", "v3.0.0", "--in-place"}, "")
	if code != 0 {
		t.Fatalf("set exit %d", code)
	}
	got, _ := os.ReadFile(base)
	if !strings.Contains(string(got), `"v3.0.0"`) {
		t.Errorf("in-place set: %s", got)
	}
}

func TestE2E_RenderFile(t *testing.T) {
	dir := t.TempDir()
	tmpl := filepath.Join(dir, "v.tpl")
	vals := filepath.Join(dir, "v.yaml")
	os.WriteFile(tmpl, []byte(`tag: {{ .Values.image.tag }}`), 0644)
	os.WriteFile(vals, []byte("image:\n  tag: v1.2.3"), 0644)

	stdout, _, code := runPipekit(t,
		[]string{"render", tmpl, "--values", vals, "--set", "image.tag=v2.0.0"}, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stdout, "tag: v2.0.0") {
		t.Errorf("got %q", stdout)
	}
}

func TestE2E_ExecMasksAndRetries(t *testing.T) {
	stdout, _, code := runPipekit(t,
		[]string{"exec", "--mask", "secret-[a-z]+", "--", "sh", "-c", "echo token=secret-abc"}, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if strings.Contains(stdout, "secret-abc") {
		t.Errorf("token leaked: %q", stdout)
	}
	if !strings.Contains(stdout, "***") {
		t.Errorf("expected mask: %q", stdout)
	}

	// Always-failing command — should retry up to N attempts.
	_, _, code = runPipekit(t,
		[]string{"exec", "--attempts", "3", "--delay", "10ms", "--", "false"}, "")
	if code == 0 {
		t.Error("expected non-zero exit")
	}
}

func TestE2E_TransformSlug(t *testing.T) {
	stdout, _, code := runPipekit(t, []string{"transform", "slug"}, "feature/My-Cool-Branch")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if strings.TrimSpace(stdout) != "feature-my-cool-branch" {
		t.Errorf("got %q", stdout)
	}
}

func TestE2E_VersionSetDoesNotRewriteDeps(t *testing.T) {
	dir := t.TempDir()
	pkg := filepath.Join(dir, "package.json")
	original := `{
  "name": "myapp",
  "dependencies": { "react": "1.2.3" },
  "version": "1.2.3"
}`
	os.WriteFile(pkg, []byte(original), 0644)

	_, _, code := runPipekit(t,
		[]string{"version", "bump", "patch", "--source", pkg}, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	got, _ := os.ReadFile(pkg)
	out := string(got)
	if !strings.Contains(out, `"react": "1.2.3"`) {
		t.Errorf("dep pin was rewritten: %s", out)
	}
	if !strings.Contains(out, `"version": "1.2.4"`) {
		t.Errorf("version not bumped: %s", out)
	}
}

func TestE2E_DiffDoubleStarGlobNote(t *testing.T) {
	// We don't have a git repo here; just smoke-test that the command runs
	// and emits a help-like signal for invalid args. Full diff matching is
	// covered in services unit tests.
	_, stderr, code := runPipekit(t, []string{"diff", "match"}, "")
	if code == 0 {
		t.Error("expected non-zero exit without args")
	}
	_ = stderr // helpful in failure mode
}

func TestE2E_URLParse(t *testing.T) {
	stdout, _, code := runPipekit(t,
		[]string{"url", "parse", "postgres://app:secret@db:5432/prod", "--prefix", "DB_"}, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{"DB_SCHEME=postgres", "DB_HOST=db", "DB_PORT=5432", "DB_USER=app"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("missing %q in %q", want, stdout)
		}
	}
}

func TestE2E_ImageParse(t *testing.T) {
	stdout, _, code := runPipekit(t,
		[]string{"image", "parse", "ghcr.io/me/app:v1.0.0@sha256:abc", "--json"}, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	expectAll(t, stdout, `"Registry":"ghcr.io"`, `"Tag":"v1.0.0"`, `"Digest":"sha256:abc"`)
}

func TestE2E_DoctorRunsAndReportsCI(t *testing.T) {
	stdout, _, code := runPipekit(t,
		[]string{"doctor"}, "",
		"GITHUB_ACTIONS=true", "GITHUB_ENV=/tmp/gh-env")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stdout, "github-actions") {
		t.Errorf("expected github-actions detection: %s", stdout)
	}
}

func expectAll(t *testing.T, haystack string, needles ...string) {
	t.Helper()
	for _, n := range needles {
		if !strings.Contains(haystack, n) {
			t.Errorf("missing %q in:\n%s", n, haystack)
		}
	}
}

// Sanity: build-info must work and report a version.
func TestE2E_BuildInfo(t *testing.T) {
	stdout, _, code := runPipekit(t, []string{"build-info"}, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stdout, "pipekit version") {
		t.Errorf("expected version line: %s", stdout)
	}
	_ = fmt.Sprintf // prevent fmt unused if we ever drop other usages
}
