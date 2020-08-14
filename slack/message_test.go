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
		{
			"./log/functions.json",
			"blue",
			"auto-dctest",
			"textPayload",
			&slack.WebhookMessage{
				Attachments: []slack.Attachment{
					{
						Color:      "blue",
						AuthorName: "GCP Slack Notifier",
						Title:      "Cloud Functions",
						Text:       "textPayload",
						Fields: []slack.AttachmentField{
							{Title: "Project", Value: projectID, Short: true},
							{Title: "Region", Value: region, Short: true},
							{Title: "Function", Value: "auto-dctest", Short: true},
							{Title: "TimeStamp", Value: "2020-08-14T04:12:47Z", Short: true},
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

		n, err := m.GetName()
		if err != nil {
			t.Fatal(err)
		}
		if n != tt.name {
			t.Errorf("expect: %s, actual: %s", tt.name, n)
		}

		x, err := m.GetText()
		if err != nil {
			t.Fatal(err)
		}
		if x != tt.text {
			t.Errorf("expect: %s, actual: %s", tt.text, x)
		}

		g, err := m.MakeSlackMessage(tt.color)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(g, tt.msg); diff != "" {
			t.Errorf("diff: %s", diff)
		}
	}
}
