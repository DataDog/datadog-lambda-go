# dd-lambda-go

## Usage

Datadog needs to be able to read context from the incoming Lambda event, in order to enable distributed tracing.
Wrap your lambda handler like so.

```go
package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/DataDog/dd-lambda-go/ddlambda"
)

func myHandler() (string, error) {
	return "Success", nil
}

func main() {
    lambda.Start( ddlambda.WrapHandler(myHandler))
}
```