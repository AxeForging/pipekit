package actions

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/AxeForging/pipekit/services"

	"github.com/urfave/cli"
)

// WaitCommand returns the wait command group.
func WaitCommand() cli.Command {
	commonFlags := []cli.Flag{
		cli.StringFlag{Name: "timeout", Value: "120s", Usage: "total time before giving up"},
		cli.StringFlag{Name: "interval", Value: "5s", Usage: "time between retries"},
		cli.BoolFlag{Name: "backoff", Usage: "exponential backoff between retries"},
		cli.BoolFlag{Name: "quiet, q", Usage: "suppress output, just exit 0/1"},
	}

	return cli.Command{
		Name:  "wait",
		Usage: "health check and readiness polling",
		Subcommands: []cli.Command{
			{
				Name:      "url",
				Usage:     "poll a URL until it returns expected status",
				ArgsUsage: "URL",
				Flags: append(commonFlags,
					cli.StringFlag{Name: "expected-status", Value: "200", Usage: "comma-separated acceptable HTTP status codes"},
					cli.StringFlag{Name: "expected-body", Usage: "match a substring in the response body"},
				),
				Action: func(c *cli.Context) error {
					urlStr := c.Args().First()
					if urlStr == "" {
						return cli.NewExitError("URL required", 1)
					}
					timeout, err := time.ParseDuration(c.String("timeout"))
					if err != nil {
						return cli.NewExitError("invalid timeout: "+err.Error(), 1)
					}
					interval, err := time.ParseDuration(c.String("interval"))
					if err != nil {
						return cli.NewExitError("invalid interval: "+err.Error(), 1)
					}
					var codes []int
					for _, s := range strings.Split(c.String("expected-status"), ",") {
						code, err := strconv.Atoi(strings.TrimSpace(s))
						if err != nil {
							return cli.NewExitError("invalid status code: "+s, 1)
						}
						codes = append(codes, code)
					}
					ctx, cancel := context.WithTimeout(context.Background(), timeout)
					defer cancel()
					if err := services.WaitForURL(ctx, urlStr, codes, c.String("expected-body"), interval, c.Bool("backoff"), c.Bool("quiet")); err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
					return nil
				},
			},
			{
				Name:      "tcp",
				Usage:     "wait for a TCP port to accept connections",
				ArgsUsage: "HOST:PORT",
				Flags:     commonFlags,
				Action: func(c *cli.Context) error {
					address := c.Args().First()
					if address == "" {
						return cli.NewExitError("address (host:port) required", 1)
					}
					timeout, err := time.ParseDuration(c.String("timeout"))
					if err != nil {
						return cli.NewExitError("invalid timeout: "+err.Error(), 1)
					}
					interval, err := time.ParseDuration(c.String("interval"))
					if err != nil {
						return cli.NewExitError("invalid interval: "+err.Error(), 1)
					}
					ctx, cancel := context.WithTimeout(context.Background(), timeout)
					defer cancel()
					if err := services.WaitForTCP(ctx, address, interval, c.Bool("backoff"), c.Bool("quiet")); err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
					return nil
				},
			},
			{
				Name:      "grpc",
				Usage:     "poll a gRPC health endpoint until it reports SERVING",
				ArgsUsage: "HOST:PORT",
				Flags: append(commonFlags,
					cli.StringFlag{Name: "service", Usage: "gRPC health service name"},
					cli.BoolFlag{Name: "tls", Usage: "use TLS for the gRPC connection"},
				),
				Action: func(c *cli.Context) error {
					address := c.Args().First()
					if address == "" {
						return cli.NewExitError("address (host:port) required", 1)
					}
					timeout, err := time.ParseDuration(c.String("timeout"))
					if err != nil {
						return cli.NewExitError("invalid timeout: "+err.Error(), 1)
					}
					interval, err := time.ParseDuration(c.String("interval"))
					if err != nil {
						return cli.NewExitError("invalid interval: "+err.Error(), 1)
					}
					ctx, cancel := context.WithTimeout(context.Background(), timeout)
					defer cancel()
					if err := services.WaitForGRPCHealth(ctx, address, c.String("service"), interval, c.Bool("backoff"), c.Bool("quiet"), c.Bool("tls")); err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
					return nil
				},
			},
			{
				Name:      "ws",
				Usage:     "poll a WebSocket endpoint until it accepts an upgrade",
				ArgsUsage: "WS_URL",
				Flags:     commonFlags,
				Action: func(c *cli.Context) error {
					urlStr := c.Args().First()
					if urlStr == "" {
						return cli.NewExitError("WebSocket URL required", 1)
					}
					timeout, err := time.ParseDuration(c.String("timeout"))
					if err != nil {
						return cli.NewExitError("invalid timeout: "+err.Error(), 1)
					}
					interval, err := time.ParseDuration(c.String("interval"))
					if err != nil {
						return cli.NewExitError("invalid interval: "+err.Error(), 1)
					}
					ctx, cancel := context.WithTimeout(context.Background(), timeout)
					defer cancel()
					if err := services.WaitForWebSocket(ctx, urlStr, interval, c.Bool("backoff"), c.Bool("quiet")); err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
					return nil
				},
			},
			{
				Name:      "command",
				Usage:     "retry a shell command until it exits 0",
				ArgsUsage: "COMMAND",
				Flags:     commonFlags,
				Action: func(c *cli.Context) error {
					command := strings.Join(c.Args(), " ")
					if command == "" {
						return cli.NewExitError("command required", 1)
					}
					timeout, err := time.ParseDuration(c.String("timeout"))
					if err != nil {
						return cli.NewExitError("invalid timeout: "+err.Error(), 1)
					}
					interval, err := time.ParseDuration(c.String("interval"))
					if err != nil {
						return cli.NewExitError("invalid interval: "+err.Error(), 1)
					}
					ctx, cancel := context.WithTimeout(context.Background(), timeout)
					defer cancel()
					if err := services.WaitForCommand(ctx, command, interval, c.Bool("backoff"), c.Bool("quiet")); err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
					return nil
				},
			},
		},
	}
}
