#!/bin/bash

# Extract the version from the geth command output
GETH_VERSION="v$(geth version | grep '^Version:' | awk '{print $2}')"

# Read the version from the .gethrc file
GETHRC_VERSION=$(cat .gethrc)

# Check if GETHRC_VERSION contains a '-'. If not, append '-stable'.
if [[ $GETHRC_VERSION != *-* ]]; then
    GETHRC_VERSION="${GETHRC_VERSION}-stable"
fi

# Compare the versions
if [[ "$GETH_VERSION" == "$GETHRC_VERSION" ]]; then
    echo "Geth version $GETH_VERSION is correct!"
    exit 0
else
    echo "Geth version does not match!"
    echo "geth version: $GETH_VERSION"
    echo ".gethrc version: $GETHRC_VERSION"
    exit 1
fi
