package actions

import (
	"fmt"

	"github.com/AxeForging/pipekit/services"

	"github.com/urfave/cli"
)

// PortCommand returns the port command group.
func PortCommand() cli.Command {
	return cli.Command{
		Name:  "port",
		Usage: "TCP port helpers",
		Subcommands: []cli.Command{
			{
				Name:  "free",
				Usage: "print an unused TCP port",
				Flags: []cli.Flag{
					cli.IntFlag{Name: "low", Usage: "lower bound of port range (optional)"},
					cli.IntFlag{Name: "high", Usage: "upper bound of port range (optional)"},
					outputKeyFlag(),
				},
				Action: func(c *cli.Context) error {
					var p int
					var err error
					if c.IsSet("low") || c.IsSet("high") {
						low := c.Int("low")
						high := c.Int("high")
						if low == 0 {
							low = 1024
						}
						if high == 0 {
							high = 65535
						}
						p, err = services.FreePortInRange(low, high)
					} else {
						p, err = services.FreePort()
					}
					if err != nil {
						return err
					}
					return emitString(c, fmt.Sprintf("%d", p))
				},
			},
		},
	}
}

// UUIDCommand returns the uuid command.
func UUIDCommand() cli.Command {
	return cli.Command{
		Name:  "uuid",
		Usage: "generate a UUID v4",
		Flags: []cli.Flag{
			cli.BoolFlag{Name: "short, s", Usage: "first 8 characters only"},
			outputKeyFlag(),
		},
		Action: func(c *cli.Context) error {
			return emitString(c, services.NewUUID(c.Bool("short")))
		},
	}
}

// RandomCommand returns the random command.
func RandomCommand() cli.Command {
	return cli.Command{
		Name:  "random",
		Usage: "generate a random string from a chosen alphabet",
		Flags: []cli.Flag{
			cli.IntFlag{Name: "length, l", Value: 16, Usage: "string length"},
			cli.StringFlag{Name: "alphabet, a", Value: "alnum", Usage: "alnum, alpha, hex, base32, digits, lower, upper"},
			outputKeyFlag(),
		},
		Action: func(c *cli.Context) error {
			s, err := services.RandomString(c.Int("length"), c.String("alphabet"))
			if err != nil {
				return err
			}
			return emitString(c, s)
		},
	}
}
