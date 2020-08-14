package slack

import (
	"time"

	"github.com/slack-go/slack"
)

// MakeSlackMessageForComputeEngine makes slack message for compute engine
func MakeSlackMessageForComputeEngine(
	color string,
	text string,
	projectID string,
	zone string,
	instanceID string,
	timestamp time.Time,
) *slack.WebhookMessage {
	attachment := slack.Attachment{
		Color:      color,
		AuthorName: "GCP Slack Notifier",
		Title:      "Compute Engine",
		Text:       text,
		Fields: []slack.AttachmentField{
			{Title: "Project", Value: projectID, Short: true},
			{Title: "Zone", Value: zone, Short: true},
			{Title: "Instance", Value: instanceID, Short: true},
			{Title: "TimeStamp", Value: timestamp.Format(time.RFC3339), Short: true},
		},
	}

	return &slack.WebhookMessage{
		Attachments: []slack.Attachment{attachment},
	}
}
