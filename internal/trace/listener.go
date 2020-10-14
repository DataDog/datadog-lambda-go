/*
 * Unless explicitly stated otherwise all files in this repository are licensed
 * under the Apache License Version 2.0.
 *
 * This product includes software developed at Datadog (https://www.datadoghq.com/).
 * Copyright 2019 Datadog, Inc.
 */

package trace

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/DataDog/datadog-lambda-go/internal/logger"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type (
	// Listener creates a function execution span and injects it into the context
	Listener struct {
		DDTraceEnabled  bool
		MergeXrayTraces bool
	}

	// Config gives options for how the Listener should work
	Config struct {
		DDTraceEnabled  bool
		MergeXrayTraces bool
	}
)

// The function execution span is the top-level span representing the current Lambda function execution
var functionExecutionSpan ddtrace.Span

// MakeListener initializes a new trace lambda Listener
func MakeListener(config Config) Listener {

	return Listener{
		DDTraceEnabled:  config.DDTraceEnabled,
		MergeXrayTraces: config.MergeXrayTraces,
	}
}

// HandlerStarted creates the function execution span representing the Lambda function execution
// and adds that span to the context so that the user can create child spans (if Datadog tracing is enabled)
func (l *Listener) HandlerStarted(ctx context.Context, msg json.RawMessage) context.Context {
	ctx, _ = ContextWithTraceContext(ctx, msg)

	if l.DDTraceEnabled {
		tracer.Start(
			tracer.WithService("aws.lambda"),
			tracer.WithLambdaMode(true),
			tracer.WithDebugMode(true),
			tracer.WithGlobalTag("__dd.origin", "lambda"),
		)

		startFunctionExecutionSpan(
			ctx,
			l.MergeXrayTraces,
		)

		// Add the span to the context so the user can create child spans
		ctx = tracer.ContextWithSpan(ctx, functionExecutionSpan)
	}

	return ctx
}

// HandlerFinished finishes the function execution span (if it was started) and stops the tracer
func (l *Listener) HandlerFinished(ctx context.Context) {
	if functionExecutionSpan != nil {
		functionExecutionSpan.Finish()
	}

	tracer.Stop()
}

func startFunctionExecutionSpan(ctx context.Context, mergeXrayTraces bool) {
	// Extract information from context
	lc, _ := lambdacontext.FromContext(ctx)
	isColdStart := ctx.Value("cold_start").(bool)
	var traceSource string
	traceContext, ok := ctx.Value(traceContextKey).(map[string]string)
	if ok {
		traceSource = traceContext[sourceType]
	} else {
		logger.Error("Error extracting Datadog trace context from context")
	}

	functionArn := lc.InvokedFunctionArn
	functionArn = strings.ToLower(functionArn)

	// Separate version from rest of function ARN
	parts := strings.Split(functionArn, ":")
	functionVersion := "$LATEST"
	if len(parts) > 7 {
		functionArn = strings.Join(parts[0:7], ":")
		functionVersion = parts[7]
	}

	// The function execution span must be made a child of the current span if the trace context came from an event OR merge X-Ray traces is enabled
	// In other words, if merge X-Ray traces is NOT enabled and the trace context came from X-Ray, we should NOT make the execution span a child of the X-Ray span
	var parentSpanContext ddtrace.SpanContext
	if (traceSource == fromEvent) || mergeXrayTraces {
		convertedSpanContext, err := convertTraceContextToSpanContext(traceContext)
		if err == nil {
			parentSpanContext = convertedSpanContext
		}
	}

	span := tracer.StartSpan(
		"aws.lambda",
		tracer.SpanType("serverless"),
		tracer.ChildOf(parentSpanContext),
		tracer.ResourceName(lambdacontext.FunctionName),
		tracer.Tag("cold_start", strconv.FormatBool(isColdStart)),
		tracer.Tag("function_arn", functionArn),
		tracer.Tag("function_version", functionVersion),
		tracer.Tag("request_id", lc.AwsRequestID),
		tracer.Tag("resource_names", lambdacontext.FunctionName),
	)

	if traceSource == fromXray && mergeXrayTraces {
		span.SetTag("_dd.parent_source", traceSource)
	}

	functionExecutionSpan = span
}

// Convert trace context headers to a SpanContext object
func convertTraceContextToSpanContext(traceContext map[string]string) (ddtrace.SpanContext, error) {
	spanContext, err := traceContextPropagator.Extract(tracer.TextMapCarrier(traceContext))

	if err == nil {
		return spanContext, nil
	}

	logger.Error("Error converting trace context to span context")
	return nil, err
}

// A Propagator that is able to extract a SpanContext object from trace context headers
var traceContextPropagatorConfig = tracer.PropagatorConfig{
	TraceHeader:    traceIDHeader,
	ParentHeader:   parentIDHeader,
	PriorityHeader: samplingPriorityHeader,
}
var traceContextPropagator = tracer.NewPropagator(&traceContextPropagatorConfig)
