package necogcpfunctions

import (
	"context"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/pubsub"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco-gcp/gcp/functions"
	"github.com/kelseyhightower/envconfig"
)

const (
	deleteInstancesMode = "delete"
	createInstancesMode = "create"

	necoBranch     = "release"
	necoAppsBranch = "release"

	machineType = "n1-standard-32"

	skipAutoDeleteLabelKey      = "skip-auto-delete"
	excludeSkipAutoDeleteFilter = "-labels." + skipAutoDeleteLabelKey + ":*"
)

// Body is body of Pub/Sub message.
type Body struct {
	Mode               string `json:"mode"`
	InstanceNamePrefix string `json:"namePrefix"`
	InstancesNum       int    `json:"num"`
	DoForceDelete      bool   `json:"doForce"`
}

// Env is cloud function environment variables
type Env struct {
	ProjectID string `envconfig:"GCP_PROJECT" required:"true"`
	Zone      string `envconfig:"ZONE" required:"true"`
}

// PubSubEntryPoint consumes a Pub/Sub message
func PubSubEntryPoint(ctx context.Context, m *pubsub.Message) error {
	log.Debug("msg body", map[string]interface{}{
		"data": string(m.Data),
	})
	var b Body
	err := json.Unmarshal(m.Data, &b)
	if err != nil {
		log.Error("failed to unmarshal json", map[string]interface{}{
			"data":      string(m.Data),
			log.FnError: err,
		})
		return err
	}
	log.Debug("unmarshalled msg body", map[string]interface{}{
		"body": b,
	})

	var e Env
	err = envconfig.Process("", &e)
	if err != nil {
		log.Error("failed to parse env vars", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}
	log.Debug("cloud functions env", map[string]interface{}{
		"env": e,
	})

	client, err := functions.NewComputeClient(ctx, e.ProjectID, e.Zone)
	if err != nil {
		log.Error("failed to create client", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}
	runner := functions.NewRunner(client)

	switch b.Mode {
	case createInstancesMode:
		log.Info("create instance(s)", map[string]interface{}{
			"prefix": b.InstanceNamePrefix,
			"num":    b.InstancesNum,
		})

		today, err := getDateStrInJST()
		if err != nil {
			log.Error("failed to get today's date", map[string]interface{}{
				log.FnError: err,
			})
			return err
		}
		log.Debug("today is "+today, map[string]interface{}{})
		if isHoliday(today, jpHolidays) {
			log.Info("today is holiday! skip creating dctest", map[string]interface{}{})
			return nil
		}

		builder, err := functions.NewNecoStartupScriptBuilder().
			WithFluentd().
			WithNeco(necoBranch).
			WithNecoApps(necoAppsBranch)
		if err != nil {
			log.Error("failed to construct startup-script builder", map[string]interface{}{
				log.FnError: err,
			})
			return err
		}
		return runner.CreateInstancesIfNotExist(
			ctx,
			b.InstanceNamePrefix,
			b.InstancesNum,
			functions.MakeNecoDevServiceAccountEmail(e.ProjectID),
			machineType,
			functions.MakeVMXEnabledImageURL(e.ProjectID),
			builder.Build(),
		)
	case deleteInstancesMode:
		log.Info("delete all instance(s)", map[string]interface{}{
			"force": b.DoForceDelete,
		})
		if b.DoForceDelete {
			return runner.DeleteInstancesMatchingFilter(ctx, "")
		}
		return runner.DeleteInstancesMatchingFilter(ctx, excludeSkipAutoDeleteFilter)
	default:
		err := fmt.Errorf("invalid mode was given: %s", b.Mode)
		log.Error(err.Error(), map[string]interface{}{})
		return err
	}
}
