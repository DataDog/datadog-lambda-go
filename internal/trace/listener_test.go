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
	"fmt"
	"testing"

	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/mocktracer"
)

func TestSeparateVersionFromFunctionArnWithVersion(t *testing.T) {
	inputArn := "arn:aws:lambda:us-east-1:123456789012:function:my-function:9"

	arnWithoutVersion, functionVersion := separateVersionFromFunctionArn(inputArn)

	expectedArnWithoutVersion := "arn:aws:lambda:us-east-1:123456789012:function:my-function"
	expectedFunctionVersion := "9"
	assert.Equal(t, expectedArnWithoutVersion, arnWithoutVersion)
	assert.Equal(t, expectedFunctionVersion, functionVersion)
}

func TestSeparateVersionFromFunctionArnWithoutVersion(t *testing.T) {
	inputArn := "arn:aws:lambda:us-east-1:123456789012:function:my-function"

	arnWithoutVersion, functionVersion := separateVersionFromFunctionArn(inputArn)

	expectedArnWithoutVersion := "arn:aws:lambda:us-east-1:123456789012:function:my-function"
	expectedFunctionVersion := "$LATEST"
	assert.Equal(t, expectedArnWithoutVersion, arnWithoutVersion)
	assert.Equal(t, expectedFunctionVersion, functionVersion)
}

var traceContextFromXray = TraceContext{
	traceIDHeader:          "1231452342",
	parentIDHeader:         "45678910",
	samplingPriorityHeader: "2",
	sourceType:             fromXray,
}

func TestStartFunctionExecutionSpanFromXrayWithMergeEnabled(t *testing.T) {
	ctx := context.Background()

	// Inject mock Lambda context into mock context
	lambdacontext.FunctionName = "MockFunctionName"
	lambdaCtx := lambdacontext.LambdaContext{
		AwsRequestID:       "abcdefgh-1234-5678-1234-abcdefghijkl",
		InvokedFunctionArn: "arn:aws:lambda:us-east-1:123456789012:function:MyFunction:11",
	}
	ctx = lambdacontext.NewContext(ctx, &lambdaCtx)

	// Inject mock TraceContext into context
	traceCtx := TraceContext{}
	traceCtx[traceIDHeader] = "1231452342"
	traceCtx[parentIDHeader] = "45678910"
	traceCtx[samplingPriorityHeader] = "2"
	traceCtx[sourceType] = fromXray
	ctx = context.WithValue(ctx, traceContextKey, traceCtx)

	ctx = context.WithValue(ctx, "cold_start", true)

	mt := mocktracer.Start()
	defer mt.Stop()

	span := startFunctionExecutionSpan(ctx, true)
	span.Finish()
	finishedSpan := mt.FinishedSpans()[0]

	fmt.Print(span)

	assert.Equal(t, "aws.lambda", finishedSpan.OperationName())

	assert.Equal(t, true, finishedSpan.Tag("cold_start"))
	// We expect the function ARN to be lowercased, and the version removed
	assert.Equal(t, "arn:aws:lambda:us-east-1:123456789012:function:myfunction", finishedSpan.Tag("function_arn"))
	assert.Equal(t, "11", finishedSpan.Tag("function_version"))
	assert.Equal(t, "abcdefgh-1234-5678-1234-abcdefghijkl", finishedSpan.Tag("request_id"))
	assert.Equal(t, "MockFunctionName", finishedSpan.Tag("resource.name"))
	assert.Equal(t, "MockFunctionName", finishedSpan.Tag("resource_names"))
	assert.Equal(t, "serverless", finishedSpan.Tag("span.type"))
	assert.Equal(t, "xray", finishedSpan.Tag("_dd.parent_source"))
}

func TestStartFunctionExecutionSpanFromXrayWithMergeDisabled(t *testing.T) {
	ctx := context.Background()

	// Inject mock Lambda context into mock context
	lambdacontext.FunctionName = "MockFunctionName"
	lambdaCtx := lambdacontext.LambdaContext{
		AwsRequestID:       "abcdefgh-1234-5678-1234-abcdefghijkl",
		InvokedFunctionArn: "arn:aws:lambda:us-east-1:123456789012:function:MyFunction:11",
	}
	ctx = lambdacontext.NewContext(ctx, &lambdaCtx)

	// Inject mock TraceContext into context
	traceCtx := TraceContext{}
	traceCtx[traceIDHeader] = "1231452342"
	traceCtx[parentIDHeader] = "45678910"
	traceCtx[samplingPriorityHeader] = "2"
	traceCtx[sourceType] = fromXray
	ctx = context.WithValue(ctx, traceContextKey, traceCtx)

	ctx = context.WithValue(ctx, "cold_start", true)

	mt := mocktracer.Start()
	defer mt.Stop()

	span := startFunctionExecutionSpan(ctx, false)
	span.Finish()
	finishedSpan := mt.FinishedSpans()[0]

	fmt.Print(span)

	assert.Equal(t, "aws.lambda", finishedSpan.OperationName())

	assert.Equal(t, true, finishedSpan.Tag("cold_start"))
	// We expect the function ARN to be lowercased, and the version removed
	assert.Equal(t, "arn:aws:lambda:us-east-1:123456789012:function:myfunction", finishedSpan.Tag("function_arn"))
	assert.Equal(t, "11", finishedSpan.Tag("function_version"))
	assert.Equal(t, "abcdefgh-1234-5678-1234-abcdefghijkl", finishedSpan.Tag("request_id"))
	assert.Equal(t, "MockFunctionName", finishedSpan.Tag("resource.name"))
	assert.Equal(t, "MockFunctionName", finishedSpan.Tag("resource_names"))
	assert.Equal(t, "serverless", finishedSpan.Tag("span.type"))
	assert.Equal(t, nil, finishedSpan.Tag("_dd.parent_source"))
}

func TestStartFunctionExecutionSpanFromEventWithMergeEnabled(t *testing.T) {
	ctx := context.Background()

	// Inject mock Lambda context into mock context
	lambdacontext.FunctionName = "MockFunctionName"
	lambdaCtx := lambdacontext.LambdaContext{
		AwsRequestID:       "abcdefgh-1234-5678-1234-abcdefghijkl",
		InvokedFunctionArn: "arn:aws:lambda:us-east-1:123456789012:function:MyFunction:11",
	}
	ctx = lambdacontext.NewContext(ctx, &lambdaCtx)

	// Inject mock TraceContext into context
	traceCtx := TraceContext{}
	traceCtx[traceIDHeader] = "1231452342"
	traceCtx[parentIDHeader] = "45678910"
	traceCtx[samplingPriorityHeader] = "2"
	traceCtx[sourceType] = fromEvent
	ctx = context.WithValue(ctx, traceContextKey, traceCtx)

	ctx = context.WithValue(ctx, "cold_start", true)

	mt := mocktracer.Start()
	defer mt.Stop()

	span := startFunctionExecutionSpan(ctx, true)
	span.Finish()
	finishedSpan := mt.FinishedSpans()[0]

	fmt.Print(span)

	assert.Equal(t, "aws.lambda", finishedSpan.OperationName())

	assert.Equal(t, true, finishedSpan.Tag("cold_start"))
	// We expect the function ARN to be lowercased, and the version removed
	assert.Equal(t, "arn:aws:lambda:us-east-1:123456789012:function:myfunction", finishedSpan.Tag("function_arn"))
	assert.Equal(t, "11", finishedSpan.Tag("function_version"))
	assert.Equal(t, "abcdefgh-1234-5678-1234-abcdefghijkl", finishedSpan.Tag("request_id"))
	assert.Equal(t, "MockFunctionName", finishedSpan.Tag("resource.name"))
	assert.Equal(t, "MockFunctionName", finishedSpan.Tag("resource_names"))
	assert.Equal(t, "serverless", finishedSpan.Tag("span.type"))
	assert.Equal(t, nil, finishedSpan.Tag("_dd.parent_source"))
}

func TestStartFunctionExecutionSpanFromEventWithMergeDisabled(t *testing.T) {
	ctx := context.Background()

	// Inject mock Lambda context into mock context
	lambdacontext.FunctionName = "MockFunctionName"
	lambdaCtx := lambdacontext.LambdaContext{
		AwsRequestID:       "abcdefgh-1234-5678-1234-abcdefghijkl",
		InvokedFunctionArn: "arn:aws:lambda:us-east-1:123456789012:function:MyFunction:11",
	}
	ctx = lambdacontext.NewContext(ctx, &lambdaCtx)

	// Inject mock TraceContext into context
	traceCtx := TraceContext{}
	traceCtx[traceIDHeader] = "1231452342"
	traceCtx[parentIDHeader] = "45678910"
	traceCtx[samplingPriorityHeader] = "2"
	traceCtx[sourceType] = fromEvent
	ctx = context.WithValue(ctx, traceContextKey, traceCtx)

	ctx = context.WithValue(ctx, "cold_start", true)

	mt := mocktracer.Start()
	defer mt.Stop()

	span := startFunctionExecutionSpan(ctx, false)
	span.Finish()
	finishedSpan := mt.FinishedSpans()[0]

	fmt.Print(span)

	assert.Equal(t, "aws.lambda", finishedSpan.OperationName())

	assert.Equal(t, true, finishedSpan.Tag("cold_start"))
	// We expect the function ARN to be lowercased, and the version removed
	assert.Equal(t, "arn:aws:lambda:us-east-1:123456789012:function:myfunction", finishedSpan.Tag("function_arn"))
	assert.Equal(t, "11", finishedSpan.Tag("function_version"))
	assert.Equal(t, "abcdefgh-1234-5678-1234-abcdefghijkl", finishedSpan.Tag("request_id"))
	assert.Equal(t, "MockFunctionName", finishedSpan.Tag("resource.name"))
	assert.Equal(t, "MockFunctionName", finishedSpan.Tag("resource_names"))
	assert.Equal(t, "serverless", finishedSpan.Tag("span.type"))
	assert.Equal(t, nil, finishedSpan.Tag("_dd.parent_source"))
}
