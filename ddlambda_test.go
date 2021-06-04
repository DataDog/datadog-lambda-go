/*
 * Unless explicitly stated otherwise all files in this repository are licensed
 * under the Apache License Version 2.0.
 *
 * This product includes software developed at Datadog (https://www.datadoghq.com/).
 * Copyright 2021 Datadog, Inc.
 */
package ddlambda

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInvokeDryRun(t *testing.T) {
	called := false
	InvokeDryRun(func(ctx context.Context) {
		called = true
		globalCtx := GetContext()
		assert.Equal(t, globalCtx, ctx)
	}, nil)
	assert.True(t, called)
}

func TestMetricsSilentFailWithoutWrapper(t *testing.T) {
	Metric("my-metric", 100, "my:tag")
}

func TestMetricsSubmitWithWrapper(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	InvokeDryRun(func(ctx context.Context) {
		Metric("my-metric", 100, "my:tag")
	}, &Config{
		APIKey: "abc-123",
		Site:   server.URL,
	})
	assert.True(t, called)
}
