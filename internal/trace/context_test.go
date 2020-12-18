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
	"io/ioutil"
	"testing"

	"github.com/aws/aws-xray-sdk-go/header"

	"github.com/aws/aws-xray-sdk-go/xray"

	"github.com/stretchr/testify/assert"
)

const (
	mockXRayEntityID      = "0b11cc4230d3e09e"
	mockXRayTraceID       = "1-5ce31dc2-2c779014b90ce44db5e03875"
	convertedXRayEntityID = "797643193680388254"
	convertedXRayTraceID  = "4110911582297405557"
)

func mockLambdaXRayTraceContext(ctx context.Context, traceID, parentID string, sampled bool) context.Context {
	decision := header.NotSampled
	if sampled {
		decision = header.Sampled
	}

	traceHeader := header.Header{
		TraceID:          traceID,
		ParentID:         parentID,
		SamplingDecision: decision,
		AdditionalData:   make(TraceContext),
	}
	headerString := traceHeader.String()
	return context.WithValue(ctx, xray.LambdaTraceHeaderKey, headerString)
}

func loadRawJSON(t *testing.T, filename string) *json.RawMessage {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		assert.Fail(t, "Couldn't find JSON file")
		return nil
	}
	msg := json.RawMessage{}
	msg.UnmarshalJSON(bytes)
	return &msg
}
func TestGetDatadogTraceContextForTraceMetadataNonProxyEvent(t *testing.T) {
	ev := loadRawJSON(t, "../testdata/apig-event-with-headers.json")

	headers, ok := getDatadogTraceContextFromEvent(*ev)
	assert.True(t, ok)

	expected := TraceContext{
		traceIDHeader:          "1231452342",
		parentIDHeader:         "45678910",
		samplingPriorityHeader: "2",
		sourceType:             "event",
	}
	assert.Equal(t, expected, headers)
}

func TestGetDatadogTraceContextForTraceMetadataWithMixedCaseHeaders(t *testing.T) {
	ev := loadRawJSON(t, "../testdata/non-proxy-with-mixed-case-headers.json")

	headers, ok := getDatadogTraceContextFromEvent(*ev)
	assert.True(t, ok)

	expected := TraceContext{
		traceIDHeader:          "1231452342",
		parentIDHeader:         "45678910",
		samplingPriorityHeader: "2",
		sourceType:             "event",
	}
	assert.Equal(t, expected, headers)
}

func TestGetDatadogTraceContextForInvalidData(t *testing.T) {
	ev := loadRawJSON(t, "../testdata/invalid.json")

	_, ok := getDatadogTraceContextFromEvent(*ev)
	assert.False(t, ok)
}

func TestGetDatadogTraceContextForMissingData(t *testing.T) {
	ev := loadRawJSON(t, "../testdata/non-proxy-no-headers.json")

	_, ok := getDatadogTraceContextFromEvent(*ev)
	assert.False(t, ok)
}

func TestConvertXRayTraceID(t *testing.T) {
	output, err := convertXRayTraceIDToDatadogTraceID(mockXRayTraceID)
	assert.NoError(t, err)
	assert.Equal(t, convertedXRayTraceID, output)
}

func TestConvertXRayTraceIDTooShort(t *testing.T) {
	output, err := convertXRayTraceIDToDatadogTraceID("1-5ce31dc2-5e03875")
	assert.Error(t, err)
	assert.Equal(t, "0", output)
}

func TestConvertXRayTraceIDInvalidFormat(t *testing.T) {
	output, err := convertXRayTraceIDToDatadogTraceID("1-2c779014b90ce44db5e03875")
	assert.Error(t, err)
	assert.Equal(t, "0", output)
}
func TestConvertXRayTraceIDIncorrectCharacters(t *testing.T) {
	output, err := convertXRayTraceIDToDatadogTraceID("1-5ce31dc2-c779014b90ce44db5e03875;")
	assert.Error(t, err)
	assert.Equal(t, "0", output)
}

func TestConvertXRayEntityID(t *testing.T) {
	output, err := convertXRayEntityIDToDatadogParentID(mockXRayEntityID)
	assert.NoError(t, err)
	assert.Equal(t, convertedXRayEntityID, output)
}

func TestConvertXRayEntityIDInvalidFormat(t *testing.T) {
	output, err := convertXRayEntityIDToDatadogParentID(";b11cc4230d3e09e")
	assert.Error(t, err)
	assert.Equal(t, "0", output)
}

func TestConvertXRayEntityIDTooShort(t *testing.T) {
	output, err := convertXRayEntityIDToDatadogParentID("c4230d3e09e")
	assert.Error(t, err)
	assert.Equal(t, "0", output)
}

func TestXrayTraceContextNoSegment(t *testing.T) {
	ctx := context.Background()

	_, err := getAndConvertXRayTraceContext(ctx)
	assert.Error(t, err)
}
func TestXrayTraceContextWithSegment(t *testing.T) {

	ctx := mockLambdaXRayTraceContext(context.Background(), mockXRayTraceID, mockXRayEntityID, true)

	headers, err := getAndConvertXRayTraceContext(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "2", headers[samplingPriorityHeader])
	assert.NotNil(t, headers[traceIDHeader])
	assert.NotNil(t, headers[parentIDHeader])
}

func TestContextWithTraceContextNoDatadogContext(t *testing.T) {
	// If there is no Datadog trace context, use the converted X-Ray trace context
	ev := loadRawJSON(t, "../testdata/apig-event-no-headers.json")
	ctx := mockLambdaXRayTraceContext(context.Background(), mockXRayTraceID, mockXRayEntityID, true)

	newCTX, _ := ContextWithTraceContext(ctx, *ev, true)
	traceContext, _ := newCTX.Value(traceContextKey).(TraceContext)

	expected := TraceContext{
		traceIDHeader:          convertedXRayTraceID,
		parentIDHeader:         convertedXRayEntityID,
		samplingPriorityHeader: "2",
		sourceType:             fromXray,
	}
	assert.Equal(t, expected, traceContext)
}

func TestContextWithTraceContextDDTraceDisabled(t *testing.T) {
	// If Datadog tracing is disabled, use the converted X-Ray trace context
	ev := loadRawJSON(t, "../testdata/apig-event-with-headers.json")
	ctx := mockLambdaXRayTraceContext(context.Background(), mockXRayTraceID, mockXRayEntityID, true)

	newCTX, _ := ContextWithTraceContext(ctx, *ev, false)
	traceContext, _ := newCTX.Value(traceContextKey).(TraceContext)

	expected := TraceContext{
		traceIDHeader:          convertedXRayTraceID,
		parentIDHeader:         convertedXRayEntityID,
		samplingPriorityHeader: "2",
		sourceType:             fromXray,
	}
	assert.Equal(t, expected, traceContext)
}
func TestContextWithTraceContextDDTraceEnabled(t *testing.T) {
	// If Datadog tracing is enabled and there is Datadog trace context, use it over the X-Ray trace context
	ev := loadRawJSON(t, "../testdata/apig-event-with-headers.json")
	ctx := mockLambdaXRayTraceContext(context.Background(), mockXRayTraceID, mockXRayEntityID, true)

	newCTX, _ := ContextWithTraceContext(ctx, *ev, true)
	traceContext, _ := newCTX.Value(traceContextKey).(TraceContext)

	expected := TraceContext{
		traceIDHeader:          "1231452342",
		parentIDHeader:         "45678910",
		samplingPriorityHeader: "2",
		sourceType:             fromEvent,
	}
	assert.Equal(t, expected, traceContext)
}

func TestContextWithTraceContextFail(t *testing.T) {
	ev := loadRawJSON(t, "../testdata/apig-event-no-headers.json")
	ctx := context.Background()

	_, err := ContextWithTraceContext(ctx, *ev, true)
	assert.Error(t, err)
}
