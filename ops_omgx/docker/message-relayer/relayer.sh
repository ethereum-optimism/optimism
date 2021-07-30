#!/bin/bash

set -x
export ADDRESS_MANAGER_ADDRESS=`/opt/secret2env -name $SECRETNAME|grep -w ADDRESS_MANAGER_ADDRESS|sed 's/ADDRESS_MANAGER_ADDRESS=//g'`
export L1_BLOCK_OFFSET=`/opt/secret2env -name $SECRETNAME|grep -w L1_BLOCK_OFFSET|sed 's/L1_BLOCK_OFFSET=//g'`
export L1_NODE_WEB3_URL=`/opt/secret2env -name $SECRETNAME|grep -w L1_NODE_WEB3_URL|sed 's/L1_NODE_WEB3_URL=//g'`
export RELAYER_PRIVATE_KEY=`/opt/secret2env -name $SECRETNAME|grep -w RELAY_L1_WALLET_PRIV_KEY	|sed 's/RELAY_L1_WALLET_PRIV_KEY=//g'`
export BLACKLIST_ENDPOINT=`/opt/secret2env -name $SECRETNAME|grep -w BLACKLIST_ENDPOINT|sed 's/BLACKLIST_ENDPOINT=//g'`


RETRIES=${RETRIES:-60}

if [[ ! -z "$URL" ]]; then
    # get the addrs from the URL provided
    ADDRESSES=$(curl --fail --show-error --silent --retry-connrefused --retry $RETRIES --retry-delay 5 $URL)
    export ADDRESS_MANAGER_ADDRESS=$(echo $ADDRESSES | jq -r '.AddressManager')
fi

# waits for l2geth to be up
curl \
    --fail \
    --show-error \
    --silent \
    --output /dev/null \
    --retry-connrefused \
    --retry $RETRIES \
    --retry-delay 1 \
    $L2_NODE_WEB3_URL

# go
exec /opt/optimism/packages/message-relayer/exec/run-message-relayer-fast.js
