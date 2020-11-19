# datadog-lambda-go

![build](https://github.com/stroem/datadog-lambda-go/workflows/build/badge.svg)
[![Code Coverage](https://img.shields.io/codecov/c/github/DataDog/datadog-lambda-go)](https://codecov.io/gh/DataDog/datadog-lambda-go)
[![Slack](https://img.shields.io/badge/slack-%23serverless-blueviolet?logo=slack)](https://datadoghq.slack.com/channels/serverless/)
[![Godoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/stroem/datadog-lambda-go)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue)](https://github.com/stroem/datadog-lambda-go/blob/master/LICENSE)

Datadog Lambda Library for Go enables enhanced Lambda metrics, distributed tracing, and custom metric submission from AWS Lambda functions.  

## Installation

Follow the installation instructions [here](https://docs.datadoghq.com/serverless/installation/go/).

## Enhanced Metrics

Once [installed](#installation), you should be able to view enhanced metrics for your Lambda function in Datadog.

Check out the official documentation on [Datadog Lambda enhanced metrics](https://docs.datadoghq.com/integrations/amazon_lambda/?tab=go#real-time-enhanced-lambda-metrics).

## Custom Metrics

Once [installed](#installation), you should be able to submit custom metrics from your Lambda function.

Check out the instructions for [submitting custom metrics from AWS Lambda functions](https://docs.datadoghq.com/integrations/amazon_lambda/?tab=go#custom-metrics).

## Tracing

Set the `DD_TRACE_ENABLED` environment variable to `true` to enable Datadog tracing. When Datadog tracing is enabled, the library will inject a span representing the Lambda's execution into the context object. You can then use the included `dd-trace-go` package to create additional spans from the context or pass the context to other services. For more information, see the [dd-trace-go documentation](https://godoc.org/gopkg.in/DataDog/dd-trace-go.v1/ddtrace).

```
import (
  "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
  httptrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/net/http"
)

func handleRequest(ctx context.Context, ev events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
  // Trace an HTTP request
  req, _ := http.NewRequestWithContext(ctx, "GET", "https://www.datadoghq.com", nil)
  client := http.Client{}
  client = *httptrace.WrapClient(&client)
  client.Do(req)

  // Create a custom span
  s, _ := tracer.StartSpanFromContext(ctx, "child.span")
  time.Sleep(100 * time.Millisecond)
  s.Finish()
}
```

You can also use the injected span to [connect your logs and traces](https://docs.datadoghq.com/tracing/connect_logs_and_traces/go/).

```
func handleRequest(ctx context.Context, ev events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
  currentSpan, _ := tracer.SpanFromContext(ctx)
  log.Printf("my log message %v", currentSpan)
}
```

If you are also using AWS X-Ray to trace your Lambda functions, you can set the `DD_MERGE_XRAY_TRACES` environment variable to `true`, and Datadog will merge your Datadog and X-Ray traces into a single, unified trace.


## Environment Variables

### DD_FLUSH_TO_LOG

Set to `true` (recommended) to send custom metrics asynchronously (with no added latency to your Lambda function executions) through CloudWatch Logs with the help of [Datadog Forwarder](https://github.com/DataDog/datadog-serverless-functions/tree/master/aws/logs_monitoring). Defaults to `false`. If set to `false`, you also need to set `DD_API_KEY` and `DD_SITE`.

### DD_API_KEY

If `DD_FLUSH_TO_LOG` is set to `false` (not recommended), the Datadog API Key must be defined.

### DD_SITE

If `DD_FLUSH_TO_LOG` is set to `false` (not recommended), and your data need to be sent to the Datadog EU site, you must set `DD_SITE` to `datadoghq.eu`. Defaults to `datadoghq.com`.

### DD_LOG_LEVEL

Set to `debug` enable debug logs from the Datadog Lambda Library. Defaults to `info`.

### DD_ENHANCED_METRICS

Generate enhanced Datadog Lambda integration metrics, such as, `aws.lambda.enhanced.invocations` and `aws.lambda.enhanced.errors`. Defaults to `true`.

### DD_TRACE_ENABLED

Initialize the Datadog tracer when set to `true`. Defaults to `false`.

### DD_MERGE_XRAY_TRACES

If you are using both X-Ray and Datadog tracing, set this to `true` to merge the X-Ray and Datadog traces. Defaults to `false`.

## Opening Issues

If you encounter a bug with this package, we want to hear about it. Before opening a new issue, search the existing issues to avoid duplicates.

When opening an issue, include the datadog-lambda-go version, `go version`, and stack trace if available. In addition, include the steps to reproduce when appropriate.

You can also open an issue for a feature request.

## Contributing

If you find an issue with this package and have a fix, please feel free to open a pull request following the [procedures](https://github.com/DataDog/dd-lambda-go/blob/master/CONTRIBUTING.md).

## License

Unless explicitly stated otherwise all files in this repository are licensed under the Apache License Version 2.0.

This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2020 Datadog, Inc.
