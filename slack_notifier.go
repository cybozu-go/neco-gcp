package necogcp

import (
	"context"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/pubsub"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco-gcp/functions"
	necogcpslack "github.com/cybozu-go/neco-gcp/slack"
	"github.com/slack-go/slack"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

const (
	slackNotifierConfigName = "gce-slack-notifier-config"

	computeEngineType = "gce_instance"
	cloudFunctionType = "cloud_function"
)

// SlackNotifierEntryPoint consumes a Pub/Sub message to send notification via Slack
func SlackNotifierEntryPoint(ctx context.Context, m *pubsub.Message) error {
	log.Debug("msg body", map[string]interface{}{
		"data": string(m.Data),
	})
	var b necogcpslack.MessageBody
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

	var name, text string
	switch b.Resource.Type {
	case computeEngineType:
		name = b.JSONPayload.Host
		text = b.JSONPayload.Message
	case cloudFunctionType:
		name = b.Resource.Labels.FunctionName
		text = b.TextPayload
	default:
		return fmt.Errorf("undefined resource type: %s", b.Resource.Type)
	}

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

	c, err := functions.NewSlackNotifierConfig(result.GetPayload().GetData())
	if err != nil {
		log.Error("failed to read config", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}

	teams, err := c.GetTeamSet(name)
	if err != nil {
		log.Error("failed to get teams", map[string]interface{}{
			"instancename": name,
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

	color, err := c.GetColorFromMessage(text)
	if err != nil {
		log.Error("failed to get color from message", map[string]interface{}{
			"message":   text,
			log.FnError: err,
		})
		return err
	}
	log.Debug("color", map[string]interface{}{
		"color": color,
	})

	var msg *slack.WebhookMessage
	switch b.Resource.Type {
	case computeEngineType:
		msg = necogcpslack.MakeMessageForComputeEngine(
			color,
			b.JSONPayload.Message,
			b.Resource.Labels.ProjectID,
			b.Resource.Labels.Zone,
			b.JSONPayload.Host,
			b.TimeStamp,
		)
	case cloudFunctionType:
		msg = necogcpslack.MakeMessageForCloudFunctions(
			color,
			b.TextPayload,
			b.Resource.Labels.ProjectID,
			b.Resource.Labels.Region,
			b.Resource.Labels.FunctionName,
			b.TimeStamp,
		)
	default:
		return fmt.Errorf("undefined resource type: %s", b.Resource.Type)
	}
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
