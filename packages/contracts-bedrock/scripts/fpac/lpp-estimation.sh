#!/bin/bash

# Default Anvil private key
PRIVATE_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
PUB_KEY="$(cast wallet addr $PRIVATE_KEY)"

# 40 gwei base fee
BASE_FEE=40000000000

# Anvil RPC
export ETH_RPC_URL="http://localhost:8545"

# Start anvil in the background
anvil --base-fee $BASE_FEE &
# Capture the process ID
ANVIL_PID=$!

# Deploy the `PreimageOracle` contract to anvil.
PO_ADDR=$(forge create PreimageOracle --private-key $PRIVATE_KEY --rpc-url $ETH_RPC_URL --json | jq -r '.deployedTo')

# Capture the balance of the submitter prior to submitting all leaves.
BALANCE_BEFORE=$(cast balance --rpc-url http://localhost:8545 "$PUB_KEY")
BASE_FEE_BEFORE=$(cast 2d "$(cast rpc 'eth_gasPrice' | jq -r)")

# Run the `SubmitLPP` script to submit the LPP to the `PreimageOracle` contract.
forge script scripts/fpac/SubmitLPP.sol \
  --sig "post(address)" "$PO_ADDR" \
  --private-key $PRIVATE_KEY \
  --rpc-url $ETH_RPC_URL \
  --broadcast

BALANCE_AFTER=$(cast balance "$PUB_KEY")
BASE_FEE_AFTER=$(cast 2d "$(cast rpc 'eth_gasPrice' | jq -r)")

echo "Base Fee Before: $BASE_FEE_BEFORE"
echo "Base Fee After: $BASE_FEE_AFTER"
echo "Balance before: $BALANCE_BEFORE"
echo "Balance after: $BALANCE_AFTER"
echo "Cost: $(cast from-wei $((BALANCE_BEFORE - BALANCE_AFTER))) ETH"

# Kill anvil
kill $ANVIL_PID
