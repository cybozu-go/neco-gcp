package autodctest

import (
	"context"
	"fmt"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco-gcp/pkg/gcp"
	"github.com/cybozu-go/well"
)

// Runner runs dctest environments on GCP instances
type Runner struct {
	compute *gcp.ComputeClient
}

// NewRunner creates Runner
func NewRunner(computeClient *gcp.ComputeClient) *Runner {
	return &Runner{compute: computeClient}
}

func (r Runner) makeInstanceName(prefix string, index int) string {
	return fmt.Sprintf("%s-%d", prefix, index)
}

// CreateInstancesIfNotExist lists instances not existing and create them
func (r Runner) CreateInstancesIfNotExist(
	ctx context.Context,
	instanceNamePrefix string,
	instancesNum int,
	serviceAccountEmail string,
	machineType string,
	numLocalSSDs int,
	imageURL string,
	startupScript string,
) error {
	set, err := r.compute.GetNameSet("")
	if err != nil {
		log.Error("failed to get instances list", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}

	log.Info("fetched instances successfully", map[string]interface{}{
		"names": set,
	})
	e := well.NewEnvironment(ctx)
	for i := 0; i < instancesNum; i++ {
		name := r.makeInstanceName(instanceNamePrefix, i)
		if _, ok := set[name]; ok {
			log.Info("skip creating instance because it already exists", map[string]interface{}{
				"name": name,
			})
			continue
		}

		e.Go(func(ctx context.Context) error {
			log.Info("start creating instance", map[string]interface{}{
				"name": name,
			})
			err := r.compute.Create(
				name,
				serviceAccountEmail,
				machineType,
				numLocalSSDs,
				imageURL,
				startupScript,
			)
			if err != nil {
				log.Error("failed to create instance", map[string]interface{}{
					log.FnError: err,
					"name":      name,
				})
				return err
			}
			log.Info("instance is created successfully", map[string]interface{}{
				"name": name,
			})

			return nil
		})
	}
	e.Stop()
	return e.Wait()
}

// DeleteFilteredInstances deletes instances which match the given filter
func (r Runner) DeleteFilteredInstances(ctx context.Context, filter string) error {
	aggregatedList, err := r.compute.GetAggregatedNameList(filter)
	if err != nil {
		log.Error("failed to get instances list", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}

	log.Info("fetched instances successfully", map[string]interface{}{
		"names": aggregatedList,
	})
	e := well.NewEnvironment(ctx)
	for z, scopedList := range aggregatedList {
		zoneName := z
		for _, n := range scopedList {
			name := n
			e.Go(func(ctx context.Context) error {
				log.Info("start deleting instance", map[string]interface{}{
					"zone": zoneName,
					"name": name,
				})
				err := r.compute.DeleteWithZone(zoneName, name)
				if err != nil {
					log.Error("failed to delete instance", map[string]interface{}{
						log.FnError: err,
						"zone":      zoneName,
						"name":      name,
					})
					return err
				}
				log.Info("instance is deleted successfully", map[string]interface{}{
					"zone": zoneName,
					"name": name,
				})
				return nil
			})
		}
	}
	e.Stop()
	return e.Wait()
}
