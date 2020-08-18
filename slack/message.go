package slack

import (
	"encoding/json"
	"time"

	"github.com/slack-go/slack"
)

const computeEngineType = "gce_instance"

// CloudLoggingMessage is a JSON-style message from Cloud Logging
type CloudLoggingMessage struct {
	TextPayload string      `json:"textPayload"`
	JSONPayload JSONPayload `json:"jsonPayload"`
	Resource    Resource    `json:"resource"`
	TimeStamp   time.Time   `json:"timestamp"`
}

// JSONPayload is a nested field of MessageBody
type JSONPayload struct {
	Host    string `json:"host"`
	Ident   string `json:"ident"`
	Message string `json:"message"`
}

// Resource is a nested field of MessageBody
type Resource struct {
	Labels Labels `json:"labels"`
	Type   string `json:"type"`
}

// Labels is a nested field of Resource
type Labels struct {
	InstanceID   string `json:"instance_id"`
	FunctionName string `json:"function_name"`
	ProjectID    string `json:"project_id"`
	Zone         string `json:"zone"`
	Region       string `json:"region"`
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
	return MakeSlackMessageForComputeEngine(
		color,
		m.JSONPayload.Message,
		m.Resource.Labels.ProjectID,
		m.Resource.Labels.Zone,
		m.JSONPayload.Host,
		m.TimeStamp,
	)
}
