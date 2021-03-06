package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/lambda"

	ddlambda "github.com/DataDog/datadog-lambda-go"
	"github.com/aws/aws-lambda-go/events"
	httptrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/net/http"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func handleRequest(ctx context.Context, ev events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	currentSpan, _ := tracer.SpanFromContext(ctx)
	currentSpanContext := currentSpan.Context()
	fmt.Println("Current span ID: " + strconv.FormatUint(currentSpanContext.SpanID(), 10))
	fmt.Println("Current trace ID: " + strconv.FormatUint(currentSpanContext.TraceID(), 10))

	// HTTP request
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://www.datadoghq.com", nil)
	client := http.Client{}
	client = *httptrace.WrapClient(&client)
	client.Do(req)

	// Metric
	ddlambda.Distribution("hello-go.dog", 1)

	// User-defined span
	for i := 0; i < 10; i++ {
		s, _ := tracer.StartSpanFromContext(ctx, "child.span")
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
