module github.com/DataDog/datadog-lambda-go/tests/integration_tests/bin/hello

go 1.12

require (
	github.com/DataDog/datadog-lambda-go v0.7.0
	github.com/andybalholm/brotli v1.0.3 // indirect
	github.com/aws/aws-lambda-go v1.26.0
	gopkg.in/DataDog/dd-trace-go.v1 v1.27.0
	gopkg.in/urfave/cli.v1 v1.20.0 // indirect
)

replace github.com/DataDog/datadog-lambda-go => ../../
