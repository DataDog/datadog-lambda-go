/*
 * Unless explicitly stated otherwise all files in this repository are licensed
 * under the Apache License Version 2.0.
 *
 * This product includes software developed at Datadog (https://www.datadoghq.com/).
 * Copyright 2021 Datadog, Inc.
 */

package extension

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/DataDog/datadog-lambda-go/internal/logger"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
)

type ddTraceContext string

const (
	DdTraceId          ddTraceContext = "x-datadog-trace-id"
	DdParentId         ddTraceContext = "x-datadog-parent-id"
	DdSpanId           ddTraceContext = "x-datadog-span-id"
	DdSamplingPriority ddTraceContext = "x-datadog-sampling-priority"
	DdInvocationError  ddTraceContext = "x-datadog-invocation-error"

	DdSeverlessSpan  ddTraceContext = "dd-tracer-serverless-span"
	DdLambdaResponse ddTraceContext = "dd-response"
)

const (
	// We don't want call to the Serverless Agent to block indefinitely for any reasons,
	// so here's a configuration of the timeout when calling the Serverless Agent. We also
	// want to let it having some time for its cold start so we should not set this too low.
	timeout = 3000 * time.Millisecond

	helloUrl           = "http://localhost:8124/lambda/hello"
	flushUrl           = "http://localhost:8124/lambda/flush"
	startInvocationUrl = "http://localhost:8124/lambda/start-invocation"
	endInvocationUrl   = "http://localhost:8124/lambda/end-invocation"

	extensionPath = "/opt/extensions/datadog-agent"
)

type ExtensionManager struct {
	helloRoute         string
	flushRoute         string
	extensionPath      string
	startInvocationUrl string
	endInvocationUrl   string
	httpClient         HTTPClient
	isExtensionRunning bool
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func BuildExtensionManager() *ExtensionManager {
	em := &ExtensionManager{
		helloRoute:         helloUrl,
		flushRoute:         flushUrl,
		startInvocationUrl: startInvocationUrl,
		endInvocationUrl:   endInvocationUrl,
		extensionPath:      extensionPath,
		httpClient:         &http.Client{Timeout: timeout},
	}
	em.checkAgentRunning()
	return em
}

func (em *ExtensionManager) checkAgentRunning() {
	if _, err := os.Stat(em.extensionPath); err != nil {
		logger.Debug("Will use the API")
		em.isExtensionRunning = false
	} else {
		logger.Debug("Will use the Serverless Agent")
		em.isExtensionRunning = true
	}
}

func (em *ExtensionManager) SendStartInvocationRequest(ctx context.Context, eventPayload json.RawMessage) context.Context {
	body := bytes.NewBuffer(eventPayload)
	req, _ := http.NewRequest(http.MethodPost, em.startInvocationUrl, body)

	if response, err := em.httpClient.Do(req); err == nil && response.StatusCode == 200 {
		// Propagate dd-trace context from the extension response if found in the response headers
		traceId := response.Header.Get(string(DdTraceId))
		if traceId != "" {
			ctx = context.WithValue(ctx, DdTraceId, traceId)
		}
		parentId := response.Header.Get(string(DdParentId))
		if parentId != "" {
			ctx = context.WithValue(ctx, DdParentId, parentId)
		}
		samplingPriority := response.Header.Get(string(DdSamplingPriority))
		if samplingPriority != "" {
			ctx = context.WithValue(ctx, DdSamplingPriority, samplingPriority)
		}
	}
	return ctx
}

func (em *ExtensionManager) SendEndInvocationRequest(ctx context.Context, functionExecutionSpan ddtrace.Span, err error) {
	// Handle Lambda response
	lambdaResponse := ctx.Value(DdLambdaResponse)
	content, responseErr := json.Marshal(lambdaResponse)
	if responseErr != nil {
		content = []byte("{}")
	}
	body := bytes.NewBuffer(content)
	req, _ := http.NewRequest(http.MethodPost, em.endInvocationUrl, body)

	// Mark the invocation as an error if any
	if err != nil {
		req.Header.Set(string(DdInvocationError), "true")
	}

	// Extract the DD trace context and pass them to the extension via request headers
	traceId, ok := ctx.Value(DdTraceId).(string)
	if ok {
		req.Header.Set(string(DdTraceId), traceId)
		if parentId, ok := ctx.Value(DdParentId).(string); ok {
			req.Header.Set(string(DdParentId), parentId)
		}
		if spanId, ok := ctx.Value(DdSpanId).(string); ok {
			req.Header.Set(string(DdSpanId), spanId)
		}
		if samplingPriority, ok := ctx.Value(DdSamplingPriority).(string); ok {
			req.Header.Set(string(DdSamplingPriority), samplingPriority)
		}
	} else {
		req.Header.Set(string(DdTraceId), fmt.Sprint(functionExecutionSpan.Context().TraceID()))
		req.Header.Set(string(DdSpanId), fmt.Sprint(functionExecutionSpan.Context().SpanID()))
	}

	resp, err := em.httpClient.Do(req)
	if err != nil || (resp.StatusCode >= 200 && resp.StatusCode <= 299) {
		logger.Error(fmt.Errorf("could not send end invocation payload to the extension"))
	}
}

func (em *ExtensionManager) IsExtensionRunning() bool {
	return em.isExtensionRunning
}

func (em *ExtensionManager) Flush() error {
	req, _ := http.NewRequest(http.MethodGet, em.flushRoute, nil)
	if response, err := em.httpClient.Do(req); err != nil {
		err := fmt.Errorf("was not able to reach the Agent to flush: %s", err)
		logger.Error(err)
		return err
	} else if response.StatusCode != 200 {
		err := fmt.Errorf("the Agent didn't returned HTTP 200: %s", response.Status)
		logger.Error(err)
		return err
	}
	return nil
}
