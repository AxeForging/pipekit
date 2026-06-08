package actions

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/AxeForging/pipekit/services"

	"github.com/urfave/cli"
)

// JSONCommand returns the json command group, defaulting to JSON format.
func JSONCommand() cli.Command {
	return dataCommand("json", "read, query, mutate, merge, convert, and pretty-print JSON", services.FormatJSON)
}

// YAMLCommand returns the yaml command group, defaulting to YAML format.
// All subcommands are identical to JSONCommand — file extensions still drive
// per-file decoding, only the stdin default differs.
func YAMLCommand() cli.Command {
	return dataCommand("yaml", "read, query, mutate, merge, convert, and pretty-print YAML", services.FormatYAML)
}

// dataCommand builds a structured-data command tree with the given default
// format for stdin input.
func dataCommand(name, usage string, defaultFormat services.DataFormat) cli.Command {
	return cli.Command{
		Name:  name,
		Usage: usage,
		Subcommands: []cli.Command{
			dataGetCmd(defaultFormat),
			dataSetCmd(defaultFormat),
			dataDelCmd(defaultFormat),
			dataMergeCmd(defaultFormat),
			dataConvertCmd(defaultFormat),
			dataPrettyCmd(defaultFormat),
			dataTableCmd(defaultFormat),
		},
	}
}

func dataGetCmd(def services.DataFormat) cli.Command {
	return cli.Command{
		Name:      "get",
		Usage:     "extract a value at a jq-style path",
		ArgsUsage: "[FILE]",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "path, p", Usage: "jq-style path expression (e.g. .image.tag)"},
			cli.BoolFlag{Name: "raw, r", Usage: "if the result is a string, print it raw (no JSON quotes)"},
			cli.StringFlag{Name: "from", Usage: "input format override (default: detect from extension)"},
			outputKeyFlag(),
		},
		Action: func(c *cli.Context) error {
			doc, _, err := readDoc(c, def)
			if err != nil {
				return err
			}
			path := c.String("path")
			if path == "" {
				return cli.NewExitError("--path required", 1)
			}
			val, err := services.JSONGet(doc, path)
			if err != nil {
				return err
			}
			out := formatJSONResult(val, c.Bool("raw"))
			return emitString(c, out)
		},
	}
}

func dataSetCmd(def services.DataFormat) cli.Command {
	return cli.Command{
		Name:      "set",
		Usage:     "set a value at a path and write back",
		ArgsUsage: "FILE",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "path, p", Usage: "path to set"},
			cli.StringFlag{Name: "value, v", Usage: "string value to set"},
			cli.StringFlag{Name: "json-value, j", Usage: "JSON-encoded value (object/array/number/bool)"},
			cli.BoolFlag{Name: "in-place, i", Usage: "write back to the file (default: stdout)"},
			cli.BoolFlag{Name: "pretty", Usage: "pretty-print output"},
			cli.BoolFlag{Name: "preserve, P", Usage: "surgical edit: keep comments/formatting, change only the target (yaml, json)"},
		},
		Action: func(c *cli.Context) error {
			file, err := firstArgOrErr(c, "FILE")
			if err != nil {
				return err
			}
			path := c.String("path")
			if path == "" {
				return cli.NewExitError("--path required", 1)
			}
			var newVal interface{}
			if c.IsSet("json-value") {
				if err := json.Unmarshal([]byte(c.String("json-value")), &newVal); err != nil {
					return fmt.Errorf("parsing --json-value: %w", err)
				}
			} else {
				newVal = c.String("value")
			}

			if c.Bool("preserve") {
				return writePreserved(c, file, def, c.Bool("in-place"), func(data []byte, format services.DataFormat) ([]byte, error) {
					return services.SetPreserving(data, format, path, newVal)
				})
			}

			doc, _, err := loadFileWithDefault(file, def)
			if err != nil {
				return err
			}
			updated, err := services.JSONSet(doc, path, newVal)
			if err != nil {
				return err
			}
			return writeResult(c, file, updated, def, c.Bool("in-place"), c.Bool("pretty"))
		},
	}
}

func dataDelCmd(def services.DataFormat) cli.Command {
	return cli.Command{
		Name:      "del",
		Usage:     "remove a value at a path",
		ArgsUsage: "FILE",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "path, p", Usage: "path to delete"},
			cli.BoolFlag{Name: "in-place, i"},
			cli.BoolFlag{Name: "pretty"},
			cli.BoolFlag{Name: "preserve, P", Usage: "surgical edit: keep comments/formatting, remove only the target (yaml, json)"},
		},
		Action: func(c *cli.Context) error {
			file, err := firstArgOrErr(c, "FILE")
			if err != nil {
				return err
			}
			path := c.String("path")
			if path == "" {
				return cli.NewExitError("--path required", 1)
			}

			if c.Bool("preserve") {
				return writePreserved(c, file, def, c.Bool("in-place"), func(data []byte, format services.DataFormat) ([]byte, error) {
					return services.DelPreserving(data, format, path)
				})
			}

			doc, _, err := loadFileWithDefault(file, def)
			if err != nil {
				return err
			}
			updated, err := services.JSONDel(doc, path)
			if err != nil {
				return err
			}
			return writeResult(c, file, updated, def, c.Bool("in-place"), c.Bool("pretty"))
		},
	}
}

func dataMergeCmd(def services.DataFormat) cli.Command {
	return cli.Command{
		Name:      "merge",
		Usage:     "deep-merge two or more files (later overrides earlier)",
		ArgsUsage: "BASE OVERLAY [OVERLAY...]",
		Flags: []cli.Flag{
			cli.BoolFlag{Name: "pretty"},
			cli.StringFlag{Name: "output, o", Usage: "write to this file (default: stdout)"},
			cli.StringFlag{Name: "format, f", Usage: "output format: json (default), yaml, toml"},
		},
		Action: func(c *cli.Context) error {
			args := c.Args()
			if len(args) < 2 {
				return cli.NewExitError("at least two files required", 1)
			}
			var merged interface{}
			for i, f := range args {
				doc, _, err := loadFileWithDefault(f, def)
				if err != nil {
					return err
				}
				if i == 0 {
					merged = doc
					continue
				}
				merged = services.DeepMerge(merged, doc)
			}
			out := c.String("output")
			format := pickFormat(c.String("format"), out, def)
			data, err := services.Encode(merged, format, c.Bool("pretty") || format == services.FormatYAML)
			if err != nil {
				return err
			}
			if out != "" {
				return os.WriteFile(out, data, 0644)
			}
			fmt.Print(string(data))
			if format == services.FormatJSON {
				fmt.Println()
			}
			return nil
		},
	}
}

func dataConvertCmd(def services.DataFormat) cli.Command {
	return cli.Command{
		Name:      "convert",
		Usage:     "convert between json/yaml/toml/csv",
		ArgsUsage: "[FILE]",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "from", Usage: "input format (default: detect from extension)"},
			cli.StringFlag{Name: "to", Usage: "output format (json/yaml/toml/csv)"},
			cli.BoolFlag{Name: "pretty"},
		},
		Action: func(c *cli.Context) error {
			doc, _, err := readDoc(c, def)
			if err != nil {
				return err
			}
			to := services.FormatString(c.String("to"))
			if c.String("to") == "" {
				return cli.NewExitError("--to required (json|yaml|toml|csv)", 1)
			}
			pretty := c.Bool("pretty") || to == services.FormatYAML
			data, err := services.Encode(doc, to, pretty)
			if err != nil {
				return err
			}
			fmt.Print(string(data))
			if to == services.FormatJSON {
				fmt.Println()
			}
			return nil
		},
	}
}

func dataPrettyCmd(def services.DataFormat) cli.Command {
	return cli.Command{
		Name:      "pretty",
		Usage:     "pretty-print structured data with configurable indent",
		ArgsUsage: "[FILE]",
		Flags: []cli.Flag{
			cli.IntFlag{Name: "indent, i", Value: 2, Usage: "indent width in spaces"},
			cli.StringFlag{Name: "from", Usage: "input format override"},
		},
		Action: func(c *cli.Context) error {
			data, err := readAllInput(c)
			if err != nil {
				return err
			}
			format := pickInputFormat(c, def)
			if format == services.FormatJSON {
				out, err := services.PrettyJSON(data, c.Int("indent"))
				if err != nil {
					return err
				}
				fmt.Println(string(out))
				return nil
			}
			doc, err := services.Decode(data, format)
			if err != nil {
				return err
			}
			out, err := services.Encode(doc, format, true)
			if err != nil {
				return err
			}
			fmt.Print(string(out))
			return nil
		},
	}
}

func dataTableCmd(def services.DataFormat) cli.Command {
	return cli.Command{
		Name:      "table",
		Usage:     "render an array of objects as an aligned text table",
		ArgsUsage: "[FILE]",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "columns, c", Usage: "comma-separated column names (default: all, sorted)"},
			cli.StringFlag{Name: "from", Usage: "input format override"},
		},
		Action: func(c *cli.Context) error {
			doc, _, err := readDoc(c, def)
			if err != nil {
				return err
			}
			records := services.AsRecords(doc)
			if records == nil {
				return cli.NewExitError("input is not an array of objects", 1)
			}
			columns := splitCSV(c.String("columns"))
			fmt.Print(services.RenderTable(records, columns))
			return nil
		},
	}
}

// readAllInput returns all bytes from the first arg (file) or stdin.
func readAllInput(c *cli.Context) ([]byte, error) {
	if path := c.Args().First(); path != "" {
		return os.ReadFile(path)
	}
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		return io.ReadAll(os.Stdin)
	}
	return nil, fmt.Errorf("no input file specified and no data on stdin")
}

// readDoc reads input bytes and decodes them. Format priority:
// 1. --from flag, 2. file extension, 3. command default.
func readDoc(c *cli.Context, def services.DataFormat) (interface{}, services.DataFormat, error) {
	data, err := readAllInput(c)
	if err != nil {
		return nil, "", err
	}
	format := pickInputFormat(c, def)
	doc, err := services.Decode(data, format)
	return doc, format, err
}

func pickInputFormat(c *cli.Context, def services.DataFormat) services.DataFormat {
	if from := c.String("from"); from != "" {
		return services.FormatString(from)
	}
	if path := c.Args().First(); path != "" {
		if f := services.DetectFormat(path); f != "" {
			return f
		}
	}
	return def
}

// loadFileWithDefault reads a file, choosing format by extension or falling
// back to the command's default.
func loadFileWithDefault(path string, def services.DataFormat) (interface{}, services.DataFormat, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	format := services.DetectFormat(path)
	if format == "" {
		format = def
	}
	doc, err := services.Decode(data, format)
	return doc, format, err
}

// writeResult emits the document either back to file (in-place) or to stdout.
func writeResult(c *cli.Context, srcPath string, doc interface{}, def services.DataFormat, inPlace, pretty bool) error {
	format := services.DetectFormat(srcPath)
	if format == "" {
		format = def
	}
	data, err := services.Encode(doc, format, pretty || format == services.FormatYAML)
	if err != nil {
		return err
	}
	if inPlace {
		return os.WriteFile(srcPath, data, 0644)
	}
	fmt.Print(string(data))
	if format == services.FormatJSON {
		fmt.Println()
	}
	return nil
}

// writePreserved reads the source file's raw bytes, applies a formatting-
// preserving edit, and either writes back in place or prints to stdout. Unlike
// writeResult it never round-trips through Decode/Encode, so the file is changed
// only where the edit lands.
func writePreserved(c *cli.Context, srcPath string, def services.DataFormat, inPlace bool, edit func([]byte, services.DataFormat) ([]byte, error)) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	format := services.DetectFormat(srcPath)
	if format == "" {
		format = def
	}
	out, err := edit(data, format)
	if err != nil {
		return err
	}
	if inPlace {
		return atomicWriteFile(srcPath, out)
	}
	fmt.Print(string(out))
	return nil
}

// atomicWriteFile writes data to a temp file in the same directory, fsyncs it,
// then renames it over the target. The rename is atomic on POSIX, so a crash or
// kill mid-write leaves the original file fully intact rather than truncated.
// The original file's permission bits are preserved.
func atomicWriteFile(path string, data []byte) error {
	mode := os.FileMode(0644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".pipekit-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return err
	}
	return nil
}

func pickFormat(flag, outputPath string, def services.DataFormat) services.DataFormat {
	if flag != "" {
		return services.FormatString(flag)
	}
	if outputPath != "" {
		if f := services.DetectFormat(outputPath); f != "" {
			return f
		}
	}
	return def
}

// formatJSONResult turns a value into a string for emit. With raw=true, a
// string scalar is printed without JSON quoting.
func formatJSONResult(v interface{}, raw bool) string {
	if raw {
		if s, ok := v.(string); ok {
			return s
		}
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}
