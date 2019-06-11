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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	mockAPIKey = "12345"
)

func TestAddAPICredentials(t *testing.T) {
	cl := MakeAPIClient(context.Background(), "", mockAPIKey)
	req, _ := http.NewRequest("GET", "http://some-api.com/endpoint", nil)
	cl.addAPICredentials(req)
	assert.Equal(t, "http://some-api.com/endpoint?api_key=12345", req.URL.String())
}

func TestPrewarmConnection(t *testing.T) {

	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		assert.Equal(t, "/validate?api_key=12345", r.URL.String())
	}))
	defer server.Close()

	cl := MakeAPIClient(context.Background(), server.URL, mockAPIKey)
	err := cl.PrewarmConnection()

	assert.NoError(t, err)
	assert.True(t, called)
}

func TestSendMetricsSuccess(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
		body, _ := ioutil.ReadAll(r.Body)
		s := string(body)

		assert.Equal(t, "/distribution_points?api_key=12345", r.URL.String())
		assert.Equal(t, "{\"series\":[{\"metric\":\"metric-1\",\"tags\":[\"a\",\"b\",\"c\"],\"type\":\"distribution\",\"points\":[[1,2],[3,4],[5,6]]}]}", s)

	}))
	defer server.Close()

	am := []APIMetric{
		{
			Name:       "metric-1",
			Host:       nil,
			Tags:       []string{"a", "b", "c"},
			MetricType: DistributionType,
			Points: [][]float64{
				{float64(1), float64(2)}, {float64(3), float64(4)}, {float64(5), float64(6)},
			},
		},
	}

	cl := MakeAPIClient(context.Background(), server.URL, mockAPIKey)
	err := cl.SendMetrics(am)

	assert.NoError(t, err)
	assert.True(t, called)
}

func TestSendMetricsBadRequest(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusForbidden)
		body, _ := ioutil.ReadAll(r.Body)
		s := string(body)

		assert.Equal(t, "/distribution_points?api_key=12345", r.URL.String())
		assert.Equal(t, "{\"series\":[{\"metric\":\"metric-1\",\"tags\":[\"a\",\"b\",\"c\"],\"type\":\"distribution\",\"points\":[[1,2],[3,4],[5,6]]}]}", s)

	}))
	defer server.Close()

	am := []APIMetric{
		{
			Name:       "metric-1",
			Host:       nil,
			Tags:       []string{"a", "b", "c"},
			MetricType: DistributionType,
			Points: [][]float64{
				{float64(1), float64(2)}, {float64(3), float64(4)}, {float64(5), float64(6)},
			},
		},
	}

	cl := MakeAPIClient(context.Background(), server.URL, mockAPIKey)
	err := cl.SendMetrics(am)

	assert.Error(t, err)
	assert.True(t, called)
}

func TestSendMetricsCantReachServer(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	am := []APIMetric{
		{
			Name:       "metric-1",
			Host:       nil,
			Tags:       []string{"a", "b", "c"},
			MetricType: DistributionType,
			Points: [][]float64{
				{float64(1), float64(2)}, {float64(3), float64(4)}, {float64(5), float64(6)},
			},
		},
	}

	cl := MakeAPIClient(context.Background(), "httpa:///badly-formatted-url", mockAPIKey)
	err := cl.SendMetrics(am)

	assert.Error(t, err)
	assert.False(t, called)
}
