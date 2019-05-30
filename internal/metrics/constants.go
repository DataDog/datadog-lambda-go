/*
 * Unless explicitly stated otherwise all files in this repository are licensed
 * under the Apache License Version 2.0.
 * 
 * This product includes software developed at Datadog (https://www.datadoghq.com/).
 * Copyright 2019 Datadog, Inc.
 */

package metrics

import "time"

const (
	baseAPIURL           = "https://api.datadoghq.com/api/v1"
	apiKeyParam          = "api_key"
	appKeyParam          = "application_key"
	defaultRetryInterval = time.Millisecond * 250
	defaultBatchInterval = time.Second * 15
)

// MetricType enumerates all the available metric types
type MetricType string

const (

	// DistributionType represents a distribution metric
	DistributionType MetricType = "distribution"
)
