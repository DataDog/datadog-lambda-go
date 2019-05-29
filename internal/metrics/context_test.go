package metrics

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetProcessorEmptyContext(t *testing.T) {
	ctx := context.Background()
	result := GetProcessor(ctx)
	assert.Nil(t, result)
}

func TestGetProcessorSuccess(t *testing.T) {
	ctx := AddProcessor(context.Background(), MakeProcessor(context.Background(), nil, nil, 0, false))
	result := GetProcessor(ctx)
	assert.NotNil(t, result)
}
