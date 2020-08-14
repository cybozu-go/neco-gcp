package slack

import (
	"time"

	"github.com/slack-go/slack"
)

// MakeMessageForCloudFunctions makes slak message for Cloud Functions
func MakeMessageForCloudFunctions(
	color string,
	text string,
	projectID string,
	region string,
	functionName string,
	timestamp time.Time,
) *slack.WebhookMessage {
	attachment := slack.Attachment{
		Color:      color,
		AuthorName: "GCP Slack Notifier",
		Title:      "Cloud Functions",
		Text:       text,
		Fields: []slack.AttachmentField{
			{Title: "Project", Value: projectID, Short: true},
			{Title: "Region", Value: region, Short: true},
			{Title: "Function", Value: functionName, Short: true},
			{Title: "TimeStamp", Value: timestamp.Format(time.RFC3339), Short: true},
		},
	}

	return &slack.WebhookMessage{
		Attachments: []slack.Attachment{attachment},
	}
}
