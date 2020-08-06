GO111MODULE = on
GOFLAGS     = -mod=vendor
export GO111MODULE GOFLAGS

GCP_PROJECT ?= tapih-test
ZONE ?= us-west1-a
ACCOUNT_JSON_PATH ?= hoge
SERVICE_ACCOUNT_NAME ?= fuga

STATIK = gcp/statik/statik.go
STATIK_FILES := $(shell find gcp/public -type f)

all:

setup:
	go install github.com/rakyll/statik

mod:
	go mod tidy
	go mod vendor
	git add -f vendor
	git add go.mod

test: build
	test -z "$$(gofmt -s -l . | grep -v '^vendor/\|^menu/assets.go\|^build/' | tee /dev/stderr)"
	test -z "$$(golint $$(go list -tags='$(GOTAGS)' ./... | grep -v /vendor/) | grep -v '/dctest/.*: should not use dot imports' | tee /dev/stderr)"
	test -z "$$(nilerr $$(go list -tags='$(GOTAGS)' ./... | grep -v /vendor/) 2>&1 | tee /dev/stderr)"
	test -z "$$(custom-checker -restrictpkg.packages=html/template,log $$(go list -tags='$(GOTAGS)' ./... | grep -v /vendor/ ) 2>&1 | tee /dev/stderr)"
	ineffassign .
	go test -tags='$(GOTAGS)' -race -v ./...
	go vet -tags='$(GOTAGS)' ./...

build: build-dev build-necogcp

build-dev:
	mkdir -p build
	go build -mod=vendor -o build/dev ./cmd/dev

build-necogcp: statik
	mkdir -p build
	go build -mod=vendor -o build/necogcp ./cmd/necogcp

statik: $(STATIK)

$(STATIK): $(STATIK_FILES)
	mkdir -p $(dir $(STATIK))
	go generate ./cmd/necogcp/...

deploy-function:
	gcloud functions deploy auto-dctest \
		--project $(GCP_PROJECT) \
		--entry-point PubSubEntryPoint \
		--runtime go113 \
		--trigger-topic autodctest-scheduler-events \
		--set-env-vars GCP_PROJECT=$(GCP_PROJECT),SERVICE_ACCOUNT_NAME=$(SERVICE_ACCOUNT_NAME),ZONE=$(ZONE) \
		--timeout 300s

deploy-create-scheduler:
	gcloud beta scheduler jobs create pubsub create-dctest \
		--project $(GCP_PROJECT) \
		--schedule '0 9 * * 1-5' \
		--topic autodctest-scheduler-events \
		--message-body '{"mode":"create", "namePrefix":"maneki", "num":2}' \
		--time-zone 'Asia/Tokyo' \
		--description 'automatically create dctest instance'

deploy-delete-scheduler:
	gcloud beta scheduler jobs create pubsub delete-dctest \
		--project $(GCP_PROJECT) \
		--schedule '0 20 * * *' \
		--topic autodctest-scheduler-events \
		--message-body '{"mode":"delete", "doForce":false}' \
		--time-zone 'Asia/Tokyo' \
		--description 'automatically delete dctest instances except for ones with skip-auto-delete label'

deploy-force-delete-scheduler:
	gcloud beta scheduler jobs create pubsub force-delete-dctest \
		--project $(GCP_PROJECT) \
		--schedule '0 23 * * *' \
		--topic autodctest-scheduler-events \
		--message-body '{"mode":"delete", "doForce":true}' \
		--time-zone 'Asia/Tokyo' \
		--description 'automatically delete dctest all instances'

clean:
	rm -rf ./gcp/statik

.PHONY: \
	setup \
	mod \
	test \
	statik \
	build \
	build-dev \
	build-necogcp \
	deploy-function \
	deploy-create-scheduler \
	deploy-delete-scheduler \
	deploy-force-delete-scheduler
