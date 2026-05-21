package actions

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/AxeForging/pipekit/services"

	"github.com/urfave/cli"
)

// ChangelogCommand returns release-note generation helpers.
func ChangelogCommand() cli.Command {
	return cli.Command{
		Name:  "changelog",
		Usage: "generate release notes from git commits",
		Subcommands: []cli.Command{
			{
				Name:  "generate",
				Usage: "generate changelog markdown for a git range",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "from", Usage: "start ref (exclusive)"},
					cli.StringFlag{Name: "to", Value: "HEAD", Usage: "end ref (inclusive)"},
					cli.BoolFlag{Name: "conventional", Usage: "group conventional commits"},
					cli.StringFlag{Name: "format, f", Value: "markdown", Usage: "markdown or json"},
					cli.StringFlag{Name: "output, o", Usage: "write output to this file"},
					outputKeyFlag(),
				},
				Action: func(c *cli.Context) error {
					markdown, entries, err := services.GenerateChangelog(services.ChangelogOptions{
						From:         c.String("from"),
						To:           c.String("to"),
						Conventional: c.Bool("conventional"),
					})
					if err != nil {
						return err
					}

					out := markdown
					switch c.String("format") {
					case "markdown", "":
					case "json":
						data, err := json.MarshalIndent(entries, "", "  ")
						if err != nil {
							return err
						}
						out = string(data) + "\n"
					default:
						return cli.NewExitError("unknown format: "+c.String("format"), 1)
					}

					if outputKey := c.String("to-github-output"); outputKey != "" {
						return services.WriteToGitHubOutputValue(outputKey, out)
					}
					if path := c.String("output"); path != "" {
						return os.WriteFile(path, []byte(out), 0644)
					}
					fmt.Print(out)
					return nil
				},
			},
			{
				Name:  "since-tag",
				Usage: "generate changelog markdown since the latest reachable tag",
				Flags: []cli.Flag{
					cli.BoolFlag{Name: "conventional", Usage: "group conventional commits"},
					cli.StringFlag{Name: "output, o", Usage: "write output to this file"},
					outputKeyFlag(),
				},
				Action: func(c *cli.Context) error {
					tag, err := services.GitPreviousTag()
					if err != nil {
						return err
					}
					markdown, _, err := services.GenerateChangelog(services.ChangelogOptions{
						From:         tag,
						To:           "HEAD",
						Conventional: c.Bool("conventional"),
					})
					if err != nil {
						return err
					}
					if outputKey := c.String("to-github-output"); outputKey != "" {
						return services.WriteToGitHubOutputValue(outputKey, markdown)
					}
					if path := c.String("output"); path != "" {
						return os.WriteFile(path, []byte(markdown), 0644)
					}
					fmt.Print(markdown)
					return nil
				},
			},
		},
	}
}
