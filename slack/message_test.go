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
			"./log/compute.json",
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

		if m.JSONPayload.Host != tt.name {
			t.Errorf("expect: %s, actual: %s", tt.name, m.JSONPayload.Host)
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
