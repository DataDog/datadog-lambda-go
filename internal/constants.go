package internal

const (
	traceIDHeader          = "x-datadog-trace-id"
	parentIDHeader         = "x-datadog-parent-id"
	samplingPriorityHeader = "x-datadog-sampling-priority"
)

const (
	userReject = "-1"
	autoReject = "0"
	autoKeep   = "1"
	userKeep   = "2"
)

const (
	xraySubsegmentName      = "datadog-metadata"
	xraySubsegmentKey       = "trace"
	xraySubsegmentNamespace = "datadog"
)
