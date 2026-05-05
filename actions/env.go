package actions

import (
	"fmt"
	"os"

	"github.com/AxeForging/pipekit/domain"
	"github.com/AxeForging/pipekit/services"

	"github.com/urfave/cli"
)

// EnvCommand returns the env command group.
func EnvCommand() cli.Command {
	commonFlags := []cli.Flag{
		cli.BoolFlag{Name: "uppercase-keys, u", Usage: "convert all keys to UPPER_SNAKE_CASE"},
		cli.StringFlag{Name: "prefix, p", Usage: "add prefix to all keys"},
		cli.BoolFlag{Name: "flatten, f", Usage: "flatten nested structures with _ separator"},
		cli.IntFlag{Name: "depth", Usage: "max depth for flattening (0 = unlimited)"},
		cli.StringFlag{Name: "filter", Usage: "jq-style filter expression"},
		cli.BoolFlag{Name: "strip-quotes", Usage: "remove surrounding quotes from values"},
		cli.BoolFlag{Name: "to-github", Usage: "write to $GITHUB_ENV"},
		cli.BoolFlag{Name: "to-github-output", Usage: "write to $GITHUB_OUTPUT"},
		cli.BoolFlag{Name: "to-gitlab", Usage: "write export statements for GitLab CI"},
	}

	return cli.Command{
		Name:  "env",
		Usage: "extract data from structured files and inject as environment variables",
		Subcommands: []cli.Command{
			{
				Name:  "from-json",
				Usage: "read JSON and export as env vars",
				Flags: commonFlags,
				Action: func(c *cli.Context) error {
					r, err := getInputReader(c)
					if err != nil {
						return err
					}

					kvs, err := services.ParseJSON(r, c.Bool("flatten"), c.Int("depth"), c.String("filter"))
					if err != nil {
						return err
					}

					return processEnvOutput(c, kvs)
				},
			},
			{
				Name:  "from-yaml",
				Usage: "read YAML and export as env vars",
				Flags: commonFlags,
				Action: func(c *cli.Context) error {
					r, err := getInputReader(c)
					if err != nil {
						return err
					}

					kvs, err := services.ParseYAML(r, c.Bool("flatten"), c.Int("depth"), c.String("filter"))
					if err != nil {
						return err
					}

					return processEnvOutput(c, kvs)
				},
			},
			{
				Name:  "from-toml",
				Usage: "read TOML and export as env vars",
				Flags: commonFlags,
				Action: func(c *cli.Context) error {
					r, err := getInputReader(c)
					if err != nil {
						return err
					}

					kvs, err := services.ParseTOML(r, c.Bool("flatten"), c.Int("depth"), c.String("filter"))
					if err != nil {
						return err
					}

					return processEnvOutput(c, kvs)
				},
			},
			{
				Name:  "from-dotenv",
				Usage: "parse .env file and re-export",
				Flags: commonFlags,
				Action: func(c *cli.Context) error {
					r, err := getInputReader(c)
					if err != nil {
						return err
					}

					kvs, err := services.ParseDotenv(r)
					if err != nil {
						return err
					}

					return processEnvOutput(c, kvs)
				},
			},
			{
				Name:  "to-github",
				Usage: "write key=value pairs to $GITHUB_ENV",
				Flags: commonFlags,
				Action: func(c *cli.Context) error {
					r, err := getInputReader(c)
					if err != nil {
						return err
					}

					kvs, err := services.ParseDotenv(r)
					if err != nil {
						return err
					}

					kvs = services.TransformKeys(kvs, c.Bool("uppercase-keys"), c.String("prefix"), c.Bool("strip-quotes"))
					return services.WriteToGitHubEnv(kvs)
				},
			},
			{
				Name:  "to-github-output",
				Usage: "write key=value pairs to $GITHUB_OUTPUT",
				Flags: commonFlags,
				Action: func(c *cli.Context) error {
					r, err := getInputReader(c)
					if err != nil {
						return err
					}

					kvs, err := services.ParseDotenv(r)
					if err != nil {
						return err
					}

					kvs = services.TransformKeys(kvs, c.Bool("uppercase-keys"), c.String("prefix"), c.Bool("strip-quotes"))
					return services.WriteToGitHubOutput(kvs)
				},
			},
			{
				Name:  "to-gitlab",
				Usage: "write export statements for GitLab CI",
				Flags: commonFlags,
				Action: func(c *cli.Context) error {
					r, err := getInputReader(c)
					if err != nil {
						return err
					}

					kvs, err := services.ParseDotenv(r)
					if err != nil {
						return err
					}

					kvs = services.TransformKeys(kvs, c.Bool("uppercase-keys"), c.String("prefix"), c.Bool("strip-quotes"))
					return services.WriteToGitLab(os.Stdout, kvs)
				},
			},
			{
				Name:  "to-shell",
				Usage: "write export KEY=VALUE statements to stdout",
				Flags: commonFlags,
				Action: func(c *cli.Context) error {
					r, err := getInputReader(c)
					if err != nil {
						return err
					}

					kvs, err := services.ParseDotenv(r)
					if err != nil {
						return err
					}

					kvs = services.TransformKeys(kvs, c.Bool("uppercase-keys"), c.String("prefix"), c.Bool("strip-quotes"))
					return services.WriteToShell(os.Stdout, kvs)
				},
			},
		},
	}
}

func processEnvOutput(c *cli.Context, kvs []domain.KeyValue) error {
	kvs = services.TransformKeys(kvs, c.Bool("uppercase-keys"), c.String("prefix"), c.Bool("strip-quotes"))

	if c.Bool("to-github") {
		return services.WriteToGitHubEnv(kvs)
	}
	if c.Bool("to-github-output") {
		return services.WriteToGitHubOutput(kvs)
	}
	if c.Bool("to-gitlab") {
		return services.WriteToGitLab(os.Stdout, kvs)
	}

	return services.WriteToShell(os.Stdout, kvs)
}

func getInputReader(c *cli.Context) (*os.File, error) {
	if c.Args().First() != "" {
		f, err := os.Open(c.Args().First())
		if err != nil {
			return nil, fmt.Errorf("opening %s: %w", c.Args().First(), err)
		}
		return f, nil
	}

	// Check if stdin has data
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		return os.Stdin, nil
	}

	return nil, fmt.Errorf("no input file specified and no data on stdin")
}
