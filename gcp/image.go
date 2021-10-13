package gcp

import (
	"context"
)

// CreateVMXEnabledImage creates vmx-enabled image
func CreateVMXEnabledImage(ctx context.Context, cc *ComputeCLIClient) error {
	cc.DeleteInstance(ctx)

	err := cc.CreateVMXEnabledInstance(ctx)
	if err != nil {
		return err
	}

	defer cc.DeleteInstance(ctx)

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

	cc.DeleteVMXEnabledImage(ctx)

	err = cc.CreateVMXEnabledImage(ctx)
	if err != nil {
		return err
	}

	return nil
}
