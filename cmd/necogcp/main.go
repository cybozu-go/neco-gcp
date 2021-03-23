package main

import (
	"github.com/cybozu-go/neco-gcp/cmd/necogcp/cmd"
	_ "github.com/cybozu-go/neco-gcp/statik"
)

func main() {
	cmd.Execute()
}
