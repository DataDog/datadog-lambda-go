package main

import (
	"context"
	"errors"

	"github.com/aws/aws-lambda-go/lambda"

	ddlambda "github.com/DataDog/datadog-lambda-go"
	"github.com/aws/aws-lambda-go/events"
)

func handleRequest(ctx context.Context, ev events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: 500,
		Body:       "error",
	}, errors.New("something went wrong")
}

func main() {
	lambda.Start(ddlambda.WrapHandler(handleRequest, nil))
}
