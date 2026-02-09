package actions

import (
	"fmt"
	"os"
	"strings"

	"github.com/AxeForging/pipekit/domain"
	"github.com/AxeForging/pipekit/services"

	"github.com/urfave/cli"
	"gopkg.in/yaml.v3"
)

// DiffCommand returns the diff command group.
func DiffCommand() cli.Command {
	commonFlags := []cli.Flag{
		cli.StringFlag{Name: "base", Value: "origin/main", Usage: "base git ref"},
		cli.StringFlag{Name: "head", Value: "HEAD", Usage: "head git ref"},
		cli.StringFlag{Name: "output", Value: "list", Usage: "output format: json, list, csv"},
		cli.StringFlag{Name: "to-github-output", Usage: "write result to GITHUB_OUTPUT with this key"},
		cli.StringSliceFlag{Name: "include", Usage: "include glob patterns"},
		cli.StringSliceFlag{Name: "exclude", Usage: "exclude glob patterns"},
	}

	return cli.Command{
		Name:  "diff",
		Usage: "detect changed files and directories between git refs",
		Subcommands: []cli.Command{
			{
				Name:  "files",
				Usage: "list changed files between two refs",
				Flags: commonFlags,
				Action: func(c *cli.Context) error {
					files, err := services.DiffFiles(c.String("base"), c.String("head"), c.StringSlice("include"), c.StringSlice("exclude"))
					if err != nil {
						return err
					}
					return outputDiff(c, files)
				},
			},
			{
				Name:  "dirs",
				Usage: "list top-level directories that have changes",
				Flags: commonFlags,
				Action: func(c *cli.Context) error {
					dirs, err := services.DiffDirs(c.String("base"), c.String("head"), c.StringSlice("include"), c.StringSlice("exclude"))
					if err != nil {
						return err
					}
					return outputDiff(c, dirs)
				},
			},
			{
				Name:      "match",
				Usage:     "check if changes match glob patterns (exit 0 if yes)",
				ArgsUsage: "PATTERN...",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "base", Value: "origin/main", Usage: "base git ref"},
					cli.StringFlag{Name: "head", Value: "HEAD", Usage: "head git ref"},
				},
				Action: func(c *cli.Context) error {
					patterns := c.Args()
					if len(patterns) == 0 {
						return cli.NewExitError("at least one pattern required", 1)
					}
					matched, err := services.DiffMatch(c.String("base"), c.String("head"), patterns)
					if err != nil {
						return err
					}
					if !matched {
						return cli.NewExitError("no changes matched the given patterns", 1)
					}
					return nil
				},
			},
			{
				Name:  "affected",
				Usage: "map changed paths to service names via a config file",
				Flags: append(commonFlags,
					cli.StringFlag{Name: "config", Value: ".pipekit-diff.yaml", Usage: "path to diff config file"},
				),
				Action: func(c *cli.Context) error {
					configPath := c.String("config")
					data, err := os.ReadFile(configPath)
					if err != nil {
						return fmt.Errorf("reading config %s: %w", configPath, err)
					}
					var config domain.DiffConfig
					if err := yaml.Unmarshal(data, &config); err != nil {
						return fmt.Errorf("parsing config: %w", err)
					}
					affected, err := services.DiffAffected(c.String("base"), c.String("head"), config)
					if err != nil {
						return err
					}
					return outputDiff(c, affected)
				},
			},
		},
	}
}

func outputDiff(c *cli.Context, items []string) error {
	formatted, err := services.FormatDiffOutput(items, c.String("output"))
	if err != nil {
		return err
	}

	if outputKey := c.String("to-github-output"); outputKey != "" {
		return services.WriteToGitHubOutputValue(outputKey, formatted)
	}

	fmt.Println(strings.TrimSpace(formatted))
	return nil
}
