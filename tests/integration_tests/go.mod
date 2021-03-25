module github.com/DataDog/datadog-lambda-go/tests/integration_tests/bin/hello

go 1.12

require (
	github.com/DataDog/datadog-lambda-go v0.7.0
	github.com/aws/aws-lambda-go v1.11.1
	gopkg.in/DataDog/dd-trace-go.v1 v1.27.0
)

replace github.com/DataDog/datadog-lambda-go => ../../
replace gopkg.in/DataDog/dd-trace-go.v1 => /Users/nicolas.hinsch/go/src/github.com/DataDog/dd-trace-go
