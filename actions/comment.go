package actions

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/AxeForging/pipekit/services"

	"github.com/urfave/cli"
)

// CommentCommand returns the markdown comment command group.
func CommentCommand() cli.Command {
	return cli.Command{
		Name:  "comment",
		Usage: "render, inspect, and amend anchored markdown comments",
		Subcommands: []cli.Command{
			{
				Name:  "anchor",
				Usage: "print a hidden pipekit anchor marker",
				Action: func(c *cli.Context) error {
					name, err := firstArgOrErr(c, "anchor name")
					if err != nil {
						return err
					}
					marker, err := services.AnchorMarker(name)
					if err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
					fmt.Println(marker)
					return nil
				},
			},
			{
				Name:  "fence",
				Usage: "render stdin or a file as a fenced markdown code block",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "language, l", Usage: "code fence language tag"},
					cli.StringFlag{Name: "output, o", Usage: "write output to this file"},
				},
				Action: func(c *cli.Context) error {
					body, err := readInputFileOrStdin(c)
					if err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
					return writeCommentOutput(c, services.RenderCodeFence(c.String("language"), string(body)))
				},
			},
			{
				Name:  "render",
				Usage: "render a markdown comment body with a hidden anchor",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "anchor, a", Usage: "hidden anchor name", Required: true},
					cli.StringFlag{Name: "body-file", Usage: "read visible markdown body from file"},
					cli.StringFlag{Name: "output, o", Usage: "write output to this file"},
				},
				Action: func(c *cli.Context) error {
					body, err := readCommentBody(c)
					if err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
					out, err := services.RenderAnchoredComment(c.String("anchor"), body)
					if err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
					return writeCommentOutput(c, out)
				},
			},
			{
				Name:  "amend",
				Usage: "replace the visible body after a hidden anchor",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "anchor, a", Usage: "hidden anchor name", Required: true},
					cli.StringFlag{Name: "body-file", Usage: "read replacement markdown body from file", Required: true},
					cli.StringFlag{Name: "output, o", Usage: "write output to this file"},
				},
				Action: func(c *cli.Context) error {
					existing, err := readInputFileOrStdin(c)
					if err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
					body, err := os.ReadFile(c.String("body-file"))
					if err != nil {
						return cli.NewExitError(fmt.Sprintf("reading body file: %v", err), 1)
					}
					out, err := services.AmendAnchoredComment(string(existing), c.String("anchor"), string(body))
					if err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
					return writeCommentOutput(c, out)
				},
			},
			{
				Name:  "inspect",
				Usage: "inspect anchors and fenced blocks in markdown or GitHub comments JSON",
				Action: func(c *cli.Context) error {
					r, err := readerFromArgOrStdin(c)
					if err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
					defer r.Close()
					comments, err := services.InspectComments(r)
					if err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
					return encodeCommentJSON(comments)
				},
			},
			{
				Name:  "select",
				Usage: "select the first GitHub comment JSON item containing an anchor",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "anchor, a", Usage: "hidden anchor name", Required: true},
					cli.StringFlag{Name: "format, f", Value: "json", Usage: "output format: json, id, body, url"},
				},
				Action: func(c *cli.Context) error {
					r, err := readerFromArgOrStdin(c)
					if err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
					defer r.Close()
					comments, err := services.InspectComments(r)
					if err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
					comment, ok := services.SelectAnchoredComment(comments, c.String("anchor"))
					if !ok {
						return cli.NewExitError("no matching anchored comment found", 1)
					}
					switch c.String("format") {
					case "json", "":
						return encodeCommentJSON(comment)
					case "id":
						fmt.Println(comment.ID)
					case "body":
						fmt.Print(comment.Body)
					case "url":
						fmt.Println(comment.URL)
					default:
						return cli.NewExitError("unsupported format: use json, id, body, or url", 1)
					}
					return nil
				},
			},
		},
	}
}

func readCommentBody(c *cli.Context) (string, error) {
	if path := c.String("body-file"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("reading body file: %w", err)
		}
		return string(data), nil
	}
	data, err := readBytesFromArgOrStdin(c)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func readInputFileOrStdin(c *cli.Context) ([]byte, error) {
	r, err := readerFromArgOrStdin(c)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func writeCommentOutput(c *cli.Context, content string) error {
	if path := c.String("output"); path != "" {
		return os.WriteFile(path, []byte(content), 0644)
	}
	fmt.Print(content)
	return nil
}

func encodeCommentJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
