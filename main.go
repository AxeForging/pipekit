package main

import (
	"fmt"
	"os"

	"github.com/AxeForging/pipekit/actions"
	"github.com/AxeForging/pipekit/helpers"

	"github.com/urfave/cli"
)

// Version information - set during build
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	helpers.SetupLogger("info")

	app := cli.NewApp()
	app.Name = "pipekit"
	app.Usage = "CI/CD pipeline Swiss Army knife"
	app.Version = Version

	app.Commands = []cli.Command{
		actions.EnvCommand(),
		actions.MaskCommand(),
		actions.TransformCommand(),
		actions.SummaryCommand(),
		actions.AssertCommand(),
		actions.MatrixCommand(),
		actions.NotifyCommand(),
		actions.WaitCommand(),
		actions.DiffCommand(),
		actions.VersionCommand(),
		actions.RetryCommand(),
		actions.CacheKeyCommand(),
		actions.ChecksumCommand(),
		actions.ArtifactCommand(),
		actions.GitCommand(),
		actions.ChangelogCommand(),
		actions.ConfigCommand(),
		actions.ParseCommand(),
		actions.CommentCommand(),
		actions.JSONCommand(),
		actions.YAMLCommand(),
		actions.RenderCommand(),
		actions.ExecCommand(),
		actions.URLCommand(),
		actions.ImageCommand(),
		actions.TimeCommand(),
		actions.PortCommand(),
		actions.UUIDCommand(),
		actions.RandomCommand(),
		actions.DoctorCommand(),
		{
			Name:  "build-info",
			Usage: "show build version information",
			Action: func(c *cli.Context) error {
				fmt.Printf("pipekit version %s\n", Version)
				fmt.Printf("Build time: %s\n", BuildTime)
				fmt.Printf("Git commit: %s\n", GitCommit)
				return nil
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
