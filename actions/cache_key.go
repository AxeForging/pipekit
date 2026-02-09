package actions

import (
	"fmt"

	"github.com/AxeForging/pipekit/services"

	"github.com/urfave/cli"
)

// CacheKeyCommand returns the cache-key command group.
func CacheKeyCommand() cli.Command {
	return cli.Command{
		Name:  "cache-key",
		Usage: "generate deterministic cache keys",
		Subcommands: []cli.Command{
			{
				Name:      "from-files",
				Usage:     "SHA256 hash of one or more files",
				ArgsUsage: "FILE...",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "prefix", Usage: "prefix for the cache key"},
					cli.StringFlag{Name: "to-github-output", Usage: "export to GITHUB_OUTPUT with this key"},
				},
				Action: func(c *cli.Context) error {
					files := c.Args()
					if len(files) == 0 {
						return cli.NewExitError("at least one file required", 1)
					}
					key, err := services.CacheKeyFromFiles(files, c.String("prefix"))
					if err != nil {
						return err
					}
					if outputKey := c.String("to-github-output"); outputKey != "" {
						return services.WriteToGitHubOutputValue(outputKey, key)
					}
					fmt.Println(key)
					return nil
				},
			},
			{
				Name:      "from-glob",
				Usage:     "hash all files matching a glob pattern",
				ArgsUsage: "PATTERN",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "prefix", Usage: "prefix for the cache key"},
					cli.StringFlag{Name: "to-github-output", Usage: "export to GITHUB_OUTPUT with this key"},
				},
				Action: func(c *cli.Context) error {
					pattern := c.Args().First()
					if pattern == "" {
						return cli.NewExitError("glob pattern required", 1)
					}
					key, err := services.CacheKeyFromGlob(pattern, c.String("prefix"))
					if err != nil {
						return err
					}
					if outputKey := c.String("to-github-output"); outputKey != "" {
						return services.WriteToGitHubOutputValue(outputKey, key)
					}
					fmt.Println(key)
					return nil
				},
			},
			{
				Name:      "composite",
				Usage:     "combine multiple inputs into one cache key",
				ArgsUsage: "PART1 PART2 ...",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "prefix", Usage: "prefix for the cache key"},
					cli.StringFlag{Name: "separator", Value: "-", Usage: "separator between parts"},
					cli.StringFlag{Name: "to-github-output", Usage: "export to GITHUB_OUTPUT with this key"},
				},
				Action: func(c *cli.Context) error {
					parts := c.Args()
					if len(parts) == 0 {
						return cli.NewExitError("at least one part required", 1)
					}
					key := services.CacheKeyComposite(parts, c.String("prefix"), c.String("separator"))
					if outputKey := c.String("to-github-output"); outputKey != "" {
						return services.WriteToGitHubOutputValue(outputKey, key)
					}
					fmt.Println(key)
					return nil
				},
			},
		},
	}
}
