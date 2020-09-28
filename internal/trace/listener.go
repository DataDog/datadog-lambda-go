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

// MakeListener initializes a new trace lambda Listener
func MakeListener(config Config) Listener {

	return Listener{
		ddTraceEnabled: config.ddTraceEnabled,
		mergeXRayTraces: config.mergeXRayTraces,
	}
}

func (l *Listener) HandlerStarted(ctx context.Context, msg json.RawMessage) context.Context {
	ctx, _ = ExtractTraceContext(ctx, msg)

	if l.ddTraceEnabled:
		createFunctionExecutionSpan(ctx, functionName, isColdStart, ddContext, l.mergeXRayTraces)

	return ctx
}

// HandlerFinished is implemented as part of the HandlerListener interface, but doesn't do anything
func (l *Listener) HandlerFinished(ctx context.Context) {
}

// TODO: Add MakeListener function that allows creation of listener with custom env vars


// TODO: Convert createFunctionExecutionSpan to Go
func createFunctionExecutionSpan(context ctx.Context, functionName string, isColdStart bool, ddContext ctx.Context, mergeXrayTraces bool) {
    
	// Python Code
    tags = {}
    if context:
        function_arn = (context.invoked_function_arn or "").lower()
        tk = function_arn.split(":")
        function_arn = ":".join(tk[0:7]) if len(tk) > 7 else function_arn
        function_version = tk[7] if len(tk) > 7 else "$LATEST"

        tags = {
            "cold_start": str(is_cold_start).lower(),
            "function_arn": function_arn,
            "function_version": function_version,
            "request_id": context.aws_request_id,
            "resource_names": context.function_name,
        }
    source = trace_context["source"]

    if source == TraceContextSource.XRAY and merge_xray_traces:
        tags["_dd.parent_source"] = source

    args = {
        "service": "aws.lambda",
        "resource": function_name,
        "span_type": "serverless",
    }

    tracer.set_tags({"_dd.origin": "lambda"})

    span = tracer.trace("aws.lambda", **args)

    if span:
        span.set_tags(tags)
    return span
}
