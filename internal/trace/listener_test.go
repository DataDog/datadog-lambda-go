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
	"os"
	"testing"

	"github.com/DataDog/datadog-lambda-go/internal/extension"
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

func TestSeparateVersionFromFunctionArnEmptyString(t *testing.T) {
	inputArn := ""

	arnWithoutVersion, functionVersion := separateVersionFromFunctionArn(inputArn)
	assert.Empty(t, arnWithoutVersion)
	assert.Empty(t, functionVersion)
}

var traceContextFromXray = TraceContext{
	traceIDHeader:  "1231452342",
	parentIDHeader: "45678910",
}

var traceContextFromEvent = TraceContext{
	traceIDHeader:  "1231452342",
	parentIDHeader: "45678910",
}

var mockLambdaContext = lambdacontext.LambdaContext{
	AwsRequestID:       "abcdefgh-1234-5678-1234-abcdefghijkl",
	InvokedFunctionArn: "arn:aws:lambda:us-east-1:123456789012:function:MyFunction:11",
}

func TestStartFunctionExecutionSpanFromXrayWithMergeEnabled(t *testing.T) {
	ctx := context.Background()

	lambdacontext.FunctionName = "MockFunctionName"
	ctx = lambdacontext.NewContext(ctx, &mockLambdaContext)
	ctx = context.WithValue(ctx, traceContextKey, traceContextFromXray)
	//nolint
	ctx = context.WithValue(ctx, "cold_start", true)

	mt := mocktracer.Start()
	defer mt.Stop()

	span := startFunctionExecutionSpan(ctx, true)
	span.Finish()
	finishedSpan := mt.FinishedSpans()[0]

	assert.Equal(t, "aws.lambda", finishedSpan.OperationName())

	assert.Equal(t, true, finishedSpan.Tag("cold_start"))
	// We expect the function ARN to be lowercased, and the version removed
	assert.Equal(t, "arn:aws:lambda:us-east-1:123456789012:function:myfunction", finishedSpan.Tag("function_arn"))
	assert.Equal(t, "11", finishedSpan.Tag("function_version"))
	assert.Equal(t, "abcdefgh-1234-5678-1234-abcdefghijkl", finishedSpan.Tag("request_id"))
	assert.Equal(t, "MockFunctionName", finishedSpan.Tag("resource.name"))
	assert.Equal(t, "MockFunctionName", finishedSpan.Tag("resource_names"))
	assert.Equal(t, "mockfunctionname", finishedSpan.Tag("functionname"))
	assert.Equal(t, "serverless", finishedSpan.Tag("span.type"))
	assert.Equal(t, "xray", finishedSpan.Tag("_dd.parent_source"))
}

func TestStartFunctionExecutionSpanFromXrayWithMergeDisabled(t *testing.T) {
	ctx := context.Background()

	lambdacontext.FunctionName = "MockFunctionName"
	ctx = lambdacontext.NewContext(ctx, &mockLambdaContext)
	ctx = context.WithValue(ctx, traceContextKey, traceContextFromXray)
	//nolint
	ctx = context.WithValue(ctx, "cold_start", true)

	mt := mocktracer.Start()
	defer mt.Stop()

	span := startFunctionExecutionSpan(ctx, false)
	span.Finish()
	finishedSpan := mt.FinishedSpans()[0]

	assert.Equal(t, nil, finishedSpan.Tag("_dd.parent_source"))
}

func TestStartFunctionExecutionSpanFromEventWithMergeEnabled(t *testing.T) {
	ctx := context.Background()

	lambdacontext.FunctionName = "MockFunctionName"
	ctx = lambdacontext.NewContext(ctx, &mockLambdaContext)
	ctx = context.WithValue(ctx, traceContextKey, traceContextFromEvent)
	//nolint
	ctx = context.WithValue(ctx, "cold_start", true)

	mt := mocktracer.Start()
	defer mt.Stop()

	span := startFunctionExecutionSpan(ctx, true)
	span.Finish()
	finishedSpan := mt.FinishedSpans()[0]

	assert.Equal(t, "xray", finishedSpan.Tag("_dd.parent_source"))
}

func TestStartFunctionExecutionSpanFromEventWithMergeDisabled(t *testing.T) {
	ctx := context.Background()

	lambdacontext.FunctionName = "MockFunctionName"
	ctx = lambdacontext.NewContext(ctx, &mockLambdaContext)
	ctx = context.WithValue(ctx, traceContextKey, traceContextFromEvent)
	//nolint
	ctx = context.WithValue(ctx, "cold_start", true)

	mt := mocktracer.Start()
	defer mt.Stop()

	span := startFunctionExecutionSpan(ctx, false)
	span.Finish()
	finishedSpan := mt.FinishedSpans()[0]

	assert.Equal(t, nil, finishedSpan.Tag("_dd.parent_source"))
}

func TestListener_buildTraceStartOptions(t *testing.T) {
	t.Run("when the DD_SERVICE neither the DD_ENV are present", func(t *testing.T) {
		os.Unsetenv("DD_SERVICE")
		os.Unsetenv("DD_ENV")

		listener := Listener{extensionManager: &extension.ExtensionManager{}}

		got := listener.buildTraceStartOptions()

		assert.Equal(t, len(got), 3)
	})

	t.Run("when the DD_SERVICE only is present", func(t *testing.T) {
		customServiceName := "my-service"

		os.Setenv("DD_SERVICE", customServiceName)
		defer os.Unsetenv("DD_SERVICE")

		os.Unsetenv("DD_ENV")

		listener := Listener{extensionManager: &extension.ExtensionManager{}}

		got := listener.buildTraceStartOptions()

		assert.Equal(t, len(got), 3)
	})

	t.Run("when the DD_ENV only is present", func(t *testing.T) {
		customEnvName := "my-env"

		os.Unsetenv("DD_SERVICE")
		os.Setenv("DD_ENV", customEnvName)
		defer os.Unsetenv("DD_ENV")

		listener := Listener{extensionManager: &extension.ExtensionManager{}}

		got := listener.buildTraceStartOptions()

		assert.Equal(t, len(got), 4)
	})

	t.Run("when the DD_ENV and DD_SERVICE are present", func(t *testing.T) {
		customEnvName := "my-env"
		customServiceName := "my-service"

		os.Setenv("DD_SERVICE", customServiceName)
		defer os.Unsetenv("DD_SERVICE")

		os.Setenv("DD_ENV", customEnvName)
		defer os.Unsetenv("DD_ENV")

		listener := Listener{extensionManager: &extension.ExtensionManager{}}

		got := listener.buildTraceStartOptions()

		assert.Equal(t, len(got), 4)
	})
}
