package gcp

import (
	"context"
)

// MakeVMXEnabledImageURL returns vmx-enabled image URL in the project
func MakeVMXEnabledImageURL(projectID string) string {
	return "https://www.googleapis.com/compute/v1/projects/" + projectID + "/global/images/vmx-enabled"
}

// CreateVMXEnabledImage creates vmx-enabled image
func CreateVMXEnabledImage(ctx context.Context, cc *ComputeCLIClient, baseImageProject, baseImage string) error {
	cc.DeleteInstance(ctx) //nolint:errcheck // best-effort cleanup; the instance may not exist yet

	err := cc.CreateVMXEnabledInstance(ctx, baseImageProject, baseImage)
	if err != nil {
		return err
	}

	defer cc.DeleteInstance(ctx) //nolint:errcheck // best-effort cleanup after image creation

	err = cc.WaitInstance(ctx)
	if err != nil {
		return err
	}

	optionalPackages := append(cc.cfg.Compute.OptionalPackages, cc.cfg.Compute.VMXEnabled.OptionalPackages...)
	err = cc.RunSetupVMXEnabled(ctx, optionalPackages)
	if err != nil {
		return err
	}

	err = cc.StopInstance(ctx)
	if err != nil {
		return err
	}

	cc.DeleteVMXEnabledImage(ctx) //nolint:errcheck // best-effort cleanup; the image may not exist yet

	err = cc.CreateVMXEnabledImage(ctx)
	if err != nil {
		return err
	}

	return nil
}
