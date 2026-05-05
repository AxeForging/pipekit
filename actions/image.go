package actions

import (
	"encoding/json"
	"fmt"

	"github.com/AxeForging/pipekit/domain"
	"github.com/AxeForging/pipekit/services"

	"github.com/urfave/cli"
)

// ImageCommand returns the image command group.
func ImageCommand() cli.Command {
	return cli.Command{
		Name:  "image",
		Usage: "container image reference helpers",
		Subcommands: []cli.Command{
			{
				Name:      "parse",
				Usage:     "split a container image ref into registry/repository/tag/digest",
				ArgsUsage: "IMAGE_REF",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "prefix, p", Value: "IMAGE_", Usage: "prefix for emitted env keys"},
					cli.BoolFlag{Name: "json", Usage: "output as JSON"},
					cli.BoolFlag{Name: "to-github", Usage: "write to $GITHUB_ENV"},
					cli.BoolFlag{Name: "to-github-output", Usage: "write to $GITHUB_OUTPUT"},
				},
				Action: func(c *cli.Context) error {
					raw, err := firstArgOrErr(c, "IMAGE_REF")
					if err != nil {
						return err
					}
					ref, err := services.ParseImage(raw)
					if err != nil {
						return err
					}
					if c.Bool("json") {
						b, _ := json.Marshal(ref)
						fmt.Println(string(b))
						return nil
					}
					prefix := c.String("prefix")
					kvs := []domain.KeyValue{
						{Key: prefix + "REGISTRY", Value: ref.Registry},
						{Key: prefix + "REPOSITORY", Value: ref.Repository},
						{Key: prefix + "TAG", Value: ref.Tag},
						{Key: prefix + "DIGEST", Value: ref.Digest},
					}
					if c.Bool("to-github") {
						return services.WriteToGitHubEnv(kvs)
					}
					if c.Bool("to-github-output") {
						return services.WriteToGitHubOutput(kvs)
					}
					for _, kv := range kvs {
						fmt.Printf("%s=%s\n", kv.Key, kv.Value)
					}
					return nil
				},
			},
		},
	}
}
