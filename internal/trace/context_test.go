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

func mockLambdaTraceContext(ctx context.Context, traceID, parentID string, sampled bool) context.Context {
	decision := header.NotSampled
	if sampled {
		decision = header.Sampled
	}

	traceHeader := header.Header{
		TraceID:          traceID,
		ParentID:         parentID,
		SamplingDecision: decision,
		AdditionalData:   make(map[string]string),
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
func TestUnmarshalEventForTraceMetadataNonProxyEvent(t *testing.T) {
	ev := loadRawJSON(t, "../testdata/apig-event-metadata.json")

	headers, ok := unmarshalEventForTraceContext(*ev)
	assert.True(t, ok)

	expected := map[string]string{
		traceIDHeader:          "1231452342",
		parentIDHeader:         "45678910",
		samplingPriorityHeader: "2",
		sourceType:             "event",
	}
	assert.Equal(t, expected, headers)
}

func TestUnmarshalEventForTraceMetadataWithMixedCaseHeaders(t *testing.T) {
	ev := loadRawJSON(t, "../testdata/non-proxy-mixed-case-metadata.json")

	headers, ok := unmarshalEventForTraceContext(*ev)
	assert.True(t, ok)

	expected := map[string]string{
		traceIDHeader:          "1231452342",
		parentIDHeader:         "45678910",
		samplingPriorityHeader: "2",
		sourceType:             "event",
	}
	assert.Equal(t, expected, headers)
}

func TestUnmarshalEventForInvalidData(t *testing.T) {
	ev := loadRawJSON(t, "../testdata/invalid.json")

	_, ok := unmarshalEventForTraceContext(*ev)
	assert.False(t, ok)
}

func TestUnmarshalEventForMissingData(t *testing.T) {
	ev := loadRawJSON(t, "../testdata/non-proxy-no-metadata.json")

	_, ok := unmarshalEventForTraceContext(*ev)
	assert.False(t, ok)
}

func TestConvertXRayTraceID(t *testing.T) {
	output, err := convertXRayTraceIDToAPMTraceID("1-5ce31dc2-2c779014b90ce44db5e03875")
	assert.NoError(t, err)
	assert.Equal(t, "4110911582297405557", output)
}

func TestConvertXRayTraceIDTooShort(t *testing.T) {
	output, err := convertXRayTraceIDToAPMTraceID("1-5ce31dc2-5e03875")
	assert.Error(t, err)
	assert.Equal(t, "0", output)
}

func TestConvertXRayTraceIDInvalidFormat(t *testing.T) {
	output, err := convertXRayTraceIDToAPMTraceID("1-2c779014b90ce44db5e03875")
	assert.Error(t, err)
	assert.Equal(t, "0", output)
}
func TestConvertXRayTraceIDIncorrectCharacters(t *testing.T) {
	output, err := convertXRayTraceIDToAPMTraceID("1-5ce31dc2-c779014b90ce44db5e03875;")
	assert.Error(t, err)
	assert.Equal(t, "0", output)
}

func TestConvertXRayEntityID(t *testing.T) {
	output, err := convertXRayEntityIDToAPMParentID("0b11cc4230d3e09e")
	assert.NoError(t, err)
	assert.Equal(t, "797643193680388254", output)
}

func TestConvertXRayEntityIDInvalidFormat(t *testing.T) {
	output, err := convertXRayEntityIDToAPMParentID(";b11cc4230d3e09e")
	assert.Error(t, err)
	assert.Equal(t, "0", output)
}

func TestConvertXRayEntityIDTooShort(t *testing.T) {
	output, err := convertXRayEntityIDToAPMParentID("c4230d3e09e")
	assert.Error(t, err)
	assert.Equal(t, "0", output)
}

func TestXrayTraceContextNoSegment(t *testing.T) {
	ctx := context.Background()

	_, err := convertTraceContextFromXRay(ctx)
	assert.Error(t, err)
}
func TestXrayTraceContextWithSegment(t *testing.T) {

	ctx := mockLambdaTraceContext(context.Background(), "1-5ce31dc2-2c779014b90ce44db5e03875", "779014b90ce44db5e03875", true)

	headers, err := convertTraceContextFromXRay(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "2", headers[samplingPriorityHeader])
	assert.NotNil(t, headers[traceIDHeader])
	assert.NotNil(t, headers[parentIDHeader])
}

func TestContextWithTraceContextFromContext(t *testing.T) {
	ev := loadRawJSON(t, "../testdata/apig-event-no-metadata.json")
	ctx := mockLambdaTraceContext(context.Background(), "1-5ce31dc2-2c779014b90ce44db5e03875", "779014b90ce44db5e03875", true)

	newCTX, err := ContextWithTraceContext(ctx, *ev)
	headers := GetTraceHeaders(newCTX, false)

	assert.NoError(t, err)
	assert.Equal(t, "2", headers[samplingPriorityHeader])
	assert.NotNil(t, headers[traceIDHeader])
	assert.NotNil(t, headers[parentIDHeader])
}
func TestContextWithTraceContextFromEvent(t *testing.T) {
	ev := loadRawJSON(t, "../testdata/apig-event-metadata.json")
	ctx := mockLambdaTraceContext(context.Background(), "1-5ce31dc2-2c779014b90ce44db5e03875", "779014b90ce44db5e03875", true)

	newCTX, err := ContextWithTraceContext(ctx, *ev)
	headers := GetTraceHeaders(newCTX, false)
	assert.NoError(t, err)

	expected := map[string]string{
		traceIDHeader:          "1231452342",
		parentIDHeader:         "13334283619152181365", // Use the parentID from the context, not the event
		samplingPriorityHeader: "2",
	}
	assert.Equal(t, expected, headers)
}

func TestContextWithTraceContextFail(t *testing.T) {
	ev := loadRawJSON(t, "../testdata/apig-event-no-metadata.json")
	ctx := context.Background()

	_, err := ContextWithTraceContext(ctx, *ev)
	assert.Error(t, err)
}

func TestGetTraceHeadersWithUpdatedParent(t *testing.T) {
	ev := loadRawJSON(t, "../testdata/apig-event-metadata.json")
	ctx := mockLambdaTraceContext(context.Background(), "1-5ce31dc2-2c779014b90ce44db5e03875", "779014b90ce44db5e03874", true)

	ctx, _ = ContextWithTraceContext(ctx, *ev)

	ctx, _ = xray.BeginSubsegment(ctx, "The Subsegment")

	headers := GetTraceHeaders(ctx, true)
	assert.Equal(t, "2", headers[samplingPriorityHeader])
	assert.Equal(t, "1231452342", headers[traceIDHeader])
	assert.NotEqual(t, "45678910", headers[parentIDHeader]) // This has changed
}
