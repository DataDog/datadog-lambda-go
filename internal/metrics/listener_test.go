/*
 * Unless explicitly stated otherwise all files in this repository are licensed
 * under the Apache License Version 2.0.
 *
 * This product includes software developed at Datadog (https://www.datadoghq.com/).
 * Copyright 2021 Datadog, Inc.
 */

package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DataDog/datadog-lambda-go/internal/extension"
	"github.com/DataDog/datadog-lambda-go/internal/logger"
	"github.com/DataDog/datadog-lambda-go/internal/version"
	"github.com/aws/aws-lambda-go/lambdacontext"

	"github.com/stretchr/testify/assert"
)

func captureOutput(f func()) string {
	var buf bytes.Buffer
	logger.SetOutput(&buf)
	f()
	logger.SetOutput(os.Stderr)
	return buf.String()
}

func TestHandlerAddsItselfToContext(t *testing.T) {
	listener := MakeListener(Config{}, &extension.ExtensionManager{})
	ctx := listener.HandlerStarted(context.Background(), json.RawMessage{})
	pr := GetListener(ctx)
	assert.NotNil(t, pr)
}

func TestHandlerFinishesProcessing(t *testing.T) {
	listener := MakeListener(Config{}, &extension.ExtensionManager{})
	ctx := listener.HandlerStarted(context.Background(), json.RawMessage{})

	listener.HandlerFinished(ctx, nil)
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

	listener := MakeListener(Config{APIKey: "12345", Site: server.URL}, &extension.ExtensionManager{})
	ctx := listener.HandlerStarted(context.Background(), json.RawMessage{})
	listener.AddDistributionMetric("the-metric", 2, time.Now(), false, "tag:a", "tag:b")
	listener.HandlerFinished(ctx, nil)
	assert.True(t, called)
}

func TestAddDistributionMetricWithLogForwarder(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	listener := MakeListener(Config{APIKey: "12345", Site: server.URL, ShouldUseLogForwarder: true}, &extension.ExtensionManager{})
	ctx := listener.HandlerStarted(context.Background(), json.RawMessage{})
	listener.AddDistributionMetric("the-metric", 2, time.Now(), false, "tag:a", "tag:b")
	listener.HandlerFinished(ctx, nil)
	assert.False(t, called)
}
func TestAddDistributionMetricWithForceLogForwarder(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	listener := MakeListener(Config{APIKey: "12345", Site: server.URL, ShouldUseLogForwarder: false}, &extension.ExtensionManager{})
	ctx := listener.HandlerStarted(context.Background(), json.RawMessage{})
	listener.AddDistributionMetric("the-metric", 2, time.Now(), true, "tag:a", "tag:b")
	listener.HandlerFinished(ctx, nil)
	assert.False(t, called)
}

func TestGetEnhancedMetricsTags(t *testing.T) {
	//nolint
	ctx := context.WithValue(context.Background(), "cold_start", false)

	lambdacontext.MemoryLimitInMB = 256
	lambdacontext.FunctionName = "go-lambda-test"
	lc := &lambdacontext.LambdaContext{
		InvokedFunctionArn: "arn:aws:lambda:us-east-1:123497558138:function:go-lambda-test:$Latest",
	}
	tags := getEnhancedMetricsTags(lambdacontext.NewContext(ctx, lc))

	assert.ElementsMatch(t, tags, []string{"functionname:go-lambda-test", "region:us-east-1", "memorysize:256", "cold_start:false", "account_id:123497558138", "resource:go-lambda-test:Latest", "datadog_lambda:v" + version.DDLambdaVersion})
}

func TestGetEnhancedMetricsTagsWithAlias(t *testing.T) {
	//nolint
	ctx := context.WithValue(context.Background(), "cold_start", false)

	lambdacontext.MemoryLimitInMB = 256
	lambdacontext.FunctionName = "go-lambda-test"
	lambdacontext.FunctionVersion = "1"
	lc := &lambdacontext.LambdaContext{
		InvokedFunctionArn: "arn:aws:lambda:us-east-1:123497558138:function:go-lambda-test:my-alias",
	}

	tags := getEnhancedMetricsTags((lambdacontext.NewContext(ctx, lc)))
	assert.ElementsMatch(t, tags, []string{"functionname:go-lambda-test", "region:us-east-1", "memorysize:256", "cold_start:false", "account_id:123497558138", "resource:go-lambda-test:my-alias", "executedversion:1", "datadog_lambda:v" + version.DDLambdaVersion})
}

func TestGetEnhancedMetricsTagsNoLambdaContext(t *testing.T) {
	//nolint
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
			APIKey:          "abc-123",
			Site:            server.URL,
			EnhancedMetrics: true,
		},
		&extension.ExtensionManager{},
	)
	//nolint
	ctx := context.WithValue(context.Background(), "cold_start", false)

	output := captureOutput(func() {
		ctx = ml.HandlerStarted(ctx, json.RawMessage{})
		ml.HandlerFinished(ctx, nil)
	})

	assert.False(t, called)
	expected := "{\"m\":\"aws.lambda.enhanced.invocations\",\"v\":1,"
	assert.True(t, strings.Contains(output, expected))
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
			APIKey:          "abc-123",
			Site:            server.URL,
			EnhancedMetrics: false,
		},
		&extension.ExtensionManager{},
	)
	//nolint
	ctx := context.WithValue(context.Background(), "cold_start", true)

	output := captureOutput(func() {
		ctx = ml.HandlerStarted(ctx, json.RawMessage{})
		ml.HandlerFinished(ctx, nil)
	})

	assert.False(t, called)
	expected := "{\"m\":\"aws.lambda.enhanced.invocations\",\"v\":1,"
	assert.False(t, strings.Contains(output, expected))
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
			APIKey:          "abc-123",
			Site:            server.URL,
			EnhancedMetrics: false,
		},
		&extension.ExtensionManager{},
	)
	//nolint
	ctx := context.WithValue(context.Background(), "cold_start", true)

	output := captureOutput(func() {
		ctx = ml.HandlerStarted(ctx, json.RawMessage{})
		ml.config.EnhancedMetrics = true
		err := errors.New("something went wrong")
		ml.HandlerFinished(ctx, err)
	})

	assert.False(t, called)
	expected := "{\"m\":\"aws.lambda.enhanced.errors\",\"v\":1,"
	assert.True(t, strings.Contains(output, expected))
}

func TestListenerHandlerFinishedFlushes(t *testing.T) {
	var called bool

	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	ts.Listener.Close()
	ts.Listener, _ = net.Listen("tcp", "127.0.0.1:8124")

	ts.Start()
	defer ts.Close()

	listener := MakeListener(Config{}, extension.BuildExtensionManager(false))
	listener.isAgentRunning = true
	for _, localTest := range []bool{true, false} {
		t.Run(fmt.Sprintf("%#v", localTest), func(t *testing.T) {
			called = false
			listener.config.LocalTest = localTest
			listener.HandlerFinished(context.TODO(), nil)
			assert.Equal(t, called, localTest)
		})
	}
}
