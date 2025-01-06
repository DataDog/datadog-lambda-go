module github.com/DataDog/datadog-lambda-go

go 1.22.7

toolchain go1.23.1

require (
	github.com/DataDog/datadog-go/v5 v5.5.1-0.20240822164813-20af2dbfabbb
	github.com/aws/aws-lambda-go v1.46.1-0.20240416201810-90a3af70ddf8
	github.com/aws/aws-sdk-go-v2/config v1.27.34-0.20240913182458-171151bb0fd1
	github.com/aws/aws-sdk-go-v2/service/kms v1.35.8-0.20240913182458-171151bb0fd1
	github.com/aws/aws-xray-sdk-go v1.8.5-0.20240715031132-eaa92cef11b1
	github.com/cenkalti/backoff/v4 v4.2.1
	github.com/sony/gobreaker v0.5.0
	github.com/stretchr/testify v1.9.1-0.20240613125739-84619f5c3cc3
	go.opentelemetry.io/otel v1.30.1-0.20240913071937-80e18a584123
	gopkg.in/DataDog/dd-trace-go.v1 v1.68.0-rc.2
)

require go.uber.org/atomic v1.11.0 // indirect

require (
	github.com/DataDog/appsec-internal-go v1.7.0 // indirect
	github.com/DataDog/datadog-agent/pkg/obfuscate v0.59.0-devel.0.20240913161137-39cd38632c79 // indirect
	github.com/DataDog/datadog-agent/pkg/remoteconfig/state v0.59.0-devel.0.20240914012957-10d974e4d276 // indirect
	github.com/DataDog/go-libddwaf/v3 v3.4.0 // indirect
	github.com/DataDog/go-sqllexer v0.0.15-0.20240906194926-cbc90c6bc0a4 // indirect
	github.com/DataDog/go-tuf v1.1.0-0.5.2 // indirect
	github.com/DataDog/sketches-go v1.4.7-0.20240802104016-7546f8f95179 // indirect
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/andybalholm/brotli v1.1.1-0.20240729165604-57434b509141 // indirect
	github.com/aws/aws-sdk-go v1.55.6-0.20240912145455-7112c0a0c2d0 // indirect
	github.com/aws/aws-sdk-go-v2 v1.30.6-0.20240913182458-171151bb0fd1 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.33-0.20240912182535-1b644bfdcae8 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.14-0.20240913182458-171151bb0fd1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.18-0.20240913182458-171151bb0fd1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.18-0.20240913182458-171151bb0fd1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.2-0.20240913182458-171151bb0fd1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.11.5-0.20240913182458-171151bb0fd1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.11.20-0.20240913182458-171151bb0fd1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.22.8-0.20240906182417-827d25db0048 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.26.8-0.20240913182458-171151bb0fd1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.30.8-0.20240913182458-171151bb0fd1 // indirect
	github.com/aws/smithy-go v1.20.4 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/ebitengine/purego v0.8.0-alpha.5.0.20240903150804-6580f25cf0bb // indirect
	github.com/go-logr/logr v1.4.3-0.20240902060449-275154abd02f // indirect
	github.com/go-logr/stdr v1.2.3-0.20220714215701-1fa2ed3fdf83 // indirect
	github.com/google/uuid v1.6.1-0.20240806143717-0e97ed3b5379 // indirect
	github.com/hashicorp/go-secure-stdlib/parseutil v0.1.9-0.20240903214937-914d7625fe0f // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.3-0.20240903214937-914d7625fe0f // indirect
	github.com/hashicorp/go-sockaddr v1.0.7-0.20240718200401-8187f9b97d0d // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/mitchellh/mapstructure v1.5.1-0.20230418172516-63cde0dfe248 // indirect
	github.com/outcaste-io/ristretto v0.2.3 // indirect
	github.com/philhofer/fwd v1.1.3-0.20240612014219-fbbf4953d986 // indirect
	github.com/pkg/errors v0.9.2-0.20201214064552-5dd12d0cfe7f // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/secure-systems-lab/go-securesystemslib v0.8.0 // indirect
	github.com/tinylib/msgp v1.2.1 // indirect
	github.com/valyala/bytebufferpool v1.0.1-0.20201104193830-18533face0df // indirect
	github.com/valyala/fasthttp v1.55.1-0.20240910180552-65e989e8b8bc // indirect
	go.opentelemetry.io/otel/metric v1.30.1-0.20240913071937-80e18a584123 // indirect
	go.opentelemetry.io/otel/trace v1.30.1-0.20240913071937-80e18a584123 // indirect
	golang.org/x/mod v0.21.0 // indirect
	golang.org/x/net v0.29.1-0.20240906182658-3c333c0c5288 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/sys v0.25.1-0.20240909193319-d58f986c8984 // indirect
	golang.org/x/text v0.18.1-0.20240911022905-38a95c2d4a4b // indirect
	golang.org/x/time v0.6.0 // indirect
	golang.org/x/tools v0.25.1-0.20240913183314-91d4bdb347ba // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240903143218-8af14fe29dc1 // indirect
	google.golang.org/grpc v1.68.0-dev.0.20240913164237-31ffeeeb001c // indirect
	google.golang.org/protobuf v1.34.3-0.20240906163944-03df6c145d96 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
