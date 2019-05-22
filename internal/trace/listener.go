package trace

import (
	"context"
	"encoding/json"
)

type (
	// Listener implements HandlerListener, injecting datadog tracing info into the context
	Listener struct{}
)

// HandlerStarted adds trace metadata to the context
func (l *Listener) HandlerStarted(ctx context.Context, msg json.RawMessage) context.Context {
	ctx, _ = ExtractTraceContext(ctx, msg)
	return ctx
}

// HandlerFinished implemented as part of the HandlerListener interface
func (l *Listener) HandlerFinished(ctx context.Context) {
}
