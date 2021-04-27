#!/bin/bash
RETRIES=${RETRIES:-60}

if [[ ! -z "$URL" ]]; then
    # get the addrs from the URL provided
    ADDRESSES=$(curl --silent --retry-connrefused --retry $RETRIES --retry-delay 5 $URL)
    # set the env
    export ADDRESS_MANAGER_ADDRESS=$(echo "$ADDRESSES" | jq -r '.AddressManager')
fi

# waits for l2geth to be up
curl \
    --silent \
    --output /dev/null \
    --retry-connrefused \
    --retry $RETRIES \
    --retry-delay 1 \
    $L2_NODE_WEB3_URL

# go
exec node ./exec/run-message-relayer.js
