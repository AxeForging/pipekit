package actions

import (
	"fmt"

	"github.com/AxeForging/pipekit/services"

	"github.com/urfave/cli"
)

// VersionCommand returns the version management command group.
func VersionCommand() cli.Command {
	return cli.Command{
		Name:  "version",
		Usage: "extract, bump, and compare semantic versions",
		Subcommands: []cli.Command{
			{
				Name:  "get",
				Usage: "extract version from a project file",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "source, s", Value: "auto", Usage: "version file (auto-detect if 'auto')"},
					cli.StringFlag{Name: "format", Value: "plain", Usage: "output format: plain, json, v-prefixed"},
					cli.StringFlag{Name: "to-github-output", Usage: "export to GITHUB_OUTPUT with this key"},
				},
				Action: func(c *cli.Context) error {
					source := c.String("source")
					if c.Args().First() != "" {
						source = c.Args().First()
					}
					version, err := services.VersionGet(source)
					if err != nil {
						return err
					}
					version = services.FormatVersion(version, c.String("format"))

					if outputKey := c.String("to-github-output"); outputKey != "" {
						return services.WriteToGitHubOutputValue(outputKey, version)
					}

					fmt.Println(version)
					return nil
				},
			},
			{
				Name:      "bump",
				Usage:     "bump major/minor/patch and write back",
				ArgsUsage: "major|minor|patch",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "source, s", Value: "auto", Usage: "version file"},
					cli.StringFlag{Name: "pre-release", Usage: "append pre-release suffix"},
					cli.StringFlag{Name: "build-meta", Usage: "append build metadata"},
					cli.StringFlag{Name: "format", Value: "plain", Usage: "output format: plain, json, v-prefixed"},
					cli.StringFlag{Name: "to-github-output", Usage: "export to GITHUB_OUTPUT with this key"},
				},
				Action: func(c *cli.Context) error {
					bumpType := c.Args().First()
					if bumpType == "" {
						return cli.NewExitError("bump type required: major, minor, or patch", 1)
					}
					newVersion, err := services.VersionBump(c.String("source"), bumpType, c.String("pre-release"), c.String("build-meta"))
					if err != nil {
						return err
					}
					newVersion = services.FormatVersion(newVersion, c.String("format"))

					if outputKey := c.String("to-github-output"); outputKey != "" {
						return services.WriteToGitHubOutputValue(outputKey, newVersion)
					}

					fmt.Println(newVersion)
					return nil
				},
			},
			{
				Name:      "set",
				Usage:     "set an explicit version and write back",
				ArgsUsage: "VERSION",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "source, s", Value: "auto", Usage: "version file"},
				},
				Action: func(c *cli.Context) error {
					v := c.Args().First()
					if v == "" {
						return cli.NewExitError("version required", 1)
					}
					if err := services.VersionSet(c.String("source"), v); err != nil {
						return err
					}
					fmt.Println(v)
					return nil
				},
			},
			{
				Name:      "compare",
				Usage:     "compare two semver strings (exit: 0=eq, 1=gt, 2=lt)",
				ArgsUsage: "V1 V2",
				Action: func(c *cli.Context) error {
					args := c.Args()
					if len(args) < 2 {
						return cli.NewExitError("two version strings required", 1)
					}
					result, err := services.VersionCompare(args[0], args[1])
					if err != nil {
						return err
					}
					switch result {
					case 0:
						fmt.Println("equal")
					case 1:
						fmt.Println("greater")
						return cli.NewExitError("", 1)
					case -1:
						fmt.Println("less")
						return cli.NewExitError("", 2)
					}
					return nil
				},
			},
			{
				Name:  "next",
				Usage: "determine next version from conventional commits",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "source, s", Value: "auto", Usage: "version file"},
					cli.StringFlag{Name: "format", Value: "plain", Usage: "output format: plain, json, v-prefixed"},
					cli.StringFlag{Name: "to-github-output", Usage: "export to GITHUB_OUTPUT with this key"},
				},
				Action: func(c *cli.Context) error {
					version, err := services.VersionNext(c.String("source"))
					if err != nil {
						return err
					}
					version = services.FormatVersion(version, c.String("format"))

					if outputKey := c.String("to-github-output"); outputKey != "" {
						return services.WriteToGitHubOutputValue(outputKey, version)
					}

					fmt.Println(version)
					return nil
				},
			},
		},
	}
}
