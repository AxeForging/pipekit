package actions

import (
	"fmt"

	"github.com/AxeForging/pipekit/services"

	"github.com/urfave/cli"
)

// GitCommand returns git metadata helpers for CI.
func GitCommand() cli.Command {
	return cli.Command{
		Name:  "git",
		Usage: "read git metadata in CI-friendly formats",
		Subcommands: []cli.Command{
			{
				Name:  "sha",
				Usage: "print the current commit SHA",
				Flags: []cli.Flag{
					cli.BoolFlag{Name: "short, s", Usage: "print short SHA"},
					outputKeyFlag(),
				},
				Action: func(c *cli.Context) error {
					sha, err := services.GitSHA(c.Bool("short"))
					if err != nil {
						return err
					}
					return emitString(c, sha)
				},
			},
			{
				Name:  "ref",
				Usage: "print the current branch or tag name",
				Flags: []cli.Flag{
					cli.BoolFlag{Name: "slug", Usage: "slugify the ref for tags, image names, and preview envs"},
					cli.IntFlag{Name: "max-length", Usage: "truncate slug to this length"},
					outputKeyFlag(),
				},
				Action: func(c *cli.Context) error {
					ref, err := services.GitRef(c.Bool("slug"), c.Int("max-length"))
					if err != nil {
						return err
					}
					return emitString(c, ref)
				},
			},
			{
				Name:  "current-tag",
				Usage: "print the tag pointing at HEAD",
				Flags: []cli.Flag{outputKeyFlag()},
				Action: func(c *cli.Context) error {
					tag, err := services.GitCurrentTag()
					if err != nil {
						return err
					}
					return emitString(c, tag)
				},
			},
			{
				Name:  "previous-tag",
				Usage: "print the latest reachable tag",
				Flags: []cli.Flag{outputKeyFlag()},
				Action: func(c *cli.Context) error {
					tag, err := services.GitPreviousTag()
					if err != nil {
						return err
					}
					return emitString(c, tag)
				},
			},
			{
				Name:  "is-dirty",
				Usage: "exit 0 when the working tree is dirty, 1 when clean",
				Flags: []cli.Flag{
					cli.BoolFlag{Name: "print", Usage: "print true or false instead of using only the exit code"},
					outputKeyFlag(),
				},
				Action: func(c *cli.Context) error {
					dirty, err := services.GitIsDirty()
					if err != nil {
						return err
					}
					if c.Bool("print") || c.String("to-github-output") != "" {
						return emitString(c, fmt.Sprintf("%t", dirty))
					}
					if dirty {
						return nil
					}
					return cli.NewExitError("working tree is clean", 1)
				},
			},
		},
	}
}
