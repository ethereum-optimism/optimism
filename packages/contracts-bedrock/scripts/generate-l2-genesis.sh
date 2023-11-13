#!/usr/bin/env bash

# Create a L2 genesis.json suitable for the solidity tests to
# ingest using `vm.loadAllocs(string)`.
# This script depends on the relative path to the op-node from
# contracts-bedrock

SCRIPTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null && pwd)"
CONTRACTS_DIR="$(realpath "$SCRIPTS_DIR/..")"
MONOREPO_BASE="$(realpath "$CONTRACTS_DIR/../..")"

DEPLOY_ARTIFACT="$CONTRACTS_DIR/deployments/hardhat/.deploy"
OP_NODE="$MONOREPO_BASE/op-node/cmd/main.go"
L1_STARTING_BLOCK_PATH="$CONTRACTS_DIR/test/mocks/block.json"
TESTDATA_DIR="$CONTRACTS_DIR/.testdata"

OUTFILE_L2="$TESTDATA_DIR/genesis.json"
OUTFILE_ROLLUP="$TESTDATA_DIR/rollup.json"
OUTFILE_ALLOC="$TESTDATA_DIR/alloc.json"

if [ ! -f "$DEPLOY_ARTIFACT" ]; then
  forge script $CONTRACTS_DIR/scripts/Deploy.s.sol:Deploy 2>&1 /dev/null
fi

if [ ! -d "$TESTDATA_DIR" ]; then
  mkdir -p "$TESTDATA_DIR"

  go run $OP_NODE genesis l2 \
    --deploy-config "$CONTRACTS_DIR/deploy-config/hardhat.json" \
    --l1-deployments "$DEPLOY_ARTIFACT" \
    --l1-starting-block "$L1_STARTING_BLOCK_PATH" \
    --outfile.l2 "$OUTFILE_L2" \
    --outfile.rollup "$OUTFILE_ROLLUP" >/dev/null 2>&1
fi
