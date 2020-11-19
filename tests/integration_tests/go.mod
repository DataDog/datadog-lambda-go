module github.com/stroem/datadog-lambda-go/tests/integration_tests/bin/hello

go 1.12

require (
	github.com/stroem/datadog-lambda-go v0.7.0
	github.com/aws/aws-lambda-go v1.11.1
	gopkg.in/DataDog/dd-trace-go.v1 v1.27.0
)

replace github.com/stroem/datadog-lambda-go => ../../
