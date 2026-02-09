package actions

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/AxeForging/pipekit/services"

	"github.com/urfave/cli"
)

// AssertCommand returns the assert command group.
func AssertCommand() cli.Command {
	return cli.Command{
		Name:  "assert",
		Usage: "lightweight pipeline assertions and guards",
		Subcommands: []cli.Command{
			{
				Name:      "env-exists",
				Usage:     "assert that env vars exist and are non-empty",
				ArgsUsage: "VAR1 VAR2 ...",
				Action: func(c *cli.Context) error {
					names := c.Args()
					if len(names) == 0 {
						return cli.NewExitError("at least one env var name required", 1)
					}
					if err := services.AssertEnvExists(names); err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
					return nil
				},
			},
			{
				Name:      "file-exists",
				Usage:     "assert files exist",
				ArgsUsage: "FILE1 FILE2 ...",
				Action: func(c *cli.Context) error {
					paths := c.Args()
					if len(paths) == 0 {
						return cli.NewExitError("at least one file path required", 1)
					}
					if err := services.AssertFileExists(paths); err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
					return nil
				},
			},
			{
				Name:  "json-path",
				Usage: "assert a value at a JSON path matches expectation",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "file", Usage: "JSON file to check"},
					cli.StringFlag{Name: "path", Usage: "jq-style path expression", Required: true},
					cli.StringFlag{Name: "expected", Usage: "expected value", Required: true},
				},
				Action: func(c *cli.Context) error {
					filePath := c.String("file")
					if filePath == "" {
						return cli.NewExitError("--file is required", 1)
					}
					data, err := os.ReadFile(filePath)
					if err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
					if err := services.AssertJSONPath(data, c.String("path"), c.String("expected")); err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
					return nil
				},
			},
			{
				Name:      "semver",
				Usage:     "assert a version string is valid semver",
				ArgsUsage: "VERSION",
				Action: func(c *cli.Context) error {
					version := c.Args().First()
					if version == "" {
						return cli.NewExitError("version string required", 1)
					}
					if err := services.AssertSemver(version); err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
					return nil
				},
			},
			{
				Name:      "compare",
				Usage:     "compare two semver versions",
				ArgsUsage: "V1 OPERATOR V2",
				Action: func(c *cli.Context) error {
					args := c.Args()
					if len(args) < 3 {
						return cli.NewExitError("usage: assert compare V1 OPERATOR V2", 1)
					}
					if err := services.AssertSemverCompare(args[0], args[1], args[2]); err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
					return nil
				},
			},
			{
				Name:      "url",
				Usage:     "assert a URL returns expected status code",
				ArgsUsage: "URL",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "expected-status", Value: "200", Usage: "comma-separated expected HTTP status codes"},
					cli.StringFlag{Name: "timeout", Value: "10s", Usage: "request timeout"},
				},
				Action: func(c *cli.Context) error {
					urlStr := c.Args().First()
					if urlStr == "" {
						return cli.NewExitError("URL required", 1)
					}
					timeout, err := time.ParseDuration(c.String("timeout"))
					if err != nil {
						return cli.NewExitError("invalid timeout: "+err.Error(), 1)
					}
					var codes []int
					for _, s := range strings.Split(c.String("expected-status"), ",") {
						code, err := strconv.Atoi(strings.TrimSpace(s))
						if err != nil {
							return cli.NewExitError("invalid status code: "+s, 1)
						}
						codes = append(codes, code)
					}
					if err := services.AssertURL(urlStr, codes, timeout); err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
					return nil
				},
			},
		},
	}
}
