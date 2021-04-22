module github.com/DataDog/datadog-lambda-go/tests/integration_tests/bin/error

go 1.13

require (
	github.com/DataDog/datadog-lambda-go v0.7.0
	github.com/aws/aws-lambda-go v1.11.1
	gopkg.in/DataDog/dd-trace-go.v1 v1.30.0
)

replace github.com/DataDog/datadog-lambda-go => ../../../
