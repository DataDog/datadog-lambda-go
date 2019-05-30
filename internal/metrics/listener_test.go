/*
 * Unless explicitly stated otherwise all files in this repository are licensed
 * under the Apache License Version 2.0.
 * 
 * This product includes software developed at Datadog (https://www.datadoghq.com/).
 * Copyright 2019 Datadog, Inc.
 */

package metrics

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandlerAddsProcessorToContext(t *testing.T) {
	listener := MakeListener(Config{})
	ctx := listener.HandlerStarted(context.Background(), json.RawMessage{})
	pr := GetProcessor(ctx)
	assert.NotNil(t, pr)
}

func TestHandlerFinishesProcessing(t *testing.T) {
	listener := MakeListener(Config{})
	ctx := listener.HandlerStarted(context.Background(), json.RawMessage{})

	pr := GetProcessor(ctx).(*processor)
	listener.HandlerFinished(ctx)
	assert.False(t, pr.isProcessing)
}
