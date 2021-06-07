#!/bin/bash

# Copyright Optimism PBC 2020
# MIT License
# github.com/ethereum-optimism

export DATA_TRANSPORT_LAYER__ADDRESS_MANAGER=`/opt/secret2env -name $SECRETNAME|grep -w DATA_TRANSPORT_LAYER__ADDRESS_MANAGER|sed 's/DATA_TRANSPORT_LAYER__ADDRESS_MANAGER=//g'`
export DATA_TRANSPORT_LAYER__L2_CHAIN_ID=`/opt/secret2env -name $SECRETNAME|grep -w DATA_TRANSPORT_LAYER__L2_CHAIN_ID|sed 's/DATA_TRANSPORT_LAYER__L2_CHAIN_ID=//g'`
export DATA_TRANSPORT_LAYER__CONFIRMATIONS=`/opt/secret2env -name $SECRETNAME|grep -w DATA_TRANSPORT_LAYER__CONFIRMATIONS|sed 's/DATA_TRANSPORT_LAYER__CONFIRMATIONS=//g'`
export DATA_TRANSPORT_LAYER__DANGEROUSLY_CATCH_ALL_ERRORS=`/opt/secret2env -name $SECRETNAME|grep -w DATA_TRANSPORT_LAYER__DANGEROUSLY_CATCH_ALL_ERRORS|sed 's/DATA_TRANSPORT_LAYER__DANGEROUSLY_CATCH_ALL_ERRORS=//g'`
export DATA_TRANSPORT_LAYER__DB_PATH=`/opt/secret2env -name $SECRETNAME|grep -w DATA_TRANSPORT_LAYER__DB_PATH|sed 's/DATA_TRANSPORT_LAYER__DB_PATH=//g'`
export DATA_TRANSPORT_LAYER__L1_RPC_ENDPOINT=`/opt/secret2env -name $SECRETNAME|grep -w DATA_TRANSPORT_LAYER__L1_RPC_ENDPOINT|sed 's/DATA_TRANSPORT_LAYER__L1_RPC_ENDPOINT=//g'`
export DATA_TRANSPORT_LAYER__LOGS_PER_POLLING_INTERVAL=`/opt/secret2env -name $SECRETNAME|grep -w DATA_TRANSPORT_LAYER__LOGS_PER_POLLING_INTERVAL|sed 's/DATA_TRANSPORT_LAYER__LOGS_PER_POLLING_INTERVAL=//g'`
export DATA_TRANSPORT_LAYER__POLLING_INTERVAL=`/opt/secret2env -name $SECRETNAME|grep -w DATA_TRANSPORT_LAYER__POLLING_INTERVAL|sed 's/DATA_TRANSPORT_LAYER__POLLING_INTERVAL=//g'`
export DATA_TRANSPORT_LAYER__SERVER_HOSTNAME=`/opt/secret2env -name $SECRETNAME|grep -w DATA_TRANSPORT_LAYER__SERVER_HOSTNAME|sed 's/DATA_TRANSPORT_LAYER__SERVER_HOSTNAME=//g'`
export DATA_TRANSPORT_LAYER__SYNC_FROM_L1=`/opt/secret2env -name $SECRETNAME|grep -w DATA_TRANSPORT_LAYER__SYNC_FROM_L1|sed 's/DATA_TRANSPORT_LAYER__SYNC_FROM_L1=//g'`
export DATA_TRANSPORT_LAYER__SYNC_FROM_L2=`/opt/secret2env -name $SECRETNAME|grep -w DATA_TRANSPORT_LAYER__SYNC_FROM_L2|sed 's/DATA_TRANSPORT_LAYER__SYNC_FROM_L2=//g'`
export DATA_TRANSPORT_LAYER__TRANSACTIONS_PER_POLLING_INTERVAL=`/opt/secret2env -name $SECRETNAME|grep -w DATA_TRANSPORT_LAYER__TRANSACTIONS_PER_POLLING_INTERVAL|sed 's/DATA_TRANSPORT_LAYER__TRANSACTIONS_PER_POLLING_INTERVAL=//g'`
export L1_NODE_WEB3_URL=`/opt/secret2env -name $SECRETNAME|grep -w L1_NODE_WEB3_URL|sed 's/L1_NODE_WEB3_URL=//g'`

rm -rf /db/LOCK
cmd="yarn start"

JSON='{"jsonrpc":"2.0","id":0,"method":"eth_chainId","params":[]}'
L1_NODE_WEB3_URL=$DATA_TRANSPORT_LAYER__L1_RPC_ENDPOINT
L2_NODE_WEB3_URL=$DATA_TRANSPORT_LAYER__L2_RPC_ENDPOINT
DATA_TRANSPORT_LAYER__L2_CHAIN_ID=$DATA_TRANSPORT_LAYER__L2_CHAIN_ID

if [[ "$DATA_TRANSPORT_LAYER__SYNC_FROM_L1" == true ]]; then
    if [[ -z "$L1_NODE_WEB3_URL" ]]; then
        echo "Missing DATA_TRANSPORT_LAYER__L1_RPC_ENDPOINT env var"
        exit 1
    fi
fi

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

if [[ "$DATA_TRANSPORT_LAYER__SYNC_FROM_L2" == true ]]; then
    if [[ -z "$L2_NODE_WEB3_URL" ]]; then
        echo "Missing DATA_TRANSPORT_LAYER__L2_RPC_ENDPOINT env var"
        exit 1
    fi

    RETRIES=${RETRIES:-20}
    until $(curl --silent --fail \
        --output /dev/null \
        -H "Content-Type: application/json" \
        --data "$JSON" "$L2_NODE_WEB3_URL"); do
      sleep 5
      echo "Will wait $((RETRIES--)) more times for L2 $L2_NODE_WEB3_URL to be up..."

      if [ "$RETRIES" -lt 0 ]; then
        echo "Timeout waiting for layer one node at $L2_NODE_WEB3_URL"
        exit 1
      fi
    done
    echo "Connected to L2 Node at $L2_NODE_WEB3_URL"

    DATA_TRANSPORT_LAYER__L2_CHAIN_ID=$(curl --silent -H \
        "Content-Type: application/json" \
        --data '{"jsonrpc":"2.0","id":0,"method":"eth_chainId","params":[]}' \
        "$L2_NODE_WEB3_URL" | jq -r .result | xargs printf '%d')
fi

if [ ! -z "$DEPLOYER_HTTP" ]; then
    RETRIES=${RETRIES:-20}
    until $(curl --silent --fail \
        --output /dev/null \
        "$DEPLOYER_HTTP/addresses.json"); do
      sleep 5
      echo "Will wait $((RETRIES--)) more times for DEPLOYER $DEPLOYER_HTTP to be up..."

      if [ "$RETRIES" -lt 0 ]; then
        echo "Timeout waiting for address list from $DEPLOYER_HTTP"
        exit 1
      fi
    done
    echo "Received address list from $DEPLOYER_HTTP"

    DATA_TRANSPORT_LAYER__ADDRESS_MANAGER=$(curl --silent $DEPLOYER_HTTP/addresses.json | jq -r .AddressManager)
    exec env \
        $cmd
else
    exec env
    $cmd
fi
