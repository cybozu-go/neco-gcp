# exp
GO111MODULE := on
export GO111MODULE

all: test

setup:
	GO111MODULE=off go get -u github.com/gostaticanalysis/nilerr/cmd/nilerr
	GOFLAGS= go install github.com/rakyll/statik
	cd /tmp; env GOFLAGS= GO111MODULE=on go get honnef.co/go/tools/cmd/staticcheck
	cd /tmp; env GOFLAGS= GO111MODULE=on go get github.com/cybozu/neco-containers/golang/analyzer/cmd/custom-checker

test: build
	test -z "$$(gofmt -s -l . | grep -v '^statik/statik.go\|^build/' | tee /dev/stderr)"
	staticcheck ./...
	test -z "$$(nilerr $$(go list -tags='$(GOTAGS)' ./...) 2>&1 | tee /dev/stderr)"
	test -z "$$(custom-checker -restrictpkg.packages=html/template,log $$(go list -tags='$(GOTAGS)' ./...) 2>&1 | tee /dev/stderr)"
	go test -tags='$(GOTAGS)' -race -v ./...
	go vet -tags='$(GOTAGS)' ./...

build: build-dev build-necogcp

build-dev:
	mkdir -p build
	go build -o ./build/dev ./cmd/dev

build-necogcp: statik
	mkdir -p build
	go build -o ./build/necogcp ./cmd/necogcp

install-necogcp: statik
	go install ./cmd/necogcp

statik:
	mkdir -p statik
	go generate ./statik/generate_rule.go

clean:
	rm -rf ./build
	rm -rf ./statik/statik.go

.PHONY: \
	setup \
	test \
	build \
	build-dev \
	build-necogcp \
	install-necogcp \
	statik \
	clean
