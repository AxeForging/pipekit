package actions

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/AxeForging/pipekit/services"

	"github.com/urfave/cli"
)

// ExecCommand returns the unified exec command (retry + mask + tee + timeout).
func ExecCommand() cli.Command {
	return cli.Command{
		Name:      "exec",
		Usage:     "run a command with retries, masking, timeout, and tee — all in one",
		ArgsUsage: "-- COMMAND [ARGS...]",
		Flags: []cli.Flag{
			cli.IntFlag{Name: "attempts, a", Value: 1, Usage: "number of attempts"},
			cli.DurationFlag{Name: "delay, d", Value: 5 * time.Second, Usage: "initial delay between retries"},
			cli.BoolFlag{Name: "backoff", Usage: "double the delay after each failed attempt"},
			cli.BoolFlag{Name: "jitter", Usage: "add up to 20% jitter to retry delays"},
			cli.DurationFlag{Name: "timeout, t", Usage: "per-attempt timeout (e.g. 30s)"},
			cli.DurationFlag{Name: "max-elapsed", Usage: "total deadline across all attempts"},
			cli.StringSliceFlag{Name: "mask", Usage: "regex pattern to mask in stdout/stderr (repeatable)"},
			cli.StringFlag{Name: "mask-preset", Usage: "comma-separated presets: aws,github,gcp,jwt,slack,stripe,pem"},
			cli.StringFlag{Name: "mask-repl", Value: "***", Usage: "replacement string for masked matches"},
			cli.StringFlag{Name: "tee", Usage: "also write combined stdout/stderr to this file"},
			cli.StringFlag{Name: "retry-on-stderr", Usage: "regex; only retry when stderr matches"},
		},
		Action: func(c *cli.Context) error {
			args := c.Args()
			if len(args) == 0 {
				return cli.NewExitError("command required after -- ", 1)
			}

			patterns := append([]string{}, c.StringSlice("mask")...)
			if preset := c.String("mask-preset"); preset != "" {
				pats, unknown := services.PresetPatterns(splitCSV(preset))
				if len(unknown) > 0 {
					return cli.NewExitError(fmt.Sprintf("unknown preset(s): %v", unknown), 1)
				}
				patterns = append(patterns, pats...)
			}
			compiled, err := services.CompilePatterns(patterns)
			if err != nil {
				return err
			}

			var retryOn *regexp.Regexp
			if pat := c.String("retry-on-stderr"); pat != "" {
				re, err := regexp.Compile(pat)
				if err != nil {
					return fmt.Errorf("invalid --retry-on-stderr: %w", err)
				}
				retryOn = re
			}

			opts := services.ExecOptions{
				Command:     args,
				Attempts:    c.Int("attempts"),
				Delay:       c.Duration("delay"),
				Backoff:     c.Bool("backoff"),
				Jitter:      c.Bool("jitter"),
				Timeout:     c.Duration("timeout"),
				MaxElapsed:  c.Duration("max-elapsed"),
				MaskRegexes: compiled,
				MaskRepl:    c.String("mask-repl"),
				TeePath:     c.String("tee"),
				RetryOn:     retryOn,
			}

			res, err := services.Run(context.Background(), opts)
			if err != nil {
				return cli.NewExitError(
					fmt.Sprintf("exec failed (attempts=%d, exit=%d, duration=%s): %v",
						res.Attempts, res.ExitCode, res.Duration, err),
					res.ExitCode,
				)
			}
			return nil
		},
	}
}
