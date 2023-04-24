#!/usr/bin/env bash

# Similar to challenge_simple.sh, but runs on L2!

SCRIPT_DIR=$(dirname "$(readlink -f "$0")")

export ETH_RPC_URL=http://127.0.0.1:9545/
export BLOCK=$(cast block-number)
export WRONG_BLOCK=$(expr $BLOCK - 1)

SKIP_NODE=1 NETWORK=l2 "$SCRIPT_DIR/challenge_simple.sh"
