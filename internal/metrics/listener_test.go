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
	"github.com/aws/aws-lambda-go/lambdacontext"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
	listener.AddDistributionMetric("the-metric", 2, time.Now(), "tag:a", "tag:b")
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
	listener.AddDistributionMetric("the-metric", 2, time.Now(), "tag:a", "tag:b")
	listener.HandlerFinished(ctx)
	assert.False(t, called)
}

func TestGetEnhancedMetricsTags(t *testing.T) {
	ctx := context.WithValue(context.Background(), "cold_start", false)

	lambdacontext.MemoryLimitInMB = 256
	lambdacontext.FunctionName = "go-lambda-test"
	lc := &lambdacontext.LambdaContext{
		InvokedFunctionArn: "arn:aws:lambda:us-east-1:123497558138:function:go-lambda-test",
	}
	tags := getEnhancedMetricsTags(lambdacontext.NewContext(ctx, lc))

	assert.ElementsMatch(t, tags, []string{"functionname:go-lambda-test", "region:us-east-1", "memorysize:256", "cold_start:false", "account_id:123497558138"})
}

func TestGetEnhancedMetricsTagsNoLambdaContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), "cold_start", true)
	tags := getEnhancedMetricsTags(ctx)

	assert.Empty(t, tags)
}

func TestSubmitEnhancedMetrics(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()
	ml := MakeListener(
		Config{
			APIKey:                "abc-123",
			Site:                  server.URL,
			EnhancedMetrics:       true,
		},
	)
	ctx := context.WithValue(context.Background(), "cold_start", false)

	ctx = ml.HandlerStarted(ctx, json.RawMessage{})
	ml.HandlerFinished(ctx)

	assert.True(t, called)
}

func TestDoNotSubmitEnhancedMetrics(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	ml := MakeListener(
		Config{
			APIKey:                "abc-123",
			Site:                  server.URL,
			EnhancedMetrics:       false,
		},
	)
	ctx := context.WithValue(context.Background(), "cold_start", true)

	ctx = ml.HandlerStarted(ctx, json.RawMessage{})
	ml.HandlerFinished(ctx)

	assert.False(t, called)
}

func TestSubmitEnhancedMetricsOnlyErrors(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	ml := MakeListener(
		Config{
			APIKey:                "abc-123",
			Site:                  server.URL,
			EnhancedMetrics:       false,
		},
	)
	ctx := context.WithValue(context.Background(), "cold_start", true)

	ctx = ml.HandlerStarted(ctx, json.RawMessage{})
	ml.config.EnhancedMetrics = true
	ctx = context.WithValue(ctx, "error", true)
	ml.HandlerFinished(ctx)

	assert.True(t, called)
}