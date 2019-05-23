package ddlambda

import (
	"context"
	"net/http"

	"github.com/DataDog/dd-lambda-go/internal/metrics"
	"github.com/DataDog/dd-lambda-go/internal/trace"
	"github.com/DataDog/dd-lambda-go/internal/wrapper"
)

// WrapHandler is used to instrument your lambda functions, reading in context from API Gateway.
// It returns a modified handler that can be passed directly to the lambda.Start function.
func WrapHandler(handler interface{}) interface{} {
	tl := trace.Listener{}
	ml := metrics.MakeListener()
	return wrapper.WrapHandlerWithListeners(handler, &tl, &ml)
}

// GetTraceHeaders reads a map containing the DataDog trace headers from a context object.
func GetTraceHeaders(ctx context.Context) map[string]string {
	result := trace.GetTraceHeaders(ctx, true)
	return result
}

// AddTraceHeaders adds DataDog trace headers to a HTTP Request
func AddTraceHeaders(ctx context.Context, req *http.Request) {
	headers := GetTraceHeaders(ctx)
	for key, value := range headers {
		req.Header.Add(key, value)
	}
}
