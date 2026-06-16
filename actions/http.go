package actions

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/AxeForging/pipekit/services"

	"github.com/urfave/cli"
)

// HTTPCommand returns curl-like HTTP helpers.
func HTTPCommand() cli.Command {
	return cli.Command{
		Name:  "http",
		Usage: "curl-like HTTP requests with status assertions and JSON extraction",
		Subcommands: []cli.Command{
			httpMethodCommand(http.MethodGet),
			httpMethodCommand(http.MethodPost),
			httpMethodCommand(http.MethodPut),
			httpMethodCommand(http.MethodPatch),
			httpMethodCommand(http.MethodDelete),
			httpChainCommand(),
		},
	}
}

func httpMethodCommand(method string) cli.Command {
	return cli.Command{
		Name:      strings.ToLower(method),
		Usage:     method + " an HTTP(S) URL",
		ArgsUsage: "URL",
		Flags: []cli.Flag{
			cli.StringSliceFlag{Name: "header, H", Usage: "request header as 'Name: value' (repeatable)"},
			cli.StringFlag{Name: "data, d", Usage: "raw request body"},
			cli.StringFlag{Name: "data-file", Usage: "read raw request body from file"},
			cli.StringFlag{Name: "json, j", Usage: "JSON request body"},
			cli.StringFlag{Name: "json-file", Usage: "read JSON request body from file"},
			cli.StringSliceFlag{Name: "form, F", Usage: "application/x-www-form-urlencoded field as key=value (repeatable)"},
			cli.StringSliceFlag{Name: "file", Usage: "multipart file field as name=path (repeatable)"},
			cli.StringFlag{Name: "expect-status", Usage: "comma-separated acceptable HTTP status codes"},
			cli.StringFlag{Name: "jq", Usage: "jq-style response JSON path to print"},
			cli.BoolFlag{Name: "raw, r", Usage: "print extracted strings raw"},
			cli.StringFlag{Name: "output, o", Usage: "write response body or extracted value to file"},
			outputKeyFlag(),
			cli.StringFlag{Name: "timeout", Value: "30s", Usage: "request timeout"},
			cli.IntFlag{Name: "retry", Value: 1, Usage: "number of attempts"},
			cli.StringFlag{Name: "retry-delay", Value: "1s", Usage: "delay between retry attempts"},
			cli.BoolFlag{Name: "backoff", Usage: "exponential backoff between retries"},
			cli.BoolFlag{Name: "insecure, k", Usage: "skip TLS certificate verification"},
			cli.BoolFlag{Name: "verbose, v", Usage: "print response status and headers to stderr"},
		},
		Action: func(c *cli.Context) error {
			urlStr, err := firstArgOrErr(c, "URL")
			if err != nil {
				return err
			}
			timeout, err := time.ParseDuration(c.String("timeout"))
			if err != nil {
				return cli.NewExitError("invalid timeout: "+err.Error(), 1)
			}
			retryDelay, err := time.ParseDuration(c.String("retry-delay"))
			if err != nil {
				return cli.NewExitError("invalid retry-delay: "+err.Error(), 1)
			}
			headers, err := services.ParseHTTPHeaders(c.StringSlice("header"))
			if err != nil {
				return cli.NewExitError(err.Error(), 1)
			}

			body, contentType, err := services.BuildHTTPBody(c.String("data"), c.String("data-file"), c.String("json"), c.String("json-file"), c.StringSlice("form"))
			if err != nil {
				return cli.NewExitError(err.Error(), 1)
			}
			if len(c.StringSlice("file")) > 0 {
				if body != nil {
					return cli.NewExitError("use --file without --data, --data-file, --json, --json-file, or --form", 1)
				}
				body, contentType, err = services.BuildMultipartBody(c.StringSlice("file"))
				if err != nil {
					return cli.NewExitError(err.Error(), 1)
				}
			}
			if contentType != "" {
				if _, exists := headers["Content-Type"]; !exists {
					headers["Content-Type"] = contentType
				}
			}

			expected, err := parseHTTPStatusList(c.String("expect-status"))
			if err != nil {
				return cli.NewExitError(err.Error(), 1)
			}

			res, err := services.ExecuteHTTPRequest(services.HTTPRequestOptions{
				Method:         method,
				URL:            urlStr,
				Headers:        headers,
				Body:           body,
				Timeout:        timeout,
				ExpectedStatus: expected,
				RetryAttempts:  c.Int("retry"),
				RetryDelay:     retryDelay,
				Backoff:        c.Bool("backoff"),
				InsecureTLS:    c.Bool("insecure"),
			})
			if err != nil {
				return cli.NewExitError(err.Error(), 1)
			}
			if c.Bool("verbose") {
				printHTTPVerbose(res)
			}

			out := res.Body
			if path := c.String("jq"); path != "" {
				val, err := services.ExtractHTTPJSON(res.Body, path)
				if err != nil {
					return cli.NewExitError(err.Error(), 1)
				}
				out = []byte(formatHTTPResult(val, c.Bool("raw")))
			}
			if output := c.String("output"); output != "" {
				return os.WriteFile(output, out, 0644)
			}
			if outputKey := c.String("to-github-output"); outputKey != "" {
				return services.WriteToGitHubOutputValue(outputKey, strings.TrimRight(string(out), "\n"))
			}
			fmt.Print(string(out))
			return nil
		},
	}
}

func httpChainCommand() cli.Command {
	return cli.Command{
		Name:      "chain",
		Usage:     "run a sequence of dependent HTTP requests from a JSON/YAML plan",
		ArgsUsage: "PLAN_FILE|-",
		Flags: []cli.Flag{
			cli.StringSliceFlag{Name: "header, H", Usage: "request header as 'Name: value' applied to every step"},
			cli.StringFlag{Name: "from", Usage: "plan format override for stdin or extensionless files: json or yaml"},
			cli.StringFlag{Name: "expect-status", Usage: "default acceptable HTTP status codes for steps"},
			cli.StringFlag{Name: "timeout", Value: "30s", Usage: "default request timeout"},
			cli.IntFlag{Name: "retry", Value: 1, Usage: "number of attempts per step"},
			cli.StringFlag{Name: "retry-delay", Value: "1s", Usage: "delay between retry attempts"},
			cli.BoolFlag{Name: "backoff", Usage: "exponential backoff between retries"},
			cli.BoolFlag{Name: "insecure, k", Usage: "skip TLS certificate verification"},
			cli.BoolFlag{Name: "verbose, v", Usage: "print per-step status to stderr"},
			outputKeyFlag(),
		},
		Action: func(c *cli.Context) error {
			file, err := firstArgOrErr(c, "PLAN_FILE")
			if err != nil {
				return err
			}
			timeout, err := time.ParseDuration(c.String("timeout"))
			if err != nil {
				return cli.NewExitError("invalid timeout: "+err.Error(), 1)
			}
			retryDelay, err := time.ParseDuration(c.String("retry-delay"))
			if err != nil {
				return cli.NewExitError("invalid retry-delay: "+err.Error(), 1)
			}
			expected, err := parseHTTPStatusList(c.String("expect-status"))
			if err != nil {
				return cli.NewExitError(err.Error(), 1)
			}
			headers, err := services.ParseHTTPHeaders(c.StringSlice("header"))
			if err != nil {
				return cli.NewExitError(err.Error(), 1)
			}
			data, format, err := readHTTPChainPlanInput(c, file)
			if err != nil {
				return cli.NewExitError(err.Error(), 1)
			}
			doc, err := services.Decode(data, format)
			if err != nil {
				return cli.NewExitError(err.Error(), 1)
			}
			plan, err := services.DecodeHTTPChainPlan(doc)
			if err != nil {
				return cli.NewExitError(err.Error(), 1)
			}
			result, err := services.ExecuteHTTPChain(plan, services.HTTPRequestOptions{
				Headers:        headers,
				Timeout:        timeout,
				ExpectedStatus: expected,
				RetryAttempts:  c.Int("retry"),
				RetryDelay:     retryDelay,
				Backoff:        c.Bool("backoff"),
				InsecureTLS:    c.Bool("insecure"),
			})
			if err != nil {
				return cli.NewExitError(err.Error(), 1)
			}
			if c.Bool("verbose") {
				for _, step := range result.Steps {
					fmt.Fprintf(os.Stderr, "%s: HTTP %d\n", step.Name, step.StatusCode)
				}
			}
			b, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return err
			}
			if outputKey := c.String("to-github-output"); outputKey != "" {
				return services.WriteToGitHubOutputValue(outputKey, string(b))
			}
			fmt.Println(string(b))
			return nil
		},
	}
}

func readHTTPChainPlanInput(c *cli.Context, file string) ([]byte, services.DataFormat, error) {
	format := services.FormatYAML
	if from := c.String("from"); from != "" {
		format = services.FormatString(from)
	} else if file != "-" {
		if detected := services.DetectFormat(file); detected != "" {
			format = detected
		}
	}
	if file == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, "", fmt.Errorf("reading stdin: %w", err)
		}
		if len(strings.TrimSpace(string(data))) == 0 {
			return nil, "", fmt.Errorf("empty HTTP chain plan on stdin")
		}
		return data, format, nil
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, "", err
	}
	return data, format, nil
}

func parseHTTPStatusList(value string) ([]int, error) {
	if value == "" {
		return nil, nil
	}
	var out []int
	for _, part := range strings.Split(value, ",") {
		code, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || code < 100 || code > 999 {
			return nil, fmt.Errorf("invalid status code: %s", part)
		}
		out = append(out, code)
	}
	return out, nil
}

func formatHTTPResult(v interface{}, raw bool) string {
	if raw {
		if s, ok := v.(string); ok {
			return s
		}
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}

func printHTTPVerbose(res *services.HTTPResult) {
	fmt.Fprintf(os.Stderr, "HTTP %d\n", res.StatusCode)
	for name, values := range res.Headers {
		for _, value := range values {
			fmt.Fprintf(os.Stderr, "%s: %s\n", name, value)
		}
	}
}
