#!/bin/sh

# Exits if any command fails
set -e

HOSTNAME=${HOSTNAME:-0.0.0.0}
PORT=${PORT:-8545}
NETWORK_ID=${NETWORK_ID:-420}
VOLUME_PATH=${VOLUME_PATH:-/mnt/l2geth}

TARGET_GAS_LIMIT=${TARGET_GAS_LIMIT:-8000000}

TX_INGESTION=${TX_INGESTION:-false}
TX_INGESTION_DB_HOST=${TX_INGESTION_DB_HOST:-localhost}
TX_INGESTION_POLL_INTERVAL=${TX_INGESTION_POLL_INTERVAL:-3s}
TX_INGESTION_DB_USER=${TX_INGESTION_DB_USER:-test}
TX_INGESTION_DB_PASSWORD=${TX_INGESTION_DB_PASSWORD:-test}

echo "Starting Geth..."
./build/bin/geth --dev \
    --datadir $VOLUME_PATH \
    --rpc \
    --rpcaddr $HOSTNAME \
    --rpcvhosts='*' \
    --rpccorsdomain='*' \
    --rpcport $PORT \
    --ipcdisable \
    --networkid $NETWORK_ID \
    --rpcapi 'eth,net,rollup' \
    --gasprice '0' \
    --targetgaslimit $TARGET_GAS_LIMIT \
    --nousb \
    --gcmode=archive \
    --verbosity "6" \
    --txingestion.enable="$TX_INGESTION" \
    --txingestion.dbhost=$TX_INGESTION_DB_HOST \
    --txingestion.pollinterval=$TX_INGESTION_POLL_INTERVAL \
    --txingestion.dbuser=$TX_INGESTION_DB_USER \
    --txingestion.dbpassword=$TX_INGESTION_DB_PASSWORD
