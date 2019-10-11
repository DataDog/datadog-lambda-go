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
	"encoding/json"
)

type (
	// Listener implements HandlerListener, injecting datadog tracing info into the context
	Listener struct {
		trace bool
	}
)

func NewListener(trace bool) *Listener {
	return &Listener{
		trace: trace,
	}
}

func (l *Listener) Trace() bool {
	return l.trace
}

func (l *Listener) Name() string {
	return "Datadog-Tracer"
}

// HandlerStarted adds trace metadata to the context
func (l *Listener) HandlerStarted(ctx context.Context, msg json.RawMessage) context.Context {
	ctx, _ = ExtractTraceContext(ctx, msg)
	return ctx
}

// HandlerFinished implemented as part of the HandlerListener interface
func (l *Listener) HandlerFinished(ctx context.Context) {
}
