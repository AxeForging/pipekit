package actions

import (
	"encoding/json"
	"fmt"

	"github.com/AxeForging/pipekit/services"

	"github.com/urfave/cli"
)

// DoctorCommand returns the doctor command — environment diagnostics.
func DoctorCommand() cli.Command {
	return cli.Command{
		Name:  "doctor",
		Usage: "diagnose pipekit's runtime environment (CI platform, vars, tools)",
		Flags: []cli.Flag{
			cli.BoolFlag{Name: "json", Usage: "output structured JSON"},
		},
		Action: func(c *cli.Context) error {
			results := services.RunDoctorChecks()
			if c.Bool("json") {
				b, err := json.MarshalIndent(results, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(b))
				return nil
			}
			fmt.Print(services.FormatDoctorText(results))
			return nil
		},
	}
}
