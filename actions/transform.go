package actions

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/AxeForging/pipekit/services"

	"github.com/urfave/cli"
)

// TransformCommand returns the transform command group.
func TransformCommand() cli.Command {
	return cli.Command{
		Name:  "transform",
		Usage: "transform values between formats",
		Subcommands: []cli.Command{
			{
				Name:  "base64-encode",
				Usage: "base64 encode stdin or a value",
				Action: func(c *cli.Context) error {
					data, err := readValueOrStdin(c)
					if err != nil {
						return err
					}
					fmt.Println(services.Base64Encode(data))
					return nil
				},
			},
			{
				Name:  "base64-decode",
				Usage: "base64 decode stdin or a value",
				Action: func(c *cli.Context) error {
					data, err := readValueOrStdin(c)
					if err != nil {
						return err
					}
					decoded, err := services.Base64Decode(string(data))
					if err != nil {
						return err
					}
					fmt.Print(string(decoded))
					return nil
				},
			},
			{
				Name:  "url-encode",
				Usage: "URL-encode a string",
				Action: func(c *cli.Context) error {
					data, err := readValueOrStdin(c)
					if err != nil {
						return err
					}
					fmt.Println(services.URLEncode(strings.TrimSpace(string(data))))
					return nil
				},
			},
			{
				Name:  "url-decode",
				Usage: "URL-decode a string",
				Action: func(c *cli.Context) error {
					data, err := readValueOrStdin(c)
					if err != nil {
						return err
					}
					decoded, err := services.URLDecode(strings.TrimSpace(string(data)))
					if err != nil {
						return err
					}
					fmt.Println(decoded)
					return nil
				},
			},
			{
				Name:  "case",
				Usage: "convert between case formats",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "to", Usage: "target case: camel, pascal, snake, upper-snake, kebab, upper, lower"},
				},
				Action: func(c *cli.Context) error {
					toCase := c.String("to")
					if toCase == "" {
						return cli.NewExitError("--to is required", 1)
					}
					data, err := readValueOrStdin(c)
					if err != nil {
						return err
					}
					fmt.Println(services.ConvertCase(strings.TrimSpace(string(data)), toCase))
					return nil
				},
			},
			{
				Name:  "regex",
				Usage: "apply regex find/replace",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "find", Usage: "regex pattern to find"},
					cli.StringFlag{Name: "replace", Usage: "replacement string"},
				},
				Action: func(c *cli.Context) error {
					find := c.String("find")
					replace := c.String("replace")
					if find == "" {
						return cli.NewExitError("--find is required", 1)
					}
					data, err := readValueOrStdin(c)
					if err != nil {
						return err
					}
					result, err := services.RegexReplace(string(data), find, replace)
					if err != nil {
						return err
					}
					fmt.Print(result)
					return nil
				},
			},
			{
				Name:  "template",
				Usage: "Go-template rendering with env vars / JSON as data",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "data", Usage: "JSON file for template data"},
				},
				Action: func(c *cli.Context) error {
					tmplData, err := readValueOrStdin(c)
					if err != nil {
						return err
					}

					var data map[string]interface{}
					if dataFile := c.String("data"); dataFile != "" {
						raw, err := os.ReadFile(dataFile)
						if err != nil {
							return fmt.Errorf("reading data file: %w", err)
						}
						if err := json.Unmarshal(raw, &data); err != nil {
							return fmt.Errorf("parsing data JSON: %w", err)
						}
					}

					result, err := services.RenderTemplate(string(tmplData), data)
					if err != nil {
						return err
					}
					fmt.Print(result)
					return nil
				},
			},
			{
				Name:  "slug",
				Usage: "generate a URL-safe slug from a string (e.g. branch name)",
				Flags: []cli.Flag{
					cli.IntFlag{Name: "max-length", Value: 63, Usage: "maximum slug length (0 = unlimited)"},
					cli.StringFlag{Name: "prefix", Usage: "prepend a prefix to the slug"},
				},
				Action: func(c *cli.Context) error {
					data, err := readValueOrStdin(c)
					if err != nil {
						return err
					}
					slug := services.Slugify(strings.TrimSpace(string(data)), c.Int("max-length"))
					if prefix := c.String("prefix"); prefix != "" {
						slug = prefix + slug
					}
					fmt.Println(slug)
					return nil
				},
			},
			{
				Name:  "hash",
				Usage: "compute hash of a value, file, or stdin",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "algorithm, a", Value: "sha256", Usage: "hash algorithm: sha256, sha1, md5"},
					cli.StringFlag{Name: "file", Usage: "file to hash"},
				},
				Action: func(c *cli.Context) error {
					algorithm := c.String("algorithm")

					var r io.Reader
					if filePath := c.String("file"); filePath != "" {
						f, err := os.Open(filePath)
						if err != nil {
							return fmt.Errorf("opening file: %w", err)
						}
						defer f.Close()
						r = f
					} else if c.Args().First() != "" {
						r = strings.NewReader(c.Args().First())
					} else {
						r = os.Stdin
					}

					hash, err := services.HashData(r, algorithm)
					if err != nil {
						return err
					}
					fmt.Println(hash)
					return nil
				},
			},
		},
	}
}

func readValueOrStdin(c *cli.Context) ([]byte, error) {
	if c.Args().First() != "" {
		return []byte(c.Args().First()), nil
	}

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		return io.ReadAll(os.Stdin)
	}

	return nil, fmt.Errorf("no value provided and no data on stdin")
}
