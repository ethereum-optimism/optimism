#!/bin/bash

# Copyright Optimism PBC 2020
# MIT License
# github.com/ethereum-optimism

export ADDRESS_MANAGER_ADDRESS=`/opt/secret2env -name $SECRETNAME|grep -w ADDRESS_MANAGER_ADDRESS|sed 's/ADDRESS_MANAGER_ADDRESS=//g'`
export L1_BLOCK_OFFSET=`/opt/secret2env -name $SECRETNAME|grep -w L1_BLOCK_OFFSET|sed 's/L1_BLOCK_OFFSET=//g'`
export L1_NODE_WEB3_URL=`/opt/secret2env -name $SECRETNAME|grep -w L1_NODE_WEB3_URL|sed 's/L1_NODE_WEB3_URL=//g'`
export L1_WALLET_KEY=`/opt/secret2env -name $SECRETNAME|grep -w L1_WALLET_KEY|sed 's/L1_WALLET_KEY=//g'`
export WHITELIST_ENDPOINT=`/opt/secret2env -name $SECRETNAME|grep -w WHITELIST_ENDPOINT|sed 's/WHITELIST_ENDPOINT=//g'`

cmd="node /opt/optimism-ts-services/build/exec/run.js"

JSON='{"jsonrpc":"2.0","id":0,"method":"net_version","params":[]}'

RETRIES=${RETRIES:-120}
until $(curl --silent --fail \
    --output /dev/null \
    -H "Content-Type: application/json" \
    --data "$JSON" "$L1_NODE_WEB3_URL"); do
  sleep 5
  echo "Will wait $((RETRIES--)) more times for L1 $L1_NODE_WEB3_URL to be up..."

  if [ "$RETRIES" -lt 0 ]; then
    echo "Timeout waiting for layer one node at $L1_NODE_WEB3_URL"
    exit 1
  fi
done
echo "Connected to L1 Node at $L1_NODE_WEB3_URL"

RETRIES=${RETRIES:-30}
until $(curl --silent --fail \
    --output /dev/null \
    -H "Content-Type: application/json" \
    --data "$JSON" "$L2_NODE_WEB3_URL"); do
  sleep 5
  echo "Will wait $((RETRIES--)) more times for L2 $L2_NODE_WEB3_URL to be up..."

  if [ "$RETRIES" -lt 0 ]; then
    echo "Timeout waiting for layer two node at $L2_NODE_WEB3_URL"
    exit 1
  fi
done
echo "Connected to L2 Node at $L2_NODE_WEB3_URL"

if [ ! -z "$DEPLOYER_HTTP" ]; then
    RETRIES=${RETRIES:-20}
    until $(curl --silent --fail \
        --output /dev/null \
        "$DEPLOYER_HTTP/addresses.json"); do
      sleep 5
      echo "Will wait $((RETRIES--)) more times for DEPLOYER $DEPLOYER_HTTP to be up..."

      if [ "$RETRIES" -lt 0 ]; then
        echo "Timeout waiting for contract deployment"
        exit 1
      fi
    done
    echo "Contracts are deployed"
    ADDRESS_MANAGER_ADDRESS=$(curl --silent $DEPLOYER_HTTP/addresses.json | jq -r .AddressManager)
    exec env \
        ADDRESS_MANAGER_ADDRESS=$ADDRESS_MANAGER_ADDRESS \
        L1_BLOCK_OFFSET=$L1_BLOCK_OFFSET \
        L1_NODE_WEB3_URL=`/opt/secret2env -name $SECRETNAME|grep -w L1_NODE_WEB3_URL|sed 's/L1_NODE_WEB3_URL=//g'` \
        L1_WALLET_KEY=`/opt/secret2env -name $SECRETNAME|grep -w L1_WALLET_KEY|sed 's/L1_WALLET_KEY=//g'` \
        WHITELIST_ENDPOINT=`/opt/secret2env -name $SECRETNAME|grep -w WHITELIST_ENDPOINT|sed 's/WHITELIST_ENDPOINT=//g'` \
        $cmd
else
   $cmd
fi
