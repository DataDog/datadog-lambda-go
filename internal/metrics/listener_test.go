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
