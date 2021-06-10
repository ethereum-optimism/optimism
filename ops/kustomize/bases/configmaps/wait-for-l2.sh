#!/bin/bash
set -eou
if [[ -z $L2_NODE_WEB3_URL ]]; then
    echo "Must pass L2_NODE_WEB3_URL"
    exit 1
fi
JSON='{"jsonrpc":"2.0","id":0,"method":"eth_chainId","params":[]}'
echo "Waiting for L2"
curl \
    -X POST \
    --silent \
    --output /dev/null \
    --retry-connrefused \
    --retry 1000 \
    --retry-delay 1 \
    -d "$JSON" \
    $L2_NODE_WEB3_URL
echo "Connected to L2"