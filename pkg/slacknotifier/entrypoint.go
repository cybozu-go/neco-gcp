package slacknotifier

import (
	"context"

	"cloud.google.com/go/pubsub"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/cybozu-go/log"
	"github.com/slack-go/slack"
)

const slackNotifierConfigName = "slack-notifier-config"

// makeConfigURL retruns config url for slack notifier
func makeConfigURL(projectID string) string {
	return "projects/" + projectID + "/secrets/" + slackNotifierConfigName + "/versions/latest"
}

// EntryPoint consumes a Pub/Sub message to send notification via Slack
func EntryPoint(ctx context.Context, m *pubsub.Message) error {
	log.Debug("msg body", map[string]interface{}{
		"data": string(m.Data),
	})

	b, err := NewComputeLog(m.Data)
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

	result, err := client.AccessSecretVersion(
		ctx,
		&secretmanagerpb.AccessSecretVersionRequest{
			Name: makeConfigURL(b.GetProjectID()),
		},
	)
	if err != nil {
		log.Error("failed to access secret version", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}
	log.Info("notifier config YAML is successfully fetched", map[string]interface{}{
		"len": len(result.GetPayload().GetData()),
	})

	c, err := NewConfig(result.GetPayload().GetData())
	if err != nil {
		log.Error("failed to read config", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}

	name := b.GetInstanceName()
	teams, err := c.FindTeamsByInstanceName(name)
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

	urls, err := c.GetWebHookURLsFromTeams(teams)
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

	logMsg := b.GetMessage()
	color, err := c.FindColorByMessage(logMsg)
	if err != nil {
		log.Error("failed to get color from message", map[string]interface{}{
			"text":      logMsg,
			log.FnError: err,
		})
		return err
	}
	log.Debug("color", map[string]interface{}{
		"color": color,
	})

	whMsg := NewSlackWebhookMessageForCompute(
		b.GetProjectID(),
		b.GetZone(),
		b.GetTimeStamp(),
		b.GetInstanceName(),
		b.GetMessage(),
		color,
	)
	log.Debug("msg", map[string]interface{}{
		"msg": whMsg,
	})

	for url := range urls {
		err = slack.PostWebhookContext(ctx, url, whMsg)
		if err != nil {
			log.Error("failed to post slack message", map[string]interface{}{
				"message":   whMsg,
				log.FnError: err,
			})
			return err
		}
	}
	return nil
}
