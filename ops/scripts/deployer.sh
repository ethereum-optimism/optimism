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

if [ -n "$DTL_REGISTRY_URL" ] ; then
  echo "Will upload addresses.json to DTL"
  curl \
      --fail \
      --show-error \
      --silent \
      -H "Content-Type: application/json" \
      --retry-connrefused \
      --retry $RETRIES \
      --retry-delay 5 \
      -T dist/dumps/addresses.json \
      "$DTL_REGISTRY_URL"
fi

# serve the addrs and the state dump
exec ./bin/serve_dump.sh
