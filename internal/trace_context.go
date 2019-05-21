package internal

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-xray-sdk-go/xray"
)

type (
	eventWithHeaders struct {
		Headers map[string]string `json:"headers"`
	}
)

func unmarshalEventForTraceContext(ev json.RawMessage) (map[string]string, bool) {
	eh := eventWithHeaders{}

	traceContext := map[string]string{}

	err := json.Unmarshal(ev, &eh)
	if err != nil {
		return traceContext, false
	}

	traceID, ok := eh.Headers[traceIDHeader]
	if !ok {
		return traceContext, false
	}

	parentID, ok := eh.Headers[parentIDHeader]
	if !ok {
		return traceContext, false
	}

	samplingPriority, ok := eh.Headers[samplingPriorityHeader]
	if !ok {
		return traceContext, false
	}

	traceContext[samplingPriorityHeader] = samplingPriority
	traceContext[traceIDHeader] = traceID
	traceContext[parentIDHeader] = parentID
	return traceContext, true
}

func readXRayTraceContext(ctx context.Context) (map[string]string, error) {
	traceContext := map[string]string{}

	segment := xray.GetSegment(ctx)
	if segment == nil {
		return traceContext, fmt.Errorf("xray segment doesn't exist, couldn't read trace context")
	}

	traceID, err := convertXRayTraceIDToAPMTraceID(segment.TraceID)
	if err != nil {
		return traceContext, fmt.Errorf("couldn't read trace id from xray: %v", err)
	}
	parentID, err := convertXRayEntityIDToAPMParentID(segment.ID)
	if err != nil {
		return traceContext, fmt.Errorf("couldn't read parent id from xray: %v", err)
	}
	samplingPriority := convertXRaySampling(segment.Sampled)

	traceContext[traceIDHeader] = traceID
	traceContext[parentIDHeader] = parentID
	traceContext[samplingPriorityHeader] = samplingPriority
	return traceContext, nil
}

// Converts the last 63 bits of an X-Ray trace ID (hex) to a Datadog trace id (uint64).
func convertXRayTraceIDToAPMTraceID(traceID string) (string, error) {
	parts := strings.Split(traceID, "-")

	if len(parts) != 3 {
		return "0", fmt.Errorf("invalid x-ray trace id; expected 3 components in id")
	}
	if len(parts[2]) != 24 {
		return "0", fmt.Errorf("x-ray trace id should be 96 bits")
	}

	traceIDLength := len(parts[2]) - 16
	traceID = parts[2][traceIDLength : traceIDLength+16] // Per XRay Team: use the last 64 bits of the trace id
	apmTraceID, err := convertHexIDToUint64(traceID)
	if err != nil {
		return "0", fmt.Errorf("while converting xray trace id: %v", err)
	}
	apmTraceID = 0x7FFFFFFFFFFFFFFF & apmTraceID // The APM Trace ID is restricted to 63 bits, so make sure the 64th bit is always 0
	return strconv.FormatUint(apmTraceID, 10), nil
}

func convertHexIDToUint64(hexNumber string) (uint64, error) {
	ba, err := hex.DecodeString(hexNumber)
	if err != nil {
		return 0, fmt.Errorf("couldn't convert hex to uint64: %v", err)
	}

	var id uint64
	id = binary.BigEndian.Uint64(ba) // TODO: Verify that this is correct

	return id, nil
}

// Converts an X-Ray entity ID (hex) to a Datadog parent id (uint64).
func convertXRayEntityIDToAPMParentID(entityID string) (string, error) {
	val, err := convertHexIDToUint64(entityID)
	if err != nil {
		return "0", fmt.Errorf("couldn't convert entity id to trace id:  %v", err)
	}
	return strconv.FormatUint(val, 10), nil
}

// Converts an X-Ray sampled flag into it's Datadog counterpart.
func convertXRaySampling(sampled bool) string {
	if sampled {
		return userKeep
	}
	return userReject
}
