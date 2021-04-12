#!/bin/bash
RETRIES=${RETRIES:-40}

# get the addrs from the URL provided
ADDRESSES=$(curl --retry-connrefused --retry $RETRIES --retry-delay 5 $URL)
# set the env
export ADDRESS_MANAGER_ADDRESS=$(echo $ADDRESSES | jq -r '.AddressManager')

# waits for l2geth to be up
curl --retry-connrefused --retry $RETRIES --retry-delay 1 $L2_NODE_WEB3_URL

# go
node ./exec/run-batch-submitter.js
