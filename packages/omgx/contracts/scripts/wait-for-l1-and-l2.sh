#!/bin/bash

# Copyright Optimism PBC 2020
# MIT License
# github.com/ethereum-optimism

cmd="$@"
JSON='{"jsonrpc":"2.0","id":0,"method":"net_version","params":[]}'

RETRIES=${RETRIES:-50}
until $(curl --silent --fail \
    --output /dev/null \
    -H "Content-Type: application/json" \
    --data "$JSON" "$L1_NODE_WEB3_URL"); do
  sleep 1
  echo "Will wait $((RETRIES--)) more times for $L1_NODE_WEB3_URL to be up..."

  if [ "$RETRIES" -lt 0 ]; then
    echo "Timeout waiting for layer one node at $L1_NODE_WEB3_URL"
    exit 1
  fi
done
echo "Connected to L1 Node at $L1_NODE_WEB3_URL"

RETRIES=${RETRIES:-50} #BUG BUG BUG - if the above takes a lot of retires, then this loop will run out of retries
until $(curl --silent --fail \
    --output /dev/null \
    -H "Content-Type: application/json" \
    --data "$JSON" "$L2_NODE_WEB3_URL"); do
  sleep 1
  echo "Will wait $((RETRIES--)) more times for $L2_NODE_WEB3_URL to be up..."

  if [ "$RETRIES" -lt 0 ]; then
    echo "Timeout waiting for layer two node at $L2_NODE_WEB3_URL"
    exit 1
  fi
done
echo "Connected to L2 Node at $L2_NODE_WEB3_URL"

if [ ! -z "$URL" ]; then
    RETRIES=${RETRIES:-50}
    until $(curl --silent --fail \
        --output /dev/null \
        "$URL"); do
      sleep 1
      echo "Will wait $((RETRIES--)) more times for $URL to be up..."

      if [ "$RETRIES" -lt 0 ]; then
        echo "Timeout waiting for contract deployment"
        exit 1
      fi
    done
    echo "Rollup contracts are deployed"
    ADDRESS_MANAGER_ADDRESS=$(curl --silent $URL | jq -r .AddressManager)
    exec env \
        ADDRESS_MANAGER_ADDRESS=$ADDRESS_MANAGER_ADDRESS \
        $cmd
else
    exec $cmd
fi
