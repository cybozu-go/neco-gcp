package slack

import (
	"time"
)

// MessageBody is a pubsub message body from GCE's fluentd or Cloud Functions
type MessageBody struct {
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
