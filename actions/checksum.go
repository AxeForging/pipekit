package actions

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/AxeForging/pipekit/services"

	"github.com/urfave/cli"
)

// ChecksumCommand returns checksum helpers for release artifacts.
func ChecksumCommand() cli.Command {
	return cli.Command{
		Name:  "checksum",
		Usage: "generate and verify file checksums",
		Subcommands: []cli.Command{
			{
				Name:      "files",
				Usage:     "hash one or more files independently",
				ArgsUsage: "FILE...",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "algorithm, a", Value: "sha256", Usage: "sha256, sha1, or md5"},
					cli.StringFlag{Name: "format, f", Value: "text", Usage: "text or json"},
					cli.StringFlag{Name: "output, o", Usage: "write output to this file"},
				},
				Action: func(c *cli.Context) error {
					files, err := argsOrErr(c, "file")
					if err != nil {
						return err
					}
					sums, err := services.ChecksumFiles(files, c.String("algorithm"))
					if err != nil {
						return err
					}
					var out string
					switch c.String("format") {
					case "json":
						data, err := json.MarshalIndent(sums, "", "  ")
						if err != nil {
							return err
						}
						out = string(data) + "\n"
					case "text", "":
						out = services.FormatChecksums(sums)
					default:
						return cli.NewExitError("unknown format: "+c.String("format"), 1)
					}
					if path := c.String("output"); path != "" {
						return os.WriteFile(path, []byte(out), 0644)
					}
					fmt.Print(out)
					return nil
				},
			},
			{
				Name:      "verify",
				Usage:     "verify a checksum file",
				ArgsUsage: "CHECKSUM_FILE",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "algorithm, a", Value: "sha256", Usage: "sha256, sha1, or md5"},
				},
				Action: func(c *cli.Context) error {
					path, err := firstArgOrErr(c, "CHECKSUM_FILE")
					if err != nil {
						return err
					}
					return services.VerifyChecksums(path, c.String("algorithm"))
				},
			},
		},
	}
}
