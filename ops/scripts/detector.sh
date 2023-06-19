#!/bin/bash

set -e

RETRIES=${RETRIES:-60}

# waits for l2geth to be up
curl \
    --fail \
    --show-error \
    --silent \
    --output /dev/null \
    --retry-connrefused \
    --retry $RETRIES \
    --retry-delay 1 \
    $FAULT_DETECTOR__L2_RPC_PROVIDER

# go
exec yarn start
