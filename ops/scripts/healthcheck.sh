#!/bin/bash

set -e

RETRIES=${RETRIES:-60}

# wait for reference RPC to be up
curl \
    --fail \
    --show-error \
    --silent \
    --output /dev/null \
    --retry-connrefused \
    --retry $RETRIES \
    --retry-delay 1 \
    $HEALTHCHECK__REFERENCE_RPC_PROVIDER

# wait for target RPC to be up
curl \
    --fail \
    --show-error \
    --silent \
    --output /dev/null \
    --retry-connrefused \
    --retry $RETRIES \
    --retry-delay 1 \
    $HEALTHCHECK__TARGET_RPC_PROVIDER

# go
exec yarn start
