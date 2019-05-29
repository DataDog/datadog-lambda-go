# dd-lambda-go

Datadog's Lambda Go client library enables distributed tracing between serverful and serverless environments.

## Installation

```bash
go get github.com/DataDog/dd-lambda-go
```

The following Datadog environment variables should be defined via the AWS CLI or Serverless Framework:

- DATADOG_API_KEY
- DATADOG_APP_KEY

## Usage

Datadog needs to be able to read headers from the incoming Lambda event. Wrap your Lambda handler function like so:

```go
package main

import (
  "github.com/aws/aws-lambda-go/lambda"
  "github.com/DataDog/dd-lambda-go"
)

func main() {
  // Wrap your lambda handler like this
  lambda.Start( ddlambda.WrapHandler(myHandler, nil))
  /* OR with manual configuration options
  lambda.Start(ddlambda.WrapHandler(myHandler, &ddlambda.Config{
    BatchInterval: time.Seconds * 15
    APIKey: "my-api-key",
    AppKey: "my-app-key",
  }))
  */
}

func myHandler(ctx context.Context, event MyEvent) (string, error) {
  // ...
}
```

## Custom Metrics

Custom metrics can be submitted using the `DistributionMetric` function. The metrics are submitted as [distribution metrics](https://docs.datadoghq.com/graphing/metrics/distributions/).

```go
ddlambda.Distribution(
  ctx, // Use the context object, (or child), that was passed into your handler
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
  req, err := http.NewRequest("GET", "http://api.youcompany.com/status")
  // Use the same Context object given to your lambda handler.
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

If you find an issue with this package and have a fix, please feel free to open a pull request following the procedures.

## License

Unless explicitly stated otherwise all files in this repository are licensed under the Apache License Version 2.0.

This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2019 Datadog, Inc.