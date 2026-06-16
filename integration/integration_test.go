// Package integration runs end-to-end tests against the built pipekit
// binary. Tests are skipped if the binary isn't available (so `go test ./...`
// alone doesn't fail on a fresh checkout). Run `make build` first, then:
//
//	go test ./integration/... -v
package integration

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
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

// TestE2E_YAMLSetPreserve verifies that `--preserve` performs a surgical in-place
// edit: only the targeted value changes, while comments, key order, and quoting
// of every other line are left byte-for-byte intact.
func TestE2E_YAMLSetPreserve(t *testing.T) {
	dir := t.TempDir()
	values := filepath.Join(dir, "values.yaml")
	original := `# Helm values
image:
  repository: myapp # do not touch
  tag: "v1.0.0"
replicas: 3
`
	if err := os.WriteFile(values, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	_, stderr, code := runPipekit(t,
		[]string{"yaml", "set", values, "--path", ".image.tag", "--value", "v2.0.0", "--in-place", "--preserve"}, "")
	if code != 0 {
		t.Fatalf("preserve set exit %d, stderr: %s", code, stderr)
	}
	got, _ := os.ReadFile(values)
	gotStr := string(got)

	for _, want := range []string{"# Helm values", "repository: myapp # do not touch", `tag: "v2.0.0"`, "replicas: 3"} {
		if !strings.Contains(gotStr, want) {
			t.Errorf("preserve lost %q:\n%s", want, gotStr)
		}
	}
	if strings.Contains(gotStr, "v1.0.0") {
		t.Errorf("old value should be gone:\n%s", gotStr)
	}

	// Backward-compat sanity: the same edit WITHOUT --preserve still works,
	// just normalizing formatting (comments dropped).
	if err := os.WriteFile(values, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}
	_, _, code = runPipekit(t,
		[]string{"yaml", "set", values, "--path", ".image.tag", "--value", "v2.0.0", "--in-place"}, "")
	if code != 0 {
		t.Fatalf("legacy set exit %d", code)
	}
	legacy, _ := os.ReadFile(values)
	if !strings.Contains(string(legacy), "v2.0.0") {
		t.Errorf("legacy set failed:\n%s", legacy)
	}
}

// TestE2E_JSONDelPreserve verifies surgical key removal keeps surrounding JSON
// formatting intact.
func TestE2E_JSONDelPreserve(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.json")
	original := "{\n  \"a\": 1,\n  \"b\": 2,\n  \"c\": 3\n}\n"
	if err := os.WriteFile(cfg, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}
	_, stderr, code := runPipekit(t,
		[]string{"json", "del", cfg, "--path", ".b", "--in-place", "--preserve"}, "")
	if code != 0 {
		t.Fatalf("preserve del exit %d, stderr: %s", code, stderr)
	}
	want := "{\n  \"a\": 1,\n  \"c\": 3\n}\n"
	if got, _ := os.ReadFile(cfg); string(got) != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

// TestE2E_JSONSetPreserveInsert verifies `set --preserve` can add a new key to
// an existing object, formatting-matched to its siblings.
func TestE2E_JSONSetPreserveInsert(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfg, []byte("{\n  \"a\": 1\n}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	_, stderr, code := runPipekit(t,
		[]string{"json", "set", cfg, "--path", ".b", "--json-value", "2", "--in-place", "--preserve"}, "")
	if code != 0 {
		t.Fatalf("insert exit %d, stderr: %s", code, stderr)
	}
	want := "{\n  \"a\": 1,\n  \"b\": 2\n}\n"
	if got, _ := os.ReadFile(cfg); string(got) != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

// TestE2E_PreservePreservesFileMode verifies the atomic in-place write keeps the
// original file's permission bits.
func TestE2E_PreservePreservesFileMode(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "values.yaml")
	if err := os.WriteFile(f, []byte("tag: v1\n"), 0640); err != nil {
		t.Fatal(err)
	}
	_, _, code := runPipekit(t,
		[]string{"yaml", "set", f, "--path", ".tag", "--value", "v2", "--in-place", "--preserve"}, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	info, err := os.Stat(f)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0640 {
		t.Errorf("mode changed: got %o want 640", info.Mode().Perm())
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

// Regression: README claimed `api/**` worked but filepath.Match never
// supported it. Build a real git repo in a tempdir, change a deep file,
// and verify `diff match "api/**"` exits 0.
func TestE2E_DiffDoubleStarGlobActuallyMatches(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	mustRunIn(t, dir, "git", "init", "-q")
	mustRunIn(t, dir, "git", "config", "user.email", "test@example.com")
	mustRunIn(t, dir, "git", "config", "user.name", "test")
	mustRunIn(t, dir, "git", "config", "commit.gpgsign", "false")

	// Initial commit with one root file.
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("hi"), 0644)
	mustRunIn(t, dir, "git", "add", ".")
	mustRunIn(t, dir, "git", "commit", "-q", "-m", "init")

	base := strings.TrimSpace(captureRunIn(t, dir, "git", "rev-parse", "HEAD"))

	// Change a deep file under api/.
	apiDir := filepath.Join(dir, "api", "v1")
	os.MkdirAll(apiDir, 0755)
	os.WriteFile(filepath.Join(apiDir, "handler.go"), []byte("package v1"), 0644)
	mustRunIn(t, dir, "git", "add", ".")
	mustRunIn(t, dir, "git", "commit", "-q", "-m", "deep change")

	// `diff match api/**` should exit 0 (match found).
	cmd := exec.Command(binaryPath(t), "diff", "match", "api/**", "--base", base)
	cmd.Dir = dir
	var so, se bytes.Buffer
	cmd.Stdout, cmd.Stderr = &so, &se
	err := cmd.Run()
	if err != nil {
		t.Fatalf("diff match api/** should exit 0, got err=%v\nstdout=%s\nstderr=%s", err, so.String(), se.String())
	}

	// And `diff match nope/**` should exit non-zero.
	cmd2 := exec.Command(binaryPath(t), "diff", "match", "nope/**", "--base", base)
	cmd2.Dir = dir
	cmd2.Stdout, cmd2.Stderr = &bytes.Buffer{}, &bytes.Buffer{}
	if err := cmd2.Run(); err == nil {
		t.Error("expected non-zero exit for non-matching glob")
	}
}

func TestE2E_RenderOutputFile(t *testing.T) {
	dir := t.TempDir()
	tmpl := filepath.Join(dir, "v.tpl")
	out := filepath.Join(dir, "out.yaml")
	os.WriteFile(tmpl, []byte(`tag: {{ .Values.tag }}`), 0644)

	_, _, code := runPipekit(t,
		[]string{"render", tmpl, "--set", "tag=v9", "--output", out}, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("output file not written: %v", err)
	}
	if !strings.Contains(string(got), "tag: v9") {
		t.Errorf("output mismatch: %s", got)
	}
}

func TestE2E_ExecTeeWritesFile(t *testing.T) {
	dir := t.TempDir()
	teeFile := filepath.Join(dir, "out.log")

	_, _, code := runPipekit(t,
		[]string{"exec", "--tee", teeFile, "--", "sh", "-c", "echo line1; echo line2 1>&2"}, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	got, err := os.ReadFile(teeFile)
	if err != nil {
		t.Fatalf("tee file not written: %v", err)
	}
	out := string(got)
	if !strings.Contains(out, "line1") || !strings.Contains(out, "line2") {
		t.Errorf("tee did not capture both streams: %q", out)
	}
}

func TestE2E_MaskMultilinePEM(t *testing.T) {
	pem := `before
-----BEGIN PRIVATE KEY-----
secret-body-1
secret-body-2
-----END PRIVATE KEY-----
after`

	stdout, _, code := runPipekit(t,
		[]string{"mask", "values", "--preset", "gcp", "--multiline"}, pem)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if strings.Contains(stdout, "secret-body-1") {
		t.Errorf("PEM body leaked: %s", stdout)
	}
	if !strings.Contains(stdout, "before") || !strings.Contains(stdout, "after") {
		t.Errorf("non-secret content lost: %s", stdout)
	}
}

func TestE2E_MatrixShard(t *testing.T) {
	stdout, _, code := runPipekit(t,
		[]string{"matrix", "shard", "--total", "3", "--index", "1", "a", "b", "c", "d", "e", "f"}, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	want := "b\ne\n"
	if strings.TrimSpace(stdout) != strings.TrimSpace(want) {
		t.Errorf("got %q, want %q", stdout, want)
	}
}

func TestE2E_CacheKeyWithEnv(t *testing.T) {
	dir := t.TempDir()
	lock := filepath.Join(dir, "lock")
	os.WriteFile(lock, []byte("contents"), 0644)

	a, _, _ := runPipekit(t,
		[]string{"cache-key", "from-files", lock, "--with-env", "PIPEKIT_TEST", "--length", "16"},
		"", "PIPEKIT_TEST=v1")
	b, _, _ := runPipekit(t,
		[]string{"cache-key", "from-files", lock, "--with-env", "PIPEKIT_TEST", "--length", "16"},
		"", "PIPEKIT_TEST=v2")

	if strings.TrimSpace(a) == strings.TrimSpace(b) {
		t.Errorf("cache keys should differ when --with-env value changes:\na=%s\nb=%s", a, b)
	}
	if len(strings.TrimSpace(a)) != 16 {
		t.Errorf("--length 16 not applied: %q", a)
	}
}

func TestE2E_ChecksumAndArtifact(t *testing.T) {
	dir := t.TempDir()
	dist := filepath.Join(dir, "dist")
	os.MkdirAll(dist, 0755)
	bin := filepath.Join(dist, "pipekit-linux-amd64")
	os.WriteFile(bin, []byte("binary"), 0644)

	checksums := filepath.Join(dist, "checksums.txt")
	stdout, _, code := runPipekit(t,
		[]string{"checksum", "files", bin, "--output", checksums}, "")
	if code != 0 {
		t.Fatalf("checksum files exit %d stdout=%s", code, stdout)
	}
	if _, err := os.Stat(checksums); err != nil {
		t.Fatalf("checksums not written: %v", err)
	}
	if _, _, code := runPipekit(t, []string{"checksum", "verify", checksums}, ""); code != 0 {
		t.Fatalf("checksum verify exit %d", code)
	}

	stdout, _, code = runPipekit(t,
		[]string{"artifact", "manifest", filepath.Join(dist, "pipekit-*"), "--pretty"}, "")
	if code != 0 {
		t.Fatalf("artifact manifest exit %d", code)
	}
	expectAll(t, stdout, `"path"`, `"size"`, `"sha256"`)
}

func TestE2E_HTTPGetAndChain(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/release":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"tag":"v1.2.3"}`))
		case "/token":
			if r.Method != http.MethodPost {
				t.Fatalf("token method = %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"abc123"}`))
		case "/deploys/abc123":
			if got := r.Header.Get("Authorization"); got != "Bearer abc123" {
				t.Fatalf("Authorization = %q", got)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":42}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	stdout, stderr, code := runPipekit(t,
		[]string{"http", "get", srv.URL + "/release", "--expect-status", "200", "--jq", ".tag", "--raw"}, "")
	if code != 0 {
		t.Fatalf("http get exit %d stderr=%s", code, stderr)
	}
	if strings.TrimSpace(stdout) != "v1.2.3" {
		t.Fatalf("http get stdout = %q", stdout)
	}

	dir := t.TempDir()
	plan := filepath.Join(dir, "flow.yaml")
	body := fmt.Sprintf(`steps:
  - name: auth
    method: POST
    url: %s/token
    json: '{"client":"ci"}'
    expectStatus: [200]
    capture:
      token: .access_token
  - name: deploy
    method: POST
    url: %s/deploys/{{token}}
    headers:
      Authorization: Bearer {{token}}
    json: '{"ref":"main"}'
    expectStatus: [201]
    capture:
      deploy_id: .id
`, srv.URL, srv.URL)
	if err := os.WriteFile(plan, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, code = runPipekit(t,
		[]string{"http", "chain", plan, "--expect-status", "200", "--verbose"}, "")
	if code != 0 {
		t.Fatalf("http chain exit %d stderr=%s stdout=%s", code, stderr, stdout)
	}
	expectAll(t, stdout, `"token": "abc123"`, `"deploy_id": "42"`, `"statusCode": 201`)
	expectAll(t, stderr, "auth: HTTP 200", "deploy: HTTP 201")
}

func TestE2E_HTTPRejectsInvalidExpectedStatus(t *testing.T) {
	_, stderr, code := runPipekit(t,
		[]string{"http", "get", "http://127.0.0.1", "--expect-status", "nope"}, "")
	if code == 0 {
		t.Fatal("expected invalid status to fail")
	}
	if !strings.Contains(stderr, "invalid status code") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestE2E_ArchivePackListUnpack(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.MkdirAll(filepath.Join(src, "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "nested", "file.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	archive := filepath.Join(dir, "bundle.tar.zst")
	_, stderr, code := runPipekit(t,
		[]string{"archive", "pack", archive, src}, "")
	if code != 0 {
		t.Fatalf("archive pack exit %d stderr=%s", code, stderr)
	}

	stdout, stderr, code := runPipekit(t,
		[]string{"archive", "list", archive}, "")
	if code != 0 {
		t.Fatalf("archive list exit %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "src/nested/file.txt") {
		t.Fatalf("archive list stdout = %q", stdout)
	}

	dest := filepath.Join(dir, "out")
	_, stderr, code = runPipekit(t,
		[]string{"archive", "unpack", archive, "--dest", dest, "--strip-components", "1"}, "")
	if code != 0 {
		t.Fatalf("archive unpack exit %d stderr=%s", code, stderr)
	}
	got, err := os.ReadFile(filepath.Join(dest, "nested", "file.txt"))
	if err != nil {
		t.Fatalf("read unpacked file: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("unpacked content = %q", got)
	}
}

func TestE2E_GitAndChangelog(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	mustRunIn(t, dir, "git", "init", "-q")
	mustRunIn(t, dir, "git", "config", "user.email", "test@example.com")
	mustRunIn(t, dir, "git", "config", "user.name", "test")
	mustRunIn(t, dir, "git", "config", "commit.gpgsign", "false")

	os.WriteFile(filepath.Join(dir, "README.md"), []byte("hi"), 0644)
	mustRunIn(t, dir, "git", "add", ".")
	mustRunIn(t, dir, "git", "commit", "-q", "-m", "feat: initial")
	mustRunIn(t, dir, "git", "tag", "v0.1.0")
	os.WriteFile(filepath.Join(dir, "fix.txt"), []byte("fix"), 0644)
	mustRunIn(t, dir, "git", "add", ".")
	mustRunIn(t, dir, "git", "commit", "-q", "-m", "fix: release artifact path")

	cmd := exec.Command(binaryPath(t), "git", "sha", "--short")
	cmd.Dir = dir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("git sha: %v", err)
	}
	if len(strings.TrimSpace(stdout.String())) < 7 {
		t.Fatalf("unexpected short sha: %q", stdout.String())
	}

	cmd = exec.Command(binaryPath(t), "changelog", "generate", "--from", "v0.1.0", "--conventional")
	cmd.Dir = dir
	stdout.Reset()
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("changelog: %v", err)
	}
	expectAll(t, stdout.String(), "### Fixes", "fix: release artifact path")
}

func TestE2E_ParseFrontmatter(t *testing.T) {
	input := `---
title: My Post
draft: true
---

Body content.`
	stdout, _, code := runPipekit(t,
		[]string{"parse", "extract-frontmatter", "--json"}, input)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stdout, `"title"`) || !strings.Contains(stdout, "My Post") {
		t.Errorf("frontmatter not parsed as JSON: %s", stdout)
	}
}

func TestE2E_AssertPath(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "exists")
	os.WriteFile(f, []byte("x"), 0644)

	if _, _, code := runPipekit(t, []string{"assert", "path", f, dir}, ""); code != 0 {
		t.Errorf("expected exit 0 for existing path/dir, got %d", code)
	}
	if _, _, code := runPipekit(t, []string{"assert", "path", filepath.Join(dir, "missing")}, ""); code == 0 {
		t.Error("expected non-zero for missing path")
	}
}

// Helpers for git-driven tests.
func mustRunIn(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	var se bytes.Buffer
	cmd.Stderr = &se
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, se.String())
	}
}

func captureRunIn(t *testing.T, dir, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var so bytes.Buffer
	cmd.Stdout = &so
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s %v: %v", name, args, err)
	}
	return so.String()
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
