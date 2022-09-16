#!/command/with-contenv bash

set -eu

GETH_DATA_DIR=/db
GETH_CHAINDATA_DIR="$GETH_DATA_DIR/geth/chaindata"
GETH_KEYSTORE_DIR="$GETH_DATA_DIR/keystore"
GENESIS_FILE_PATH="/etc/op-geth/genesis.json"

mkdir -p /etc/secrets

if [ "$OP_NODE_L1_ETH_RPC" = "dummy" ]; then
	echo "You must specify the OP_NODE_L1_ETH_RPC environment variable."
	exit 1	
fi

if [ "$JWT_SECRET" = "dummy" ]; then
	echo "Regenerating JWT secret."
	hexdump -vn32 -e'4/4 "%08X" 1 ""' /dev/urandom > /etc/secrets/jwt-secret.txt
else
	echo "Found JWT secret."
fi

if [ "$P2P_SECRET" = "dummy" ]; then
	echo "Regenerating P2P private key."
	hexdump -vn32 -e'4/4 "%08X" 1 ""' /dev/urandom > /etc/secrets/p2p-private-key.txt
else
	echo "Found P2P private key."
fi

if [ ! -d "$GETH_CHAINDATA_DIR" ]; then
	echo "$GETH_CHAINDATA_DIR missing, running init"
	echo "Initializing genesis."
	geth --verbosity="$OP_GETH_VERBOSITY" init \
		--datadir="$GETH_DATA_DIR" \
		"$GENESIS_FILE_PATH"
else
	echo "$GETH_CHAINDATA_DIR exists."
fi