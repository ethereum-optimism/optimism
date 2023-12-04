#!/bin/bash

# Check if a directory path is provided
# Directory must contain the output of batch_decoder's reassemble command
if [ -z "$1" ]; then
    echo "Usage: $0 /path/to/directory"
    exit 1
fi

directory_path=$1
valid_count=0
invalid_count=0
invalid_channels=()

# Loop over every .json file in the specified directory
for file in "$directory_path"/*.json; do
    # If channel is ready, all batches must valid.
    # If delta is activated, all batch types must be span batch.
    result=$(jq 'if .is_ready then (.invalid_batches == false and all(.batch_types[]; . == 1)) else empty end' "$file")
    if [[ $result == "true" ]]; then
        ((valid_count++))
    elif [[ $result == "false" ]]; then
        ((invalid_count++))
        invalid_channels+=("$file")
    fi
done

# Display the counts
echo "Valid count: $valid_count"
echo "Invalid count: $invalid_count"

# Display the files that returned invalid results
if [ ${#invalid_channels[@]} -gt 0 ]; then
    echo "Channels with invalid results:"
    printf '[*] %s\n' "${invalid_channels[@]}"
else
    echo "All processed channels are valid and contains successfully derived span batches."
fi
