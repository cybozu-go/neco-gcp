package slack

import (
	"encoding/json"
	"time"

	"github.com/slack-go/slack"
)

const (
	computeEngineType    = "gce_instance"
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

// MakeSlackMessage gets message by resource type
func (m CloudLoggingMessage) MakeSlackMessage(color string) *slack.WebhookMessage {
	if len(m.JSONPayload.Message) != 0 {
		return MakeSlackMessageForComputeEngine(
			color,
			m.JSONPayload.Message,
			m.Resource.Labels.ProjectID,
			m.Resource.Labels.Zone,
			m.JSONPayload.Host,
			m.TimeStamp,
		)
	}

	return MakeSlackMessageForComputeEngine(
		color,
		computeDeleteMessage,
		m.Resource.Labels.ProjectID,
		m.Resource.Labels.Zone,
		m.JSONPayload.PayloadResource.Name,
		m.TimeStamp,
	)
}
