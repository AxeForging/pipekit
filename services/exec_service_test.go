package services

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestRun_Success(t *testing.T) {
	var stdout, stderr bytes.Buffer
	res, err := Run(context.Background(), ExecOptions{
		Command: []string{"sh", "-c", "echo hello"},
		Stdout:  &stdout,
		Stderr:  &stderr,
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !res.Success {
		t.Errorf("expected success")
	}
	if res.Attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", res.Attempts)
	}
	if !strings.Contains(stdout.String(), "hello") {
		t.Errorf("stdout: %q", stdout.String())
	}
}

func TestRun_RetriesUntilSuccess(t *testing.T) {
	tmp := t.TempDir() + "/counter"
	// `sh -c "n=$(...); ... exit X"` to fail twice, succeed third.
	script := `if [ -f "` + tmp + `" ]; then
  n=$(cat "` + tmp + `")
else
  n=0
fi
n=$((n+1))
echo $n > "` + tmp + `"
if [ $n -lt 3 ]; then exit 1; fi
echo done`

	var stdout bytes.Buffer
	res, err := Run(context.Background(), ExecOptions{
		Command:  []string{"sh", "-c", script},
		Attempts: 5,
		Delay:    10 * time.Millisecond,
		Stdout:   &stdout,
		Stderr:   &bytes.Buffer{},
	})
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if res.Attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", res.Attempts)
	}
}

func TestRun_FailsAfterAttempts(t *testing.T) {
	res, err := Run(context.Background(), ExecOptions{
		Command:  []string{"false"},
		Attempts: 3,
		Delay:    10 * time.Millisecond,
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if res.Attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", res.Attempts)
	}
	if res.ExitCode == 0 {
		t.Errorf("expected non-zero exit, got %d", res.ExitCode)
	}
}

func TestRun_MasksStdout(t *testing.T) {
	patterns, err := CompilePatterns([]string{`secret-[a-z0-9]+`})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	var stdout bytes.Buffer
	_, err = Run(context.Background(), ExecOptions{
		Command:     []string{"sh", "-c", `echo "token is secret-abc123"`},
		MaskRegexes: patterns,
		Stdout:      &stdout,
		Stderr:      &bytes.Buffer{},
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if strings.Contains(stdout.String(), "secret-abc123") {
		t.Errorf("token leaked: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "***") {
		t.Errorf("expected mask: %q", stdout.String())
	}
}

func TestRun_RetryOnStderrPattern(t *testing.T) {
	// Script always fails with "rate limited". With retry-on regex matching,
	// it should retry up to attempts times.
	script := `echo "rate limited" >&2; exit 1`
	res, err := Run(context.Background(), ExecOptions{
		Command:  []string{"sh", "-c", script},
		Attempts: 3,
		Delay:    10 * time.Millisecond,
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		RetryOn:  regexp.MustCompile("rate limited"),
	})
	if err == nil {
		t.Fatal("expected eventual failure")
	}
	if res.Attempts != 3 {
		t.Errorf("expected 3 attempts (matched retry pattern), got %d", res.Attempts)
	}

	// Same script but stderr doesn't match — should bail after first attempt.
	res2, err := Run(context.Background(), ExecOptions{
		Command:  []string{"sh", "-c", script},
		Attempts: 3,
		Delay:    10 * time.Millisecond,
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		RetryOn:  regexp.MustCompile("not-this-pattern"),
	})
	if err == nil {
		t.Fatal("expected failure")
	}
	if res2.Attempts != 1 {
		t.Errorf("expected 1 attempt (stderr did not match), got %d", res2.Attempts)
	}
}
