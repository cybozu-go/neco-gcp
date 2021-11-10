package necogcp

import (
	"context"
	"net/http"

	"cloud.google.com/go/pubsub"
	"github.com/cybozu-go/neco-gcp/pkg/api"
	"github.com/cybozu-go/neco-gcp/pkg/autodctest"
	"github.com/cybozu-go/neco-gcp/pkg/slacknotifier"
)

// NOTE: An entrypoint of the Cloud Functions must be in the root of this project.
// ref: https://cloud.google.com/functions/docs/writing#structuring_source_code
// >> For the Go runtime, your function must be in a Go package at the root of your project. Your function cannot be in package main. Sub-packages are only supported when using Go modules.

func ExtendEntryPoint(w http.ResponseWriter, r *http.Request) {
	api.ExtendEntryPoint(w, r)
}

func ShutdownEntryPoint(ctx context.Context, m *pubsub.Message) error {
	return api.ShutdownEntryPoint(ctx, m)
}

// SlackNotifierEntryPoint is the entrypoint for the "slack-notifier" function.
func SlackNotifierEntryPoint(ctx context.Context, m *pubsub.Message) error {
	return slacknotifier.EntryPoint(ctx, m)
}

// AutoDCTestEntryPoint is the entrypoint for the "auto-dctest" function.
func AutoDCTestEntryPoint(ctx context.Context, m *pubsub.Message) error {
	return autodctest.EntryPoint(ctx, m, autoDCTestMachineType, autoDCTestZone, autoDCTestJPHolidays)
}
