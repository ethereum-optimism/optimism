#!/bin/bash
set -e

RETRIES=${RETRIES:-20}
JSON='{"jsonrpc":"2.0","id":0,"method":"net_version","params":[]}'

# wait for the base layer to be up
curl \
    --fail \
    --show-error \
    --silent \
    -H "Content-Type: application/json" \
    --retry-connrefused \
    --retry $RETRIES \
    --retry-delay 1 \
    -d $JSON \
    $L1_NODE_WEB3_URL

yarn run deploy

# serve the addrs and the state dump
exec ./bin/serve_dump.sh
