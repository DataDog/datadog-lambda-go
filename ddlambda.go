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
	"time"

	"github.com/DataDog/dd-lambda-go/internal/metrics"
	"github.com/DataDog/dd-lambda-go/internal/trace"
	"github.com/DataDog/dd-lambda-go/internal/wrapper"
)

type (
	// Config gives options for how ddlambda should behave
	Config struct {
		// APIKey is your Datadog API key. This is used for sending metrics.
		APIKey string
		// AppKey is your Datadog App key. This is used for sending metrics.
		AppKey string
		// ShouldRetryOnFailure is used to turn on retry logic when sending metrics via the API. This can negatively effect the performance of your lambda,
		// and should only be turned on if you can't afford to lose metrics data under poor network conditions.
		ShouldRetryOnFailure bool
		// BatchInterval is the period of time which metrics are grouped together for processing to be sent to the API or written to logs.
		// Any pending metrics are flushed at the end of the lambda.
		BatchInterval time.Duration
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
		return
	}

	// We add our own runtime tag to the metric for version tracking
	tags = append(tags, getRuntimeTag())

	m := metrics.Distribution{
		Name:   metric,
		Tags:   tags,
		Values: []float64{},
	}
	m.AddPoint(value)
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

func getRuntimeTag() string {
	v := runtime.Version()
	return fmt.Sprintf("dd_lambda_layer:datadog-%s", v)
}
