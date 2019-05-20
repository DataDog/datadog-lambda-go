package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshalEventForTraceMetadataNonProxyEvent(t *testing.T) {
	ev := loadRawJSON(t, "testdata/apig-event-metadata.json")

	headers, ok := unmarshalEventForTraceMetadata(*ev)
	assert.True(t, ok)

	expected := map[string]string{
		traceIDHeader:          "1231452342",
		parentIDHeader:         "45678910",
		samplingPriorityHeader: "12",
	}
	assert.Equal(t, expected, headers)
}

func TestUnmarshalEventForInvalidData(t *testing.T) {
	ev := loadRawJSON(t, "testdata/invalid.json")

	_, ok := unmarshalEventForTraceMetadata(*ev)
	assert.False(t, ok)
}

func TestUnmarshalEventForMissingData(t *testing.T) {
	ev := loadRawJSON(t, "testdata/non-proxy-no-metadata.json")

	_, ok := unmarshalEventForTraceMetadata(*ev)
	assert.False(t, ok)
}
