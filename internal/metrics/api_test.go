/*
 * Unless explicitly stated otherwise all files in this repository are licensed
 * under the Apache License Version 2.0.
 *
 * This product includes software developed at Datadog (https://www.datadoghq.com/).
 * Copyright 2021 Datadog, Inc.
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
	mockAPIKey          = "12345"
	mockEncryptedAPIKey = "mockEncrypted"
	mockDecryptedAPIKey = "mockDecrypted"
	mockAPIKeySecretARN = "arn:aws:secretsmanager:us-east-1:123456789012:secret:apiKey"
)

type (
	mockDecrypter struct {
		returnValue string
		returnError error
	}
)

func (md *mockDecrypter) Decrypt(cipherText string) (string, error) {
	return md.returnValue, md.returnError
}

type mockSecretFetcher struct {
	returnValue string
	returnError error
}

func (m *mockSecretFetcher) FetchSecret(secretID string) (string, error) {
	return m.returnValue, m.returnError
}

func TestAddAPICredentials(t *testing.T) {
	cl := MakeAPIClient(context.Background(), APIClientOptions{baseAPIURL: "", apiKey: mockAPIKey})
	req, _ := http.NewRequest("GET", "http://some-api.com/endpoint", nil)
	cl.addAPICredentials(req)
	assert.Equal(t, "http://some-api.com/endpoint?api_key=12345", req.URL.String())
}

func TestSendMetricsSuccess(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
		body, _ := ioutil.ReadAll(r.Body)
		s := string(body)

		assert.Equal(t, "/distribution_points?api_key=12345", r.URL.String())
		assert.Equal(t, "{\"series\":[{\"metric\":\"metric-1\",\"tags\":[\"a\",\"b\",\"c\"],\"type\":\"distribution\",\"points\":[[1,[2]],[3,[4]],[5,[6]]]}]}", s)

	}))
	defer server.Close()

	am := []APIMetric{
		{
			Name:       "metric-1",
			Host:       nil,
			Tags:       []string{"a", "b", "c"},
			MetricType: DistributionType,
			Points: []interface{}{
				[]interface{}{float64(1), []interface{}{float64(2)}},
				[]interface{}{float64(3), []interface{}{float64(4)}},
				[]interface{}{float64(5), []interface{}{float64(6)}},
			},
		},
	}

	cl := MakeAPIClient(context.Background(), APIClientOptions{baseAPIURL: server.URL, apiKey: mockAPIKey})
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
		assert.Equal(t, "{\"series\":[{\"metric\":\"metric-1\",\"tags\":[\"a\",\"b\",\"c\"],\"type\":\"distribution\",\"points\":[[1,[2]],[3,[4]],[5,[6]]]}]}", s)

	}))
	defer server.Close()

	am := []APIMetric{
		{
			Name:       "metric-1",
			Host:       nil,
			Tags:       []string{"a", "b", "c"},
			MetricType: DistributionType,
			Points: []interface{}{
				[]interface{}{float64(1), []interface{}{float64(2)}},
				[]interface{}{float64(3), []interface{}{float64(4)}},
				[]interface{}{float64(5), []interface{}{float64(6)}},
			},
		},
	}

	cl := MakeAPIClient(context.Background(), APIClientOptions{baseAPIURL: server.URL, apiKey: mockAPIKey})
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
			Points: []interface{}{
				[]interface{}{float64(1), []interface{}{float64(2)}},
				[]interface{}{float64(3), []interface{}{float64(4)}},
				[]interface{}{float64(5), []interface{}{float64(6)}},
			},
		},
	}

	cl := MakeAPIClient(context.Background(), APIClientOptions{baseAPIURL: "httpa:///badly-formatted-url", apiKey: mockAPIKey})
	err := cl.SendMetrics(am)

	assert.Error(t, err)
	assert.False(t, called)
}

func TestLazyLoadAPIKey(t *testing.T) {
	tests := map[string]struct {
		clientOptions  APIClientOptions
		expectedAPIKey string
	}{
		"decrypt using KMS key": {
			clientOptions: APIClientOptions{
				kmsAPIKey: mockEncryptedAPIKey,
				decrypter: &mockDecrypter{returnValue: mockDecryptedAPIKey},
			},
			expectedAPIKey: mockDecryptedAPIKey,
		},
		"fetch from secret ARN": {
			clientOptions: APIClientOptions{
				apiKeySecretARN: mockAPIKeySecretARN,
				secretFetcher:   &mockSecretFetcher{returnValue: mockAPIKey},
			},
			expectedAPIKey: mockAPIKey,
		},
	}

	for name, tt := range tests {
		tt := tt // https://go.dev/doc/faq#closures_and_goroutines
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			called := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				assert.Equal(t, "/distribution_points?api_key="+tt.expectedAPIKey, r.URL.String())
			}))
			defer server.Close()

			am := []APIMetric{
				{
					Name:       "metric-1",
					Host:       nil,
					Tags:       []string{"a", "b", "c"},
					MetricType: DistributionType,
					Points: []interface{}{
						[]interface{}{float64(1), []interface{}{float64(2)}},
						[]interface{}{float64(3), []interface{}{float64(4)}},
						[]interface{}{float64(5), []interface{}{float64(6)}},
					},
				},
			}
			tt.clientOptions.baseAPIURL = server.URL
			err := MakeAPIClient(context.Background(), tt.clientOptions).SendMetrics(am)

			assert.NoError(t, err)
			assert.True(t, called)
		})
	}
}
