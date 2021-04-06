#!/bin/bash
RETRIES=${RETRIES:-20}

JSON='{"jsonrpc":"2.0","id":0,"method":"net_version","params":[]}'

# wait for the base layer to be up
curl -H "Content-Type: application/json" --retry-connrefused --retry $RETRIES --retry-delay 1 -d $JSON $L1_NODE_WEB3_URL
# get the addrs to a var
ADDRESSES=$(yarn run --silent deploy)
# sent them to the file
echo $ADDRESSES > dist/dumps/addresses.json
# serve the addrs and the state dump
./bin/serve_dump.sh
