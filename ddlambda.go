package ddlambda

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/DataDog/dd-lambda-go/internal/trace"
)

type (
	handlerListener struct{}
)

// WrapHandler is used to instrument your lambda functions, reading in context from API Gateway.
// It returns a modified handler that can be passed directly to the lambda.Start function.
func WrapHandler(handler interface{}) interface{} {
	hl := handlerListener{}
	return trace.WrapHandlerWithListener(handler, &hl)
}

// GetTraceHeaders reads a map containing the DataDog trace headers from a context object.
func GetTraceHeaders(ctx context.Context) map[string]string {
	result := trace.GetTraceHeaders(ctx, true)
	return result
}

// AddTraceHeaders adds DataDog trace headers to a HTTP Request
func AddTraceHeaders(req *http.Request) {
	headers := GetTraceHeaders(req.Context())
	for key, value := range headers {
		req.Header.Add(key, value)
	}
}

func (hl *handlerListener) HandlerStarted(ctx context.Context, msg json.RawMessage) context.Context {
	ctx, _ = trace.ExtractTraceContext(ctx, msg)
	return ctx
}

func (hl *handlerListener) HandlerFinished(ctx context.Context) {
}
