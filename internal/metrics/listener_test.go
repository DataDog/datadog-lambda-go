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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandlerAddsItselfToContext(t *testing.T) {
	listener := MakeListener(Config{})
	ctx := listener.HandlerStarted(context.Background(), json.RawMessage{})
	pr := GetListener(ctx)
	assert.NotNil(t, pr)
}

func TestHandlerFinishesProcessing(t *testing.T) {
	listener := MakeListener(Config{})
	ctx := listener.HandlerStarted(context.Background(), json.RawMessage{})

	listener.HandlerFinished(ctx)
	assert.False(t, listener.processor.IsProcessing())
}

func TestAddDistributionMetricWithAPI(t *testing.T) {

	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/distribution_points?api_key=12345", r.URL.String())
		called = true
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	listener := MakeListener(Config{APIKey: "12345", Site: server.URL})
	ctx := listener.HandlerStarted(context.Background(), json.RawMessage{})
	listener.AddDistributionMetric("the-metric", 2, "tag:a", "tag:b")
	listener.HandlerFinished(ctx)
	assert.True(t, called)
}

func TestAddDistributionMetricWithLogForwarder(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	listener := MakeListener(Config{APIKey: "12345", Site: server.URL, ShouldUseLogForwarder: true})
	ctx := listener.HandlerStarted(context.Background(), json.RawMessage{})
	listener.AddDistributionMetric("the-metric", 2, "tag:a", "tag:b")
	listener.HandlerFinished(ctx)
	assert.False(t, called)
}
