package actions

import (
	"fmt"
	"os"
	"strings"

	"github.com/AxeForging/pipekit/domain"
	"github.com/AxeForging/pipekit/services"

	"github.com/urfave/cli"
)

// NotifyCommand returns the notify command group.
func NotifyCommand() cli.Command {
	commonFlags := []cli.Flag{
		cli.StringFlag{Name: "status", Value: "info", Usage: "message status: success, failure, warning, info"},
		cli.StringFlag{Name: "title", Usage: "message title"},
		cli.StringFlag{Name: "message, m", Usage: "message body"},
		cli.StringSliceFlag{Name: "field", Usage: "key=value fields (repeatable)"},
		cli.StringFlag{Name: "url", Usage: "webhook URL (or use env var)"},
	}

	return cli.Command{
		Name:  "notify",
		Usage: "send webhook notifications",
		Subcommands: []cli.Command{
			{
				Name:  "slack",
				Usage: "send a formatted message to a Slack webhook",
				Flags: commonFlags,
				Action: func(c *cli.Context) error {
					msg, err := buildNotifyMessage(c, "SLACK_WEBHOOK_URL")
					if err != nil {
						return err
					}
					return services.SendSlack(msg)
				},
			},
			{
				Name:  "discord",
				Usage: "send a formatted message to a Discord webhook",
				Flags: commonFlags,
				Action: func(c *cli.Context) error {
					msg, err := buildNotifyMessage(c, "DISCORD_WEBHOOK_URL")
					if err != nil {
						return err
					}
					return services.SendDiscord(msg)
				},
			},
			{
				Name:  "teams",
				Usage: "send an Adaptive Card to MS Teams webhook",
				Flags: commonFlags,
				Action: func(c *cli.Context) error {
					msg, err := buildNotifyMessage(c, "TEAMS_WEBHOOK_URL")
					if err != nil {
						return err
					}
					return services.SendTeams(msg)
				},
			},
			{
				Name:  "webhook",
				Usage: "POST a JSON payload to any URL",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "url", Usage: "webhook URL", Required: true},
					cli.StringFlag{Name: "from-json", Usage: "JSON file to send as payload"},
				},
				Action: func(c *cli.Context) error {
					url := c.String("url")
					if url == "" {
						return cli.NewExitError("--url is required", 1)
					}

					if jsonFile := c.String("from-json"); jsonFile != "" {
						f, err := os.Open(jsonFile)
						if err != nil {
							return fmt.Errorf("opening %s: %w", jsonFile, err)
						}
						defer f.Close()
						return services.SendWebhook(url, f)
					}

					// Read from stdin
					return services.SendWebhook(url, os.Stdin)
				},
			},
		},
	}
}

func buildNotifyMessage(c *cli.Context, envVarName string) (domain.NotifyMessage, error) {
	url := c.String("url")
	if url == "" {
		url = os.Getenv(envVarName)
	}
	if url == "" {
		return domain.NotifyMessage{}, fmt.Errorf("webhook URL required via --url or $%s", envVarName)
	}

	fields := make(map[string]string)
	for _, f := range c.StringSlice("field") {
		parts := strings.SplitN(f, "=", 2)
		if len(parts) == 2 {
			fields[parts[0]] = parts[1]
		}
	}

	return domain.NotifyMessage{
		Status:  c.String("status"),
		Title:   c.String("title"),
		Message: c.String("message"),
		Fields:  fields,
		URL:     url,
	}, nil
}
