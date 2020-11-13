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
)

func TestComputeLogShouldFail(t *testing.T) {
	for _, n := range []string{
		"./log/invalid/from_cloud_function.json",
		"./log/invalid/non_target_event_type.json",
		"./log/invalid/non_target_event_subtype.json",
	} {
		json, err := ioutil.ReadFile(n)
		if err != nil {
			t.Fatal(err)
		}

		m, err := NewComputeLogFromJSON(json)
		if err == nil {
			t.Errorf("filename: %s log: %#v", n, m)
		}
	}
}

func TestComputeLogShouldSucceed(t *testing.T) {
	testCases := []struct {
		inputFileName string
		color         string
		name          string
		text          string
		msg           *slack.WebhookMessage
	}{
		{
			"./log/valid/startup_script.json",
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
			"./log/valid/delete.json",
			"green",
			"sample-1",
			"Instance Deleted",
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
			"./log/valid/insert.json",
			"green",
			"sample-2",
			"Instance Inserted",
			&slack.WebhookMessage{
				Attachments: []slack.Attachment{
					{
						Color:      "green",
						AuthorName: "GCP Slack Notifier",
						Title:      "Compute Engine",
						Text:       "Instance Inserted",
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

		m, err := NewComputeLogFromJSON(json)
		if err != nil {
			t.Fatal(err)
		}

		name := m.GetInstanceName()
		if name != tt.name {
			t.Errorf("expect: %s, actual: %s", tt.name, name)
		}

		text := m.GetPayloadMessage()
		if text != tt.text {
			t.Errorf("expect: %s, actual: %s", tt.text, text)
		}

		g := m.MakeWebhookMessage(tt.color)
		if diff := cmp.Diff(g, tt.msg); diff != "" {
			t.Errorf("diff: %s", diff)
		}
	}
}
