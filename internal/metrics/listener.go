package metrics

import (
	"context"
	"encoding/json"
)

type (
	// Listener implements wrapper.HandlerListener, injecting metrics into the context
	Listener struct{}
)

// HandlerStarted adds metrics service to the context
func (l *Listener) HandlerStarted(ctx context.Context, msg json.RawMessage) context.Context {
	return ctx
}

// HandlerFinished implemented as part of the wrapper.HandlerListener interface
func (l *Listener) HandlerFinished(ctx context.Context) {
}
