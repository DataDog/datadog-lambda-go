package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/lambda"

	ddlambda "github.com/DataDog/datadog-lambda-go"
	"github.com/aws/aws-lambda-go/events"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

var (
	invokeCount = 0
)

func handleRequest(ctx context.Context, ev events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	req, _ := http.NewRequest("GET", "https://www.datadoghq.com", nil)
	ddlambda.AddTraceHeaders(ctx, req)
	client := http.Client{}
	client.Do(req)

	headers := ddlambda.GetTraceHeaders(ctx)

	ddlambda.Distribution("hello-go.dog", float64(invokeCount))
	invokeCount++

	fmt.Println("Start Logging Trace Headers")
	fmt.Println("x-datadog-parent-id:" + headers["x-datadog-parent-id"])
	fmt.Println("x-datadog-trace-id:" + headers["x-datadog-trace-id"])
	fmt.Println("x-datadog-sampling-priority:" + headers["x-datadog-sampling-priority"])
	fmt.Println("End Logging Trace Headers")

	// Test tracing
	for i := 0; i < 10; i++ {
		s := tracer.StartSpanFromContext(ctx, "child.span")
		time.Sleep(100 * time.Millisecond)
		s.Finish()
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "hello, dog!",
	}, nil
}

func main() {
	lambda.Start(ddlambda.WrapHandler(handleRequest, nil))
}
