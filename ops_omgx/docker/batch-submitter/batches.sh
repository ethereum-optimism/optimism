#!/bin/bash

# Copyright Optimism PBC 2020
# MIT License
# github.com/ethereum-optimism

export SEQUENCER_PRIVATE_KEY=`/opt/secret2env -name $SECRETNAME|grep -w SEQUENCER_PRIVATE_KEY|sed 's/SEQUENCER_PRIVATE_KEY=//g'`
export PROPOSER_PRIVATE_KEY=`/opt/secret2env -name $SECRETNAME|grep -w PROPOSER_PRIVATE_KEY|sed 's/PROPOSER_PRIVATE_KEY=//g'`
export CLEAR_PENDING_TXS=`/opt/secret2env -name $SECRETNAME|grep -w CLEAR_PENDING_TXS|sed 's/CLEAR_PENDING_TXS=//g'`
export DEBUG=`/opt/secret2env -name $SECRETNAME|grep -w DEBUG|sed 's/DEBUG=//g'`
export FINALITY_CONFIRMATIONS=`/opt/secret2env -name $SECRETNAME|grep -w FINALITY_CONFIRMATIONS|sed 's/FINALITY_CONFIRMATIONS=//g'`
export L1_NODE_WEB3_URL=`/opt/secret2env -name $SECRETNAME|grep -w L1_NODE_WEB3_URL|sed 's/L1_NODE_WEB3_URL=//g'`
export MAX_BATCH_SUBMISSION_TIME=`/opt/secret2env -name $SECRETNAME|grep -w MAX_BATCH_SUBMISSION_TIME|sed 's/MAX_BATCH_SUBMISSION_TIME=//g'`
export MAX_L1_TX_SIZE=`/opt/secret2env -name $SECRETNAME|grep -w MAX_L1_TX_SIZE|sed 's/MAX_L1_TX_SIZE=//g'`
export MAX_STATE_BATCH_COUNT=`/opt/secret2env -name $SECRETNAME|grep -w MAX_STATE_BATCH_COUNT|sed 's/MAX_STATE_BATCH_COUNT=//g'`
export MAX_TX_BATCH_COUNT=`/opt/secret2env -name $SECRETNAME|grep -w MAX_TX_BATCH_COUNT|sed 's/MAX_TX_BATCH_COUNT=//g'`
export MIN_L1_TX_SIZE=`/opt/secret2env -name $SECRETNAME|grep -w MIN_L1_TX_SIZE|sed 's/MIN_L1_TX_SIZE=//g'`
export NUM_CONFIRMATIONS=`/opt/secret2env -name $SECRETNAME|grep -w NUM_CONFIRMATIONS|sed 's/NUM_CONFIRMATIONS=//g'`
export POLL_INTERVAL=`/opt/secret2env -name $SECRETNAME|grep -w POLL_INTERVAL|sed 's/POLL_INTERVAL=//g'`
export RESUBMISSION_TIMEOUT=`/opt/secret2env -name $SECRETNAME|grep -w RESUBMISSION_TIMEOUT|sed 's/RESUBMISSION_TIMEOUT=//g'`
export RUN_STATE_BATCH_SUBMITTER=`/opt/secret2env -name $SECRETNAME|grep -w RUN_STATE_BATCH_SUBMITTER|sed 's/RUN_STATE_BATCH_SUBMITTER=//g'`
export RUN_TX_BATCH_SUBMITTER=`/opt/secret2env -name $SECRETNAME|grep -w RUN_TX_BATCH_SUBMITTER|sed 's/RUN_TX_BATCH_SUBMITTER=//g'`
export SAFE_MINIMUM_ETHER_BALANCE=`/opt/secret2env -name $SECRETNAME|grep -w SAFE_MINIMUM_ETHER_BALANCE|sed 's/SAFE_MINIMUM_ETHER_BALANCE=//g'`
export ADDRESS_MANAGER_ADDRESS=`/opt/secret2env -name $SECRETNAME|grep -w ADDRESS_MANAGER_ADDRESS|sed 's/ADDRESS_MANAGER_ADDRESS=//g'`

set -e

RETRIES=${RETRIES:-40}
JSON='{"jsonrpc":"2.0","id":0,"method":"net_version","params":[]}'

if [[ ! -z "$URL" ]]; then
    # get the addrs from the URL provided
    ADDRESSES=$(curl --fail --show-error --silent --retry-connrefused --retry $RETRIES --retry-delay 5 $URL)
    # set the env
    export ADDRESS_MANAGER_ADDRESS=$(echo $ADDRESSES | jq -r '.AddressManager')
fi

# waits for l2geth to be up
curl --silent --fail \
    --output /dev/null \
    -retry-connrefused \
    --retry $RETRIES \
    --retry-delay 1 \
    -H "Content-Type: application/json" \
    --data "$JSON" "$L2_NODE_WEB3_URL"

# go
exec node ./exec/run-batch-submitter.js
