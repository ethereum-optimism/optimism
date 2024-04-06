#!/bin/bash

set -euo pipefail

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <test name> <json file>"
    exit 1
fi

TEST_NAME="$1"
JSON_FILE="$2"

jq --raw-output --join-output --arg testName "${TEST_NAME}" 'select(.Test == $testName and .Action == "output").Output' "${JSON_FILE}"
