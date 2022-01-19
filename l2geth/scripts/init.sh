#!/bin/bash

# script to help simplify l2geth initialization
# it needs a path on the filesystem to the state
# dump

set -eou pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" > /dev/null && pwd )"
REPO=$DIR/..
STATE_DUMP=${STATE_DUMP:-$REPO/../packages/contracts/dist/dumps/state-dump.latest.json}
DATADIR=${DATADIR:-$HOME/.ethereum}

# These are the initial key and address that must be used for the clique
# signer on the optimism network. All nodes must be initialized with this
# key before they are able to join the network and sync correctly.
# The signer address needs to be in the genesis block's extradata.
SIGNER_KEY=6587ae678cf4fc9a33000cdbf9f35226b71dcc6a4684a31203241f9bcfd55d27
SIGNER=0x00000398232e2064f896018496b4b44b3d62751f

mkdir -p $DATADIR

if [[ ! -f $STATE_DUMP ]]; then
    echo "Cannot find $STATE_DUMP"
    exit 1
fi

# Add funds to the signer account so that it can be used to send transactions
if [[ ! -z "$DEVELOPMENT" ]]; then
    echo "Setting up development genesis"
    echo "Assuming that the initial clique signer is $SIGNER, this is configured in genesis extradata"
    DUMP=$(cat $STATE_DUMP | jq '.alloc += {"0x00000398232e2064f896018496b4b44b3d62751f": {balance: "10000000000000000000"}}')
    TEMP=$(mktemp)
    echo "$DUMP" | jq . > $TEMP
    STATE_DUMP=$TEMP
fi

geth="$REPO/build/bin/geth"
USING_OVM=true $geth init --datadir $DATADIR $STATE_DUMP

echo "6587ae678cf4fc9a33000cdbf9f35226b71dcc6a4684a31203241f9bcfd55d27" \
    > $DATADIR/keyfile

echo "password" > $DATADIR/password

USING_OVM=true $geth account import \
    --datadir $DATADIR --password $DATADIR/password $DATADIR/keyfile
