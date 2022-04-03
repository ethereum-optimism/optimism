#!/bin/sh
set -eou

echo running "${0}"
if [ -z "$DATADIR" ]; then
    echo "Must pass DATADIR"
    exit 1
fi
if [ -z "$BLOCK_SIGNER_PRIVATE_KEY" ]; then
    echo "Must pass BLOCK_SIGNER_PRIVATE_KEY"
    exit 1
fi
if [ -z "$BLOCK_SIGNER_PRIVATE_KEY_PASSWORD" ]; then
    echo "Must pass BLOCK_SIGNER_PRIVATE_KEY_PASSWORD"
    exit 1
fi
if [ -z "$L2GETH_GENESIS_URL" ]; then
    echo "Must pass L2GETH_GENESIS_URL"
    exit 1
fi
if [ -z "$L2GETH_GENESIS_URL_SHA256SUM" ]; then
    echo "Must pass L2GETH_GENESIS_URL_SHA256SUM"
    exit 1
fi
# Check for an existing chaindata folder.
# If it exists, assume it's correct and skip geth init step
GETH_CHAINDATA_DIR=$DATADIR/geth/chaindata

if [ -d "$GETH_CHAINDATA_DIR" ]; then
    echo "$GETH_CHAINDATA_DIR existing, skipping geth init"
else
    echo "$GETH_CHAINDATA_DIR missing, running geth init"
    echo "Retrieving genesis file $L2GETH_GENESIS_URL"
    TEMP_DIR=$(mktemp -d)
    wget -O "$TEMP_DIR"/genesis.json "$L2GETH_GENESIS_URL"
    GENESIS_SHA256SUM=$(sha256sum "$TEMP_DIR"/genesis.json | awk '{print $1}')
    if [ "$GENESIS_SHA256SUM" != "$L2GETH_GENESIS_URL_SHA256SUM" ];then
        echo GENESIS_SHA256SUM: "$GENESIS_SHA256SUM" != L2GETH_GENESIS_URL_SHA256SUM: "$L2GETH_GENESIS_URL_SHA256SUM"
        exit 1
    fi
    echo checksums match
    echo GENESIS_SHA256SUM: "$GENESIS_SHA256SUM" == L2GETH_GENESIS_URL_SHA256SUM: "$L2GETH_GENESIS_URL_SHA256SUM"
    geth init \
        --datadir=/"$DATADIR" \
        "$TEMP_DIR"/genesis.json
    echo geth init complete
fi

# Check for an existing keystore folder.
# If it exists, assume it's correct and skip geth acount import step
GETH_KEYSTORE_DIR=$DATADIR/keystore
mkdir -p "$GETH_KEYSTORE_DIR"
GETH_KEYSTORE_KEYS=$(find "$GETH_KEYSTORE_DIR" -type f)

if [ ! -z "$GETH_KEYSTORE_KEYS" ]; then
    echo "$GETH_KEYSTORE_KEYS exist, skipping account import if any keys are present"
else
    echo "$GETH_KEYSTORE_DIR missing, running account import"
    echo -n "$BLOCK_SIGNER_PRIVATE_KEY_PASSWORD" > "$DATADIR"/password
    echo -n "$BLOCK_SIGNER_PRIVATE_KEY" > "$DATADIR"/block-signer-key
    geth account import \
        --datadir=/"$DATADIR" \
        --password "$DATADIR"/password \
        "$DATADIR"/block-signer-key
fi

echo "l2geth setup complete"
