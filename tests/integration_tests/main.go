package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/lambda"

	ddlambda "github.com/DataDog/datadog-lambda-go"
	"github.com/aws/aws-lambda-go/events"
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

	fmt.Println("Start Logging Headers")
	for key, value := range headers {
		fmt.Printf("Request header: %s: %s\n", key, value)
	}
	fmt.Println("End Logging Headers")

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "hello, dog!",
	}, nil
}

func main() {
	lambda.Start(ddlambda.WrapHandler(handleRequest, nil))
}
