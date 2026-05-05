package actions

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/AxeForging/pipekit/services"

	"github.com/urfave/cli"
)

// firstArgOrErr returns the first positional argument or a CLI exit error
// using the given argument name in the message.
func firstArgOrErr(c *cli.Context, name string) (string, error) {
	v := c.Args().First()
	if v == "" {
		return "", cli.NewExitError(fmt.Sprintf("%s required", name), 1)
	}
	return v, nil
}

// argsOrErr returns all positional arguments, erroring if none were given.
func argsOrErr(c *cli.Context, name string) ([]string, error) {
	args := c.Args()
	if len(args) == 0 {
		return nil, cli.NewExitError(fmt.Sprintf("at least one %s required", name), 1)
	}
	return args, nil
}

// emitString writes a single string value to the right place:
// $GITHUB_OUTPUT (when --to-github-output=<key> is set), otherwise stdout.
func emitString(c *cli.Context, value string) error {
	if outputKey := c.String("to-github-output"); outputKey != "" {
		return services.WriteToGitHubOutputValue(outputKey, value)
	}
	fmt.Println(value)
	return nil
}

// outputKeyFlag returns the standard --to-github-output flag.
func outputKeyFlag() cli.StringFlag {
	return cli.StringFlag{Name: "to-github-output", Usage: "write to $GITHUB_OUTPUT under this key"}
}

// prefixFlag returns the standard --prefix flag.
func prefixFlag() cli.StringFlag {
	return cli.StringFlag{Name: "prefix", Usage: "prefix to prepend to the result"}
}

// readerFromArgOrStdin opens the first positional arg as a file, or returns
// stdin if no arg is given and stdin is piped. Caller closes if file.
func readerFromArgOrStdin(c *cli.Context) (io.ReadCloser, error) {
	if path := c.Args().First(); path != "" {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("opening %s: %w", path, err)
		}
		return f, nil
	}
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		return io.NopCloser(os.Stdin), nil
	}
	return nil, fmt.Errorf("no input file specified and no data on stdin")
}

// readBytesFromArgOrStdin reads the entire input — either the value of the
// first positional arg, or all of stdin if it's piped.
func readBytesFromArgOrStdin(c *cli.Context) ([]byte, error) {
	if v := c.Args().First(); v != "" {
		return []byte(v), nil
	}
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		return io.ReadAll(os.Stdin)
	}
	return nil, fmt.Errorf("no value provided and no data on stdin")
}

// splitCSV splits a comma-separated flag value, trimming whitespace and
// dropping empty entries.
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
