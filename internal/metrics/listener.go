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
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambdacontext"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/stroem/datadog-lambda-go/internal/logger"
)

const datadogLambdaVersion = "v0.9.0"

type (
	// Listener implements wrapper.HandlerListener, injecting metrics into the context
	Listener struct {
		apiClient          *APIClient
		statsdClient       *statsd.Client
		config             *Config
		processor          Processor
		useServerlessAgent bool
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

const (
	// We don't want call to the Serverless Agent to block indefinitely for any reasons,
	// so here's a configuration of the timeout when calling the Serverless Agent. We also
	// want to let it having some time for its cold start so we should not set this too low.
	AgentCallTimeout = 3000 * time.Millisecond
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

	var statsdClient *statsd.Client
	// immediate call to the Agent, if not a 200, fallback to API
	// TODO(remy): we may want to use an environment var to force the use of the
	// Agent instead of using this "discovery" implementation.
	if isServerlessAgentRunning() {
		var err error
		if statsdClient, err = statsd.New("127.0.0.1:8125"); err != nil {
			statsdClient = nil // force nil if an error occurred during statsd client init
		}
	}

	return Listener{
		apiClient:          apiClient,
		config:             &config,
		useServerlessAgent: statsdClient != nil,
		statsdClient:       statsdClient,
		processor:          nil,
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
	if l.useServerlessAgent {
		// use the agent
		// flush the metrics from the DogStatsD client to the Agent
		if l.statsdClient != nil {
			if err := l.statsdClient.Flush(); err != nil {
				logger.Error(fmt.Errorf("can't flush the DogStatsD client: %s", err))
			}
		}
		// send a message to the Agent to flush the metrics
		if err := flushServerlessAgent(); err != nil {
			logger.Error(fmt.Errorf("error while flushing the metrics: %s", err))
		}
	} else {
		// use the api
		if l.processor != nil {
			if ctx.Value("error") != nil {
				l.submitEnhancedMetrics("errors", ctx)
			}
			l.processor.FinishProcessing()
		}
	}
}

// AddDistributionMetric sends a distribution metric
func (l *Listener) AddDistributionMetric(metric string, value float64, timestamp time.Time, forceLogForwarder bool, tags ...string) {

	// We add our own runtime tag to the metric for version tracking
	tags = append(tags, getRuntimeTag())

	if l.useServerlessAgent {
		l.statsdClient.Distribution(metric, value, tags, 1)
		return
	}

	if l.config.ShouldUseLogForwarder || forceLogForwarder {
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
		logger.Raw(payload)
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
		l.AddDistributionMetric(fmt.Sprintf("aws.lambda.enhanced.%s", metricName), 1, time.Now(), true, tags...)
	}
}

func getEnhancedMetricsTags(ctx context.Context) []string {
	isColdStart := ctx.Value("cold_start")

	if lc, ok := lambdacontext.FromContext(ctx); ok {
		// ex: arn:aws:lambda:us-east-1:123497558138:function:golang-layer:alias
		splitArn := strings.Split(lc.InvokedFunctionArn, ":")

		// malformed arn string
		if len(splitArn) < 5 {
			logger.Debug("malformed arn string in the LambdaContext")
			return []string{}
		}

		var alias string
		var executedVersion string

		functionName := fmt.Sprintf("functionname:%s", lambdacontext.FunctionName)
		region := fmt.Sprintf("region:%s", splitArn[3])
		accountId := fmt.Sprintf("account_id:%s", splitArn[4])
		memorySize := fmt.Sprintf("memorysize:%d", lambdacontext.MemoryLimitInMB)
		coldStart := fmt.Sprintf("cold_start:%t", isColdStart.(bool))
		resource := fmt.Sprintf("resource:%s", lambdacontext.FunctionName)
		datadogLambda := fmt.Sprintf("datadog_lambda:%s", datadogLambdaVersion)

		tags := []string{functionName, region, accountId, memorySize, coldStart, datadogLambda}

		// Check if our slice contains an alias or version
		if len(splitArn) > 7 {
			alias = splitArn[7]

			// If we have an alias...
			switch alias != "" {
			// If the alias is $Latest, drop the $ for ddog tag conventio
			case strings.HasPrefix(alias, "$"):
				alias = strings.TrimPrefix(alias, "$")
			// If this is not a version number, we will have an alias and executed version
			case isNotNumeric(alias):
				executedVersion = fmt.Sprintf("executedversion:%s", lambdacontext.FunctionVersion)
				tags = append(tags, executedVersion)
			}

			resource = fmt.Sprintf("resource:%s:%s", lambdacontext.FunctionName, alias)
		}

		tags = append(tags, resource)

		return tags
	}

	logger.Debug("could not retrieve the LambdaContext from Context")
	return []string{}
}

func isNotNumeric(s string) bool {
	_, err := strconv.ParseInt(s, 0, 64)
	return err != nil
}

func isServerlessAgentRunning() bool {
	client := &http.Client{Timeout: AgentCallTimeout}
	req, _ := http.NewRequest(http.MethodGet, "http://localhost:8124/lambda/hello", nil)
	if response, err := client.Do(req); err == nil && response.StatusCode == 200 {
		logger.Debug("Will use the Serverless Agent")
		return true
	}
	logger.Debug("Will use the API")
	return false
}

func flushServerlessAgent() error {
	client := &http.Client{Timeout: AgentCallTimeout}
	req, _ := http.NewRequest(http.MethodGet, "http://localhost:8124/lambda/flush", nil)
	if response, err := client.Do(req); err != nil {
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
