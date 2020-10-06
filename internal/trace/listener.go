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
    "strings"
    "strconv"
    "gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
    "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
    "github.com/aws/aws-lambda-go/lambdacontext"
)

type (
	Listener struct{
		ddTraceEnabled		  bool
		mergeXRayTraces       bool
	}

	// Config gives options for how the Listener should work
	Config struct {
		ddTraceEnabled		  bool
		mergeXRayTraces       bool
	}
)

t := tracer

// The function execution span is the top-level span representing the Lambda function execution
// If Datadog tracing is enabled, we use this 
var functionExecutionSpan ddtrace.Span

// MakeListener initializes a new trace lambda Listener
func MakeListener(config Config) Listener {

	return Listener{
		ddTraceEnabled: config.ddTraceEnabled,
		mergeXRayTraces: config.mergeXRayTraces,
	}
}

func (l *Listener) HandlerStarted(ctx context.Context, msg json.RawMessage) context.Context {
	traceContext, _ = ExtractTraceContext(ctx, msg)

    if l.ddTraceEnabled:
        t.Start(
            // service name?
            tracer.WithLambdaMode(true),
            tracer.WithDebugMode(true),
            tracer.WithGlobalTag("__dd.origin", "lambda")
        )

		startFunctionExecutionSpan(
            ctx,
            traceContext,
            l.mergeXRayTraces
        )

	return traceContext
}

func (l *Listener) HandlerFinished(ctx context.Context) {
    if functionExecutionSpan:
        functionExecutionSpan.finish()

    t.Stop()
}

func startFunctionExecutionSpan(context ctx.Context, traceContext ctx.Context, mergeXrayTraces bool) {
    lc, _ := lambdacontext.FromContext(ctx)

    functionArn := lc.InvokedFunctionArn
    if functionArn === nil:
        functionArn = ""
    functionArn = strings.ToLower(functionArn)

    // Separate version from rest of function ARN
    parts = strings.Split(functionArn, ":")
    functionVersion = "$LATEST"
    if len(parts) > 7:
        functionArn = strings.Join(parts[0:7], ":")
        functionVersion = parts[7]

    isColdStart := ctx.Value("cold_start")

    span := tracer.StartSpan(
        "aws.lambda",
        tracer.WithService("aws.lambda"),
        tracer.SpanType("serverless"),
        tracer.ResourceName(functionName),
        tracer.Tag("cold_start", strconv.FormatBool(isColdStart)),
        tracer.Tag("function_arn", functionArn),
        tracer.Tag("function_version", functionVersion),
        tracer.Tag("request_id", lc.AwsRequestId,
        tracer.Tag("resource_names", lc.FunctionName,
    )

    traceSource := traceContext[sourceType]

    if traceSource == fromXray && mergeXrayTraces:
        span.SetTag("_dd.parent_source", traceSource)

    functionExecutionSpan = span
}
