package actions

import (
	"fmt"
	"time"

	"github.com/AxeForging/pipekit/services"

	"github.com/urfave/cli"
)

// TimeCommand returns the time command group.
func TimeCommand() cli.Command {
	return cli.Command{
		Name:  "time",
		Usage: "timestamps, formatting, and arithmetic",
		Subcommands: []cli.Command{
			{
				Name:  "now",
				Usage: "print the current time in the given format",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "format, f", Value: "rfc3339", Usage: "named layout (rfc3339, unix, compact, tag, date, datetime, iso) or Go time layout"},
					cli.BoolFlag{Name: "utc", Usage: "use UTC instead of local time", EnvVar: ""},
					outputKeyFlag(),
				},
				Action: func(c *cli.Context) error {
					t := time.Now()
					if c.Bool("utc") || true {
						t = t.UTC()
					}
					return emitString(c, services.FormatTime(t, c.String("format")))
				},
			},
			{
				Name:      "format",
				Usage:     "reformat a timestamp from --from to --to",
				ArgsUsage: "TIMESTAMP",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "from", Value: "rfc3339", Usage: "input layout name or Go layout"},
					cli.StringFlag{Name: "to", Value: "rfc3339", Usage: "output layout name or Go layout"},
				},
				Action: func(c *cli.Context) error {
					raw, err := firstArgOrErr(c, "TIMESTAMP")
					if err != nil {
						return err
					}
					t, err := services.ParseTime(raw, c.String("from"))
					if err != nil {
						return err
					}
					fmt.Println(services.FormatTime(t, c.String("to")))
					return nil
				},
			},
			{
				Name:      "add",
				Usage:     "add a duration to a timestamp (or to now)",
				ArgsUsage: "DURATION",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "from", Usage: "starting timestamp (default: now)"},
					cli.StringFlag{Name: "input-format", Value: "rfc3339", Usage: "layout used to parse --from"},
					cli.StringFlag{Name: "format, f", Value: "rfc3339", Usage: "output layout"},
				},
				Action: func(c *cli.Context) error {
					raw, err := firstArgOrErr(c, "DURATION")
					if err != nil {
						return err
					}
					d, err := time.ParseDuration(raw)
					if err != nil {
						return fmt.Errorf("parsing duration: %w", err)
					}
					var base time.Time
					if from := c.String("from"); from != "" {
						base, err = services.ParseTime(from, c.String("input-format"))
						if err != nil {
							return err
						}
					} else {
						base = time.Now().UTC()
					}
					fmt.Println(services.FormatTime(services.AddDuration(base, d), c.String("format")))
					return nil
				},
			},
		},
	}
}
