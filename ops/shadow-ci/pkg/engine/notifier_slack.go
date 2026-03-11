package engine

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// SlackNotifier sends revert notifications to Slack via webhook.
type SlackNotifier struct {
	webhookURL string
	channel    string
}

// NewSlackNotifier creates a new SlackNotifier.
func NewSlackNotifier(webhookURL, channel string) *SlackNotifier {
	return &SlackNotifier{webhookURL: webhookURL, channel: channel}
}

// NotifyRevert sends a Slack message about a revert decision.
func (sn *SlackNotifier) NotifyRevert(decision *RevertDecision) error {
	payload := sn.formatMessage(decision)

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling slack payload: %w", err)
	}

	resp, err := http.Post(sn.webhookURL, "application/json", strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("sending slack webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (sn *SlackNotifier) formatMessage(decision *RevertDecision) map[string]any {
	text := fmt.Sprintf(":rotating_light: *Auto-Revert Decision*\n"+
		"*Commit:* `%s`\n"+
		"*PR:* #%d\n"+
		"*Reason:* %s\n"+
		"*Failed Tests:* %d\n",
		decision.CulpritCommit,
		decision.CulpritPR,
		decision.Reason,
		len(decision.FailedTests),
	)

	if len(decision.FailedTests) > 0 {
		text += "*Tests:*\n"
		for i, t := range decision.FailedTests {
			if i >= 10 {
				text += fmt.Sprintf("  ... and %d more\n", len(decision.FailedTests)-10)
				break
			}
			text += fmt.Sprintf("  • `%s`\n", t)
		}
	}

	msg := map[string]any{"text": text}
	if sn.channel != "" {
		msg["channel"] = sn.channel
	}
	return msg
}
