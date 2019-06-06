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
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-xray-sdk-go/header"
	"github.com/aws/aws-xray-sdk-go/xray"
)

type (
	eventWithHeaders struct {
		Headers map[string]string `json:"headers"`
	}
)

type contextKeytype int

var traceContextKey = new(contextKeytype)

// ExtractTraceContext returns a list of headers with the current
func ExtractTraceContext(ctx context.Context, ev json.RawMessage) (context.Context, error) {

	// First priority is always any trace context from incoming headers
	traceContext, ok := unmarshalEventForTraceContext(ev)
	if ok {
		// If we detect the trace headers, we should save metadata to xray so it can be read by the converter.
		err := addTraceContextToXRay(ctx, traceContext)
		if err != nil {
			return ctx, err
		}
		// Set parent ID to the functions parent.
		xrayTraceContext, err := convertTraceContextFromXRay(ctx)
		if err == nil {
			traceContext[parentIDHeader] = xrayTraceContext[parentIDHeader]
		}

		return context.WithValue(ctx, traceContextKey, traceContext), nil
	}

	// Second priority is any trace
	traceContext, err := convertTraceContextFromXRay(ctx)
	if err != nil {
		return ctx, fmt.Errorf("couldn't convert trace context: %v", err)
	}

	return context.WithValue(ctx, traceContextKey, traceContext), nil
}

// GetTraceHeaders retrieves the current trace headers that should be added to outbound requests
func GetTraceHeaders(ctx context.Context, useCurrentSegmentAsParent bool) map[string]string {
	if traceContext, ok := ctx.Value(traceContextKey).(map[string]string); ok {
		// Change the trace context to include the current segment/subsegment id as the parent ID.
		parentID := traceContext[parentIDHeader]

		if useCurrentSegmentAsParent {
			segment := xray.GetSegment(ctx)
			if segment != nil {
				println("Xray current segment is set")
				newParentID, err := convertXRayEntityIDToAPMParentID(segment.ID)
				if err == nil {
					parentID = newParentID
				}
			} else {
				println("Xray current segment is empty, defaulting to parent segment")
			}
		}

		newTraceContext := map[string]string{}
		newTraceContext[traceIDHeader] = traceContext[traceIDHeader]
		newTraceContext[samplingPriorityHeader] = traceContext[samplingPriorityHeader]
		newTraceContext[parentIDHeader] = parentID

		return newTraceContext
	}
	return map[string]string{}
}

func addTraceContextToXRay(ctx context.Context, traceContext map[string]string) error {
	_, segment := xray.BeginSubsegment(ctx, xraySubsegmentName)

	traceID := traceContext[traceIDHeader]
	parentID := traceContext[parentIDHeader]
	sampled := traceContext[samplingPriorityHeader]
	metadata := map[string]string{
		"trace-id":          traceID,
		"parent-id":         parentID,
		"sampling-priority": sampled,
	}

	err := segment.AddMetadataToNamespace(xraySubsegmentNamespace, xraySubsegmentKey, metadata)
	if err != nil {
		return fmt.Errorf("couldn't save trace context to XRay: %v", err)
	}
	segment.Close(nil)
	return nil
}

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

func convertTraceContextFromXRay(ctx context.Context) (map[string]string, error) {
	traceContext := map[string]string{}

	header := getXrayTraceHeaderFromContext(ctx)
	if header == nil {
		return traceContext, fmt.Errorf("xray segment doesn't exist, couldn't read trace context")
	}

	traceID, err := convertXRayTraceIDToAPMTraceID(header.TraceID)
	if err != nil {
		return traceContext, fmt.Errorf("couldn't read trace id from xray: %v", err)
	}
	parentID, err := convertXRayEntityIDToAPMParentID(header.ParentID)
	if err != nil {
		return traceContext, fmt.Errorf("couldn't read parent id from xray: %v", err)
	}
	samplingPriority := convertXRaySamplingDecision(header.SamplingDecision)

	traceContext[traceIDHeader] = traceID
	traceContext[parentIDHeader] = parentID
	traceContext[samplingPriorityHeader] = samplingPriority
	return traceContext, nil
}

// getXrayTraceHeaderFromContext is used to extract xray segment metadata from the lambda context object.
// By default, the context object won't have any Segment, (xray.GetSegment(ctx) will return nil). However it
// will have the "LambdaTraceHeader" object, which contains the traceID/parentID/sampling info.
func getXrayTraceHeaderFromContext(ctx context.Context) *header.Header {
	var traceHeader string

	if traceHeaderValue := ctx.Value(xray.LambdaTraceHeaderKey); traceHeaderValue != nil {
		traceHeader = traceHeaderValue.(string)
		return header.FromString(traceHeader)
	}
	return nil
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
	if len(entityID) < 16 {
		return "0", fmt.Errorf("couldn't convert to trace id, too short")
	}
	val, err := convertHexIDToUint64(entityID[len(entityID)-16:])
	if err != nil {
		return "0", fmt.Errorf("couldn't convert entity id to trace id:  %v", err)
	}
	return strconv.FormatUint(val, 10), nil
}

// Converts an X-Ray sampled flag into it's Datadog counterpart.
func convertXRaySamplingDecision(decision header.SamplingDecision) string {
	if decision == header.Sampled {
		return userKeep
	}
	return userReject
}
