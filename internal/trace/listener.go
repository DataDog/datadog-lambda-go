/*
 * Unless explicitly stated otherwise all files in this repository are licensed
 * under the Apache License Version 2.0.
 *
 * This product includes software developed at Datadog (https://www.datadoghq.com/).
 * Copyright 2021 Datadog, Inc.
 */

package trace

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/DataDog/datadog-lambda-go/internal/logger"
	"github.com/DataDog/datadog-lambda-go/internal/version"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type (
	// Listener creates a function execution span and injects it into the context
	Listener struct {
		ddTraceEnabled  bool
		mergeXrayTraces bool
	}

	// Config gives options for how the Listener should work
	Config struct {
		DDTraceEnabled  bool
		MergeXrayTraces bool
	}
)

// The function execution span is the top-level span representing the current Lambda function execution
var functionExecutionSpan ddtrace.Span

var tracerInitialized = false

// MakeListener initializes a new trace lambda Listener
func MakeListener(config Config) Listener {

	return Listener{
		ddTraceEnabled:  config.DDTraceEnabled,
		mergeXrayTraces: config.MergeXrayTraces,
	}
}

// HandlerStarted sets up tracing and starts the function execution span if Datadog tracing is enabled
func (l *Listener) HandlerStarted(ctx context.Context, msg json.RawMessage) context.Context {
	if !l.ddTraceEnabled {
		return ctx
	}

	ctx, _ = contextWithRootTraceContext(ctx, msg, l.mergeXrayTraces)

	if !tracerInitialized {
		tracer.Start(
			tracer.WithService("aws.lambda"),
			tracer.WithLambdaMode(true),
			tracer.WithGlobalTag("_dd.origin", "lambda"),
		)
		tracerInitialized = true
	}

	functionExecutionSpan = startFunctionExecutionSpan(ctx, l.mergeXrayTraces)

	// Add the span to the context so the user can create child spans
	ctx = tracer.ContextWithSpan(ctx, functionExecutionSpan)

	return ctx
}

// HandlerFinished ends the function execution span and stops the tracer
func (l *Listener) HandlerFinished(ctx context.Context, err error) {
	if functionExecutionSpan != nil {
		functionExecutionSpan.Finish(tracer.WithError(err))
	}
	tracer.Flush()
}

// startFunctionExecutionSpan starts a span that represents the current Lambda function execution
// and returns the span so that it can be finished when the function execution is complete
func startFunctionExecutionSpan(ctx context.Context, mergeXrayTraces bool) tracer.Span {
	// Extract information from context
	lambdaCtx, _ := lambdacontext.FromContext(ctx)
	rootTraceContext, ok := ctx.Value(traceContextKey).(TraceContext)
	if !ok {
		logger.Error(fmt.Errorf("Error extracting trace context from context object"))
	}

	functionArn := lambdaCtx.InvokedFunctionArn
	functionArn = strings.ToLower(functionArn)
	functionArn, functionVersion := separateVersionFromFunctionArn(functionArn)

	// Set the root trace context as the parent of the function execution span
	var parentSpanContext ddtrace.SpanContext
	convertedSpanContext, err := ConvertTraceContextToSpanContext(rootTraceContext)
	if err == nil {
		parentSpanContext = convertedSpanContext
	}

	span := tracer.StartSpan(
		"aws.lambda", // This operation name will be replaced with the value of the service tag by the Forwarder
		tracer.SpanType("serverless"),
		tracer.ChildOf(parentSpanContext),
		tracer.ResourceName(lambdacontext.FunctionName),
		tracer.Tag("cold_start", ctx.Value("cold_start")),
		tracer.Tag("function_arn", functionArn),
		tracer.Tag("function_version", functionVersion),
		tracer.Tag("request_id", lambdaCtx.AwsRequestID),
		tracer.Tag("resource_names", lambdacontext.FunctionName),
		tracer.Tag("datadog_lambda", version.DDLambdaVersion),
		tracer.Tag("dd_trace", version.DDTraceVersion),
	)

	if parentSpanContext != nil && mergeXrayTraces {
		// This tag will cause the Forwarder to drop the span (to avoid redundancy with X-Ray)
		span.SetTag("_dd.parent_source", "xray")
	}

	return span
}

func separateVersionFromFunctionArn(functionArn string) (arnWithoutVersion string, functionVersion string) {
	arnSegments := strings.Split(functionArn, ":")
	if cap(arnSegments) < 7 {
		return "", ""
	}
	functionVersion = "$LATEST"
	arnWithoutVersion = strings.Join(arnSegments[0:7], ":")
	if len(arnSegments) > 7 {
		functionVersion = arnSegments[7]
	}
	return arnWithoutVersion, functionVersion
}
