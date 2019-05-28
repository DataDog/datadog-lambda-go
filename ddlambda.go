package ddlambda

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/DataDog/dd-lambda-go/internal/metrics"
	"github.com/DataDog/dd-lambda-go/internal/trace"
	"github.com/DataDog/dd-lambda-go/internal/wrapper"
)

type (
	// Config gives options for how ddlambda should behave
	Config struct {
		APIKey               string
		AppKey               string
		ShouldRetryOnFailure bool
		BatchInterval        time.Duration
	}
)

const (
	// DatadogAPIKeyEnvVar is the environment variable that will be used as an API key by default
	DatadogAPIKeyEnvVar = "DATADOG_API_KEY"
	// DatadogAPPKeyEnvVar is the environment variable that will be used as an API key by default
	DatadogAPPKeyEnvVar = "DATADOG_APP_KEY"
)

// WrapHandler is used to instrument your lambda functions, reading in context from API Gateway.
// It returns a modified handler that can be passed directly to the lambda.Start function.
func WrapHandler(handler interface{}, cfg *Config) interface{} {
	tl := trace.Listener{}
	ml := metrics.MakeListener(cfg.toMetricsConfig())
	return wrapper.WrapHandlerWithListeners(handler, &tl, &ml)
}

// GetTraceHeaders reads a map containing the DataDog trace headers from a context object.
func GetTraceHeaders(ctx context.Context) map[string]string {
	result := trace.GetTraceHeaders(ctx, true)
	return result
}

// AddTraceHeaders adds DataDog trace headers to a HTTP Request
func AddTraceHeaders(ctx context.Context, req *http.Request) {
	headers := GetTraceHeaders(ctx)
	for key, value := range headers {
		req.Header.Add(key, value)
	}
}

// DistributionMetric sends a distribution metric to DataDog
func DistributionMetric(ctx context.Context, metric string, value float64, tags ...string) {
	pr := metrics.GetProcessor(ctx)
	if pr == nil {
		return
	}
	m := metrics.Distribution{
		Name:   metric,
		Tags:   tags,
		Values: []float64{},
	}
	m.AddPoint(value)
	pr.AddMetric(&m)
}

func (cfg *Config) toMetricsConfig() metrics.Config {

	mc := metrics.Config{
		ShouldRetryOnFailure: false,
	}

	if cfg != nil {
		mc.BatchInterval = cfg.BatchInterval
		mc.ShouldRetryOnFailure = cfg.ShouldRetryOnFailure
		mc.APIKey = cfg.APIKey
		mc.AppKey = cfg.AppKey
	}

	if mc.APIKey == "" {
		mc.APIKey = os.Getenv(DatadogAPIKeyEnvVar)

	}
	if mc.AppKey == "" {
		mc.AppKey = os.Getenv(DatadogAPIKeyEnvVar)
	}
	return mc
}
