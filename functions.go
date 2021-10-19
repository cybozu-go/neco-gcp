package necogcp

import (
	"context"

	"cloud.google.com/go/pubsub"
	"github.com/cybozu-go/neco-gcp/pkg/slacknotifier"
)

// NOTE: A entrypoint of the Cloud Functions must be in the root of this project.
// ref: https://cloud.google.com/functions/docs/writing#structuring_source_code
// >> For the Go runtime, your function must be in a Go package at the root of your project. Your function cannot be in package main. Sub-packages are only supported when using Go modules.

// SlackNotifierEntryPoint is a entrypoint for the slack-notifier.
func SlackNotifierEntryPoint(ctx context.Context, m *pubsub.Message) error {
	return slacknotifier.SendNotification(ctx, slackNotifierConfigName, m)
}
