.PHONY: all
all: build

.PHONY: check-generated
check-generated:
	go mod tidy
	$(MAKE) fmt
	git diff --exit-code --name-only

.PHONY: fmt
fmt:
	go tool golangci-lint fmt

.PHONY: lint
lint:
	go tool golangci-lint run

.PHONY: test
test:
	go test -tags='$(GOTAGS)' -race -v ./...

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
