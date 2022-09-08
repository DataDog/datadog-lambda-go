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
		// The hello route marks in the extension that a lambda library is active. This prevents certain trace features in the extension
		em.isExtensionRunning = true

		// req, _ := http.NewRequest(http.MethodGet, em.helloRoute, nil)
		// if response, err := em.httpClient.Do(req); err == nil && response.StatusCode == 200 {
		// 	logger.Debug("Will use the Serverless Agent")
		// 	em.isExtensionRunning = true
		// } else {
		// 	logger.Debug("Will use the API since the Serverless Agent was detected but the hello route was unreachable")
		// 	em.isExtensionRunning = false
		// }
	}
}

func (em *ExtensionManager) SendStartInvocationRequest(lambdaContext context.Context, eventPayload json.RawMessage) {
	body := bytes.NewBuffer(eventPayload)
	req, _ := http.NewRequest(http.MethodPost, em.startInvocationUrl, body)
	// For the Lambda context, we need to put each k:v into the request headers
	logger.Debug(fmt.Sprintf("Context: %v", lambdaContext))

	// TODO: send dummy x-datadog headers
	// req.Header = map[string][]string{"x-datadog-trace-id": {"0"}}

	if response, err := em.httpClient.Do(req); err == nil && response.StatusCode == 200 {
		logger.Debug(fmt.Sprintf("Response Body: %v", response.Body))
		logger.Debug(fmt.Sprintf("Response Header: %v", response.Header))
	}
}

func (em *ExtensionManager) SendEndInvocationRequest(traceCtx map[string]string, err error) {
	content, _ := json.Marshal(err)
	// content, err := json.Marshal(err)
	// if err != nil {
	// 	logger.Debug("Uhoh")
	// }

	body := bytes.NewBuffer(content)
	// body := bytes.NewBuffer([]byte(content))

	// We should try to extract any trace context from the lambda context if available
	req, _ := http.NewRequest(http.MethodPost, em.endInvocationUrl, body)

	// Add trace context as headers
	for k, v := range traceCtx {
		if k == "x-datadog-sampling-priority" {
			req.Header[k] = append(req.Header[k], "1")
			logger.Debug("override sampling priority")
			continue
		}
		req.Header[k] = append(req.Header[k], v)
	}

	// For the Lambda context, we need to put each k:v into the request headers
	logger.Debug(fmt.Sprintf("Request Header: %v", req.Header))

	if response, err := em.httpClient.Do(req); err == nil && response.StatusCode == 200 {
		logger.Debug(fmt.Sprintf("Response Body: %v", response.Body))
		logger.Debug(fmt.Sprintf("Response Header: %v", response.Header))
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
