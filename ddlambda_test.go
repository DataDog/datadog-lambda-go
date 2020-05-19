/*
 * Unless explicitly stated otherwise all files in this repository are licensed
 * under the Apache License Version 2.0.
 *
 * This product includes software developed at Datadog (https://www.datadoghq.com/).
 * Copyright 2019 Datadog, Inc.
 */
package ddlambda

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/DataDog/datadog-lambda-go/internal/metrics"
	"github.com/stretchr/testify/assert"
)

func TestInvokeDryRun(t *testing.T) {
	called := false
	InvokeDryRun(func(ctx context.Context) {
		called = true
		globalCtx := GetContext()
		assert.Equal(t, globalCtx, ctx)
	}, nil)
	assert.True(t, called)
}

func TestMetricsSilentFailWithoutWrapper(t *testing.T) {
	Metric("my-metric", 100, "my:tag")
}

func TestMetricsSubmitWithWrapper(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	InvokeDryRun(func(ctx context.Context) {
		Metric("my-metric", 100, "my:tag")
	}, &Config{
		APIKey: "abc-123",
		Site:   server.URL,
	})
	assert.True(t, called)
}

func TestConfig_toMetricsConfig(t *testing.T) {
	type envvars struct {
		DatadogAPIKeyEnvVar    string
		DatadogKMSAPIKeyEnvVar string
		DatadogSiteEnvVar      string
		DatadogGlobalTags      string
	}
	tests := map[string]struct {
		config  *Config
		envvars envvars
		want    metrics.Config
	}{
		"with no env vars": {
			config:  &Config{},
			envvars: envvars{},
			want: metrics.Config{
				APIKey:                "",
				KMSAPIKey:             "",
				Site:                  "https://api.datadoghq.com/api/v1",
				ShouldRetryOnFailure:  false,
				ShouldUseLogForwarder: false,
				BatchInterval:         0,
				EnhancedMetrics:       true,
				GlobalTags:            []string{},
			},
		},
		"with env vars": {
			config: &Config{},
			envvars: envvars{
				DatadogAPIKeyEnvVar:    "fooAPIKeyEnvVar",
				DatadogKMSAPIKeyEnvVar: "fooKMSAPIKeyEnvVar",
				DatadogSiteEnvVar:      "fooSiteEnvVar.com",
				DatadogGlobalTags:      "key1:val1,key2:val2",
			},
			want: metrics.Config{
				APIKey:                "fooAPIKeyEnvVar",
				KMSAPIKey:             "fooKMSAPIKeyEnvVar",
				Site:                  "https://api.fooSiteEnvVar.com/api/v1",
				ShouldRetryOnFailure:  false,
				ShouldUseLogForwarder: false,
				BatchInterval:         0,
				EnhancedMetrics:       true,
				GlobalTags:            []string{"key1:val1", "key2:val2"},
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			// set env vars
			os.Setenv(DatadogAPIKeyEnvVar, test.envvars.DatadogAPIKeyEnvVar)
			os.Setenv(DatadogKMSAPIKeyEnvVar, test.envvars.DatadogKMSAPIKeyEnvVar)
			os.Setenv(DatadogSiteEnvVar, test.envvars.DatadogSiteEnvVar)
			os.Setenv(DatadogGlobalTags, test.envvars.DatadogGlobalTags)

			// run
			got := test.config.toMetricsConfig()

			// assert
			assert.EqualValues(t, test.want, got)

			// cleanup
			os.Unsetenv(DatadogAPIKeyEnvVar)
			os.Unsetenv(DatadogKMSAPIKeyEnvVar)
			os.Unsetenv(DatadogSiteEnvVar)
			os.Unsetenv(DatadogGlobalTags)
		})
	}
}
