package necogcp

import (
	"context"

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
)

// SlackNotifierEntryPoint consumes a Pub/Sub message to send notification via Slack
func SlackNotifierEntryPoint(ctx context.Context, m *pubsub.Message) error {
	log.Debug("msg body", map[string]interface{}{
		"data": string(m.Data),
	})

	b, err := necogcpslack.NewCloudLoggingMessage(m.Data)
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

	name, err := b.GetName()
	if err != nil {
		log.Error("failed to get name", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}
	log.Debug("name", map[string]interface{}{
		"name": name,
	})

	text, err := b.GetText()
	if err != nil {
		log.Error("failed to get text", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}
	log.Debug("text", map[string]interface{}{
		"text": text,
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

	msg, err := b.MakeSlackMessage(color)
	if err != nil {
		log.Error("failed to get slack message", map[string]interface{}{
			"message":   text,
			log.FnError: err,
		})
		return err
	}
	log.Debug("msg", map[string]interface{}{
		"msg": msg,
	})

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
