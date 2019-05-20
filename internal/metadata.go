package internal

import "encoding/json"

type (
	eventWithHeaders struct {
		Headers map[string]string `json:"headers"`
	}
)

func unmarshalEventForTraceMetadata(ev json.RawMessage) (map[string]string, bool) {
	eh := eventWithHeaders{}

	headers := map[string]string{}

	err := json.Unmarshal(ev, &eh)
	if err != nil {
		return headers, false
	}

	traceID, ok := eh.Headers[traceIDHeader]
	if !ok {
		return headers, false
	}

	parentID, ok := eh.Headers[parentIDHeader]
	if !ok {
		return headers, false
	}

	samplingPriority, ok := eh.Headers[samplingPriorityHeader]
	if !ok {
		return headers, false
	}

	headers[samplingPriorityHeader] = samplingPriority
	headers[traceIDHeader] = traceID
	headers[parentIDHeader] = parentID
	return headers, true
}
