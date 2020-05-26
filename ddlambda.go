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
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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
		// ShouldUseLogForwarder enabled the log forwarding method for sending metrics to Datadog. This approach requires the user to set up a custom lambda
		// function that forwards metrics from cloudwatch to the Datadog api. This approach doesn't have any impact on the performance of your lambda function.
		ShouldUseLogForwarder bool
		// BatchInterval is the period of time which metrics are grouped together for processing to be sent to the API or written to logs.
		// Any pending metrics are flushed at the end of the lambda.
		BatchInterval time.Duration
		// Site is the host to send metrics to. If empty, this value is read from the 'DD_SITE' environment variable, or if that is empty
		// will default to 'datadoghq.com'.
		Site string

		// DebugLogging will turn on extended debug logging.
		DebugLogging bool
		// EnhancedMetrics enables the reporting of enhanced metrics under `aws.lambda.enhanced*` and adds enhanced metric tags
		EnhancedMetrics bool

		// GlobalTags are normal tags which are appended to every metric.
		GlobalTags []string
	}
)

const (
	// DatadogAPIKeyEnvVar is the environment variable that will be used as an API key by default
	DatadogAPIKeyEnvVar = "DD_API_KEY"
	// DatadogKMSAPIKeyEnvVar is the environment variable that will be sent to KMS for decryption, then used as an API key.
	DatadogKMSAPIKeyEnvVar = "DD_KMS_API_KEY"
	// DatadogSiteEnvVar is the environment variable that will be used as the API host.
	DatadogSiteEnvVar = "DD_SITE"
	// DatadogLogLevelEnvVar is the environment variable that will be used to check the log level.
	// if it equals "debug" everything will be logged.
	DatadogLogLevelEnvVar = "DD_LOG_LEVEL"
	// DatadogShouldUseLogForwarderEnvVar is the environment variable that is used to enable log forwarding of metrics.
	DatadogShouldUseLogForwarderEnvVar = "DD_FLUSH_TO_LOG"
	// DatadogGlobalTags is the environment variable used to add tags to all metrics
	DatadogGlobalTags = "DD_TAGS"
	// DefaultSite to send API messages to.
	DefaultSite = "datadoghq.com"
	// DefaultEnhancedMetrics enables enhanced metrics by default.
	DefaultEnhancedMetrics = true
)

// WrapHandler is used to instrument your lambda functions, reading in context from API Gateway.
// It returns a modified handler that can be passed directly to the lambda.Start function.
func WrapHandler(handler interface{}, cfg *Config) interface{} {

	logLevel := os.Getenv(DatadogLogLevelEnvVar)
	if strings.EqualFold(logLevel, "debug") || (cfg != nil && cfg.DebugLogging) {
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

// Distribution sends a distribution metric to DataDog
// Deprecated: Use Metric method instead
func Distribution(metric string, value float64, tags ...string) {
	Metric(metric, value, tags...)
}

// Metric sends a distribution metric to DataDog
func Metric(metric string, value float64, tags ...string) {
	MetricWithTimestamp(metric, value, time.Now(), tags...)
}

// MetricWithTimestamp sends a distribution metric to DataDog with a custom timestamp
func MetricWithTimestamp(metric string, value float64, timestamp time.Time, tags ...string) {
	ctx := GetContext()

	if ctx == nil {
		logger.Debug("no context available, did you wrap your handler?")
		return
	}

	listener := metrics.GetListener(ctx)

	if listener == nil {
		logger.Error(fmt.Errorf("couldn't get metrics listener from current context"))
		return
	}
	listener.AddDistributionMetric(metric, value, timestamp, false, tags...)
}

// InvokeDryRun is a utility to easily run your lambda for testing
func InvokeDryRun(callback func(ctx context.Context), cfg *Config) (interface{}, error) {
	wrapped := WrapHandler(callback, cfg)
	// Convert the wrapped handler to it's underlying raw handler type
	handler, ok := wrapped.(func(ctx context.Context, msg json.RawMessage) (interface{}, error))
	if !ok {
		logger.Debug("Could not unwrap lambda during dry run")
	}
	return handler(context.Background(), json.RawMessage("{}"))
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
		mc.Site = cfg.Site
		mc.ShouldUseLogForwarder = cfg.ShouldUseLogForwarder
	}

	if mc.Site == "" {
		mc.Site = os.Getenv(DatadogSiteEnvVar)
	}
	if mc.Site == "" {
		mc.Site = DefaultSite
	}
	if strings.HasPrefix(mc.Site, "https://") || strings.HasPrefix(mc.Site, "http://") {
		mc.Site = fmt.Sprintf("%s/api/v1", mc.Site)
	} else {
		mc.Site = fmt.Sprintf("https://api.%s/api/v1", mc.Site)
	}

	if !mc.ShouldUseLogForwarder {
		shouldUseLogForwarder := os.Getenv(DatadogShouldUseLogForwarderEnvVar)
		mc.ShouldUseLogForwarder = strings.EqualFold(shouldUseLogForwarder, "true")
	}

	if mc.APIKey == "" {
		mc.APIKey = os.Getenv(DatadogAPIKeyEnvVar)

	}
	if mc.KMSAPIKey == "" {
		mc.KMSAPIKey = os.Getenv(DatadogKMSAPIKeyEnvVar)
	}
	if mc.APIKey == "" && mc.KMSAPIKey == "" && !mc.ShouldUseLogForwarder {
		logger.Error(fmt.Errorf("couldn't read DD_API_KEY or DD_KMS_API_KEY from environment"))
	}

	enhancedMetrics := os.Getenv("DD_ENHANCED_METRICS")
	if enhancedMetrics == "" {
		mc.EnhancedMetrics = DefaultEnhancedMetrics
	}
	if !mc.EnhancedMetrics {
		mc.EnhancedMetrics = strings.EqualFold(enhancedMetrics, "true")
	}

	if mc.GlobalTags == nil {
		val := os.Getenv(DatadogGlobalTags)
		mc.GlobalTags = []string{}
		if len(val) > 0 {
			mc.GlobalTags = strings.Split(val, ",")
		}
	}

	return mc
}
