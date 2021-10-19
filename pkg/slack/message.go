package slack

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"time"

	"github.com/slack-go/slack"
)

const (
	computeServiceType = "gce_instance"

	computeInsertMethodName = "v1.compute.instances.insert"
	computeDeleteMethodName = "v1.compute.instances.delete"
	startupScriptMethodName = "startup-script"

	computeInsertMessage = "Instance Inserted"
	computeDeleteMessage = "Instance Deleted"

	computeAPILogName    = "cloudaudit.googleapis.com%2Factivity"
	startupScriptLogName = "systemd"
)

// ComputeLog is a JSON-style log from Compute Engine
type ComputeLog interface {
	GetInstanceName() string
	GetMethodName() string
	GetMessage() string
	GetProjectID() string
	GetZone() string
	GetTimeStamp() string
}

type computeLogCommon struct {
	Resource  Resource  `json:"resource"`
	TimeStamp time.Time `json:"timestamp"`
	LogName   string
}

// Resource is a nested field of MessageBody
type Resource struct {
	Labels Labels `json:"labels"`
	Type   string `json:"type"`
}

// Labels is a nested field of Resource
type Labels struct {
	ProjectID string `json:"project_id"`
	Zone      string `json:"zone"`
}

func newComputeLogCommon(jsonPayload []byte) (*computeLogCommon, error) {
	var m computeLogCommon
	err := json.Unmarshal(jsonPayload, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (l computeLogCommon) getType() string {
	return l.Resource.Type
}

// GetProjectID returns GCP project ID
func (l computeLogCommon) GetProjectID() string {
	return l.Resource.Labels.ProjectID
}

// GetZone returns GCP zone the instance locates in
func (l computeLogCommon) GetZone() string {
	return l.Resource.Labels.Zone
}

// GetTimeStamp returns timestamp of log
func (l computeLogCommon) GetTimeStamp() string {
	return l.TimeStamp.Format(time.RFC3339)
}

// GetLogName returns name of log
func (l computeLogCommon) GetLogName() string {
	return path.Base(l.LogName)
}

// ComputeAPILog is a JSON-style log from Compute Engine API
type ComputeAPILog struct {
	ProtoPayload ProtoPayload `json:"protoPayload"`
	Operation    Operation    `json:"operation"`
	computeLogCommon
}

// ProtoPayload is a nested field of ProtoPayload
type ProtoPayload struct {
	MethodName   string `json:"methodName"`
	ResourceName string `json:"resourceName"`
}

// Operation is a nested field of Operation
type Operation struct {
	Last bool `json:"last"`
}

// NewComputeAPILog parses JSON log from Compute Engine API and creates ComputeAPILog
func NewComputeAPILog(jsonPayload []byte) (*ComputeAPILog, error) {
	var m ComputeAPILog
	err := json.Unmarshal(jsonPayload, &m)
	if err != nil {
		return nil, err
	}

	// If operation is not last, do not catch this log to avoid sending too many alerts
	if !m.Operation.Last {
		return nil, errors.New("operation should be last")
	}

	// If method is neither insert nor delete, do not catch this log to avoid sending too many alerts
	if m.ProtoPayload.MethodName != computeInsertMethodName && m.ProtoPayload.MethodName != computeDeleteMethodName {
		return nil, errors.New("method should be insert or delete")
	}

	return &m, nil
}

// GetInstanceName returns Compute Engine instance name
func (l ComputeAPILog) GetInstanceName() string {
	return path.Base(l.ProtoPayload.ResourceName)
}

// GetMethodName returns Compute Engine method name
func (l ComputeAPILog) GetMethodName() string {
	return l.ProtoPayload.MethodName
}

// GetMessage returns Compute Engine log message
func (l ComputeAPILog) GetMessage() string {
	switch l.ProtoPayload.MethodName {
	case computeInsertMethodName:
		return computeInsertMessage
	case computeDeleteMethodName:
		return computeDeleteMessage
	default:
		return ""
	}
}

// ComputeStartupScriptLog is a JSON-style message of Compute Engine startup script on Cloud Logging
type ComputeStartupScriptLog struct {
	JSONPayload JSONPayload `json:"jsonPayload"`
	computeLogCommon
}

// JSONPayload is a nested field of MessageBody
type JSONPayload struct {
	HostName string `json:"_HOSTNAME"`
	Message  string `json:"MESSAGE"`
}

// NewComputeStartupScriptLog parses JSON log from Compute Engine startup script and creates ComputeStartupScriptLog
func NewComputeStartupScriptLog(jsonPayload []byte) (*ComputeStartupScriptLog, error) {
	var m ComputeStartupScriptLog
	err := json.Unmarshal(jsonPayload, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// GetInstanceName returns Compute Engine instance name
func (l ComputeStartupScriptLog) GetInstanceName() string {
	return l.JSONPayload.HostName
}

// GetMethodName returns Compute Engine method name
func (l ComputeStartupScriptLog) GetMethodName() string {
	return startupScriptMethodName
}

// GetMessage returns Compute Engine log message
func (l ComputeStartupScriptLog) GetMessage() string {
	return l.JSONPayload.Message
}

// NewComputeLog parses JSON log from any Compute Engine and creates slack webhook message
func NewComputeLog(jsonPayload []byte) (ComputeLog, error) {
	c, err := newComputeLogCommon(jsonPayload)
	if err != nil {
		return nil, err
	}

	// NOTE:
	// This function invocation MUST NOT BE DELETED because the logs from cloud function can cause
	// an infinite loop as below.
	// Cloud Functions -> Cloud Logging Sink -> Cloud Functions -> ...
	t := c.getType()
	if t != computeServiceType {
		return nil, fmt.Errorf("this log is not from compute engine: %s", t)
	}

	n := c.GetLogName()
	switch n {
	case computeAPILogName:
		return NewComputeAPILog(jsonPayload)
	case startupScriptLogName:
		return NewComputeStartupScriptLog(jsonPayload)
	default:
		return nil, fmt.Errorf("invalid log name: %s", n)
	}
}

// NewSlackWebhookMessageForCompute gets message for Slack WebHook
func NewSlackWebhookMessageForCompute(
	projectID string,
	zone string,
	timestamp string,
	instanceName string,
	message string,
	color string,
) *slack.WebhookMessage {
	attachment := slack.Attachment{
		Color:      color,
		AuthorName: "GCP Slack Notifier",
		Title:      "Compute Engine",
		Text:       message,
		Fields: []slack.AttachmentField{
			{Title: "Project", Value: projectID, Short: true},
			{Title: "Zone", Value: zone, Short: true},
			{Title: "Instance", Value: instanceName, Short: true},
			{Title: "TimeStamp", Value: timestamp, Short: true},
		},
	}

	return &slack.WebhookMessage{
		Attachments: []slack.Attachment{attachment},
	}
}
