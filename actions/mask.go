package actions

import (
	"os"
	"strings"

	"github.com/AxeForging/pipekit/services"

	"github.com/urfave/cli"
)

// MaskCommand returns the mask command group.
func MaskCommand() cli.Command {
	return cli.Command{
		Name:  "mask",
		Usage: "prevent secrets from leaking in logs",
		Subcommands: []cli.Command{
			{
				Name:  "values",
				Usage: "mask specific values in stdin stream",
				Flags: []cli.Flag{
					cli.StringSliceFlag{Name: "pattern", Usage: "regex pattern(s) to match values to mask"},
					cli.StringFlag{Name: "replacement", Value: "***", Usage: "replacement string"},
					cli.IntFlag{Name: "partial", Usage: "show first/last N chars"},
				},
				Action: func(c *cli.Context) error {
					patterns := c.StringSlice("pattern")
					if len(patterns) == 0 {
						return cli.NewExitError("at least one --pattern is required", 1)
					}
					return services.MaskValues(os.Stdin, os.Stdout, patterns, c.String("replacement"), c.Int("partial"))
				},
			},
			{
				Name:  "file",
				Usage: "redact sensitive values from a file",
				Flags: []cli.Flag{
					cli.StringSliceFlag{Name: "pattern", Usage: "regex pattern(s) to match values to mask"},
					cli.StringFlag{Name: "replacement", Value: "***", Usage: "replacement string"},
					cli.IntFlag{Name: "partial", Usage: "show first/last N chars"},
				},
				Action: func(c *cli.Context) error {
					path := c.Args().First()
					if path == "" {
						return cli.NewExitError("file path required", 1)
					}
					patterns := c.StringSlice("pattern")
					if len(patterns) == 0 {
						return cli.NewExitError("at least one --pattern is required", 1)
					}
					return services.MaskFile(path, os.Stdout, patterns, c.String("replacement"), c.Int("partial"))
				},
			},
			{
				Name:  "github",
				Usage: "emit ::add-mask:: commands for GitHub Actions",
				Action: func(c *cli.Context) error {
					values := c.Args()
					if len(values) == 0 {
						return cli.NewExitError("at least one value required", 1)
					}
					return services.MaskGitHub(os.Stdout, values)
				},
			},
			{
				Name:  "env",
				Usage: "mask env vars matching a pattern in GitHub Actions",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "env-match", Usage: "comma-separated glob patterns for env var names"},
					cli.BoolFlag{Name: "github", Usage: "emit ::add-mask:: commands"},
				},
				Action: func(c *cli.Context) error {
					matchStr := c.String("env-match")
					if matchStr == "" {
						return cli.NewExitError("--env-match is required", 1)
					}
					patterns := strings.Split(matchStr, ",")
					for i := range patterns {
						patterns[i] = strings.TrimSpace(patterns[i])
					}
					return services.MaskEnvVars(os.Stdout, patterns, c.Bool("github"))
				},
			},
		},
	}
}
