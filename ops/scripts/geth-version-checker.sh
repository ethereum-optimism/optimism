#!/bin/bash

SCRIPTS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
MONOREPO_DIR=$(cd "$SCRIPTS_DIR/../../" && pwd)

# Extract the version from the geth command output
GETH_VERSION="v$(geth version | grep '^Version:' | awk '{print $2}')"

# Read the version from the versions file
EXPECTED_GETH_VERSION=$(jq -r .geth < "$MONOREPO_DIR"/versions.json)

# Check if EXPECTED_GETH_VERSION contains a '-'. If not, append '-stable'.
if [[ $EXPECTED_GETH_VERSION != *-* ]]; then
    EXPECTED_GETH_VERSION="${EXPECTED_GETH_VERSION}-stable"
fi

# Compare the versions
if [[ "$GETH_VERSION" == "$EXPECTED_GETH_VERSION" ]]; then
    echo "Geth version $GETH_VERSION is correct!"
    exit 0
else
    echo "Geth version does not match!"
    echo "Local geth version: $GETH_VERSION"
    echo "Expected geth version: $EXPECTED_GETH_VERSION"
    exit 1
fi


