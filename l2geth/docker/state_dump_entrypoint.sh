#!/bin/sh

HOSTNAME=${HOSTNAME:-0.0.0.0}
PORT=${PORT:-8545}
NETWORK_ID=${NETWORK_ID:-420}
VOLUME_PATH=${VOLUME_PATH:-/mnt/l2geth}

TARGET_GAS_LIMIT=${TARGET_GAS_LIMIT:-8000000}

echo "Starting Geth in debug mode"
## Command to kick off geth
geth --dev \
  --datadir $VOLUME_PATH \
  --rpc \
  --rpcaddr $HOSTNAME \
  --rpcvhosts='*' \
  --rpccorsdomain='*' \
  --rpcport $PORT \
  --networkid $NETWORK_ID \
  --ipcdisable \
  --rpcapi 'debug' \
  --gasprice '0' \
  --targetgaslimit $TARGET_GAS_LIMIT \
  --nousb \
  --gcmode=archive \
  --verbosity "6" \
  --txingestion.enable=false
