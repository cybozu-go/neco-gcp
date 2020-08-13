package necogcp

import (
	"context"
	"encoding/json"
	"time"

	"cloud.google.com/go/pubsub"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco-gcp/functions"
	"github.com/slack-go/slack"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

// SlackNotifierMessageBody is a pubsub message body from GCE's fluentd
type SlackNotifierMessageBody struct {
	JSONPayload JSONPayload `json:"jsonPayload"`
	Resource    Resource    `json:"resource"`
	TimeStamp   time.Time   `json:"timestamp"`
}

// JSONPayload is a nested field of SlackBody
type JSONPayload struct {
	Host    string `json:"host"`
	Ident   string `json:"ident"`
	Message string `json:"message"`
}

// Resource is a nested field of SlackBody
type Resource struct {
	Labels Labels `json:"labels"`
}

// Labels is a nested field of Resource
type Labels struct {
	InstanceID string `json:"instance_id"`
	ProjectID  string `json:"project_id"`
	Zone       string `json:"zone"`
}

// SlackNotifierEntryPoint consumes a Pub/Sub message to send notification via Slack
func SlackNotifierEntryPoint(ctx context.Context, m *pubsub.Message) error {
	log.Debug("msg body", map[string]interface{}{
		"data": string(m.Data),
	})
	var b SlackNotifierMessageBody
	err := json.Unmarshal(m.Data, &b)
	if err != nil {
		log.Error("failed to unmarshal json", map[string]interface{}{
			"data":      string(m.Data),
			log.FnError: err,
		})
		return err
	}
	log.Debug("unmarshalled msg body", map[string]interface{}{
		"body": b,
	})

	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		log.Error("failed to setup client", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}

	accessRequest := &secretmanagerpb.AccessSecretVersionRequest{
		Name: "projects/" + b.Resource.Labels.ProjectID + "/secrets/" + slackNotifierConfigName + "/versions/latest",
	}

	result, err := client.AccessSecretVersion(ctx, accessRequest)
	if err != nil {
		log.Error("failed to access secret version", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}
	log.Info("notifier config YAML is successfully fetched", map[string]interface{}{
		"len": len(result.GetPayload().GetData()),
	})

	c, err := functions.NewConfig(result.GetPayload().GetData())
	if err != nil {
		log.Error("failed to read config", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}

	teams, err := c.GetTeamSet(b.JSONPayload.Host)
	if err != nil {
		log.Error("failed to get teams", map[string]interface{}{
			"instancename": b.JSONPayload.Host,
			log.FnError:    err,
		})
		return err
	}
	log.Info("target teams", map[string]interface{}{
		"teams": teams,
	})

	urls, err := c.ConvertTeamsToURLs(teams)
	if err != nil {
		log.Error("failed to convert teams to URLs", map[string]interface{}{
			"teams":     teams,
			log.FnError: err,
		})
		return err
	}
	if len(urls) == 0 {
		log.Info("No target URL is selected.", map[string]interface{}{
			"teams": teams,
		})
		return nil
	}

	color, err := c.GetColorFromMessage(b.JSONPayload.Message)
	if err != nil {
		log.Error("failed to get color from message", map[string]interface{}{
			"message":   b.JSONPayload.Message,
			log.FnError: err,
		})
		return err
	}
	log.Debug("color", map[string]interface{}{
		"color": color,
	})

	msg := makeSlackMessageForGCE(
		color,
		b.JSONPayload.Message,
		b.Resource.Labels.ProjectID,
		b.Resource.Labels.Zone,
		b.JSONPayload.Host,
		b.TimeStamp,
	)
	for url := range urls {
		err = slack.PostWebhookContext(ctx, url, msg)
		if err != nil {
			log.Error("failed to post message from message", map[string]interface{}{
				"message":   msg,
				log.FnError: err,
			})
			return err
		}
	}
	return nil
}

func makeSlackMessageForGCE(
	color string,
	text string,
	projectID string,
	zone string,
	instanceID string,
	timestamp time.Time,
) *slack.WebhookMessage {
	attachment := slack.Attachment{
		Color:      color,
		AuthorName: "GCE Slack Notifier",
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
