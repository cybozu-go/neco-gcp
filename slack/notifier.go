package slack

import (
	"context"
	"fmt"
	"regexp"

	"github.com/slack-go/slack"
	"sigs.k8s.io/yaml"
)

const defaultColor = "good"

// Notifier is a Slack notifier
type Notifier struct {
	Teams    map[string]string `yaml:"teams"`
	Severity []Severity        `yaml:"severity"`
	Rules    []Rule            `yaml:"rules"`
}

// Severity represents relationships betwen color and regex
type Severity struct {
	Color string `yaml:"color"`
	Regex string `yaml:"regex"`
}

// Rule is a notification rule for Slack
type Rule struct {
	Name        string   `yaml:"name"`
	Regex       string   `yaml:"regex"`
	TargetTeams []string `yaml:"targetTeams"`
}

// NewNotifier creates new Notifier from config YAML
func NewNotifier(configYAML string) (*Notifier, error) {
	var n Notifier
	err := yaml.Unmarshal([]byte(configYAML), &n)
	if err != nil {
		return nil, err
	}

	return &n, nil
}

// Notify notifies message via Slack webhook
func (n Notifier) Notify(ctx context.Context, webhookURL, color, message string) error {
	attachment := slack.Attachment{
		Color: color,
		Text:  message,
	}
	msg := slack.WebhookMessage{
		Attachments: []slack.Attachment{attachment},
	}

	return slack.PostWebhookContext(
		ctx,
		webhookURL,
		&msg,
	)
}

// GetURLSetOfMatchedTeams returns matched URLs of target teams
func (n Notifier) GetURLSetOfMatchedTeams(target string) (map[string]struct{}, error) {
	urls := make(map[string]struct{})
	for _, r := range n.Rules {
		matched, err := regexp.Match(r.Regex, []byte(target))
		if err != nil {
			return nil, err
		}
		if !matched {
			continue
		}
		for _, t := range r.TargetTeams {
			v, ok := n.Teams[t]
			if !ok {
				return nil, fmt.Errorf("cannot find %s in teams field", t)
			}
			urls[v] = struct{}{}
		}
	}
	return urls, nil
}

// GetColorFromMessage returns color by maching regex with message
func (n Notifier) GetColorFromMessage(message string) (string, error) {
	for _, s := range n.Severity {
		matched, err := regexp.Match(s.Regex, []byte(message))
		if err != nil {
			return "", err
		}
		if !matched {
			continue
		}
		return s.Color, nil
	}

	return defaultColor, nil
}
