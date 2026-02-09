package actions

import (
	"fmt"
	"os"

	"github.com/AxeForging/pipekit/services"

	"github.com/urfave/cli"
)

// SummaryCommand returns the summary command group.
func SummaryCommand() cli.Command {
	return cli.Command{
		Name:  "summary",
		Usage: "generate formatted summaries for pipeline UIs",
		Subcommands: []cli.Command{
			{
				Name:  "github",
				Usage: "append Markdown to $GITHUB_STEP_SUMMARY",
				Action: func(c *cli.Context) error {
					content := c.Args().First()
					if content == "" {
						data, err := readValueOrStdin(c)
						if err != nil {
							return cli.NewExitError("markdown content required as argument or via stdin", 1)
						}
						content = string(data)
					}
					return services.AppendToGitHubSummary(content)
				},
			},
			{
				Name:  "table",
				Usage: "generate a Markdown table from JSON/CSV",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "title", Usage: "table title"},
					cli.StringFlag{Name: "format", Value: "json", Usage: "input format: json or csv"},
					cli.BoolFlag{Name: "to-github-summary", Usage: "append to $GITHUB_STEP_SUMMARY"},
				},
				Action: func(c *cli.Context) error {
					r, err := getInputReader(c)
					if err != nil {
						return err
					}

					table, err := services.GenerateTable(r, c.String("title"), c.String("format"))
					if err != nil {
						return err
					}

					if c.Bool("to-github-summary") {
						return services.AppendToGitHubSummary(table)
					}

					fmt.Print(table)
					return nil
				},
			},
			{
				Name:  "badge",
				Usage: "generate a status badge line",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "label", Usage: "badge label", Required: true},
					cli.StringFlag{Name: "status", Usage: "badge status: success, failure, warning, info", Required: true},
					cli.BoolFlag{Name: "to-github-summary", Usage: "append to $GITHUB_STEP_SUMMARY"},
				},
				Action: func(c *cli.Context) error {
					badge := services.GenerateBadge(c.String("label"), c.String("status"))

					if c.Bool("to-github-summary") {
						return services.AppendToGitHubSummary(badge)
					}

					fmt.Println(badge)
					return nil
				},
			},
			{
				Name:  "section",
				Usage: "generate a collapsible details section",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "title", Usage: "section title", Required: true},
					cli.BoolFlag{Name: "to-github-summary", Usage: "append to $GITHUB_STEP_SUMMARY"},
				},
				Action: func(c *cli.Context) error {
					body := c.Args().First()
					if body == "" {
						data, err := readValueOrStdin(c)
						if err != nil {
							return cli.NewExitError("body content required as argument or via stdin", 1)
						}
						body = string(data)
					}

					section := services.GenerateSection(c.String("title"), body)

					if c.Bool("to-github-summary") {
						return services.AppendToGitHubSummary(section)
					}

					fmt.Print(section)
					return nil
				},
			},
		},
	}
}

// getOutputFile returns a file for writing or stdout.
func getOutputFile(c *cli.Context) (*os.File, error) {
	if output := c.String("output"); output != "" {
		return os.Create(output)
	}
	return os.Stdout, nil
}
