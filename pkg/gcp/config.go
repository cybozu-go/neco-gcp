package gcp

import (
	"fmt"
	"time"
)

const (
	// DefaultExpiration is default expiration time
	defaultExpiration = "0s"
	// DefaultNumLocalSSDs is the default number of local SSDs
	defaultNumLocalSSDs = 1
	// DefaultBootDiskSizeGB is default instance boot disk size
	defaultBootDiskSizeGB = 30
	// DefaultHomeDisk is default value for attaching home disk image in host-vm
	defaultHomeDisk = false
	// DefaultHomeDiskSizeGB is default home disk size
	defaultHomeDiskSizeGB = 50
	// DefaultPreemptible is default value for enabling preemptible
	// https://cloud.google.com/compute/docs/instances/preemptible
	defaultPreemptible = false
	// defaultAppShutdownAt is default time for test instance auto-shutdown
	defaultAppShutdownAt = "20:00"
	// DefaultShutdownAt is default time for instance auto-shutdown
	defaultShutdownAt = "21:00"
	// DefaultTimeZone is default timezone for instance auto-shutdown
	defaultTimeZone = "Asia/Tokyo"
)

// Config is configuration for necogcp command and GAE app
type Config struct {
	Common  CommonConfig  `yaml:"common"`
	App     AppConfig     `yaml:"app"`
	Compute ComputeConfig `yaml:"compute"`
}

// CommonConfig is common configuration for GCP
type CommonConfig struct {
	Project        string `yaml:"project"`
	ServiceAccount string `yaml:"serviceaccount"`
	Zone           string `yaml:"zone"`
}

// AppConfig is configuration for GAE app
type AppConfig struct {
	Shutdown ShutdownConfig `yaml:"shutdown"`
}

// ShutdownConfig is automatic shutdown configuration
type ShutdownConfig struct {
	Stop            []string      `yaml:"stop"`
	Exclude         []string      `yaml:"exclude"`
	Expiration      time.Duration `yaml:"expiration"`
	Timezone        string        `yaml:"timezone"`
	ShutdownAt      string        `yaml:"shutdown-at"`
	AdditionalZones []string      `yaml:"additional-zones"`
}

// ComputeConfig is configuration for GCE
type ComputeConfig struct {
	MachineType      string             `yaml:"machine-type"`
	NumLocalSSDs     int                `yaml:"local-ssd"`
	BootDiskSizeGB   int                `yaml:"boot-disk-sizeGB"`
	OptionalPackages []string           `yaml:"optional-packages"`
	HostVM           HostVMConfig       `yaml:"host-vm"`
	AutoShutdown     AutoShutdownConfig `yaml:"auto-shutdown"`

	// backward compatibility
	VMXEnabled struct {
		OptionalPackages []string `yaml:"optional-packages"`
	} `yaml:"vmx-enabled"`
}

// HostVMConfig is configuration for host-vm instance
type HostVMConfig struct {
	HomeDisk       bool `yaml:"home-disk"`
	HomeDiskSizeGB int  `yaml:"home-disk-sizeGB"`
	Preemptible    bool `yaml:"preemptible"`
}

// AutoShutdownConfig is configuration for automatically shutting down host-vm instance
type AutoShutdownConfig struct {
	Timezone   string `yaml:"timezone"`
	ShutdownAt string `yaml:"shutdown-at"`
}

// NewConfig returns Config
func NewConfig() (*Config, error) {
	expiration, err := time.ParseDuration(defaultExpiration)
	if err != nil {
		return nil, err
	}

	return &Config{
		App: AppConfig{
			Shutdown: ShutdownConfig{
				Expiration: expiration,
				Timezone:   defaultTimeZone,
				ShutdownAt: defaultAppShutdownAt,
			},
		},
		Compute: ComputeConfig{
			NumLocalSSDs:   defaultNumLocalSSDs,
			BootDiskSizeGB: defaultBootDiskSizeGB,
			AutoShutdown: AutoShutdownConfig{
				Timezone:   defaultTimeZone,
				ShutdownAt: defaultShutdownAt,
			},
			HostVM: HostVMConfig{
				HomeDisk:       defaultHomeDisk,
				HomeDiskSizeGB: defaultHomeDiskSizeGB,
				Preemptible:    defaultPreemptible,
			},
		},
	}, nil
}

// NecoTestConfig returns configuration for neco-test
func NecoTestConfig(projectID, zone string) *Config {
	return &Config{
		Common: CommonConfig{
			Project:        projectID,
			ServiceAccount: fmt.Sprintf("%s@%s.iam.gserviceaccount.com", projectID, projectID),
			Zone:           zone,
		},
		App: AppConfig{
			Shutdown: ShutdownConfig{
				Exclude: []string{
					"neco-ops",
					"neco-apps-release",
					"neco-apps-master",
				},
				Expiration: 2 * time.Hour,
				Timezone:   defaultTimeZone,
				ShutdownAt: defaultAppShutdownAt,
				AdditionalZones: []string{
					"asia-northeast1-a",
					"asia-northeast1-b",
					"asia-northeast1-c",
				},
			},
		},
		Compute: ComputeConfig{
			MachineType:    "n1-standard-64",
			NumLocalSSDs:   defaultNumLocalSSDs,
			BootDiskSizeGB: defaultBootDiskSizeGB,
			HostVM: HostVMConfig{
				HomeDisk:       defaultHomeDisk,
				HomeDiskSizeGB: defaultHomeDiskSizeGB,
				Preemptible:    defaultPreemptible,
			},
		},
	}
}
