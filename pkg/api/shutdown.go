package api

import (
	"context"
	"net/http"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco-gcp/pkg/gcp"
	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

func ShutdownEntryPoint(ctx context.Context, m *pubsub.Message) error {
	client, err := google.DefaultClient(context.Background(), "https://www.googleapis.com/auth/compute")
	if err != nil {
		log.ErrorExit(err)
	}
	cfg := gcp.NecoTestConfig("yamatchas-test", "asia-northeast2-c")
	return Shutdown(ctx, m, client, cfg)
}

func Shutdown(ctx context.Context, m *pubsub.Message, client *http.Client, cfg *gcp.Config) error {
	project := cfg.Common.Project
	commonZone := cfg.Common.Zone
	addZones := cfg.App.Shutdown.AdditionalZones
	exclude := cfg.App.Shutdown.Exclude
	stop := cfg.App.Shutdown.Stop
	status := ShutdownStatus{}
	now := time.Now().UTC()
	expiration := cfg.App.Shutdown.Expiration

	service, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return err
	}
	targetZones := append([]string{commonZone}, addZones...)
	var errList []error
	for _, zone := range targetZones {
		instanceList, err := service.Instances.List(project, zone).Do()
		if err != nil {
			errList = append(errList, err)
			continue
		}

		for _, instance := range instanceList.Items {
			if contain(instance.Name, exclude) {
				continue
			}

			shutdownAt, err := getShutdownAt(instance)
			switch err {
			case errShutdownMetadataNotFound:
			case nil:
				if now.Sub(shutdownAt) >= 0 {
					_, err := service.Instances.Delete(project, zone, instance.Name).Do()
					if err != nil {
						errList = append(errList, err)
						continue
					}
					status.Deleted = append(status.Deleted, instance.Name)
				}
				continue
			default:
				errList = append(errList, err)
				continue
			}

			creationTime, err := time.Parse(time.RFC3339, instance.CreationTimestamp)
			if err != nil {
				errList = append(errList, err)
				continue
			}
			elapsed := now.Sub(creationTime)
			if elapsed.Seconds() < expiration.Seconds() {
				continue
			}

			if contain(instance.Name, stop) {
				_, err := service.Instances.Stop(project, zone, instance.Name).Do()
				if err != nil {
					continue
				}
				status.Stopped = append(status.Stopped, instance.Name)
			} else {
				_, err := service.Instances.Delete(project, zone, instance.Name).Do()
				if err != nil {
					continue
				}
				status.Deleted = append(status.Deleted, instance.Name)
			}
		}
	}
	log.Info("shutdown instances", map[string]interface{}{
		"deleted": status.Deleted,
		"stopped": status.Stopped,
	})
	if len(errList) != 0 {
		log.Error("shutdown failed", map[string]interface{}{
			"errors": errList,
		})
		return errList[0]
	}
	return nil
}
