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
		req, _ := http.NewRequest(http.MethodGet, em.helloRoute, nil)
		if response, err := em.httpClient.Do(req); err == nil && response.StatusCode == 200 {
			logger.Debug("Will use the Serverless Agent")
			em.isExtensionRunning = true
		} else {
			logger.Debug("Will use the API since the Serverless Agent was detected but the hello route was unreachable")
			em.isExtensionRunning = false
		}
	}
}

func (em *ExtensionManager) SendStartInvocationRequest(lambdaContext context.Context, eventPayload json.RawMessage) {
	content, err := json.Marshal(eventPayload)
	if err != nil {
		logger.Debug("Uhoh")
	}
	body := bytes.NewBuffer(content)
	req, _ := http.NewRequest(http.MethodPost, em.startInvocationUrl, body)
	// For the Lambda context, we need to put each k:v into the request headers
	logger.Debug(fmt.Sprintf("Context: %v", lambdaContext))

	if response, err := em.httpClient.Do(req); err == nil && response.StatusCode == 200 {
		logger.Debug(fmt.Sprintf("NICE: %v", response))
	} else {
		logger.Debug("BAD")
	}
}

func (em *ExtensionManager) SendEndInvocationRequest(lambdaContext context.Context, err error) {
	content, err := json.Marshal(err)
	if err != nil {
		logger.Debug("Uhoh")
	}
	body := bytes.NewBuffer(content)
	req, _ := http.NewRequest(http.MethodPost, em.endInvocationUrl, body)
	// For the Lambda context, we need to put each k:v into the request headers
	logger.Debug(fmt.Sprintf("Context: %v", lambdaContext))
	if response, err := em.httpClient.Do(req); err == nil && response.StatusCode == 200 {
		logger.Debug(fmt.Sprintf("NICE: %v", response))
	} else {
		logger.Debug("BAD")
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
