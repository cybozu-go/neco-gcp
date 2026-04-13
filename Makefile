.PHONY: all
all: build

.PHONY: setup
setup:
	env GOFLAGS= go install honnef.co/go/tools/cmd/staticcheck@ff63afafc529279f454e02f1d060210bd4263951 # v0.7.0
	env GOFLAGS= go install github.com/cybozu-go/golang-custom-analyzer/cmd/custom-checker@5cda2f85e31dbe2453825f6520710a76465f197e # v0.1.5

.PHONY: check-generated
check-generated:
	go mod tidy
	git diff --exit-code --name-only

.PHONY: test
test:
	test -z "$$(gofmt -s -l . | grep -v '^build/' | tee /dev/stderr)"
	staticcheck ./...
	test -z "$$(custom-checker -restrictpkg.packages=html/template,log $$(go list -tags='$(GOTAGS)' ./...) 2>&1 | tee /dev/stderr)"
	go test -tags='$(GOTAGS)' -race -v ./...
	go vet -tags='$(GOTAGS)' ./...

.PHONY: build
build: build-dev build-necogcp

.PNOHY: build-dev
build-dev:
	mkdir -p build
	go build -o ./build/dev ./cmd/dev

.PHONY: build-setup
build-setup:
	GOOS=linux GOARCH=amd64 go build -o ./pkg/gcp/bin/ ./cmd/setup

.PHONY: build-necogcp
build-necogcp: build-setup
	mkdir -p build
	go build -o ./build/necogcp ./cmd/necogcp

.PHONY: install-necogcp
install-necogcp: build-setup
	go install ./cmd/necogcp

.PHONY: clean
clean:
	rm -rf ./build ./pkg/gcp/bin/setup
