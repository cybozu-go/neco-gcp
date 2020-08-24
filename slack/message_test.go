package slack

import (
	"io/ioutil"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/slack-go/slack"
)

const (
	projectID = "neco-dev"
	zone      = "asia-northeast1-c"
	region    = "asia-northeast1"
)

func TestCloudLoggingMessage(t *testing.T) {
	testCases := []struct {
		inputFileName string
		color         string
		name          string
		text          string
		msg           *slack.WebhookMessage
	}{
		{
			"./log/startup.json",
			"red",
			"sample-0",
			"startup",
			&slack.WebhookMessage{
				Attachments: []slack.Attachment{
					{
						Color:      "red",
						AuthorName: "GCP Slack Notifier",
						Title:      "Compute Engine",
						Text:       "startup",
						Fields: []slack.AttachmentField{
							{Title: "Project", Value: projectID, Short: true},
							{Title: "Zone", Value: zone, Short: true},
							{Title: "Instance", Value: "sample-0", Short: true},
							{Title: "TimeStamp", Value: "2020-08-14T00:21:34Z", Short: true},
						},
					},
				},
			},
		},
		{
			"./log/delete.json",
			"green",
			"sample-1",
			"",
			&slack.WebhookMessage{
				Attachments: []slack.Attachment{
					{
						Color:      "green",
						AuthorName: "GCP Slack Notifier",
						Title:      "Compute Engine",
						Text:       "Instance Deleted",
						Fields: []slack.AttachmentField{
							{Title: "Project", Value: projectID, Short: true},
							{Title: "Zone", Value: zone, Short: true},
							{Title: "Instance", Value: "sample-1", Short: true},
							{Title: "TimeStamp", Value: "2020-08-24T04:29:30Z", Short: true},
						},
					},
				},
			},
		},
		{
			"./log/insert.json",
			"green",
			"sample-2",
			"",
			&slack.WebhookMessage{
				Attachments: []slack.Attachment{
					{
						Color:      "green",
						AuthorName: "GCP Slack Notifier",
						Title:      "Compute Engine",
						Text:       "Instance Created",
						Fields: []slack.AttachmentField{
							{Title: "Project", Value: projectID, Short: true},
							{Title: "Zone", Value: zone, Short: true},
							{Title: "Instance", Value: "sample-2", Short: true},
							{Title: "TimeStamp", Value: "2020-08-24T06:23:07Z", Short: true},
						},
					},
				},
			},
		},
	}

	for _, tt := range testCases {
		json, err := ioutil.ReadFile(tt.inputFileName)
		if err != nil {
			t.Fatal(err)
		}

		m, err := NewCloudLoggingMessage(json)
		if err != nil {
			t.Fatal(err)
		}

		if m.JSONPayload.Host != tt.name && m.JSONPayload.PayloadResource.Name != tt.name {
			name := m.JSONPayload.Host
			if len(name) == 0 {
				name = m.JSONPayload.PayloadResource.Name
			}
			t.Errorf("expect: %s, actual: %s", tt.name, name)
		}

		if m.JSONPayload.Message != tt.text {
			t.Errorf("expect: %s, actual: %s", tt.text, m.JSONPayload.Message)
		}

		g := m.MakeSlackMessage(tt.color)
		if diff := cmp.Diff(g, tt.msg); diff != "" {
			t.Errorf("diff: %s", diff)
		}
	}
}
