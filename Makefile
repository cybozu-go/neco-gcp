all: test

setup:
	env GOFLAGS= go install github.com/gostaticanalysis/nilerr/cmd/nilerr@latest
	env GOFLAGS= go install honnef.co/go/tools/cmd/staticcheck@latest
	env GOFLAGS= go install github.com/cybozu/neco-containers/golang/analyzer/cmd/custom-checker@latest

test: build
	test -z "$$(gofmt -s -l . | grep -v '^build/' | tee /dev/stderr)"
	staticcheck ./...
	test -z "$$(nilerr $$(go list -tags='$(GOTAGS)' ./...) 2>&1 | tee /dev/stderr)"
	test -z "$$(custom-checker -restrictpkg.packages=html/template,log $$(go list -tags='$(GOTAGS)' ./...) 2>&1 | tee /dev/stderr)"
	go test -tags='$(GOTAGS)' -race -v ./...
	go vet -tags='$(GOTAGS)' ./...

build: build-dev build-necogcp

build-dev:
	mkdir -p build
	go build -o ./build/dev ./cmd/dev

build-setup:
	GOOS=linux GOARCH=amd64 go build -o ./pkg/gcp/bin/ ./cmd/setup

build-necogcp: build-setup
	mkdir -p build
	go build -o ./build/necogcp ./cmd/necogcp

install-necogcp: build-setup
	go install ./cmd/necogcp

clean:
	rm -rf ./build ./pkg/gcp/bin/setup

.PHONY: \
	setup \
	test \
	build \
	build-dev \
	build-necogcp \
	install-necogcp \
	clean
