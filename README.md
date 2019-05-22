# dd-lambda-go

Datadog's Lambda Go client library enables distributed tracing between serverful and serverless environments.

## Installation

```bash
go get github.com/DataDog/dd-lambda-go
```

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
  lambda.Start( ddlambda.WrapHandler(myHandler))
}

func myHandler(ctx context.Context, event MyEvent) (string, error) {
  // ...
}
```

Make sure any outbound requests have Datadog's tracing headers.

```go
  req, err := http.NewRequest("GET", "http://api.youcompany.com/status")
  // Use the same Context object given to your lambda handler.
  ddlambda.AddTraceHeaders(ctx, req)

  client := http.Client{}
  client.Do(req)
}
```

## Non-proxy integration

If your Lambda function is triggered by API Gateway via the non-proxy integration, then you have to set up a mapping template, which passes the Datadog trace context from the incoming HTTP request headers to the Lambda function via the event object.

If your Lambda function is deployed by the Serverless Framework, such a mapping template gets created by default.