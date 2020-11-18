package necogcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"cloud.google.com/go/pubsub"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco-gcp/functions"
	"github.com/cybozu-go/neco-gcp/gcp"
)

const (
	deleteInstancesMode = "delete"
	createInstancesMode = "create"

	skipAutoDeleteLabelKey      = "skip-auto-delete"
	excludeSkipAutoDeleteFilter = "-labels." + skipAutoDeleteLabelKey + ":*"

	projectIDEnvName = "GCP_PROJECT"
)

// AutoDCTestMessageBody is body of Pub/Sub message.
type AutoDCTestMessageBody struct {
	Mode               string `json:"mode"`
	InstanceNamePrefix string `json:"namePrefix"`
	InstancesNum       int    `json:"num"`
	DoForceDelete      bool   `json:"doForce"`
}

// AutoDCTestEntryPoint consumes a Pub/Sub message
func AutoDCTestEntryPoint(ctx context.Context, m *pubsub.Message) error {
	log.Debug("msg body", map[string]interface{}{
		"data": string(m.Data),
	})
	var b AutoDCTestMessageBody
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

	projectID := os.Getenv(projectIDEnvName)
	if len(projectID) == 0 {
		err := errors.New(projectIDEnvName + " env should not be empty")
		log.Error(err.Error(), map[string]interface{}{})
		return err
	}
	log.Debug("project id", map[string]interface{}{
		"projectid": projectID,
	})

	client, err := gcp.NewComputeClient(ctx, projectID, zone)
	if err != nil {
		log.Error("failed to create client", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}
	runner := functions.NewAutoDCTestRunner(client)

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

		builder, err := NewNecoStartupScriptBuilder().
			WithFluentd().
			WithNeco(necoBranch).
			WithNecoApps(necoAppsBranch)
		if err != nil {
			log.Error("failed to construct startup-script builder", map[string]interface{}{
				log.FnError: err,
			})
			return err
		}

		sa := MakeNecoDevServiceAccountEmail(projectID)
		imageURL := MakeVMXEnabledImageURL(projectID)
		err = runner.CreateInstancesIfNotExist(
			ctx,
			b.InstanceNamePrefix,
			b.InstancesNum,
			sa,
			machineType,
			imageURL,
			builder.Build(),
		)
		if err != nil {
			log.Error("failed to create instance(s)", map[string]interface{}{
				"prefix":         b.InstanceNamePrefix,
				"num":            b.InstancesNum,
				"serviceaccount": sa,
				"machinetype":    machineType,
				"imageurl":       imageURL,
				log.FnError:      err,
			})
			return err
		}
		log.Info("created instance(s) successfully", map[string]interface{}{
			"prefix":         b.InstanceNamePrefix,
			"num":            b.InstancesNum,
			"serviceaccount": sa,
			"machinetype":    machineType,
			"imageurl":       imageURL,
		})

		return nil
	case deleteInstancesMode:
		log.Info("delete all instance(s)", map[string]interface{}{
			"force": b.DoForceDelete,
		})
		var filter string
		if !b.DoForceDelete {
			filter = excludeSkipAutoDeleteFilter
		}
		err := runner.DeleteFilteredInstances(ctx, filter)
		if err != nil {
			log.Error("failed to delete instance(s)", map[string]interface{}{
				"force":     b.DoForceDelete,
				log.FnError: err,
			})
			return err
		}
		log.Info("deleted all instance(s) successfully", map[string]interface{}{
			"force": b.DoForceDelete,
		})
		return nil
	default:
		err := fmt.Errorf("invalid mode was given: %s", b.Mode)
		log.Error(err.Error(), map[string]interface{}{})
		return err
	}
}
