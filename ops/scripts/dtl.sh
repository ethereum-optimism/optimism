#!/bin/bash
RETRIES=${RETRIES:-60}

# get the addrs from the URL provided
ADDRESSES=$(curl --silent --retry-connrefused --retry $RETRIES --retry-delay 5 $URL)
# set the env
export DATA_TRANSPORT_LAYER__ADDRESS_MANAGER=$(echo $ADDRESSES | jq -r '.AddressManager')

# go
exec node dist/src/services/run.js
