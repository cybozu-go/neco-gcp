package necogcpfunctions

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/pubsub"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco-gcp/gcp/functions"
	necogcpslack "github.com/cybozu-go/neco-gcp/slack"
	"github.com/kelseyhightower/envconfig"
	"github.com/slack-go/slack"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

const (
	deleteInstancesMode = "delete"
	createInstancesMode = "create"

	necoBranch     = "release"
	necoAppsBranch = "release"

	machineType = "n1-standard-32"

	skipAutoDeleteLabelKey      = "skip-auto-delete"
	excludeSkipAutoDeleteFilter = "-labels." + skipAutoDeleteLabelKey + ":*"

	slackNotifierConfigName = "gce-slack-notifier-config"
)

// Body is body of Pub/Sub message.
type Body struct {
	Mode               string `json:"mode"`
	InstanceNamePrefix string `json:"namePrefix"`
	InstancesNum       int    `json:"num"`
	DoForceDelete      bool   `json:"doForce"`
}

// Env is cloud function environment variables
type Env struct {
	ProjectID string `envconfig:"GCP_PROJECT" required:"true"`
	Zone      string `envconfig:"ZONE" required:"true"`
}

// AutoDCTestEntryPoint consumes a Pub/Sub message
func AutoDCTestEntryPoint(ctx context.Context, m *pubsub.Message) error {
	log.Debug("msg body", map[string]interface{}{
		"data": string(m.Data),
	})
	var b Body
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

	var e Env
	err = envconfig.Process("", &e)
	if err != nil {
		log.Error("failed to parse env vars", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}
	log.Debug("cloud functions env", map[string]interface{}{
		"env": e,
	})

	client, err := functions.NewComputeClient(ctx, e.ProjectID, e.Zone)
	if err != nil {
		log.Error("failed to create client", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}
	runner := functions.NewRunner(client)

	switch b.Mode {
	case createInstancesMode:
		log.Info("create instance(s)", map[string]interface{}{
			"prefix": b.InstanceNamePrefix,
			"num":    b.InstancesNum,
		})

		today, err := getDateStrInJST()
		if err != nil {
			log.Error("failed to get today's date", map[string]interface{}{
				log.FnError: err,
			})
			return err
		}
		log.Debug("today is "+today, map[string]interface{}{})
		if isHoliday(today, jpHolidays) {
			log.Info("today is holiday! skip creating dctest", map[string]interface{}{})
			return nil
		}

		builder, err := functions.NewNecoStartupScriptBuilder().
			WithFluentd().
			WithNeco(necoBranch).
			WithNecoApps(necoAppsBranch)
		if err != nil {
			log.Error("failed to construct startup-script builder", map[string]interface{}{
				log.FnError: err,
			})
			return err
		}
		return runner.CreateInstancesIfNotExist(
			ctx,
			b.InstanceNamePrefix,
			b.InstancesNum,
			functions.MakeNecoDevServiceAccountEmail(e.ProjectID),
			machineType,
			functions.MakeVMXEnabledImageURL(e.ProjectID),
			builder.Build(),
		)
	case deleteInstancesMode:
		log.Info("delete all instance(s)", map[string]interface{}{
			"force": b.DoForceDelete,
		})
		if b.DoForceDelete {
			return runner.DeleteFilteredInstances(ctx, "")
		}
		return runner.DeleteFilteredInstances(ctx, excludeSkipAutoDeleteFilter)
	default:
		err := fmt.Errorf("invalid mode was given: %s", b.Mode)
		log.Error(err.Error(), map[string]interface{}{})
		return err
	}
}

// SlackBody is a pubsub message body from GCE's fluentd
type SlackBody struct {
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
	var b SlackBody
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

	c, err := necogcpslack.NewConfig(result.GetPayload().GetData())
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

	msg := makeGCEMessage(
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

func makeGCEMessage(
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
