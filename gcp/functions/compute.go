package functions

import (
	"context"
	"errors"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco-gcp/gcp"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iam/v1"
)

// ComputeClient is GCP compute client with go client
type ComputeClient struct {
	ctx             context.Context
	service         *compute.Service
	projectID       string
	zone            string
	projectEndpoint string
}

// NewComputeClient returns ComputeClient
func NewComputeClient(
	ctx context.Context,
	projectID string,
	zone string,
) (*ComputeClient, error) {
	s, err := compute.NewService(ctx)
	if err != nil {
		return nil, err
	}

	return &ComputeClient{
		ctx:             ctx,
		service:         s,
		projectID:       projectID,
		zone:            zone,
		projectEndpoint: "https://www.googleapis.com/compute/v1/projects/" + projectID,
	}, nil
}

// Create creates a compute instance with running startup script
func (c *ComputeClient) Create(
	instanceName string,
	serviceAccountName string,
	machineType string,
	imageURL string,
	startupScript string,
) error {
	instance := &compute.Instance{
		Name:        instanceName,
		MachineType: c.projectEndpoint + "/zones/" + c.zone + "/machineTypes/" + machineType,
		Metadata: &compute.Metadata{
			Items: []*compute.MetadataItems{
				{
					Key:   "startup-script",
					Value: &startupScript,
				},
			},
		},
		Disks: []*compute.AttachedDisk{
			{
				AutoDelete: true,
				Boot:       true,
				Type:       "PERSISTENT",
				InitializeParams: &compute.AttachedDiskInitializeParams{
					// DiskName must be unique to create multiple instances simultaneously
					DiskName:    instanceName,
					SourceImage: imageURL,
				},
			},
			{
				Type: "SCRATCH",
				InitializeParams: &compute.AttachedDiskInitializeParams{
					DiskType: "zones/" + c.zone + "/diskTypes/local-ssd",
				},
				AutoDelete: true,
				Interface:  "SCSI",
			},
		},
		NetworkInterfaces: []*compute.NetworkInterface{
			{
				AccessConfigs: []*compute.AccessConfig{
					{
						Type: "ONE_TO_ONE_NAT",
						Name: "External NAT",
					},
				},
				Network: c.projectEndpoint + "/global/networks/default",
			},
		},
		ServiceAccounts: []*compute.ServiceAccount{
			{
				Email: serviceAccountName,
				Scopes: []string{
					compute.DevstorageFullControlScope,
					compute.ComputeScope,
					// Scopes is legacy method. We should set appropriate permissions with IAM
					// https://cloud.google.com/compute/docs/access/create-enable-service-accounts-for-instances#best_practices
					iam.CloudPlatformScope,
				},
			},
		},
	}

	_, err := c.service.Instances.Insert(c.projectID, c.zone, instance).Do()
	if err != nil {
		return err
	}

	return gcp.RetryWithSleep(c.ctx, 10, time.Second, func(ctx context.Context) error {
		ci, err := c.service.Instances.Get(c.projectID, c.zone, instance.Name).Do()
		if err != nil {
			return err
		}
		if ci.Status != "RUNNING" {
			return errors.New("instance is not running yet")
		}
		return nil
	}, func(err error) {
		log.Info("waiting for creating instance...", nil)
	},
	)
}

// GetNameSet gets a list of existing GCP instances with the given filter
func (c *ComputeClient) GetNameSet(filter string) (map[string]struct{}, error) {
	list, err := c.service.Instances.List(c.projectID, c.zone).Filter(filter).Do()
	if err != nil {
		return nil, err
	}

	res := map[string]struct{}{}
	for _, n := range list.Items {
		res[n.Name] = struct{}{}
	}
	return res, nil
}

// Delete deletes a GCP instance
func (c *ComputeClient) Delete(name string) error {
	_, err := c.service.Instances.Delete(c.projectID, c.zone, name).Do()
	return err
}
