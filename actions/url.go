package actions

import (
	"fmt"

	"github.com/AxeForging/pipekit/services"

	"github.com/urfave/cli"
)

// URLCommand returns the url command group.
func URLCommand() cli.Command {
	return cli.Command{
		Name:  "url",
		Usage: "URL parsing helpers",
		Subcommands: []cli.Command{
			{
				Name:      "parse",
				Usage:     "split a URL into components and emit as env vars or KV pairs",
				ArgsUsage: "URL",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "prefix, p", Usage: "prefix for emitted keys (e.g. DB_)"},
					cli.BoolFlag{Name: "to-github", Usage: "write to $GITHUB_ENV"},
					cli.BoolFlag{Name: "to-github-output", Usage: "write to $GITHUB_OUTPUT"},
				},
				Action: func(c *cli.Context) error {
					raw, err := firstArgOrErr(c, "URL")
					if err != nil {
						return err
					}
					kvs, err := services.ParseURL(raw, c.String("prefix"))
					if err != nil {
						return err
					}
					if c.Bool("to-github") {
						return services.WriteToGitHubEnv(kvs)
					}
					if c.Bool("to-github-output") {
						return services.WriteToGitHubOutput(kvs)
					}
					for _, kv := range kvs {
						fmt.Printf("%s=%s\n", kv.Key, kv.Value)
					}
					return nil
				},
			},
		},
	}
}
