/*
 * Unless explicitly stated otherwise all files in this repository are licensed
 * under the Apache License Version 2.0.
 * 
 * This product includes software developed at Datadog (https://www.datadoghq.com/).
 * Copyright 2019 Datadog, Inc.
 */

package metrics

import "context"

type contextKeytype int

var traceContextKey = new(contextKeytype)

// GetProcessor retrieves the processor from a context object.
func GetProcessor(ctx context.Context) Processor {
	result := ctx.Value(traceContextKey)
	if result == nil {
		return nil
	}
	return result.(Processor)
}

// AddProcessor adds a processor to a context object
func AddProcessor(ctx context.Context, processor Processor) context.Context {
	return context.WithValue(ctx, traceContextKey, processor)
}
