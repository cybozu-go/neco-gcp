// This server can run on Google App Engine.
package necogcp

import (
	"io/ioutil"
	"net/http"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco-gcp/pkg/app"
	"github.com/cybozu-go/neco-gcp/pkg/gcp"
	"gopkg.in/yaml.v2"
)

const (
	cfgFile = ".necogcp.yml"
)

func loadConfig() (*gcp.Config, error) {
	cfg, err := gcp.NewConfig()
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		// lint:ignore nilerr If cfgFile does not exist, use neco-test config
		return gcp.NecoTestConfig("neco-test", "asia-northeast2-c"), nil
	}
	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func ExtendEntryPoint(w http.ResponseWriter, r *http.Request) {
	cfg, err := loadConfig()
	if err != nil {
		log.ErrorExit(err)
	}

	s, err := app.NewServer(cfg)
	if err != nil {
		log.ErrorExit(err)
	}

	s.Extend(w, r)
}

func ShutdownEntryPoint(w http.ResponseWriter, r *http.Request) {
	cfg, err := loadConfig()
	if err != nil {
		log.ErrorExit(err)
	}

	s, err := app.NewServer(cfg)
	if err != nil {
		log.ErrorExit(err)
	}

	s.Shutdown(w, r)
}
