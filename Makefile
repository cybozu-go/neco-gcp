

# For Go
GO111MODULE = on
GOFLAGS     = -mod=vendor
export GO111MODULE GOFLAGS

all: test

test:
	test -z "$$(gofmt -s -l . | grep -v '^vendor' | tee /dev/stderr)"
	test -z "$$(golint $$(go list ./... | grep -v /vendor/) | tee /dev/stderr)"
	test -z "$$(nilerr ./... 2>&1 | tee /dev/stderr)"
	test -z "$(custom-checker -restrictpkg.packages=html/template,log $(go list -tags='' ./... | grep -v /vendor/ ) 2>&1 | tee /dev/stderr)"
	go build ./...
	go test -race -v ./...
	go vet ./...
	ineffassign .

mod:
	go mod tidy
	go mod vendor
	git add -f vendor
	git add go.mod

.PHONY:	all test mod
