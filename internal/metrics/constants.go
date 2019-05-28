package metrics

import "time"

const (
	baseAPIURL           = "https://api.datadoghq.com/api/v1"
	apiKeyParam          = "api_key"
	appKeyParam          = "application_key"
	defaultRetryInterval = time.Millisecond * 250
)

// MetricType enumerates all the available metric types
type MetricType string

const (

	// DistributionType represents a distribution metric
	DistributionType MetricType = "distribution"
)
