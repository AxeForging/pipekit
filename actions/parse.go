package actions

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/AxeForging/pipekit/domain"
	"github.com/AxeForging/pipekit/services"

	"github.com/urfave/cli"
)

// ParseCommand returns the parse command group.
func ParseCommand() cli.Command {
	return cli.Command{
		Name:  "parse",
		Usage: "extract structured data from markdown and text",
		Subcommands: []cli.Command{
			{
				Name:  "extract-block",
				Usage: "extract fenced code blocks from markdown/text (stdin or file)",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "language, l", Usage: "filter by language tag (e.g. yaml, json, python, bash)"},
					cli.IntFlag{Name: "index, i", Value: -1, Usage: "return only the Nth block (0-based); -1 returns all"},
					cli.BoolFlag{Name: "content-only", Usage: "output raw content only (no JSON wrapping)"},
					cli.BoolFlag{Name: "to-github-output", Usage: "write first block content to $GITHUB_OUTPUT as PARSED_BLOCK"},
					cli.StringFlag{Name: "output-key", Value: "PARSED_BLOCK", Usage: "output variable name for --to-github-output"},
				},
				Action: func(c *cli.Context) error {
					r, err := getInputReader(c)
					if err != nil {
						return cli.NewExitError(err.Error(), 1)
					}

					blocks, err := services.ExtractCodeBlocks(r, c.String("language"))
					if err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
					if len(blocks) == 0 {
						return cli.NewExitError("no code blocks found", 1)
					}

					idx := c.Int("index")
					if idx >= 0 {
						if idx >= len(blocks) {
							return cli.NewExitError(fmt.Sprintf("block index %d out of range (found %d blocks)", idx, len(blocks)), 1)
						}
						blocks = []services.CodeBlock{blocks[idx]}
					}

					if c.Bool("to-github-output") {
						content := blocks[0].Content
						kvs := []domain.KeyValue{{Key: c.String("output-key"), Value: content}}
						return services.WriteToGitHubOutput(kvs)
					}

					if c.Bool("content-only") {
						for i, b := range blocks {
							if i > 0 {
								fmt.Println("---")
							}
							fmt.Println(b.Content)
						}
						return nil
					}

					jsonStr, err := services.FormatCodeBlocksJSON(blocks)
					if err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
					fmt.Println(jsonStr)
					return nil
				},
			},
			{
				Name:  "extract-yaml",
				Usage: "extract YAML code blocks from markdown and parse to JSON",
				Flags: []cli.Flag{
					cli.IntFlag{Name: "index, i", Value: -1, Usage: "return only the Nth YAML block (0-based); -1 returns all"},
					cli.BoolFlag{Name: "to-github-output", Usage: "write parsed YAML as JSON to $GITHUB_OUTPUT"},
					cli.StringFlag{Name: "output-key", Value: "PARSED_YAML", Usage: "output variable name for --to-github-output"},
					cli.BoolFlag{Name: "to-env", Usage: "export top-level keys as env vars (first block only)"},
					cli.BoolFlag{Name: "to-github", Usage: "write top-level keys to $GITHUB_ENV (first block only)"},
					cli.BoolFlag{Name: "uppercase-keys, u", Usage: "convert keys to UPPER_SNAKE_CASE when using --to-env/--to-github"},
				},
				Action: func(c *cli.Context) error {
					r, err := getInputReader(c)
					if err != nil {
						return cli.NewExitError(err.Error(), 1)
					}

					results, err := services.ExtractAndParseYAML(r)
					if err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
					if len(results) == 0 {
						return cli.NewExitError("no YAML blocks found", 1)
					}

					idx := c.Int("index")
					if idx >= 0 {
						if idx >= len(results) {
							return cli.NewExitError(fmt.Sprintf("block index %d out of range (found %d blocks)", idx, len(results)), 1)
						}
						results = []map[string]interface{}{results[idx]}
					}

					// Export as env vars
					if c.Bool("to-env") || c.Bool("to-github") {
						block := results[0]
						var kvs []domain.KeyValue
						for k, v := range block {
							kvs = append(kvs, domain.KeyValue{Key: k, Value: fmt.Sprintf("%v", v)})
						}
						kvs = services.TransformKeys(kvs, c.Bool("uppercase-keys"), "", false)

						if c.Bool("to-github") {
							return services.WriteToGitHubEnv(kvs)
						}
						return services.WriteToShell(os.Stdout, kvs)
					}

					if c.Bool("to-github-output") {
						jsonBytes, err := json.Marshal(results)
						if err != nil {
							return cli.NewExitError(err.Error(), 1)
						}
						kvs := []domain.KeyValue{{Key: c.String("output-key"), Value: string(jsonBytes)}}
						return services.WriteToGitHubOutput(kvs)
					}

					jsonStr, err := services.FormatParsedYAMLJSON(results)
					if err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
					fmt.Println(jsonStr)
					return nil
				},
			},
		},
	}
}
