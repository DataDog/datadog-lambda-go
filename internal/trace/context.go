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

	// TraceContext is map of headers containing a Datadog trace context
	TraceContext map[string]string
)

type contextKeytype int

// traceContextKey is the key used to store a TraceContext in a Context object
var traceContextKey = new(contextKeytype)

// AddRootTraceContextToContext uses the incoming event and/or context object payloads to determine
// the root TraceContext and then adds that TraceContext to the context object.
func AddRootTraceContextToContext(ctx context.Context, ev json.RawMessage, isDDTraceEnabled bool) (context.Context, error) {

	traceCtx, ok := getDatadogTraceContextFromEvent(ctx, ev)

	if ok && isDDTraceEnabled {
		return context.WithValue(ctx, traceContextKey, traceCtx), nil
	}

	// If there is no Datadog trace context, or Datadog tracing is disabled, use the converted X-Ray trace context instead
	traceCtx, err := getAndConvertXRayTraceContext(ctx)
	if err != nil {
		return ctx, fmt.Errorf("couldn't convert X-Ray trace context: %v", err)
	}
	return context.WithValue(ctx, traceContextKey, traceCtx), nil

}

// GetCurrentTraceContext extracts the current Datadog trace context from the ctx object,
// taking into account both the current Datadog span and X-Ray subsegment
func GetCurrentTraceContext(ctx context.Context) TraceContext {
	// Get the current X-Ray trace context (including subsegment)
	// If there is X-Ray trace context but no Datadog trace context
	//// use X-Ray trace context
	// If there is both X-Ray trace context and Datadog trace context
	//// Use the Datadog context, but set the parent ID to that of the X-Ray context
	// Return the context as a traceContext
}

// GetTraceHeaders retrieves the current trace headers that should be added to outbound requests
// TODO: Needs to be replaced!
func GetTraceHeaders(ctx context.Context) TraceContext {
	if traceCtx, ok := ctx.Value(traceContextKey).(TraceContext); ok {
		parentID := traceCtx[parentIDHeader]

		segment := xray.GetSegment(ctx)
		if segment != nil {
			newParentID, err := convertXRayEntityIDToDatadogParentID(segment.ID)
			if err == nil {
				parentID = newParentID
			}
		}

		newTraceContext := map[string]string{}
		newTraceContext[traceIDHeader] = traceCtx[traceIDHeader]
		newTraceContext[samplingPriorityHeader] = traceCtx[samplingPriorityHeader]
		newTraceContext[parentIDHeader] = parentID

		return newTraceContext
	}
	return map[string]string{}
}

// createDummySubsegmentForXrayConverter creates a dummy X-Ray subsegment containing Datadog trace context metadata.
// This metadata is used by the Datadog X-Ray converter to parent the X-Ray trace under the Datadog trace.
// This subsegment will be dropped by the X-Ray converter and will not appear in Datadog.
func createDummySubsegmentForXrayConverter(ctx context.Context, traceCtx TraceContext) error {
	_, segment := xray.BeginSubsegment(ctx, xraySubsegmentName)

	traceID := traceCtx[traceIDHeader]
	parentID := traceCtx[parentIDHeader]
	sampled := traceCtx[samplingPriorityHeader]
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

// getDatadogTraceContextFromEvent extracts the Datadog trace context from an incoming Lambda event payload
// and creates a dummy X-Ray subsegment containing this information
func getDatadogTraceContextFromEvent(ctx context.Context, ev json.RawMessage) (TraceContext, bool) {
	eh := eventWithHeaders{}

	traceCtx := map[string]string{}

	err := json.Unmarshal(ev, &eh)
	if err != nil {
		return traceCtx, false
	}

	lowercaseHeaders := map[string]string{}
	for k, v := range eh.Headers {
		lowercaseHeaders[strings.ToLower(k)] = v
	}

	traceID, ok := lowercaseHeaders[traceIDHeader]
	if !ok {
		return traceCtx, false
	}

	parentID, ok := lowercaseHeaders[parentIDHeader]
	if !ok {
		return traceCtx, false
	}

	samplingPriority, ok := lowercaseHeaders[samplingPriorityHeader]
	if !ok {
		return traceCtx, false
	}

	traceCtx[samplingPriorityHeader] = samplingPriority
	traceCtx[traceIDHeader] = traceID
	traceCtx[parentIDHeader] = parentID
	traceCtx[sourceType] = fromEvent

	createDummySubsegmentForXrayConverter(ctx, traceCtx)

	return traceCtx, true
}

func getAndConvertXRayTraceContext(ctx context.Context) (TraceContext, error) {
	traceCtx := map[string]string{}

	header := getXrayTraceHeaderFromContext(ctx)
	if header == nil {
		return traceCtx, fmt.Errorf("xray segment doesn't exist, couldn't read trace context")
	}

	traceID, err := convertXRayTraceIDToDatadogTraceID(header.TraceID)
	if err != nil {
		return traceCtx, fmt.Errorf("couldn't read trace id from xray: %v", err)
	}
	parentID, err := convertXRayEntityIDToDatadogParentID(header.ParentID)
	if err != nil {
		return traceCtx, fmt.Errorf("couldn't read parent id from xray: %v", err)
	}
	samplingPriority := convertXRaySamplingDecision(header.SamplingDecision)

	traceCtx[traceIDHeader] = traceID
	traceCtx[parentIDHeader] = parentID
	traceCtx[samplingPriorityHeader] = samplingPriority
	traceCtx[sourceType] = fromXray
	return traceCtx, nil
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
func convertXRayTraceIDToDatadogTraceID(traceID string) (string, error) {
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
func convertXRayEntityIDToDatadogParentID(entityID string) (string, error) {
	if len(entityID) < 16 {
		return "0", fmt.Errorf("couldn't convert to trace id, too short")
	}
	val, err := convertHexIDToUint64(entityID[len(entityID)-16:])
	if err != nil {
		return "0", fmt.Errorf("couldn't convert entity id to trace id:  %v", err)
	}
	return strconv.FormatUint(val, 10), nil
}

// Converts an X-Ray sampled flag into its Datadog counterpart.
func convertXRaySamplingDecision(decision header.SamplingDecision) string {
	if decision == header.Sampled {
		return userKeep
	}
	return userReject
}
