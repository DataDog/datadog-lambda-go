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
