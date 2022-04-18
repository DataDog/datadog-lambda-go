module github.com/DataDog/datadog-lambda-go/tests/integration_tests/bin/error

go 1.17

require (
	github.com/DataDog/datadog-lambda-go v0.7.0
	github.com/aws/aws-lambda-go v1.11.1
)

replace github.com/DataDog/datadog-lambda-go => ../../../
