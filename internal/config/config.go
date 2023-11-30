package config

import "time"

type ServiceThreshold struct {

	// Percent indicates the percent (up to 1)
	// how to scale the resource.
	// E.g. for CPUs that should scale when it reaches 20% usage
	// the value here will be 0.2
	Percent float32

	// default: `average`
	Metric string

	// How long to watch for usage changes before scaling up
	// Mininum value is `10s`
	ScaleUpDuration string

	// How long to watch for usage changes before scaling down.
	// Mininum value is `10s`
	ScaleDownDuration string
}

// Threshold definitions
type ConfigServiceThreshold struct {
	Cpu    ServiceThreshold
	Memory ServiceThreshold
}

type ConfigService struct {
	Name        string
	StackName   string
	Enabled     bool
	MaxReplicas uint16

	Thresholds ConfigServiceThreshold
}

// Parsed information from an YAML file
// Or service labels
type Config struct {
	Interval time.Duration
	Services []ConfigService
}
