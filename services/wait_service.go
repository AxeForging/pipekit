package services

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/AxeForging/pipekit/helpers"
)

// WaitForURL polls a URL until it returns an expected status code.
func WaitForURL(ctx context.Context, urlStr string, expectedCodes []int, expectedBody string, interval time.Duration, backoff bool, quiet bool) error {
	attempt := 0
	delay := interval
	client := &http.Client{Timeout: 10 * time.Second}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for %s", urlStr)
		default:
		}

		attempt++
		ready, err := tryURL(client, urlStr, expectedCodes, expectedBody)
		if err == nil && ready {
			if !quiet {
				helpers.Log.Info().Msgf("URL %s is ready (attempt %d)", urlStr, attempt)
			}
			return nil
		}

		if !quiet {
			helpers.Log.Info().Msgf("Waiting for %s (attempt %d)...", urlStr, attempt)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for %s after %d attempts", urlStr, attempt)
		case <-time.After(delay):
		}

		if backoff {
			delay = delay * 2
		}
	}
}

// tryURL performs one probe and always closes the response body before
// returning, so the polling loop in WaitForURL doesn't leak connections.
func tryURL(client *http.Client, urlStr string, expectedCodes []int, expectedBody string) (bool, error) {
	resp, err := client.Get(urlStr)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	codeMatch := false
	for _, code := range expectedCodes {
		if resp.StatusCode == code {
			codeMatch = true
			break
		}
	}
	if !codeMatch {
		return false, nil
	}
	if expectedBody == "" {
		return true, nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	return strings.Contains(string(body), expectedBody), nil
}

// WaitForTCP polls a TCP address until a connection can be established.
func WaitForTCP(ctx context.Context, address string, interval time.Duration, backoff bool, quiet bool) error {
	attempt := 0
	delay := interval

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for TCP %s", address)
		default:
		}

		attempt++
		conn, err := net.DialTimeout("tcp", address, 5*time.Second)
		if err == nil {
			conn.Close()
			if !quiet {
				helpers.Log.Info().Msgf("TCP %s is ready (attempt %d)", address, attempt)
			}
			return nil
		}

		if !quiet {
			helpers.Log.Info().Msgf("Waiting for TCP %s (attempt %d)...", address, attempt)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for TCP %s after %d attempts", address, attempt)
		case <-time.After(delay):
		}

		if backoff {
			delay = delay * 2
		}
	}
}

// WaitForCommand retries a shell command until it exits 0.
func WaitForCommand(ctx context.Context, command string, interval time.Duration, backoff bool, quiet bool) error {
	attempt := 0
	delay := interval

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for command %q", command)
		default:
		}

		attempt++
		cmd := exec.CommandContext(ctx, "sh", "-c", command)
		if err := cmd.Run(); err == nil {
			if !quiet {
				helpers.Log.Info().Msgf("Command succeeded (attempt %d)", attempt)
			}
			return nil
		}

		if !quiet {
			helpers.Log.Info().Msgf("Command failed, retrying (attempt %d)...", attempt)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for command %q after %d attempts", command, attempt)
		case <-time.After(delay):
		}

		if backoff {
			delay = delay * 2
		}
	}
}
