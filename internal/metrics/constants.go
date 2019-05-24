package metrics

const (
	baseAPIURL  = "https://api.datadoghq.com/api/v1"
	apiKeyParam = "api_key"
	appKeyParam = "application_key"
)

type MetricType string

const (
	DistributionType MetricType = "distribution"
)
