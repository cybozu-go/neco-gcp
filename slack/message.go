package slack

import (
	"encoding/json"
	"time"

	"github.com/slack-go/slack"
)

const (
	computeEngineType    = "gce_instance"
	computeCreateMessage = "Instance Created"
	computeDeleteMessage = "Instance Deleted"
)

// CloudLoggingMessage is a JSON-style message from Cloud Logging
type CloudLoggingMessage struct {
	TextPayload string      `json:"textPayload"`
	JSONPayload JSONPayload `json:"jsonPayload"`
	Resource    Resource    `json:"resource"`
	TimeStamp   time.Time   `json:"timestamp"`
}

// JSONPayload is a nested field of MessageBody
type JSONPayload struct {
	Host            string          `json:"host"`
	Ident           string          `json:"ident"`
	Message         string          `json:"message"`
	EventType       string          `json:"event_type"`
	EventSubType    string          `json:"event_subtype"`
	PayloadResource PayloadResource `json:"resource"`
}

// PayloadResource is a nested field of JSONPayload
type PayloadResource struct {
	Name string `json:"name"`
}

// Resource is a nested field of MessageBody
type Resource struct {
	Labels Labels `json:"labels"`
	Type   string `json:"type"`
}

// Labels is a nested field of Resource
type Labels struct {
	InstanceID string `json:"instance_id"`
	ProjectID  string `json:"project_id"`
	Zone       string `json:"zone"`
	Region     string `json:"region"`
}

// NewCloudLoggingMessage creates CloudLoggingMessage from JSON
func NewCloudLoggingMessage(jsonPayload []byte) (*CloudLoggingMessage, error) {
	var m CloudLoggingMessage
	err := json.Unmarshal(jsonPayload, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// GetName returns instance name
func (m CloudLoggingMessage) GetName() string {
	switch m.JSONPayload.EventSubType {
	case "compute.instances.insert":
		return m.JSONPayload.PayloadResource.Name
	case "compute.instances.delete":
		return m.JSONPayload.PayloadResource.Name
	default:
		return m.JSONPayload.Host
	}
}

// GetText returns payload text to notify
func (m CloudLoggingMessage) GetText() string {
	switch m.JSONPayload.EventSubType {
	case "compute.instances.insert":
		return computeCreateMessage
	case "compute.instances.delete":
		return computeDeleteMessage
	default:
		return m.JSONPayload.Message
	}
}

// MakeSlackMessage gets message by resource type
func (m CloudLoggingMessage) MakeSlackMessage(color string) *slack.WebhookMessage {
	attachment := slack.Attachment{
		Color:      color,
		AuthorName: "GCP Slack Notifier",
		Title:      "Compute Engine",
		Text:       m.GetText(),
		Fields: []slack.AttachmentField{
			{Title: "Project", Value: m.Resource.Labels.ProjectID, Short: true},
			{Title: "Zone", Value: m.Resource.Labels.Zone, Short: true},
			{Title: "Instance", Value: m.GetName(), Short: true},
			{Title: "TimeStamp", Value: m.TimeStamp.Format(time.RFC3339), Short: true},
		},
	}

	return &slack.WebhookMessage{
		Attachments: []slack.Attachment{attachment},
	}
}
