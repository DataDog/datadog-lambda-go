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
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"runtime"
	"strings"
	"time"

	"github.com/DataDog/datadog-lambda-go/internal/logger"
)

type (
	// Listener implements wrapper.HandlerListener, injecting metrics into the context
	Listener struct {
		apiClient *APIClient
		config    *Config
		processor Processor
	}

	// Config gives options for how the listener should work
	Config struct {
		APIKey                string
		KMSAPIKey             string
		Site                  string
		ShouldRetryOnFailure  bool
		ShouldUseLogForwarder bool
		BatchInterval         time.Duration
		EnhancedMetrics       bool
	}

	logMetric struct {
		MetricName string   `json:"m"`
		Value      float64  `json:"v"`
		Timestamp  int64    `json:"e"`
		Tags       []string `json:"t"`
	}
)

// MakeListener initializes a new metrics lambda listener
func MakeListener(config Config) Listener {

	apiClient := MakeAPIClient(context.Background(), APIClientOptions{
		baseAPIURL: config.Site,
		apiKey:     config.APIKey,
		decrypter:  MakeKMSDecrypter(),
		kmsAPIKey:  config.KMSAPIKey,
	})
	if config.BatchInterval <= 0 {
		config.BatchInterval = defaultBatchInterval
	}

	return Listener{
		apiClient: apiClient,
		config:    &config,
		processor: nil,
	}
}

// HandlerStarted adds metrics service to the context
func (l *Listener) HandlerStarted(ctx context.Context, msg json.RawMessage) context.Context {
	if l.apiClient.apiKey == "" && l.config.KMSAPIKey == "" && !l.config.ShouldUseLogForwarder {
		logger.Error(fmt.Errorf("datadog api key isn't set, won't be able to send metrics"))
	}

	ts := MakeTimeService()
	pr := MakeProcessor(ctx, l.apiClient, ts, l.config.BatchInterval, l.config.ShouldRetryOnFailure)
	l.processor = pr

	ctx = AddListener(ctx, l)
	// Setting the context on the client will mean that future requests will be cancelled correctly
	// if the lambda times out.
	l.apiClient.context = ctx

	pr.StartProcessing()
	l.submitEnhancedMetrics("invocations", ctx)

	return ctx
}

// HandlerFinished implemented as part of the wrapper.HandlerListener interface
func (l *Listener) HandlerFinished(ctx context.Context) {
	if l.processor != nil {
		if ctx.Value("error") != nil {
			l.submitEnhancedMetrics("errors", ctx)
		}
		l.processor.FinishProcessing()
	}
}

// AddDistributionMetric sends a distribution metric
func (l *Listener) AddDistributionMetric(metric string, value float64, timestamp time.Time, tags ...string) {

	// We add our own runtime tag to the metric for version tracking
	tags = append(tags, getRuntimeTag())

	if l.config.ShouldUseLogForwarder {
		logger.Debug("sending metric via log forwarder")
		unixTime := timestamp.Unix()
		lm := logMetric{
			MetricName: metric,
			Value:      value,
			Timestamp:  unixTime,
			Tags:       tags,
		}
		result, err := json.Marshal(lm)
		if err != nil {
			logger.Error(fmt.Errorf("failed to marshall metric for log forwarder with error %v", err))
			return
		}
		payload := string(result)
		println(payload)
		return
	}
	m := Distribution{
		Name:   metric,
		Tags:   tags,
		Values: []MetricValue{},
	}
	m.AddPoint(timestamp, value)
	logger.Debug(fmt.Sprintf("adding metric \"%s\", with value %f", metric, value))
	l.processor.AddMetric(&m)
}

func getRuntimeTag() string {
	v := runtime.Version()
	return fmt.Sprintf("dd_lambda_layer:datadog-%s", v)
}

func (l *Listener) submitEnhancedMetrics(metricName string, ctx context.Context) {
	if l.config.EnhancedMetrics {
		tags := getEnhancedMetricsTags(ctx)
		l.AddDistributionMetric(fmt.Sprintf("aws.lambda.enhanced.%s", metricName), 1, time.Now(), tags...)
	}
}

func getEnhancedMetricsTags(ctx context.Context) []string {
	isColdStart := ctx.Value("cold_start")

	if lc, ok := lambdacontext.FromContext(ctx); ok {
		// ex: arn:aws:lambda:us-east-1:123497558138:function:golang-layer
		splitArn := strings.Split(lc.InvokedFunctionArn, ":")

		functionName := fmt.Sprintf("functionname:%s", lambdacontext.FunctionName)
		region := fmt.Sprintf("region:%s", splitArn[3])
		accountId := fmt.Sprintf("account_id:%s", splitArn[4])
		memorySize := fmt.Sprintf("memorysize:%d", lambdacontext.MemoryLimitInMB)
		coldStart := fmt.Sprintf("cold_start:%t", isColdStart.(bool))
		return []string{functionName, region, accountId, memorySize, coldStart}
	}
	logger.Debug("could not retrieve the LambdaContext from Context")
	return []string{}
}
