# datadog-lambda-go

[![CircleCI](https://img.shields.io/circleci/build/github/DataDog/datadog-lambda-go)](https://circleci.com/gh/DataDog/datadog-lambda-go)
[![Code Coverage](https://img.shields.io/codecov/c/github/DataDog/datadog-lambda-go)](https://codecov.io/gh/DataDog/datadog-lambda-go)
[![Slack](https://img.shields.io/badge/slack-%23serverless-blueviolet?logo=slack)](https://datadoghq.slack.com/channels/serverless/)
[![Godoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/DataDog/datadog-lambda-go)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue)](https://github.com/DataDog/datadog-lambda-go/blob/master/LICENSE)

Datadog's Lambda Go client library enables distributed tracing between serverful and serverless environments.

## Installation

```bash
go get github.com/DataDog/datadog-lambda-go
```

You can set the following environment variables via the AWS CLI or Serverless Framework

### DD_API_KEY

Your datadog API key

### DD_SITE

Which Datadog site to use. Set this to `datadoghq.eu` to send your data to the Datadog EU site.

### DD_LOG_LEVEL

How much logging datadog-lambda-go should do. Set this to "debug" for extensive logs.

### DD_FLUSH_TO_LOG

If your Lambda function powers a performance-critical task (e.g., a consumer-facing API). You can avoid the added latencies of metric-submitting API calls, by setting this value to true. Custom metrics will be submitted asynchronously through CloudWatch Logs and the Datadog log forwarder.

### DD_ENHANCED_METRICS

The Lambda layer will increment a Lambda integration metric called `aws.lambda.enhanced.invocations` with each invocation and `aws.lambda.enhanced.errors` if the invocation results in an error. These metrics are tagged with the function name, region, account, runtime, memorysize, and `cold_start:true|false`. This is enabled by default.

### DD_TAGS

A comma-delimetered list of tag:value pairs which are added to all metrics emitted (ie. in addition to any tags used when calling the `.Metric()` function).

Example - This will add 2 tags to all metrics emitted from this runtime:
```
DD_TAGS=layer:api,team:intake
```

## Usage

Datadog needs to be able to read headers from the incoming Lambda event. Wrap your Lambda handler function like so:

```go
package main

import (
  "github.com/aws/aws-lambda-go/lambda"
  "github.com/DataDog/datadog-lambda-go"
)

func main() {
  // Wrap your lambda handler like this
  lambda.Start(ddlambda.WrapHandler(myHandler, nil))
  /* OR with manual configuration options
  lambda.Start(ddlambda.WrapHandler(myHandler, &ddlambda.Config{
    BatchInterval: time.Second * 15
    APIKey: "my-api-key",
  }))
  */
}

func myHandler(ctx context.Context, event MyEvent) (string, error) {
  // ...
}
```

## Custom Metrics

Custom metrics can be submitted using the `Metric` function. The metrics are submitted as [distribution metrics](https://docs.datadoghq.com/graphing/metrics/distributions/).

**IMPORTANT NOTE:** If you have already been submitting the same custom metric as non-distribution metric (e.g., gauge, count, or histogram) without using the datadog-lambda-go lib, you MUST pick a new metric name to use for `ddlambda.Metric`. Otherwise that existing metric will be converted to a distribution metric and the historical data prior to the conversion will be no longer queryable.

```go
ddlambda.Metric(
  "coffee_house.order_value", // Metric name
  12.45, // The value
  "product:latte", "order:online" // Associated tags
)
```

### VPC

If your Lambda function is associated with a VPC, you need to ensure it has access to the [public internet](https://aws.amazon.com/premiumsupport/knowledge-center/internet-access-lambda-function/).

## Distributed Tracing

[Distributed tracing](https://docs.datadoghq.com/tracing/guide/distributed_tracing/?tab=python) allows you to propagate a trace context from a service running on a host to a service running on AWS Lambda, and vice versa, so you can see performance end-to-end. Linking is implemented by injecting Datadog trace context into the HTTP request headers.

Distributed tracing headers are language agnostic, e.g., a trace can be propagated between a Java service running on a host to a Lambda function written in Go.

Because the trace context is propagated through HTTP request headers, the Lambda function needs to be triggered by AWS API Gateway or AWS Application Load Balancer.

To enable this feature, make sure any outbound requests have Datadog's tracing headers.

```go
  req, err := http.NewRequest("GET", "http://example.com/status")
  // Use the same Context object given to your lambda handler.
  // If you don't want to pass the context through your call hierarchy, you can use ddlambda.GetContext()
  ddlambda.AddTraceHeaders(ctx, req)

  client := http.Client{}
  client.Do(req)
}
```

## Sampling

The traces for your Lambda function are converted by Datadog from AWS X-Ray traces. X-Ray needs to sample the traces that the Datadog tracing agent decides to sample, in order to collect as many complete traces as possible. You can create X-Ray sampling rules to ensure requests with header `x-datadog-sampling-priority:1` or `x-datadog-sampling-priority:2` via API Gateway always get sampled by X-Ray.

These rules can be created using the following AWS CLI command.

```bash
aws xray create-sampling-rule --cli-input-json file://datadog-sampling-priority-1.json
aws xray create-sampling-rule --cli-input-json file://datadog-sampling-priority-2.json
```

The file content for `datadog-sampling-priority-1.json`:

```json
{
  "SamplingRule": {
    "RuleName": "Datadog-Sampling-Priority-1",
    "ResourceARN": "*",
    "Priority": 9998,
    "FixedRate": 1,
    "ReservoirSize": 100,
    "ServiceName": "*",
    "ServiceType": "AWS::APIGateway::Stage",
    "Host": "*",
    "HTTPMethod": "*",
    "URLPath": "*",
    "Version": 1,
    "Attributes": {
      "x-datadog-sampling-priority": "1"
    }
  }
}
```

The file content for `datadog-sampling-priority-2.json`:

```json
{
  "SamplingRule": {
    "RuleName": "Datadog-Sampling-Priority-2",
    "ResourceARN": "*",
    "Priority": 9999,
    "FixedRate": 1,
    "ReservoirSize": 100,
    "ServiceName": "*",
    "ServiceType": "AWS::APIGateway::Stage",
    "Host": "*",
    "HTTPMethod": "*",
    "URLPath": "*",
    "Version": 1,
    "Attributes": {
      "x-datadog-sampling-priority": "2"
    }
  }
}
```

## Non-proxy integration

If your Lambda function is triggered by API Gateway via the non-proxy integration, then you have to set up a mapping template, which passes the Datadog trace context from the incoming HTTP request headers to the Lambda function via the event object.

If your Lambda function is deployed by the Serverless Framework, such a mapping template gets created by default.

## Opening Issues

If you encounter a bug with this package, we want to hear about it. Before opening a new issue, search the existing issues to avoid duplicates.

When opening an issue, include the Datadog Lambda Layer version, Python version, and stack trace if available. In addition, include the steps to reproduce when appropriate.

You can also open an issue for a feature request.

## Contributing

If you find an issue with this package and have a fix, please feel free to open a pull request following the [procedures](https://github.com/DataDog/dd-lambda-go/blob/master/CONTRIBUTING.md).

## License

Unless explicitly stated otherwise all files in this repository are licensed under the Apache License Version 2.0.

This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2019 Datadog, Inc.
