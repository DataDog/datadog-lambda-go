# dd-lambda-go

## Usage

Datadog needs to be able to read headers from the incoming Lambda event, in order to add datadog metadata to the go context.
Wrap your lambda handler like so.

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
	// Add headers to outbound using the context object
	req, err := http.NewRequest("GET", "http://api.youcompany.com/status")
	ddlambda.AddTraceHeaders(ctx)

	client := http.Client{}
	client.Do(req)

	return "Success", nil
}
```