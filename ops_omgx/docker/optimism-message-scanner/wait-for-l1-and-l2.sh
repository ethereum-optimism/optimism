#!/bin/bash

# Copyright Optimism PBC 2020
# MIT License
# github.com/ethereum-optimism

export DEPLOYER_PRIVATE_KEY=`/opt/secret2env -name $SECRETNAME|grep -w DEPLOYER_PRIVATE_KEY|sed 's/DEPLOYER_PRIVATE_KEY=//g'`
export L1_NODE_WEB3_URL=`/opt/secret2env -name $SECRETNAME|grep -w L1_NODE_WEB3_URL|sed 's/L1_NODE_WEB3_URL=//g'`
export MYSQL_DATABASE_NAME=`/opt/secret2env -name $SECRETNAME|grep -w MYSQL_DATABASE_NAME|sed 's/MYSQL_DATABASE_NAME=//g'`
export MYSQL_HOST_URL=`/opt/secret2env -name $SECRETNAME|grep -w MYSQL_HOST_URL|sed 's/MYSQL_HOST_URL=//g'`
export MYSQL_PASSWORD=`/opt/secret2env -name $SECRETNAME|grep -w MYSQL_PASSWORD|sed 's/MYSQL_PASSWORD=//g'`
export MYSQL_PORT=`/opt/secret2env -name $SECRETNAME|grep -w MYSQL_PORT|sed 's/MYSQL_PORT=//g'`
export MYSQL_USERNAME=`/opt/secret2env -name $SECRETNAME|grep -w MYSQL_USERNAME|sed 's/MYSQL_USERNAME=//g'`

cmd="node /usr/local/bin/run-L2ToL1Message-scanner.js"
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
  sleep 1
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
      sleep 1
      echo "Will wait $((RETRIES--)) more times for $DEPLOYER_HTTP to be up..."

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
        $cmd
else
    $cmd
fi
