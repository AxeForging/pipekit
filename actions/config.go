package actions

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/AxeForging/pipekit/domain"
	"github.com/AxeForging/pipekit/services"

	"github.com/urfave/cli"
)

// ConfigCommand returns the config command group.
func ConfigCommand() cli.Command {
	outputFlags := []cli.Flag{
		cli.BoolFlag{Name: "uppercase-keys, u", Usage: "convert all keys to UPPER_SNAKE_CASE"},
		cli.StringFlag{Name: "prefix, p", Usage: "add prefix to all keys"},
		cli.BoolFlag{Name: "to-github", Usage: "write to $GITHUB_ENV"},
		cli.BoolFlag{Name: "to-github-output", Usage: "write to $GITHUB_OUTPUT"},
		cli.BoolFlag{Name: "json", Usage: "output resolved config as compact JSON"},
	}

	return cli.Command{
		Name:  "config",
		Usage: "resolve environment configuration from structured maps",
		Subcommands: []cli.Command{
			{
				Name:      "resolve",
				Usage:     "resolve environment config from a JSON/YAML map with alias support",
				ArgsUsage: "CONFIG_FILE",
				Flags: append(outputFlags,
					cli.StringFlag{Name: "env, e", Usage: "environment name (supports aliases like dev, staging, prod)", Required: true},
					cli.StringFlag{Name: "format, f", Value: "json", Usage: "config format: json, yaml"},
					cli.StringFlag{Name: "aliases", Usage: "custom aliases as JSON (e.g. '{\"preview\": \"staging\"}')"},
				),
				Action: func(c *cli.Context) error {
					r, err := getInputReader(c)
					if err != nil {
						return cli.NewExitError("config file required: "+err.Error(), 1)
					}

					var customAliases map[string]string
					if aliasStr := c.String("aliases"); aliasStr != "" {
						if err := json.Unmarshal([]byte(aliasStr), &customAliases); err != nil {
							return cli.NewExitError("invalid aliases JSON: "+err.Error(), 1)
						}
					}

					envName := c.String("env")
					format := c.String("format")

					if c.Bool("json") {
						jsonStr, normalized, err := services.ResolveConfigJSON(r, envName, format, customAliases)
						if err != nil {
							return cli.NewExitError(err.Error(), 1)
						}

						kvs := []domain.KeyValue{{Key: "json_map", Value: jsonStr}}

						if c.Bool("to-github") {
							return services.WriteToGitHubEnv(kvs)
						}
						if c.Bool("to-github-output") {
							return services.WriteToGitHubOutput(kvs)
						}

						fmt.Println(jsonStr)
						fmt.Fprintf(os.Stderr, "resolved environment: %s\n", normalized)
						return nil
					}

					kvs, normalized, err := services.ResolveConfig(r, envName, format, customAliases)
					if err != nil {
						return cli.NewExitError(err.Error(), 1)
					}

					kvs = services.TransformKeys(kvs, c.Bool("uppercase-keys"), c.String("prefix"), false)

					fmt.Fprintf(os.Stderr, "resolved environment: %s\n", normalized)

					if c.Bool("to-github") {
						return services.WriteToGitHubEnv(kvs)
					}
					if c.Bool("to-github-output") {
						return services.WriteToGitHubOutput(kvs)
					}

					return services.WriteToShell(os.Stdout, kvs)
				},
			},
			{
				Name:      "branch-env",
				Usage:     "map a git branch name to an environment using a JSON mapping",
				ArgsUsage: "BRANCH",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "mapping, m", Usage: `branch-to-env JSON mapping (e.g. '{"main":"production","develop":"dev","release/*":"staging"}')`, Required: true},
					cli.BoolFlag{Name: "to-github", Usage: "write TARGET_ENV to $GITHUB_ENV"},
					cli.BoolFlag{Name: "to-github-output", Usage: "write TARGET_ENV to $GITHUB_OUTPUT"},
					cli.StringFlag{Name: "output-key", Value: "TARGET_ENV", Usage: "output variable name"},
				},
				Action: func(c *cli.Context) error {
					branch := c.Args().First()
					if branch == "" {
						branch = os.Getenv("GITHUB_REF")
						if branch == "" {
							branch = os.Getenv("GITHUB_HEAD_REF")
						}
					}
					if branch == "" {
						return cli.NewExitError("branch name required (as argument or via $GITHUB_REF)", 1)
					}
					branch = strings.TrimPrefix(branch, "refs/heads/")

					env, err := services.BranchToEnv(branch, c.String("mapping"))
					if err != nil {
						return cli.NewExitError(err.Error(), 1)
					}

					outputKey := c.String("output-key")
					kvs := []domain.KeyValue{{Key: outputKey, Value: env}}

					if c.Bool("to-github") {
						return services.WriteToGitHubEnv(kvs)
					}
					if c.Bool("to-github-output") {
						return services.WriteToGitHubOutput(kvs)
					}

					fmt.Println(env)
					return nil
				},
			},
		},
	}
}
