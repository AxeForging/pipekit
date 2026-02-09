package services

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/AxeForging/pipekit/helpers"
)

// RetryRun executes a command with retry logic.
func RetryRun(command []string, attempts int, delay time.Duration, backoff bool, exitCodes []int, quiet bool) error {
	if len(command) == 0 {
		return fmt.Errorf("no command specified")
	}

	currentDelay := delay
	var lastErr error

	for i := 1; i <= attempts; i++ {
		cmd := exec.Command(command[0], command[1:]...)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if !quiet {
			cmd.Stdout = nil // Let output through
			cmd.Stderr = nil
		}

		// Use combined output to capture and pass through
		output, err := cmd.CombinedOutput()
		if !quiet && len(output) > 0 {
			fmt.Print(string(output))
		}

		if err == nil {
			if !quiet && i > 1 {
				helpers.Log.Info().Msgf("Command succeeded on attempt %d", i)
			}
			return nil
		}

		lastErr = err

		// Check if we should retry based on exit code
		if len(exitCodes) > 0 {
			exitCode := getExitCode(err)
			shouldRetry := false
			for _, code := range exitCodes {
				if exitCode == code {
					shouldRetry = true
					break
				}
			}
			if !shouldRetry {
				return fmt.Errorf("command failed with exit code %d (not in retry list): %w", exitCode, err)
			}
		}

		if i < attempts {
			if !quiet {
				helpers.Log.Warn().Msgf("Attempt %d/%d failed, retrying in %s...", i, attempts, currentDelay)
			}
			time.Sleep(currentDelay)
			if backoff {
				currentDelay = currentDelay * 2
			}
		}
	}

	return fmt.Errorf("command failed after %d attempts: %w", attempts, lastErr)
}

func getExitCode(err error) int {
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	return -1
}
