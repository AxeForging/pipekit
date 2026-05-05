package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"time"
)

// ExecOptions configures a single Run() invocation.
type ExecOptions struct {
	Command     []string
	Attempts    int
	Delay       time.Duration
	Backoff     bool
	Jitter      bool
	Timeout     time.Duration // per-attempt
	MaxElapsed  time.Duration // total deadline (0 = none)
	MaskRegexes []*regexp.Regexp
	MaskRepl    string
	TeePath     string
	RetryOn     *regexp.Regexp // retry if stderr matches this
	Stdout      io.Writer
	Stderr      io.Writer
}

// ExecResult summarizes an Run() execution.
type ExecResult struct {
	Attempts int
	Success  bool
	ExitCode int
	Duration time.Duration
}

// Run executes opts.Command with retries, optional per-attempt timeouts, and
// stdout/stderr filtered through the supplied mask regexes (and optionally
// teed to a file). It returns a structured result and any final error.
func Run(parent context.Context, opts ExecOptions) (ExecResult, error) {
	if len(opts.Command) == 0 {
		return ExecResult{}, fmt.Errorf("no command specified")
	}
	if opts.Attempts < 1 {
		opts.Attempts = 1
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	repl := opts.MaskRepl
	if repl == "" {
		repl = "***"
	}

	var teeFile *os.File
	if opts.TeePath != "" {
		f, err := os.Create(opts.TeePath)
		if err != nil {
			return ExecResult{}, fmt.Errorf("opening tee file: %w", err)
		}
		teeFile = f
		defer teeFile.Close()
	}

	overall := parent
	if opts.MaxElapsed > 0 {
		var cancel context.CancelFunc
		overall, cancel = context.WithTimeout(parent, opts.MaxElapsed)
		defer cancel()
	}

	delay := opts.Delay
	start := time.Now()
	res := ExecResult{}

	for attempt := 1; attempt <= opts.Attempts; attempt++ {
		res.Attempts = attempt

		ctx := overall
		if opts.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(overall, opts.Timeout)
			defer cancel()
		}

		stderrCapture := newCircularBuffer(64 * 1024)

		// stdout: mask -> (tee + parent stdout)
		stdoutSinks := []io.Writer{opts.Stdout}
		if teeFile != nil {
			stdoutSinks = append(stdoutSinks, teeFile)
		}
		stdoutW := newMaskingWriter(io.MultiWriter(stdoutSinks...), opts.MaskRegexes, repl)

		stderrSinks := []io.Writer{opts.Stderr, stderrCapture}
		if teeFile != nil {
			stderrSinks = append(stderrSinks, teeFile)
		}
		stderrW := newMaskingWriter(io.MultiWriter(stderrSinks...), opts.MaskRegexes, repl)

		cmd := exec.CommandContext(ctx, opts.Command[0], opts.Command[1:]...)
		cmd.Stdout = stdoutW
		cmd.Stderr = stderrW

		err := cmd.Run()
		_ = stdoutW.Flush()
		_ = stderrW.Flush()

		if err == nil {
			res.Success = true
			res.ExitCode = 0
			res.Duration = time.Since(start)
			return res, nil
		}

		exitCode := 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		res.ExitCode = exitCode

		if attempt == opts.Attempts {
			res.Duration = time.Since(start)
			return res, fmt.Errorf("command failed after %d attempts: %w", attempt, err)
		}

		if opts.RetryOn != nil && !opts.RetryOn.Match(stderrCapture.Bytes()) {
			res.Duration = time.Since(start)
			return res, fmt.Errorf("command failed (stderr did not match retry pattern): %w", err)
		}

		select {
		case <-overall.Done():
			res.Duration = time.Since(start)
			return res, fmt.Errorf("max-elapsed reached after %d attempts", attempt)
		case <-time.After(applyJitter(delay, opts.Jitter)):
		}

		if opts.Backoff {
			delay *= 2
		}
	}
	res.Duration = time.Since(start)
	return res, fmt.Errorf("unreachable")
}

func applyJitter(base time.Duration, on bool) time.Duration {
	if !on || base <= 0 {
		return base
	}
	// Add up to 20% jitter using a deterministic-but-good-enough source.
	delta := time.Duration(int64(base) / 5)
	return base + time.Duration(time.Now().UnixNano()%int64(delta+1))
}

// CompilePatterns is a convenience for callers — same compilation as masking.
func CompilePatterns(patterns []string) ([]*regexp.Regexp, error) {
	return compilePatterns(patterns)
}
