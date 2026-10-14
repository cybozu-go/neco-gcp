package necogcp

import (
	"context"
	"net/http"

	"cloud.google.com/go/pubsub/v2"

	"github.com/cybozu-go/neco-gcp/pkg/instancedeleter"
)

// NOTE: An entrypoint of the Cloud Functions must be in the root of this project.
// ref: https://cloud.google.com/functions/docs/writing#structuring_source_code
// >> For the Go runtime, your function must be in a Go package at the root of your project. Your function cannot be in package main. Sub-packages are only supported when using Go modules.

func ExtendEntryPoint(w http.ResponseWriter, r *http.Request) {
	instancedeleter.ExtendEntryPoint(w, r, NecoTestProject, NecoTestZone)
}

func ShutdownEntryPoint(ctx context.Context, m *pubsub.Message) error {
	return instancedeleter.ShutdownEntryPoint(ctx, m, NecoTestProject, NecoTestZone)
}
