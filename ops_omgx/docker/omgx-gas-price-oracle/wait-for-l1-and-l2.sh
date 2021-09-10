#!/bin/sh
export GAS_PRICE_ORACLE_ADDRESS=`/opt/secret2env -name $SECRETNAME|grep -w GAS_PRICE_ORACLE_ADDRESS|sed 's/GAS_PRICE_ORACLE_ADDRESS=//g'`
export DEPLOYER_PRIVATE_KEY=`/opt/secret2env -name $SECRETNAME|grep -w DEPLOYER_PRIVATE_KEY|sed 's/DEPLOYER_PRIVATE_KEY=//g'`
export SEQUENCER_PRIVATE_KEY=`/opt/secret2env -name $SECRETNAME|grep -w SEQUENCER_PRIVATE_KEY|sed 's/SEQUENCER_PRIVATE_KEY=//g'`
export PROPOSER_PRIVATE_KEY=`/opt/secret2env -name $SECRETNAME|grep -w PROPOSER_PRIVATE_KEY|sed 's/PROPOSER_PRIVATE_KEY=//g'`
export RELAYER_PRIVATE_KEY=`/opt/secret2env -name $SECRETNAME|grep -w RELAYER_PRIVATE_KEY|sed 's/RELAYER_PRIVATE_KEY=//g'`
export FAST_RELAYER_PRIVATE_KEY=`/opt/secret2env -name $SECRETNAME|grep -w FAST_RELAYER_PRIVATE_KEY|sed 's/FAST_RELAYER_PRIVATE_KEY=//g'`
export ETHERSCAN_API=`/opt/secret2env -name $SECRETNAME|grep -w ETHERSCAN_API|sed 's/ETHERSCAN_API=//g'`
export L1_NODE_WEB3_URL=`/opt/secret2env -name $SECRETNAME|grep -w L1_NODE_WEB3_URL|sed 's/L1_NODE_WEB3_URL=//g'`
export GAS_PRICE_ORACLE_OWNER_PRIVATE_KEY=`/opt/secret2env -name $SECRETNAME|grep -w ROLLUP_GAS_PRICE_ORACLE_OWNER_ADDRESS|sed 's/ROLLUP_GAS_PRICE_ORACLE_OWNER_ADDRESS=//g'`
RETRIES=${RETRIES:-40}

cmd="$@"
JSON='{"jsonrpc":"2.0","id":0,"method":"net_version","params":[]}'

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
    exec $cmd
fi
