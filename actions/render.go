package actions

import (
	"fmt"
	"os"

	"github.com/AxeForging/pipekit/services"

	"github.com/urfave/cli"
)

// RenderCommand returns the render command.
func RenderCommand() cli.Command {
	return cli.Command{
		Name:      "render",
		Usage:     "render a Go template file with values + sprig-like funcs",
		ArgsUsage: "[TEMPLATE_FILE]",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "template, t", Usage: "template file (alt to positional)"},
			cli.StringSliceFlag{Name: "values, v", Usage: "JSON/YAML/TOML values file (repeatable; later wins)"},
			cli.StringSliceFlag{Name: "set, s", Usage: "key=value override (repeatable, dotted keys ok)"},
			cli.StringFlag{Name: "output, o", Usage: "write output to this file (default: stdout)"},
		},
		Action: func(c *cli.Context) error {
			tmplPath := c.String("template")
			if tmplPath == "" {
				tmplPath = c.Args().First()
			}
			if tmplPath == "" {
				return cli.NewExitError("template file required (positional or --template)", 1)
			}

			values := make(map[string]interface{})
			for _, vp := range c.StringSlice("values") {
				v, err := services.LoadValues(vp)
				if err != nil {
					return err
				}
				values = mergeMaps(values, v)
			}
			if err := services.ApplySetOverrides(values, c.StringSlice("set")); err != nil {
				return err
			}

			out, err := services.RenderTemplateFile(tmplPath, values)
			if err != nil {
				return err
			}
			if outPath := c.String("output"); outPath != "" {
				return os.WriteFile(outPath, []byte(out), 0644)
			}
			fmt.Print(out)
			return nil
		},
	}
}

// mergeMaps deep-merges b into a (returns the merged map). Used for stacking
// multiple --values files.
func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	merged := services.DeepMerge(a, b)
	if m, ok := merged.(map[string]interface{}); ok {
		return m
	}
	return a
}
