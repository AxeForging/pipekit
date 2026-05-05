package actions

import (
	"fmt"
	"strings"

	"github.com/AxeForging/pipekit/services"

	"github.com/urfave/cli"
)

// MatrixCommand returns the matrix command group.
func MatrixCommand() cli.Command {
	return cli.Command{
		Name:  "matrix",
		Usage: "dynamic CI matrix generation",
		Subcommands: []cli.Command{
			{
				Name:      "from-dirs",
				Usage:     "generate matrix from directory names",
				ArgsUsage: "PATH",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "key", Value: "dir", Usage: "matrix key name"},
					cli.StringFlag{Name: "to-github-output", Usage: "write to GITHUB_OUTPUT with this key"},
				},
				Action: func(c *cli.Context) error {
					dirPath := c.Args().First()
					if dirPath == "" {
						return cli.NewExitError("directory path required", 1)
					}
					result, err := services.MatrixFromDirs(dirPath, c.String("key"))
					if err != nil {
						return err
					}
					return outputMatrix(c, result)
				},
			},
			{
				Name:      "from-files",
				Usage:     "generate matrix from file list (glob pattern)",
				ArgsUsage: "PATTERN",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "key", Value: "file", Usage: "matrix key name"},
					cli.StringFlag{Name: "to-github-output", Usage: "write to GITHUB_OUTPUT with this key"},
				},
				Action: func(c *cli.Context) error {
					pattern := c.Args().First()
					if pattern == "" {
						return cli.NewExitError("glob pattern required", 1)
					}
					result, err := services.MatrixFromFiles(pattern, c.String("key"))
					if err != nil {
						return err
					}
					return outputMatrix(c, result)
				},
			},
			{
				Name:  "from-json",
				Usage: "transform/filter a JSON array into matrix format",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "key", Value: "item", Usage: "matrix key name"},
					cli.StringFlag{Name: "filter-field", Usage: "field name to filter on"},
					cli.StringFlag{Name: "filter-value", Usage: "required value for filter field"},
					cli.StringFlag{Name: "to-github-output", Usage: "write to GITHUB_OUTPUT with this key"},
				},
				Action: func(c *cli.Context) error {
					r, err := getInputReader(c)
					if err != nil {
						return err
					}
					result, err := services.MatrixFromJSON(r, c.String("key"), c.String("filter-field"), c.String("filter-value"))
					if err != nil {
						return err
					}
					return outputMatrix(c, result)
				},
			},
			{
				Name:  "combine",
				Usage: "Cartesian product of multiple arrays",
				Flags: []cli.Flag{
					cli.StringSliceFlag{Name: "set", Usage: "key=val1,val2,val3 (repeatable)"},
					cli.StringFlag{Name: "to-github-output", Usage: "write to GITHUB_OUTPUT with this key"},
				},
				Action: func(c *cli.Context) error {
					sets := c.StringSlice("set")
					if len(sets) == 0 {
						return cli.NewExitError("at least one --set required", 1)
					}
					arrays := make(map[string][]string)
					for _, s := range sets {
						parts := strings.SplitN(s, "=", 2)
						if len(parts) != 2 {
							return cli.NewExitError("invalid --set format, expected key=val1,val2,...", 1)
						}
						arrays[parts[0]] = strings.Split(parts[1], ",")
					}
					result, err := services.MatrixCombine(arrays)
					if err != nil {
						return err
					}
					return outputMatrix(c, result)
				},
			},
			{
				Name:      "shard",
				Usage:     "pick the N-th shard of items (for splitting test suites)",
				ArgsUsage: "ITEM [ITEM...]",
				Flags: []cli.Flag{
					cli.IntFlag{Name: "total", Usage: "total number of shards (required)"},
					cli.IntFlag{Name: "index", Usage: "0-based index of the shard to return (required)"},
					cli.StringFlag{Name: "from-stdin-lines", Usage: "read items as newline-separated from stdin"},
					cli.StringFlag{Name: "format", Value: "list", Usage: "output format: list (newlines), csv, json"},
				},
				Action: func(c *cli.Context) error {
					if !c.IsSet("total") || !c.IsSet("index") {
						return cli.NewExitError("--total and --index are required", 1)
					}
					var items []string
					if c.Bool("from-stdin-lines") || c.IsSet("from-stdin-lines") {
						data, err := readAllInput(c)
						if err != nil {
							return err
						}
						for _, l := range strings.Split(strings.TrimSpace(string(data)), "\n") {
							if l != "" {
								items = append(items, l)
							}
						}
					} else {
						items = []string(c.Args())
					}
					out, err := services.MatrixShard(items, c.Int("total"), c.Int("index"))
					if err != nil {
						return err
					}
					return printList(out, c.String("format"))
				},
			},
		},
	}
}

func printList(items []string, format string) error {
	out, err := services.FormatDiffOutput(items, format)
	if err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}

func outputMatrix(c *cli.Context, result string) error {
	if outputKey := c.String("to-github-output"); outputKey != "" {
		return services.WriteToGitHubOutputValue(outputKey, result)
	}
	fmt.Println(result)
	return nil
}
