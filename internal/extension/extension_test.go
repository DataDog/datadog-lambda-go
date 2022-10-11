/*
 * Unless explicitly stated otherwise all files in this repository are licensed
 * under the Apache License Version 2.0.
 *
 * This product includes software developed at Datadog (https://www.datadoghq.com/).
 * Copyright 2021 Datadog, Inc.
 */

package extension

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

type ClientErrorMock struct {
}

type ClientSuccessMock struct {
}

type ClientSuccess202Mock struct {
}

type ClientSuccessStartInvoke struct {
}

type ClientSuccessEndInvoke struct {
}

const (
	mockTraceId          = "1"
	mockParentId         = "2"
	mockSamplingPriority = "3"
)

func (c *ClientErrorMock) Do(req *http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("KO")
}

func (c *ClientSuccessMock) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 200}, nil
}

func (c *ClientSuccess202Mock) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 202, Status: "KO"}, nil
}

func (c *ClientSuccessStartInvoke) Do(req *http.Request) (*http.Response, error) {
	header := http.Header{}
	header.Set(string(DdTraceId), mockTraceId)
	header.Set(string(DdParentId), mockParentId)
	header.Set(string(DdSamplingPriority), mockSamplingPriority)
	return &http.Response{StatusCode: 200, Status: "KO", Header: header}, nil
}

func (c *ClientSuccessEndInvoke) Do(req *http.Request) (*http.Response, error) {
	header := map[string][]string{}
	return &http.Response{StatusCode: 200, Status: "KO", Header: header}, nil
}

func TestBuildExtensionManager(t *testing.T) {
	em := BuildExtensionManager()
	assert.Equal(t, "http://localhost:8124/lambda/hello", em.helloRoute)
	assert.Equal(t, "http://localhost:8124/lambda/flush", em.flushRoute)
	assert.Equal(t, "http://localhost:8124/lambda/start-invocation", em.startInvocationUrl)
	assert.Equal(t, "http://localhost:8124/lambda/end-invocation", em.endInvocationUrl)
	assert.Equal(t, "/opt/extensions/datadog-agent", em.extensionPath)
	assert.NotNil(t, em.httpClient)
}

func TestIsAgentRunningFalse(t *testing.T) {
	em := &ExtensionManager{
		httpClient: &ClientErrorMock{},
	}
	assert.False(t, em.IsExtensionRunning())
}

func TestIsAgentRunningFalseSinceTheAgentIsNotHere(t *testing.T) {
	em := &ExtensionManager{
		extensionPath: "/impossible/path/test",
	}
	em.checkAgentRunning()
	assert.False(t, em.IsExtensionRunning())
}

func TestIsAgentRunningTrue(t *testing.T) {
	existingPath, err := os.Getwd()
	assert.Nil(t, err)

	em := &ExtensionManager{
		httpClient:    &ClientSuccessMock{},
		extensionPath: existingPath,
	}
	em.checkAgentRunning()
	assert.True(t, em.IsExtensionRunning())
}

func TestFlushErrorNot200(t *testing.T) {
	em := &ExtensionManager{
		httpClient: &ClientSuccess202Mock{},
	}
	err := em.Flush()
	assert.Equal(t, "the Agent didn't returned HTTP 200: KO", err.Error())
}

func TestFlushError(t *testing.T) {
	em := &ExtensionManager{
		httpClient: &ClientErrorMock{},
	}
	err := em.Flush()
	assert.Equal(t, "was not able to reach the Agent to flush: KO", err.Error())
}

func TestFlushSuccess(t *testing.T) {
	em := &ExtensionManager{
		httpClient: &ClientSuccessMock{},
	}
	err := em.Flush()
	assert.Nil(t, err)
}

func TestExtensionStartInvokeWithTraceContext(t *testing.T) {
	em := &ExtensionManager{
		startInvocationUrl: startInvocationUrl,
		httpClient:         &ClientSuccessStartInvoke{},
	}
	ctx := em.SendStartInvocationRequest(context.TODO(), []byte{})

	// Ensure that we pull the DD trace context if this is a distributed trace
	traceId := ctx.Value(DdTraceId)
	parentId := ctx.Value(DdParentId)
	samplingPriority := ctx.Value(DdSamplingPriority)
	err := em.Flush()

	assert.Nil(t, err)
	assert.Equal(t, mockTraceId, traceId)
	assert.Equal(t, mockParentId, parentId)
	assert.Equal(t, mockSamplingPriority, samplingPriority)
}
