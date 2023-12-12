#!/bin/bash
set -uo pipefail

# Check if a directory path is provided
# Directory must contain the output of batch_decoder's reassemble command, which are channels in json form
if [ -z "${1:-}" ]; then
    echo "Usage: $0 /path/to/directory"
    exit 1
fi

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo "Error: jq is not installed"
    exit 1
fi

directory_path=$1
# Check if directory exists and is not empty
if [ ! -d "$directory_path" ] || [ -z "$(ls -A "$directory_path")" ]; then
    echo "Error: Directory does not exist or is empty"
    exit 1
fi

invalid_json_count=0
invalid_jsons=()

not_ready_channel_count=0
not_ready_channels=()
ready_channel_count=0

span_batch_count=0
singular_batch_count=0

channels_with_invalid_batches=()

batch_type_counter_jq_script='reduce .batch_types[] as $batch_type (
    {"span_batch_count": 0, "singular_batch_count": 0};
    if $batch_type == 1 then
        .span_batch_count += 1
    else
        .singular_batch_count += 1
    end) | .span_batch_count, .singular_batch_count'

# Loop over every .json file in the specified directory
for file in "$directory_path"/*.json; do
    # check file is valid json
    if ! jq empty "$file" 2>/dev/null; then
        ((invalid_json_count++))
        invalid_jsons+=("$file")
        continue
    fi
    # check channels are ready
    if [ $(jq -r ".is_ready" "$file") == "false" ] ; then
        # not ready channel have no batches so invalid_batches field is always false
        ((not_ready_channel_count++))
        not_ready_channels+=("$file")
        continue
    else
        ((ready_channel_count++))
    fi
    # check channels contain invalid batches
    if [ $(jq -r ".invalid_batches" "$file") == "true" ] ; then
        channels_with_invalid_batches+=("$file")
    fi
    # count singular batch count and span batch count
    jq_result=$(jq "$batch_type_counter_jq_script" "$file")
    read span_batch_count_per_channel singular_batch_count_per_channel <<< $jq_result
    span_batch_count=$((span_batch_count+span_batch_count_per_channel))
    singular_batch_count=$((singular_batch_count+singular_batch_count_per_channel))
done

# Display the counts
echo "Singular batch count: $singular_batch_count"
echo "Span batch count: $span_batch_count"
echo "Ready channel count: $ready_channel_count"
echo "Not ready channel count: $not_ready_channel_count"
echo "Invalid json count: $invalid_json_count"

# Display the files which are invalid jsons
if [ ${#invalid_jsons[@]} -gt 0 ]; then
    echo "Invalid jsons"
    printf '[*] %s\n' "${invalid_jsons[@]}"
else
    echo "All processed channels are valid jsons."
fi

# Display the files which are channels not ready
if [ ${#not_ready_channels[@]} -gt 0 ]; then
    echo "Not ready channels"
    printf '[*] %s\n' "${not_ready_channels[@]}"
else
    echo "All processed channels are ready."
fi

# Display the files including invalid batches
if [ ${#channels_with_invalid_batches[@]} -gt 0 ]; then
    echo "Channels with invalid batches"
    printf '[*] %s\n' "${channels_with_invalid_batches[@]}"
else
    echo "All processed ready channels contain valid batches."
fi
