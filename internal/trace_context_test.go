package internal

import (
	"context"
	"testing"

	"github.com/aws/aws-xray-sdk-go/xray"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshalEventForTraceMetadataNonProxyEvent(t *testing.T) {
	ev := loadRawJSON(t, "testdata/apig-event-metadata.json")

	headers, ok := unmarshalEventForTraceContext(*ev)
	assert.True(t, ok)

	expected := map[string]string{
		traceIDHeader:          "1231452342",
		parentIDHeader:         "45678910",
		samplingPriorityHeader: "2",
	}
	assert.Equal(t, expected, headers)
}

func TestUnmarshalEventForInvalidData(t *testing.T) {
	ev := loadRawJSON(t, "testdata/invalid.json")

	_, ok := unmarshalEventForTraceContext(*ev)
	assert.False(t, ok)
}

func TestUnmarshalEventForMissingData(t *testing.T) {
	ev := loadRawJSON(t, "testdata/non-proxy-no-metadata.json")

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
	output, err := convertXRayEntityIDToAPMParentID("779014b90ce44db5e03875")
	assert.NoError(t, err)
	assert.Equal(t, "8615408872177552821", output)
}

func TestConvertXRayEntityIDInvalidFormat(t *testing.T) {
	output, err := convertXRayEntityIDToAPMParentID(";79014b90ce44db5e03875")
	assert.Error(t, err)
	assert.Equal(t, "0", output)
}

func TestXrayTraceContextNoSegment(t *testing.T) {
	ctx := context.Background()

	_, err := convertTraceContextFromXRay(ctx)
	assert.Error(t, err)
}
func TestXrayTraceContextWithSegment(t *testing.T) {
	ctx, _ := xray.BeginSegment(context.Background(), "Test-Segment")

	headers, err := convertTraceContextFromXRay(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "2", headers[samplingPriorityHeader])
	assert.NotNil(t, headers[traceIDHeader])
	assert.NotNil(t, headers[parentIDHeader])
}

func TestExtractTraceContextFromContext(t *testing.T) {
	ev := loadRawJSON(t, "testdata/apig-event-no-metadata.json")
	ctx, _ := xray.BeginSegment(context.Background(), "Test-Segment")

	headers, err := ExtractTraceContext(ctx, *ev)
	assert.NoError(t, err)
	assert.Equal(t, "2", headers[samplingPriorityHeader])
	assert.NotNil(t, headers[traceIDHeader])
	assert.NotNil(t, headers[parentIDHeader])
}
func TestExtractTraceContextFromEvent(t *testing.T) {
	ev := loadRawJSON(t, "testdata/apig-event-metadata.json")
	ctx, _ := xray.BeginSegment(context.Background(), "Test-Segment")

	headers, err := ExtractTraceContext(ctx, *ev)
	assert.NoError(t, err)

	expected := map[string]string{
		traceIDHeader:          "1231452342",
		parentIDHeader:         "45678910",
		samplingPriorityHeader: "2",
	}
	assert.Equal(t, expected, headers)
}

func TestExtractTraceContextFail(t *testing.T) {
	ev := loadRawJSON(t, "testdata/apig-event-no-metadata.json")
	ctx := context.Background()

	_, err := ExtractTraceContext(ctx, *ev)
	assert.Error(t, err)
}
