package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/AxeForging/pipekit/domain"
)

// sortedFieldKeys returns the keys of a map in lexicographic order, so that
// rendered notification payloads are deterministic across runs.
func sortedFieldKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// BuildSlackPayload returns the JSON-serializable payload that SendSlack
// would post — exposed for testing deterministic field ordering.
func BuildSlackPayload(msg domain.NotifyMessage) map[string]interface{} {
	color := statusColor(msg.Status)
	emoji := statusEmoji(msg.Status)

	var fields []map[string]interface{}
	for _, k := range sortedFieldKeys(msg.Fields) {
		fields = append(fields, map[string]interface{}{
			"type": "mrkdwn",
			"text": fmt.Sprintf("*%s:* %s", k, msg.Fields[k]),
		})
	}

	blocks := []map[string]interface{}{{
		"type": "section",
		"text": map[string]interface{}{
			"type": "mrkdwn",
			"text": fmt.Sprintf("%s *%s*", emoji, msg.Title),
		},
	}}

	if msg.Message != "" {
		blocks = append(blocks, map[string]interface{}{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": msg.Message,
			},
		})
	}

	if len(fields) > 0 {
		blocks = append(blocks, map[string]interface{}{
			"type":   "section",
			"fields": fields,
		})
	}

	return map[string]interface{}{
		"blocks": blocks,
		"attachments": []map[string]interface{}{
			{"color": color},
		},
	}
}

// SendSlack sends a formatted message to a Slack webhook.
func SendSlack(msg domain.NotifyMessage) error {
	color := statusColor(msg.Status)
	emoji := statusEmoji(msg.Status)

	var fields []map[string]interface{}
	for _, k := range sortedFieldKeys(msg.Fields) {
		fields = append(fields, map[string]interface{}{
			"type": "mrkdwn",
			"text": fmt.Sprintf("*%s:* %s", k, msg.Fields[k]),
		})
	}

	blocks := []map[string]interface{}{
		{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": fmt.Sprintf("%s *%s*", emoji, msg.Title),
			},
		},
	}

	if msg.Message != "" {
		blocks = append(blocks, map[string]interface{}{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": msg.Message,
			},
		})
	}

	if len(fields) > 0 {
		blocks = append(blocks, map[string]interface{}{
			"type":   "section",
			"fields": fields,
		})
	}

	payload := map[string]interface{}{
		"blocks": blocks,
		"attachments": []map[string]interface{}{
			{"color": color},
		},
	}

	return postJSON(msg.URL, payload)
}

// SendDiscord sends a formatted message to a Discord webhook.
func SendDiscord(msg domain.NotifyMessage) error {
	color := statusColorInt(msg.Status)
	emoji := statusEmoji(msg.Status)

	var fieldsList []map[string]interface{}
	for _, k := range sortedFieldKeys(msg.Fields) {
		fieldsList = append(fieldsList, map[string]interface{}{
			"name":   k,
			"value":  msg.Fields[k],
			"inline": true,
		})
	}

	embed := map[string]interface{}{
		"title":       fmt.Sprintf("%s %s", emoji, msg.Title),
		"color":       color,
		"description": msg.Message,
	}
	if len(fieldsList) > 0 {
		embed["fields"] = fieldsList
	}

	payload := map[string]interface{}{
		"embeds": []map[string]interface{}{embed},
	}

	return postJSON(msg.URL, payload)
}

// SendTeams sends an Adaptive Card to a Microsoft Teams webhook.
func SendTeams(msg domain.NotifyMessage) error {
	emoji := statusEmoji(msg.Status)

	var bodyItems []map[string]interface{}
	bodyItems = append(bodyItems, map[string]interface{}{
		"type":   "TextBlock",
		"size":   "Medium",
		"weight": "Bolder",
		"text":   fmt.Sprintf("%s %s", emoji, msg.Title),
	})

	if msg.Message != "" {
		bodyItems = append(bodyItems, map[string]interface{}{
			"type": "TextBlock",
			"text": msg.Message,
			"wrap": true,
		})
	}

	if len(msg.Fields) > 0 {
		var facts []map[string]string
		for _, k := range sortedFieldKeys(msg.Fields) {
			facts = append(facts, map[string]string{"title": k, "value": msg.Fields[k]})
		}
		bodyItems = append(bodyItems, map[string]interface{}{
			"type":  "FactSet",
			"facts": facts,
		})
	}

	payload := map[string]interface{}{
		"type":    "message",
		"summary": msg.Title,
		"attachments": []map[string]interface{}{
			{
				"contentType": "application/vnd.microsoft.card.adaptive",
				"content": map[string]interface{}{
					"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
					"type":    "AdaptiveCard",
					"version": "1.4",
					"body":    bodyItems,
				},
			},
		},
	}

	return postJSON(msg.URL, payload)
}

// SendWebhook sends a raw JSON payload to a URL.
func SendWebhook(url string, payload io.Reader) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(url, "application/json", payload)
	if err != nil {
		return fmt.Errorf("sending webhook: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func postJSON(url string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request returned status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func statusColor(status string) string {
	switch strings.ToLower(status) {
	case "success":
		return "#36a64f"
	case "failure":
		return "#e01e5a"
	case "warning":
		return "#ecb22e"
	default:
		return "#1264a3"
	}
}

func statusColorInt(status string) int {
	switch strings.ToLower(status) {
	case "success":
		return 3066993
	case "failure":
		return 15158332
	case "warning":
		return 15844367
	default:
		return 3447003
	}
}

func statusEmoji(status string) string {
	switch strings.ToLower(status) {
	case "success":
		return "\u2705"
	case "failure":
		return "\u274c"
	case "warning":
		return "\u26a0\ufe0f"
	default:
		return "\u2139\ufe0f"
	}
}
