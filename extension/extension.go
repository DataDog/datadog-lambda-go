/*
 * Unless explicitly stated otherwise all files in this repository are licensed
 * under the Apache License Version 2.0.
 *
 * This product includes software developed at Datadog (https://www.datadoghq.com/).
 * Copyright 2021 Datadog, Inc.
 */

package extension

import (
	"fmt"
	"net/http"
	"time"

	"github.com/DataDog/datadog-lambda-go/internal/logger"
)

const (
	// We don't want call to the Serverless Agent to block indefinitely for any reasons,
	// so here's a configuration of the timeout when calling the Serverless Agent. We also
	// want to let it having some time for its cold start so we should not set this too low.
	timeout = 3000 * time.Millisecond

	// helloUrl is the hello route url
	helloUrl = "http://localhost:8124/lambda/hello"

	// flushUrl is the flush route url
	flushUrl = "http://localhost:8124/lambda/flush"
)

type ExtensionManager struct {
	helloRoute         string
	flushRoute         string
	httpClient         HTTPClient
	isExtensionRunning bool
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func BuildExtensionManager() *ExtensionManager {
	em := &ExtensionManager{
		helloRoute: helloUrl,
		flushRoute: flushUrl,
		httpClient: &http.Client{Timeout: timeout},
	}
	em.checkAgentRunning()
	return em
}

func (em *ExtensionManager) checkAgentRunning() {
	req, _ := http.NewRequest(http.MethodGet, em.helloRoute, nil)
	if response, err := em.httpClient.Do(req); err == nil && response.StatusCode == 200 {
		logger.Debug("Will use the Serverless Agent")
		em.isExtensionRunning = true
	} else {
		logger.Debug("Will use the API")
		em.isExtensionRunning = false
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
