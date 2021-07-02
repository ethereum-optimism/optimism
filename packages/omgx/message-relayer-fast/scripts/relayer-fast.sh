#!/bin/bash

set -e

RETRIES=${RETRIES:-60}

if [[ ! -z "$URL" ]]; then
    # get the addrs from the URL provided
    ADDRESSES=$(curl --fail --show-error --silent --retry-connrefused --retry $RETRIES --retry-delay 5 $URL)
    # set the env
    export ADDRESS_MANAGER_ADDRESS=$(echo $ADDRESSES | jq -r '.AddressManager')
fi

if [[ ! -z "$OMGX_URL" ]]; then
    # get the addrs from the URL provided
    ADDRESSES=$(curl --fail --show-error --silent --retry-connrefused --retry $RETRIES --retry-delay 5 $OMGX_URL)
    # set the env
    export L1_MESSENGER_FAST=$(echo $ADDRESSES | jq -r '.OVM_L1CrossDomainMessengerFast')
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
exec node ./exec/run-message-relayer-fast.js