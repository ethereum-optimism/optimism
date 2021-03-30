#!/bin/sh

## Passed in from environment variables:
HOSTNAME=${HOSTNAME:-0.0.0.0}
PORT=${PORT:-8545}
NETWORK_ID=${NETWORK_ID:-420}
VOLUME_PATH=${VOLUME_PATH:-/mnt/l2geth}

CLEAR_DATA_FILE_PATH="${VOLUME_PATH}/.clear_data_key_${CLEAR_DATA_KEY}"
TARGET_GAS_LIMIT=${TARGET_GAS_LIMIT:-8000000}

ETH1_SYNC_SERVICE=${ETH1_SYNC_SERVICE:-true}
ETH1_CTC_DEPLOYMENT_HEIGHT=${ETH1_CTC_DEPLOYMENT_HEIGHT-:1}
ETH1_ADDRESS_RESOLVER_ADDRESS=${ETH1_ADDRESS_RESOLVER_ADDRESS:-0x0000000000000000000000000000000000000000}
ETH1_CHAINID=${ETH1_CHAINID:-1}
ETH1_NETWORKID=${ETH1_NETWORKID:-1}

if [ $ETH1_SYNC_SERVICE == 'false' ]; then
    ETH1_HTTP=''
fi

echo "Starting Sequencer"
geth --dev \
    --datadir $VOLUME_PATH \
    --rpc \
    --rpcaddr $HOSTNAME \
    --rpcvhosts='*' \
    --rpccorsdomain='*' \
    --rpcport $PORT \
    --networkid $NETWORK_ID \
    --ipcdisable \
    --rpcapi 'eth,net,rollup' \
    --gasprice '0' \
    --targetgaslimit $TARGET_GAS_LIMIT \
    --nousb \
    --gcmode=archive \
    --verbosity "6" \
    --eth1.syncservice=$ETH1_SYNC_SERVICE \
    --eth1.ctcdeploymentheight $ETH1_CTC_DEPLOYMENT_HEIGHT \
    --eth1.addressresolveraddress $ETH1_ADDRESS_RESOLVER_ADDRESS \
    --eth1.chainid $ETH1_CHAINID \
    --eth1.networkid $ETH1_NETWORKID \
    --eth1.http $ETH1_HTTP
