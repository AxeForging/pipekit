package actions

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/AxeForging/pipekit/services"

	"github.com/urfave/cli"
)

// ArchiveCommand returns archive pack/list/unpack helpers.
func ArchiveCommand() cli.Command {
	return cli.Command{
		Name:  "archive",
		Usage: "pack, list, and unpack tar/zip archives",
		Subcommands: []cli.Command{
			{
				Name:      "pack",
				Usage:     "create an archive from files or directories",
				ArgsUsage: "OUTPUT INPUT...",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "format, f", Usage: "zip, tar, tar.gz, tar.xz, or tar.zst (default: detect from output)"},
				},
				Action: func(c *cli.Context) error {
					args := c.Args()
					if len(args) < 2 {
						return cli.NewExitError("usage: archive pack OUTPUT INPUT...", 1)
					}
					if err := services.PackArchive(args[0], args[1:], c.String("format")); err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
					return nil
				},
			},
			{
				Name:      "unpack",
				Usage:     "extract an archive",
				ArgsUsage: "ARCHIVE",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "dest, C", Value: ".", Usage: "destination directory"},
					cli.StringFlag{Name: "format, f", Usage: "zip, tar, tar.gz, tar.xz, or tar.zst (default: detect from input)"},
					cli.IntFlag{Name: "strip-components", Usage: "strip leading path components while extracting"},
				},
				Action: func(c *cli.Context) error {
					input, err := firstArgOrErr(c, "ARCHIVE")
					if err != nil {
						return err
					}
					if err := services.UnpackArchive(input, c.String("dest"), c.String("format"), c.Int("strip-components")); err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
					return nil
				},
			},
			{
				Name:      "list",
				Usage:     "list archive entries",
				ArgsUsage: "ARCHIVE",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "format, f", Usage: "zip, tar, tar.gz, tar.xz, or tar.zst (default: detect from input)"},
					cli.BoolFlag{Name: "json", Usage: "output entry metadata as JSON"},
				},
				Action: func(c *cli.Context) error {
					input, err := firstArgOrErr(c, "ARCHIVE")
					if err != nil {
						return err
					}
					entries, err := services.ListArchive(input, c.String("format"))
					if err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
					if c.Bool("json") {
						b, err := json.MarshalIndent(entries, "", "  ")
						if err != nil {
							return err
						}
						fmt.Println(string(b))
						return nil
					}
					names := make([]string, 0, len(entries))
					for _, entry := range entries {
						names = append(names, entry.Name)
					}
					fmt.Println(strings.Join(names, "\n"))
					return nil
				},
			},
		},
	}
}
