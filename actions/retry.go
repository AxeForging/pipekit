package actions

import (
	"strconv"
	"strings"
	"time"

	"github.com/AxeForging/pipekit/services"

	"github.com/urfave/cli"
)

// RetryCommand returns the retry command group.
func RetryCommand() cli.Command {
	return cli.Command{
		Name:  "retry",
		Usage: "run commands with configurable retry logic",
		Subcommands: []cli.Command{
			{
				Name:      "run",
				Usage:     "execute a command with retries on failure",
				ArgsUsage: "-- COMMAND [ARGS...]",
				Flags: []cli.Flag{
					cli.IntFlag{Name: "attempts", Value: 3, Usage: "max number of attempts"},
					cli.StringFlag{Name: "delay", Value: "10s", Usage: "delay between attempts"},
					cli.BoolFlag{Name: "backoff", Usage: "exponential backoff (2x each retry)"},
					cli.StringFlag{Name: "on-exit-codes", Usage: "comma-separated exit codes to retry on"},
					cli.BoolFlag{Name: "quiet, q", Usage: "suppress retry messages"},
				},
				Action: func(c *cli.Context) error {
					args := c.Args()
					if len(args) == 0 {
						return cli.NewExitError("command required after --", 1)
					}
					delay, err := time.ParseDuration(c.String("delay"))
					if err != nil {
						return cli.NewExitError("invalid delay: "+err.Error(), 1)
					}

					var exitCodes []int
					if codes := c.String("on-exit-codes"); codes != "" {
						for _, s := range strings.Split(codes, ",") {
							code, err := strconv.Atoi(strings.TrimSpace(s))
							if err != nil {
								return cli.NewExitError("invalid exit code: "+s, 1)
							}
							exitCodes = append(exitCodes, code)
						}
					}

					if err := services.RetryRun(args, c.Int("attempts"), delay, c.Bool("backoff"), exitCodes, c.Bool("quiet")); err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
					return nil
				},
			},
		},
	}
}
