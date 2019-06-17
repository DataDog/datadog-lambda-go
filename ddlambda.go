/*
 * Unless explicitly stated otherwise all files in this repository are licensed
 * under the Apache License Version 2.0.
 *
 * This product includes software developed at Datadog (https://www.datadoghq.com/).
 * Copyright 2019 Datadog, Inc.
 */

package ddlambda

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/DataDog/datadog-lambda-go/internal/logger"
	"github.com/DataDog/datadog-lambda-go/internal/metrics"
	"github.com/DataDog/datadog-lambda-go/internal/trace"
	"github.com/DataDog/datadog-lambda-go/internal/wrapper"
)

type (
	// Config gives options for how ddlambda should behave
	Config struct {
		// APIKey is your Datadog API key. This is used for sending metrics.
		APIKey string
		// KMSAPIKey is your Datadog API key, encrypted using the AWS KMS service. This is used for sending metrics.
		KMSAPIKey string
		// ShouldRetryOnFailure is used to turn on retry logic when sending metrics via the API. This can negatively effect the performance of your lambda,
		// and should only be turned on if you can't afford to lose metrics data under poor network conditions.
		ShouldRetryOnFailure bool
		// BatchInterval is the period of time which metrics are grouped together for processing to be sent to the API or written to logs.
		// Any pending metrics are flushed at the end of the lambda.
		BatchInterval time.Duration
		// Site is the host to send metrics to. If empty, this value is read from the 'DD_SITE' environment variable, or if that is empty
		// will default to 'datadoghq.com'.
		Site string

		// DebugLogging will turn on extended debug logging.
		DebugLogging bool
	}
)

const (
	// DatadogAPIKeyEnvVar is the environment variable that will be used as an API key by default
	DatadogAPIKeyEnvVar    = "DD_API_KEY"
	DatadogKMSAPIKeyEnvVar = "DD_KMS_API_KEY"
	// DatadogSiteEnvVar is the environment variable that will be used as the API host.
	DatadogSiteEnvVar = "DD_SITE"
	// DatadogLogLevelEnvVar is the environment variable that will be used to check the log level.
	// if it equals "debug" everything will be logged.
	DatadogLogLevelEnvVar = "DD_LOG_LEVEL"
)

// WrapHandler is used to instrument your lambda functions, reading in context from API Gateway.
// It returns a modified handler that can be passed directly to the lambda.Start function.
func WrapHandler(handler interface{}, cfg *Config) interface{} {

	logLevel := os.Getenv(DatadogLogLevelEnvVar)
	if strings.EqualFold(logLevel, "debug") {
		logger.SetLogLevel(logger.LevelDebug)
	}

	if cfg != nil && cfg.DebugLogging {
		logger.SetLogLevel(logger.LevelDebug)
	}

	// Set up state that is shared between handler invocations
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

// GetContext retrieves the last created lambda context.
// Only use this if you aren't manually passing context through your call hierarchy.
func GetContext() context.Context {
	return wrapper.CurrentContext
}

// DistributionWithContext sends a distribution metric to DataDog
func DistributionWithContext(ctx context.Context, metric string, value float64, tags ...string) {
	pr := metrics.GetProcessor(GetContext())
	if pr == nil {
		logger.Error(fmt.Errorf("couldn't get metrics processor from current context"))
		return
	}

	// We add our own runtime tag to the metric for version tracking
	tags = append(tags, getRuntimeTag())

	m := metrics.Distribution{
		Name:   metric,
		Tags:   tags,
		Values: []metrics.MetricValue{},
	}
	m.AddPoint(time.Now(), value)
	logger.Debug(fmt.Sprintf("adding metric \"%s\", with value %f", metric, value))
	pr.AddMetric(&m)
}

// Distribution sends a distribution metric to DataDog
func Distribution(metric string, value float64, tags ...string) {
	DistributionWithContext(GetContext(), metric, value, tags...)
}

func (cfg *Config) toMetricsConfig() metrics.Config {

	mc := metrics.Config{
		ShouldRetryOnFailure: false,
	}

	if cfg != nil {
		mc.BatchInterval = cfg.BatchInterval
		mc.ShouldRetryOnFailure = cfg.ShouldRetryOnFailure
		mc.APIKey = cfg.APIKey
		mc.KMSAPIKey = cfg.KMSAPIKey
	}

	if mc.APIKey == "" {
		mc.APIKey = os.Getenv(DatadogAPIKeyEnvVar)

	}

	if mc.KMSAPIKey == "" {
		mc.KMSAPIKey = os.Getenv(DatadogKMSAPIKeyEnvVar)
	}
	if mc.APIKey == "" && mc.KMSAPIKey == "" {
		logger.Error(fmt.Errorf("couldn't read DD_API_KEY or DD_KMS_API_KEY from environment"))
	}
	if mc.Site == "" {
		mc.Site = os.Getenv(DatadogSiteEnvVar)
	}

	return mc
}

func getRuntimeTag() string {
	v := runtime.Version()
	return fmt.Sprintf("dd_lambda_layer:datadog-%s", v)
}
