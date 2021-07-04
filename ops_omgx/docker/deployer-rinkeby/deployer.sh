#!/bin/bash

# Copyright Optimism PBC 2020
# MIT License
# github.com/ethereum-optimism

export DEPLOYER_PRIVATE_KEY=`/opt/secret2env -name $SECRETNAME|grep -w DEPLOYER_PRIVATE_KEY|sed 's/DEPLOYER_PRIVATE_KEY=//g'`
export FRAUD_PROOF_WINDOW_SECONDS=`/opt/secret2env -name $SECRETNAME|grep -w FRAUD_PROOF_WINDOW_SECONDS|sed 's/FRAUD_PROOF_WINDOW_SECONDS=//g'`
export HARDHAT=`/opt/secret2env -name $SECRETNAME|grep -w HARDHAT|sed 's/HARDHAT=//g'`
export L1_NODE_WEB3_URL=`/opt/secret2env -name $SECRETNAME|grep -w L1_NODE_WEB3_URL|sed 's/L1_NODE_WEB3_URL=//g'`
export SEQUENCER_PRIVATE_KEY=`/opt/secret2env -name $SECRETNAME|grep -w SEQUENCER_PRIVATE_KEY|sed 's/SEQUENCER_PRIVATE_KEY=//g'`
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
