/*
 * Unless explicitly stated otherwise all files in this repository are licensed
 * under the Apache License Version 2.0.
 *
 * This product includes software developed at Datadog (https://www.datadoghq.com/).
 * Copyright 2019 Datadog, Inc.
 */

package wrapper

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/DataDog/datadog-lambda-go/internal/metrics"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

type (
	mockHandlerListener struct {
		inputCTX  context.Context
		inputMSG  json.RawMessage
		outputCTX context.Context
	}

	mockNonProxyEvent struct {
		MyCustomEvent map[string]int `json:"my-custom-event"`
		FakeID        string         `json:"fake-id"`
	}
)

func (mhl *mockHandlerListener) HandlerStarted(ctx context.Context, msg json.RawMessage) context.Context {
	mhl.inputCTX = ctx
	mhl.inputMSG = msg
	return ctx
}

func (mhl *mockHandlerListener) HandlerFinished(ctx context.Context) {
	mhl.outputCTX = ctx
}

func runHandlerWithJSON(t *testing.T, filename string, handler interface{}) (*mockHandlerListener, interface{}, error) {
	ctx := context.Background()
	payload := loadRawJSON(t, filename)

	mhl := mockHandlerListener{}

	wrappedHandler := WrapHandlerWithListeners(handler, &mhl).(func(context.Context, json.RawMessage) (interface{}, error))

	response, err := wrappedHandler(ctx, *payload)
	return &mhl, response, err
}

func loadRawJSON(t *testing.T, filename string) *json.RawMessage {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		assert.Fail(t, "Couldn't find JSON file")
		return nil
	}
	msg := json.RawMessage{}
	msg.UnmarshalJSON(bytes)
	return &msg
}

func TestValidateHandlerNotFunction(t *testing.T) {
	nonFunction := 1

	err := validateHandler(nonFunction)
	assert.EqualError(t, err, "handler is not a function")
}
func TestValidateHandlerToManyArguments(t *testing.T) {
	tooManyArgs := func(a, b, c int) {
	}

	err := validateHandler(tooManyArgs)
	assert.EqualError(t, err, "handler takes too many arguments")
}

func TestValidateHandlerContextIsNotFirstArgument(t *testing.T) {
	firstArgNotContext := func(arg1, arg2 int) {
	}

	err := validateHandler(firstArgNotContext)
	assert.EqualError(t, err, "handler should take context as first argument")
}

func TestValidateHandlerTwoArguments(t *testing.T) {
	twoArguments := func(arg1 context.Context, arg2 int) {
	}

	err := validateHandler(twoArguments)
	assert.NoError(t, err)
}

func TestValidateHandlerOneArgument(t *testing.T) {
	oneArgument := func(arg1 int) {
	}

	err := validateHandler(oneArgument)
	assert.NoError(t, err)
}

func TestValidateHandlerTooManyReturnValues(t *testing.T) {
	tooManyReturns := func() (int, int, error) {
		return 0, 0, nil
	}

	err := validateHandler(tooManyReturns)
	assert.EqualError(t, err, "handler returns more than two values")
}
func TestValidateHandlerLastReturnValueNotError(t *testing.T) {
	lastNotError := func() (int, int) {
		return 0, 0
	}

	err := validateHandler(lastNotError)
	assert.EqualError(t, err, "handler doesn't return error as it's last value")
}
func TestValidateHandlerCorrectFormat(t *testing.T) {
	correct := func(context context.Context) (int, error) {
		return 0, nil
	}

	err := validateHandler(correct)
	assert.NoError(t, err)
}

func TestWrapHandlerAPIGEvent(t *testing.T) {
	called := false

	handler := func(ctx context.Context, request events.APIGatewayProxyRequest) (int, error) {
		called = true
		assert.Equal(t, "c6af9ac6-7b61-11e6-9a41-93e8deadbeef", request.RequestContext.RequestID)
		return 5, nil
	}

	_, response, err := runHandlerWithJSON(t, "../testdata/apig-event-no-metadata.json", handler)

	assert.True(t, called)
	assert.NoError(t, err)
	assert.Equal(t, 5, response)
}

func TestWrapHandlerNonProxyEvent(t *testing.T) {
	called := false

	handler := func(ctx context.Context, request mockNonProxyEvent) (int, error) {
		called = true
		assert.Equal(t, "12345678910", request.FakeID)
		return 5, nil
	}

	_, response, err := runHandlerWithJSON(t, "../testdata/non-proxy-no-metadata.json", handler)

	assert.True(t, called)
	assert.NoError(t, err)
	assert.Equal(t, 5, response)
}

func TestWrapHandlerEventArgumentOnly(t *testing.T) {
	called := false

	handler := func(request mockNonProxyEvent) (int, error) {
		called = true
		assert.Equal(t, "12345678910", request.FakeID)
		return 5, nil
	}

	_, response, err := runHandlerWithJSON(t, "../testdata/non-proxy-no-metadata.json", handler)

	assert.True(t, called)
	assert.NoError(t, err)
	assert.Equal(t, 5, response)
}

func TestWrapHandlerContextArgumentOnly(t *testing.T) {
	called := true
	var handler = func(ctx context.Context) (interface{}, error) {
		return nil, nil
	}

	mhl := mockHandlerListener{}
	wrappedHandler := WrapHandlerWithListeners(handler, &mhl).(func(context.Context, json.RawMessage) (interface{}, error))

	_, err := wrappedHandler(context.Background(), nil)
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestWrapHandlerNoArguments(t *testing.T) {
	called := false

	handler := func() (int, error) {
		called = true
		return 5, nil
	}

	_, response, err := runHandlerWithJSON(t, "../testdata/non-proxy-no-metadata.json", handler)

	assert.True(t, called)
	assert.NoError(t, err)
	assert.Equal(t, 5, response)
}

func TestWrapHandlerInvalidData(t *testing.T) {
	called := false

	handler := func(request mockNonProxyEvent) (int, error) {
		called = true
		return 5, nil
	}

	_, response, err := runHandlerWithJSON(t, "../testdata/invalid.json", handler)

	assert.False(t, called)
	assert.Error(t, err)
	assert.Equal(t, nil, response)
}

func TestWrapHandlerReturnsError(t *testing.T) {
	called := false
	defaultErr := errors.New("Some error")

	handler := func(request mockNonProxyEvent) (int, error) {
		called = true
		return 5, defaultErr
	}

	_, response, err := runHandlerWithJSON(t, "../testdata/non-proxy-no-metadata.json", handler)

	assert.True(t, called)
	assert.Equal(t, defaultErr, err)
	assert.Equal(t, 5, response)
}

func TestWrapHandlerReturnsErrorOnly(t *testing.T) {
	called := false
	defaultErr := errors.New("Some error")

	handler := func(request mockNonProxyEvent) error {
		called = true
		return defaultErr
	}

	_, response, err := runHandlerWithJSON(t, "../testdata/non-proxy-no-metadata.json", handler)

	assert.True(t, called)
	assert.Equal(t, defaultErr, err)
	assert.Equal(t, nil, response)
}

func TestWrapHandlerReturnsOriginalHandlerIfInvalid(t *testing.T) {

	var handler interface{} = func(arg1, arg2, arg3 int) (int, error) {
		return 0, nil
	}
	mhl := mockHandlerListener{}

	wrappedHandler := WrapHandlerWithListeners(handler, &mhl)

	assert.Equal(t, reflect.ValueOf(handler).Pointer(), reflect.ValueOf(wrappedHandler).Pointer())

}

func TestGetEnhancedMetricsTags(t *testing.T) {
	ctx := context.WithValue(context.Background(), "cold_start", false)

	lambdacontext.MemoryLimitInMB = 256
	lambdacontext.FunctionName = "go-lambda-test"
	lc := &lambdacontext.LambdaContext{
		InvokedFunctionArn: "arn:aws:lambda:us-east-1:123497558138:function:go-lambda-test",
	}
	tags := getEnhancedMetricsTags(lambdacontext.NewContext(ctx, lc))

	assert.ElementsMatch(t, tags, []string{"functionname:go-lambda-test", "region:us-east-1", "memorysize:256", "cold_start:false", "account_id:123497558138"})
}

func TestGetEnhancedMetricsTagsNoLambdaContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), "cold_start", true)
	tags := getEnhancedMetricsTags(ctx)

	assert.Empty(t, tags)
}

func TestSubmitEnhancedMetrics(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	ml := metrics.MakeListener(
		metrics.Config{
			APIKey:                "abc-123",
			Site:                  server.URL,
			EnhancedMetrics:       true,
		},
	)
	ctx := context.WithValue(context.Background(), "cold_start", false)

	ctx = ml.HandlerStarted(ctx, json.RawMessage{})
	submitEnhancedMetrics("invocations", ctx)
	submitEnhancedMetrics("errors", ctx)
	ml.HandlerFinished(ctx)

	assert.True(t, called)
}

func TestDoNotSubmitEnhancedMetrics(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	ml := metrics.MakeListener(
		metrics.Config{
			APIKey:                "abc-123",
			Site:                  server.URL,
			EnhancedMetrics:       false,
		},
	)
	ctx := context.WithValue(context.Background(), "cold_start", true)

	ctx = ml.HandlerStarted(ctx, json.RawMessage{})
	submitEnhancedMetrics("invocations", ctx)
	submitEnhancedMetrics("errors", ctx)
	ml.HandlerFinished(ctx)

	assert.False(t, called)
}