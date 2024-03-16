/*
 * Unless explicitly stated otherwise all files in this repository are licensed
 * under the Apache License Version 2.0.
 *
 * This product includes software developed at Datadog (https://www.datadoghq.com/).
 * Copyright 2021 Datadog, Inc.
 */
package ddlambda

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInvokeDryRun(t *testing.T) {
	t.Setenv(UniversalInstrumentation, "false")
	t.Setenv(DatadogTraceEnabledEnvVar, "false")

	called := false
	_, err := InvokeDryRun(func(ctx context.Context) {
		called = true
		globalCtx := GetContext()
		assert.Equal(t, globalCtx, ctx)
	}, nil)
	assert.NoError(t, err)
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

	_, err := InvokeDryRun(func(ctx context.Context) {
		Metric("my-metric", 100, "my:tag")
	}, &Config{
		APIKey: "abc-123",
		Site:   server.URL,
	})
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestToMetricConfigLocalTest(t *testing.T) {
	testcases := []struct {
		envs map[string]string
		cval bool
	}{
		{
			envs: map[string]string{"DD_LOCAL_TEST": "True"},
			cval: true,
		},
		{
			envs: map[string]string{"DD_LOCAL_TEST": "true"},
			cval: true,
		},
		{
			envs: map[string]string{"DD_LOCAL_TEST": "1"},
			cval: true,
		},
		{
			envs: map[string]string{"DD_LOCAL_TEST": "False"},
			cval: false,
		},
		{
			envs: map[string]string{"DD_LOCAL_TEST": "false"},
			cval: false,
		},
		{
			envs: map[string]string{"DD_LOCAL_TEST": "0"},
			cval: false,
		},
		{
			envs: map[string]string{"DD_LOCAL_TEST": ""},
			cval: false,
		},
		{
			envs: map[string]string{},
			cval: false,
		},
	}

	cfg := Config{}
	for _, tc := range testcases {
		t.Run(fmt.Sprintf("%#v", tc.envs), func(t *testing.T) {
			for k, v := range tc.envs {
				os.Setenv(k, v)
			}
			mc := cfg.toMetricsConfig(true)
			assert.Equal(t, tc.cval, mc.LocalTest)
		})
	}
}
