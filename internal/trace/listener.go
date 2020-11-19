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
	"fmt"
	"strings"

	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/stroem/datadog-lambda-go/internal/logger"
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

// MakeListener initializes a new trace lambda Listener
func MakeListener(config Config) Listener {

	return Listener{
		ddTraceEnabled:  config.DDTraceEnabled,
		mergeXrayTraces: config.MergeXrayTraces,
	}
}

// HandlerStarted creates the function execution span representing the Lambda function execution
// and adds that span to the context so that the user can create child spans (if Datadog tracing is enabled)
func (l *Listener) HandlerStarted(ctx context.Context, msg json.RawMessage) context.Context {
	ctx, _ = ContextWithTraceContext(ctx, msg)

	if l.ddTraceEnabled {
		tracer.Start(
			tracer.WithService("aws.lambda"),
			tracer.WithLambdaMode(true),
			tracer.WithDebugMode(true),
			tracer.WithGlobalTag("__dd.origin", "lambda"),
		)

		functionExecutionSpan = startFunctionExecutionSpan(ctx, l.mergeXrayTraces)

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

	// Stop the tracer, forcing it to flush any traces it's holding
	// Without this, we might drop traces
	tracer.Stop()
}

// startFunctionExecutionSpan starts a span that represents the current Lambda function execution
// and returns the span so that it can be finished when the function execution is complete
func startFunctionExecutionSpan(ctx context.Context, mergeXrayTraces bool) tracer.Span {
	// Extract information from context
	lambdaCtx, _ := lambdacontext.FromContext(ctx)
	var traceSource string
	traceContext, ok := ctx.Value(traceContextKey).(TraceContext)
	if ok {
		traceSource = traceContext[sourceType]
	} else {
		logger.Error(fmt.Errorf("Error extracting Datadog trace context from context"))
	}

	functionArn := lambdaCtx.InvokedFunctionArn
	functionArn = strings.ToLower(functionArn)
	functionArn, functionVersion := separateVersionFromFunctionArn(functionArn)

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
		tracer.Tag("cold_start", ctx.Value("cold_start")),
		tracer.Tag("function_arn", functionArn),
		tracer.Tag("function_version", functionVersion),
		tracer.Tag("request_id", lambdaCtx.AwsRequestID),
		tracer.Tag("resource_names", lambdacontext.FunctionName),
	)

	if traceSource == fromXray && mergeXrayTraces {
		span.SetTag("_dd.parent_source", traceSource)
	}

	return span
}

func separateVersionFromFunctionArn(functionArn string) (arnWithoutVersion string, functionVersion string) {
	arnSegments := strings.Split(functionArn, ":")
	functionVersion = "$LATEST"
	arnWithoutVersion = strings.Join(arnSegments[0:7], ":")
	if len(arnSegments) > 7 {
		functionVersion = arnSegments[7]
	}
	return arnWithoutVersion, functionVersion
}

func convertTraceContextToSpanContext(traceCtx TraceContext) (ddtrace.SpanContext, error) {
	spanCtx, err := propagator.Extract(tracer.TextMapCarrier(traceCtx))

	if err != nil {
		logger.Error(fmt.Errorf("Error extracting Datadog trace context from context: %v", err))
		return nil, err
	}

	return spanCtx, nil
}

// propagator is able to extract a SpanContext object from a TraceContext object
var propagator = tracer.NewPropagator(&tracer.PropagatorConfig{
	TraceHeader:    traceIDHeader,
	ParentHeader:   parentIDHeader,
	PriorityHeader: samplingPriorityHeader,
})
