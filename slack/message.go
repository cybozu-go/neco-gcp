package slack

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/slack-go/slack"
)

const (
	computeResourceType    = "gce_instance"
	computeInsertEventName = "compute.instances.insert"
	computeDeleteEventName = "compute.instances.delete"
)

// ComputeEngineLog is a JSON-style message of Compute Engine on Cloud Logging
type ComputeEngineLog struct {
	TextPayload string      `json:"textPayload"`
	JSONPayload JSONPayload `json:"jsonPayload"`
	Resource    Resource    `json:"resource"`
	TimeStamp   time.Time   `json:"timestamp"`
}

// JSONPayload is a nested field of MessageBody
type JSONPayload struct {
	Host            string              `json:"host"`
	Ident           string              `json:"ident"`
	Message         string              `json:"message"`
	EventType       string              `json:"event_type"`
	EventSubType    string              `json:"event_subtype"`
	PayloadResource JSONPayloadResource `json:"resource"`
}

// JSONPayloadResource is a nested field of JSONPayload
type JSONPayloadResource struct {
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

// NewComputeEngineLog parses JSON log from Compute Engine API or startup-script, and creates ComputeEngineLog
func NewComputeEngineLog(jsonPayload []byte) (*ComputeEngineLog, error) {
	var m ComputeEngineLog
	err := json.Unmarshal(jsonPayload, &m)
	if err != nil {
		return nil, err
	}

	// NOTE: This JSON-styled log is parsed and may invoke slack notification.
	// If the message includes an invalid JSON value, the cloud function might fall into an infinite loop,
	// like Cloud Functions -> Cloud Logging Sink -> Cloud Functions -> ... .
	// This block is intended to prevent the infinite loop from being invoked.
	if m.Resource.Type != computeResourceType {
		return nil, fmt.Errorf("invalid resource type: %s", m.Resource.Type)
	}

	// This means the log is from startup-script.
	if len(m.JSONPayload.Message) > 0 {
		return &m, nil
	}

	// This means the log is from Compute Engine API but operation has not completed yet.
	if m.JSONPayload.EventType != "GCE_OPERATION_DONE" {
		return nil, fmt.Errorf("invalid event type: %s", m.JSONPayload.EventType)
	}

	// This means the log is from Compute Engine API but the event subtype is neither `create` nor `delete`.
	if m.JSONPayload.EventSubType != computeInsertEventName && m.JSONPayload.EventSubType != computeDeleteEventName {
		return nil, fmt.Errorf("invalid event subtype: %s", m.JSONPayload.EventType)
	}
	return &m, nil
}

// GetName returns instance name
func (m ComputeEngineLog) GetName() string {
	switch m.JSONPayload.EventSubType {
	case computeInsertEventName:
		return m.JSONPayload.PayloadResource.Name
	case computeDeleteEventName:
		return m.JSONPayload.PayloadResource.Name
	default:
		return m.JSONPayload.Host
	}
}

// GetText returns payload text to notify
func (m ComputeEngineLog) GetText() string {
	switch m.JSONPayload.EventSubType {
	case computeInsertEventName:
		return "Instance Inserted"
	case computeDeleteEventName:
		return "Instance Deleted"
	default:
		return m.JSONPayload.Message
	}
}

// GetSlackMessage gets message for Slack WebHook
func (m ComputeEngineLog) GetSlackMessage(color string) *slack.WebhookMessage {
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
