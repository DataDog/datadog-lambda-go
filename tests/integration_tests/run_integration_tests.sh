#!/bin/bash

# Usage - run commands from the /integration_tests directory:
# To check if new changes to the library cause changes to any snapshots:
#   DD_API_KEY=XXXX aws-vault exec sandbox-account-admin -- ./run_integration_tests.sh
# To regenerate snapshots:
#   UPDATE_SNAPSHOTS=true DD_API_KEY=XXXX aws-vault exec sandbox-account-admin -- ./run_integration_tests.sh

set -e

# These values need to be in sync with serverless.yml, where there needs to be a function
# defined for every handler_runtime combination
LAMBDA_HANDLERS=("hello-go")

LOGS_WAIT_SECONDS=20

integration_tests_dir=$(cd `dirname $0` && pwd)
echo $integration_tests_dir

script_start_time=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

mismatch_found=false

if [ -z "$DD_API_KEY" ]; then
    echo "No DD_API_KEY env var set, exiting"
    exit 1
fi

if [ -n "$UPDATE_SNAPSHOTS" ]; then
    echo "Overwriting snapshots in this execution"
fi

echo "Building Go binary"
GOOS=linux go build -ldflags="-s -w" -o bin/hello

echo "Deploying function"
sls deploy --api-key $DD_API_KEY

cd $integration_tests_dir

input_event_files=$(ls ./input_events)
# Sort event files by name so that snapshots stay consistent
input_event_files=($(for file_name in ${input_event_files[@]}; do echo $file_name; done | sort))

echo "Invoking functions"
set +e # Don't exit this script if an invocation fails or there's a diff
for input_event_file in "${input_event_files[@]}"; do
    for function_name in "${LAMBDA_HANDLERS[@]}"; do
        # Get event name without trailing ".json" so we can build the snapshot file name
        input_event_name=$(echo "$input_event_file" | sed "s/.json//")
        # Return value snapshot file format is snapshots/return_values/{handler}_{runtime}_{input-event}
        snapshot_path="$integration_tests_dir/snapshots/return_values/${function_name}_${input_event_name}.json"

        return_value=$(sls invoke -f $function_name --path "$integration_tests_dir/input_events/$input_event_file" --api-key=$DD_API_KEY)

        if [ ! -f $snapshot_path ]; then
            # If the snapshot file doesn't exist yet, we create it
            echo "Writing return value to $snapshot_path because no snapshot exists yet"
            echo "$return_value" >$snapshot_path
        elif [ -n "$UPDATE_SNAPSHOTS" ]; then
            # If $UPDATE_SNAPSHOTS is set to true, write the new logs over the current snapshot
            echo "Overwriting return value snapshot for $snapshot_path"
            echo "$return_value" >$snapshot_path
        else
            # Compare new return value to snapshot
            diff_output=$(echo "$return_value" | diff - $snapshot_path)
            if [ $? -eq 1 ]; then
                echo "Failed: Return value for $function_name does not match snapshot:"
                echo "$diff_output"
                mismatch_found=true
            else
                echo "Ok: Return value for $function_name with $input_event_name event matches snapshot"
            fi
        fi
    done
done
set -e

echo "Sleeping $LOGS_WAIT_SECONDS seconds to wait for logs to appear in CloudWatch..."
sleep $LOGS_WAIT_SECONDS

echo "Fetching logs for invocations and comparing to snapshots"
for function_name in "${LAMBDA_HANDLERS[@]}"; do
    function_snapshot_path="./snapshots/logs/$function_name.log"

    # Fetch logs with serverless cli
    raw_logs=$(serverless logs -f $function_name --startTime $script_start_time)

    # Replace invocation-specific data like timestamps and IDs with XXXX to normalize logs across executions
    logs=$(
        echo "$raw_logs" |
            # Filter serverless cli errors
            sed '/Serverless: Recoverable error occurred/d' |
            # Normalize Lambda runtime report logs
            sed -E 's/(RequestId|TraceId|SegmentId|Duration|Memory Used|"e"):( )?[a-z0-9\.\-]+/\1:\2XXXX/g' |
            # Normalize DD APM headers and AWS account ID
            sed -E "s/(x-datadog-parent-id:|x-datadog-trace-id:|account_id:) ?[0-9]+/\1XXXX/g" |
            # Strip API key from logged requests
            sed -E "s/(api_key=|'api_key': ')[a-z0-9\.\-]+/\1XXXX/g" |
            # Normalize ISO combined date-time
            sed -E "s/[0-9]{4}\-[0-9]{2}\-[0-9]{2}(T?)[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]+ \(\-?[0-9:]+\))?Z/XXXX-XX-XXTXX:XX:XX.XXXZ/" |
            # Normalize log timestamps
            sed -E "s/[0-9]{4}(\-|\/)[0-9]{2}(\-|\/)[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]+( \(\-?[0-9:]+\))?)?/XXXX-XX-XX XX:XX:XX.XXX/" |
            # Normalize DD trace ID injection
            sed -E "s/(dd\.trace_id=)[0-9]+ (dd\.span_id=)[0-9]+/\1XXXX \2XXXX/" |
            # Normalize execution ID in logs prefix
            sed -E $'s/[0-9a-z]+\-[0-9a-z]+\-[0-9a-z]+\-[0-9a-z]+\-[0-9a-z]+\t/XXXX-XXXX-XXXX-XXXX-XXXX\t/' |
            # Normalize minor package version tag so that these snapshots aren't broken on version bumps
            sed -E "s/(dd_lambda_layer:datadog-go[0-9]+\.)[0-9]+\.[0-9]+/\1XX\.X/g" |
            # Normalize data in logged traces
            sed -E 's/"(span_id|parent_id|trace_id|start|duration|tcp\.local\.address|tcp\.local\.port|dns\.address|request_id|function_arn)":("?)[a-zA-Z0-9\.:\-]+("?)/"\1":\2XXXX\3/g' |
            # Normalize data in logged traces
            sed -E 's/"(points\\\":\[\[)([0-9]+)/\1XXXX/g'

    )

    if [ ! -f $function_snapshot_path ]; then
        # If no snapshot file exists yet, we create one
        echo "Writing logs to $function_snapshot_path because no snapshot exists yet"
        echo "$logs" >$function_snapshot_path
    elif [ -n "$UPDATE_SNAPSHOTS" ]; then
        # If $UPDATE_SNAPSHOTS is set to true write the new logs over the current snapshot
        echo "Overwriting log snapshot for $function_snapshot_path"
        echo "$logs" >$function_snapshot_path
    else
        # Compare new logs to snapshots
        set +e # Don't exit this script if there is a diff
        diff_output=$(echo "$logs" | diff - $function_snapshot_path)
        if [ $? -eq 1 ]; then
            echo "Failed: Mismatch found between new $function_name logs (first) and snapshot (second):"
            echo "$diff_output"
            mismatch_found=true
        else
            echo "Ok: New logs for $function_name match snapshot"
        fi
        set -e
    fi
done

if [ "$mismatch_found" = true ]; then
    echo "FAILURE: A mismatch between new data and a snapshot was found and printed above."
    echo "If the change is expected, generate new snapshots by running 'UPDATE_SNAPSHOTS=true DD_API_KEY=XXXX ./scripts/run_integration_tests.sh'"
    exit 1
fi

if [ -n "$UPDATE_SNAPSHOTS" ]; then
    echo "SUCCESS: Wrote new snapshots for all functions"
    exit 0
fi

echo "SUCCESS: No difference found between snapshots and new return values or logs"
