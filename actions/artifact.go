package actions

import (
	"fmt"
	"os"

	"github.com/AxeForging/pipekit/services"

	"github.com/urfave/cli"
)

// ArtifactCommand returns CI artifact helpers.
func ArtifactCommand() cli.Command {
	return cli.Command{
		Name:  "artifact",
		Usage: "inspect and validate release artifacts",
		Subcommands: []cli.Command{
			{
				Name:      "manifest",
				Usage:     "generate a JSON manifest for files or globs",
				ArgsUsage: "PATH_OR_GLOB...",
				Flags: []cli.Flag{
					cli.BoolFlag{Name: "pretty", Usage: "pretty-print JSON"},
					cli.StringFlag{Name: "output, o", Usage: "write manifest to this file"},
				},
				Action: func(c *cli.Context) error {
					patterns, err := argsOrErr(c, "artifact path or glob")
					if err != nil {
						return err
					}
					entries, err := services.ArtifactManifest(patterns)
					if err != nil {
						return err
					}
					out, err := services.FormatArtifactManifestJSON(entries, c.Bool("pretty"))
					if err != nil {
						return err
					}
					out += "\n"
					if path := c.String("output"); path != "" {
						return os.WriteFile(path, []byte(out), 0644)
					}
					fmt.Print(out)
					return nil
				},
			},
			{
				Name:      "assert",
				Usage:     "fail unless each artifact path or glob resolves",
				ArgsUsage: "PATH_OR_GLOB...",
				Action: func(c *cli.Context) error {
					patterns, err := argsOrErr(c, "artifact path or glob")
					if err != nil {
						return err
					}
					return services.AssertArtifacts(patterns)
				},
			},
		},
	}
}
