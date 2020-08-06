//go:generate statik -f -include * -src=./../../gcp/public -dest=./../../gcp

package main

import (
	"github.com/cybozu-go/neco-gcp/cmd/necogcp/cmd"
	_ "github.com/cybozu-go/neco-gcp/gcp/statik"
)

func main() {
	cmd.Execute()
}
