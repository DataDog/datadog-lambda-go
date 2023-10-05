/*
 * Unless explicitly stated otherwise all files in this repository are licensed
 * under the Apache License Version 2.0.
 *
 * This product includes software developed at Datadog (https://www.datadoghq.com/).
 * Copyright 2021 Datadog, Inc.
 */

package ddlambda

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/DataDog/datadog-lambda-go/internal/extension"
	"github.com/DataDog/datadog-lambda-go/internal/logger"
	"github.com/DataDog/datadog-lambda-go/internal/metrics"
	"github.com/DataDog/datadog-lambda-go/internal/trace"
	"github.com/DataDog/datadog-lambda-go/internal/wrapper"
	"github.com/aws/aws-lambda-go/lambda"
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
		// DDTraceEnabled enables the Datadog tracer.
		DDTraceEnabled bool
		// MergeXrayTraces will cause Datadog traces to be merged with traces from AWS X-Ray.
		MergeXrayTraces bool
		// HTTPClientTimeout specifies a time limit for requests to the API. It defaults to 5s.
		HTTPClientTimeout time.Duration
		// CircuitBreakerInterval is the cyclic period of the closed state
		// for the CircuitBreaker to clear the internal Counts.
		// default: 30s
		CircuitBreakerInterval time.Duration
		// CircuitBreakerTimeout is the period of the open state,
		// after which the state of the CircuitBreaker becomes half-open.
		// default: 60s
		CircuitBreakerTimeout time.Duration
		// CircuitBreakerTotalFailures after this amount of times
		// of a request failing in the closed state, the state will become open.
		// the counter will get totally reset after CircuitBreakerInterval
		// default: 4
		CircuitBreakerTotalFailures uint32
		// TraceContextExtractor is the function that extracts a root/parent trace context from the Lambda event body.
		// See trace.DefaultTraceExtractor for an example.
		TraceContextExtractor trace.ContextExtractor
	}
)

const (
	// DatadogAPIKeyEnvVar is the environment variable that will be used to set the API key.
	DatadogAPIKeyEnvVar = "DD_API_KEY"
	// DatadogKMSAPIKeyEnvVar is the environment variable that will be sent to KMS for decryption, then used as an API key.
	DatadogKMSAPIKeyEnvVar = "DD_KMS_API_KEY"
	// DatadogSiteEnvVar is the environment variable that will be used as the API host.
	DatadogSiteEnvVar = "DD_SITE"
	// LogLevelEnvVar is the environment variable that will be used to set the log level.
	LogLevelEnvVar = "DD_LOG_LEVEL"
	// ShouldUseLogForwarderEnvVar is the environment variable that enables log forwarding of metrics.
	ShouldUseLogForwarderEnvVar = "DD_FLUSH_TO_LOG"
	// DatadogTraceEnabledEnvVar is the environment variable that enables Datadog tracing.
	DatadogTraceEnabledEnvVar = "DD_TRACE_ENABLED"
	// MergeXrayTracesEnvVar is the environment variable that enables the merging of X-Ray and Datadog traces.
	MergeXrayTracesEnvVar = "DD_MERGE_XRAY_TRACES"
	// UniversalInstrumentation is the environment variable that enables universal instrumentation with the DD Extension
	UniversalInstrumentation = "DD_UNIVERSAL_INSTRUMENTATION"

	// DefaultSite to send API messages to.
	DefaultSite = "datadoghq.com"
	// DefaultEnhancedMetrics enables enhanced metrics by default.
	DefaultEnhancedMetrics = true
)

// WrapLambdaHandlerInterface is used to instrument your lambda functions.
// It returns a modified handler that can be passed directly to the lambda.StartHandler function from aws-lambda-go.
func WrapLambdaHandlerInterface(handler lambda.Handler, cfg *Config) lambda.Handler {
	listeners := initializeListeners(cfg)
	applyLambdaExecWrapperConfiguration()
	return wrapper.WrapHandlerInterfaceWithListeners(handler, listeners...)
}

// WrapFunction is used to instrument your lambda functions.
// It returns a modified handler that can be passed directly to the lambda.Start function from aws-lambda-go.
func WrapFunction(handler interface{}, cfg *Config) interface{} {
	listeners := initializeListeners(cfg)
	applyLambdaExecWrapperConfiguration()
	return wrapper.WrapHandlerWithListeners(handler, listeners...)
}

// WrapHandler is used to instrument your lambda functions.
// It returns a modified handler that can be passed directly to the lambda.Start function from aws-lambda-go.
// Deprecated: use WrapFunction instead
func WrapHandler(handler interface{}, cfg *Config) interface{} {
	return WrapFunction(handler, cfg)
}

// GetTraceHeaders returns a map containing Datadog trace headers that reflect the
// current X-Ray subsegment.
// Deprecated: use native Datadog tracing instead.
func GetTraceHeaders(ctx context.Context) map[string]string {
	result := trace.ConvertCurrentXrayTraceContext(ctx)
	return result
}

// AddTraceHeaders adds Datadog trace headers to a HTTP Request reflecting the current X-Ray
// subsegment.
// Deprecated: use native Datadog tracing instead.
func AddTraceHeaders(ctx context.Context, req *http.Request) {
	headers := trace.ConvertCurrentXrayTraceContext(ctx)
	for key, value := range headers {
		req.Header.Add(key, value)
	}
}

// GetContext retrieves the last created lambda context.
// Only use this if you aren't manually passing context through your call hierarchy.
func GetContext() context.Context {
	return wrapper.CurrentContext
}

// Distribution sends a distribution metric to Datadog
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

func (cfg *Config) toTraceConfig() trace.Config {
	traceConfig := trace.Config{
		DDTraceEnabled:           false,
		MergeXrayTraces:          false,
		UniversalInstrumentation: false,
	}

	if cfg != nil {
		traceConfig.DDTraceEnabled = cfg.DDTraceEnabled
		traceConfig.MergeXrayTraces = cfg.MergeXrayTraces
		traceConfig.TraceContextExtractor = cfg.TraceContextExtractor
	}

	if traceConfig.TraceContextExtractor == nil {
		traceConfig.TraceContextExtractor = trace.DefaultTraceExtractor
	}

	if !traceConfig.DDTraceEnabled {
		traceConfig.DDTraceEnabled, _ = strconv.ParseBool(os.Getenv(DatadogTraceEnabledEnvVar))
	}

	if !traceConfig.MergeXrayTraces {
		traceConfig.MergeXrayTraces, _ = strconv.ParseBool(os.Getenv(MergeXrayTracesEnvVar))
	}

	if !traceConfig.UniversalInstrumentation {
		traceConfig.UniversalInstrumentation, _ = strconv.ParseBool(os.Getenv(UniversalInstrumentation))
	}

	return traceConfig
}

func initializeListeners(cfg *Config) []wrapper.HandlerListener {
	logLevel := os.Getenv(LogLevelEnvVar)
	if strings.EqualFold(logLevel, "debug") || (cfg != nil && cfg.DebugLogging) {
		logger.SetLogLevel(logger.LevelDebug)
	}
	traceConfig := cfg.toTraceConfig()
	extensionManager := extension.BuildExtensionManager(traceConfig.UniversalInstrumentation)
	isExtensionRunning := extensionManager.IsExtensionRunning()
	metricsConfig := cfg.toMetricsConfig(isExtensionRunning)

	// Wrap the handler with listeners that add instrumentation for traces and metrics.
	tl := trace.MakeListener(traceConfig, extensionManager)
	ml := metrics.MakeListener(metricsConfig, extensionManager)
	return []wrapper.HandlerListener{
		&tl, &ml,
	}
}

func (cfg *Config) toMetricsConfig(isExtensionRunning bool) metrics.Config {

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
		mc.HTTPClientTimeout = cfg.HTTPClientTimeout
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
		shouldUseLogForwarder := os.Getenv(ShouldUseLogForwarderEnvVar)
		mc.ShouldUseLogForwarder = strings.EqualFold(shouldUseLogForwarder, "true")
	}

	if mc.APIKey == "" {
		mc.APIKey = os.Getenv(DatadogAPIKeyEnvVar)

	}
	if mc.KMSAPIKey == "" {
		mc.KMSAPIKey = os.Getenv(DatadogKMSAPIKeyEnvVar)
	}
	if !isExtensionRunning && mc.APIKey == "" && mc.KMSAPIKey == "" && !mc.ShouldUseLogForwarder {
		logger.Error(fmt.Errorf(
			"couldn't read %s or %s from environment", DatadogAPIKeyEnvVar, DatadogKMSAPIKeyEnvVar,
		))
	}

	enhancedMetrics := os.Getenv("DD_ENHANCED_METRICS")
	if enhancedMetrics == "" {
		mc.EnhancedMetrics = DefaultEnhancedMetrics
	}
	if !mc.EnhancedMetrics {
		mc.EnhancedMetrics = strings.EqualFold(enhancedMetrics, "true")
	}

	return mc
}

// applyLambdaExecWrapperConfiguration applies environment variables set by the wrapper configured in the
// `AWS_LAMBDA_EXEC_WRAPPER` environment variable if present. This is done here because the AWS Lambda runtimes used by
// go applications (`go1.x` and `provided.al2`) do not honor this setting, while other runtimes do. This assumes the
// wrapper script does nothing other than setting environment variables, and will only honor environment variables with
// names starting with `DD_` or `AWS_LAMBDA_`. In particular, `AWS_LAMBDA_RUNTIME_API` is used to re-route the Lambda
// control flow API through a proxy, which is used by ASM.
//
// See: https://docs.aws.amazon.com/lambda/latest/dg/runtimes-modify.html#runtime-wrapper.
// See: https://github.com/DataDog/datadog-lambda-extension/blob/main/scripts/datadog_wrapper.
func applyLambdaExecWrapperConfiguration() {
	const AwsExecutionEnvEnvVar = "AWS_EXECUTION_ENV"
	const AwsLambdaExecWrapperEnvVar = "AWS_LAMBDA_EXEC_WRAPPER"

	// Only perform this operation if the Lambda runtime doesn't do it by itself, meaning we only do something for
	// provided runtimes (in which `AWS_EXECUTION_ENV` is not set) and `go1.x`.
	if env := os.Getenv(AwsExecutionEnvEnvVar); env != "" && env != "AWS_Lambda_go1.x" {
		logger.Debug(fmt.Sprintf("Skipping applyLambdaExecWrapperConfiguration, runtime is %s", env))
		return
	}

	script := os.Getenv(AwsLambdaExecWrapperEnvVar)
	if script == "" {
		// Nothing to do
		return
	}

	cmd := exec.Command("env", "-u", AwsLambdaExecWrapperEnvVar, script, "sh", "-c", "env")
	logger.Debug(fmt.Sprintf("[%s] Command: %s", AwsLambdaExecWrapperEnvVar, cmd.String()))

	if stdout, err := cmd.Output(); err != nil {
		logger.Debug(fmt.Sprintf("[%s] Failed to run: %s", AwsLambdaExecWrapperEnvVar, err))
	} else {
		for _, line := range strings.Split(string(stdout), "\n") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				// Forward what the wrapper script prints to the standard output...
				fmt.Println(line)
				continue
			}
			name := parts[0]
			val := parts[1]

			if os.Getenv(name) == val {
				// Not changed, nothing to do.
				continue
			}

			if !strings.HasPrefix(name, "DD_") && !strings.HasPrefix(name, "AWS_LAMBDA_") {
				logger.Debug(fmt.Sprintf("[%s] Skip %s=<redacted>", AwsLambdaExecWrapperEnvVar, name))
				continue
			}

			if err := os.Setenv(name, val); err != nil {
				logger.Debug(fmt.Sprintf("[%s] Failed %s=%s: %s", AwsLambdaExecWrapperEnvVar, name, val, err))
			} else {
				logger.Debug(fmt.Sprintf("[%s] Set %s=%s", AwsLambdaExecWrapperEnvVar, name, val))
			}
		}
	}
}
