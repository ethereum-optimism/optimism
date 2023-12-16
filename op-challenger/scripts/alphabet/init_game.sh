#!/usr/bin/env bash

set -euo pipefail

SOURCE_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
CHALLENGER_DIR="${SOURCE_DIR%/*/*}"
MONOREPO_DIR="${SOURCE_DIR%/*/*/*}"

cd "$CHALLENGER_DIR"
make

cd "$MONOREPO_DIR"
make devnet-clean
make cannon-prestate
make devnet-up

DEVNET_SPONSOR="ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
DISPUTE_GAME_FACTORY=$(jq -r .DisputeGameFactoryProxy "$MONOREPO_DIR"/.devnet/addresses.json)

echo "----------------------------------------------------------------"
echo " Dispute Game Factory at $DISPUTE_GAME_FACTORY"
echo "----------------------------------------------------------------"

L2_OUTPUT_ORACLE_PROXY=$(jq -r .L2OutputOracleProxy "$MONOREPO_DIR"/.devnet/addresses.json)

echo "----------------------------------------------------------------"
echo " L2 Output Oracle Proxy at $L2_OUTPUT_ORACLE_PROXY"
echo "----------------------------------------------------------------"

BLOCK_ORACLE_PROXY=$(jq -r .BlockOracle "$MONOREPO_DIR"/.devnet/addresses.json)

echo "----------------------------------------------------------------"
echo " Block Oracle Proxy at $BLOCK_ORACLE_PROXY"
echo "----------------------------------------------------------------"

CHARLIE_ADDRESS="0xF45B7537828CB2fffBC69996B054c2Aaf36DC778"
MALLORY_ADDRESS="0x4641c704a6c743f73ee1f36C7568Fbf4b80681e4"

echo "----------------------------------------------------------------"
echo " - Fetching balance of the sponsor"
echo " - Balance: $(cast balance 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266)"
echo "----------------------------------------------------------------"

echo "Funding Charlie"
cast send "$CHARLIE_ADDRESS" --value 5ether --private-key "$DEVNET_SPONSOR"

echo "Funding Mallory"
cast send "$MALLORY_ADDRESS" --value 5ether --private-key "$DEVNET_SPONSOR"

# Loop and wait until there are at least 2 outputs in the l2 output oracle
echo "Waiting until 2 output proposals are in the l2 output oracle..."
echo "NOTE: This may show errors if no output proposals are in the oracle yet."
while [[ $(cast call "$L2_OUTPUT_ORACLE_PROXY" "latestOutputIndex()" | cast to-dec) -lt 2 ]]
do
  echo "[BLOCK: $(cast block-number)] Waiting for output proposals..."
  sleep 2
done

# Root claim commits to the entire trace.
# Alphabet game claim construction: keccak256(abi.encode(trace_index, trace[trace_index]))
ROOT_CLAIM=$(cast keccak "$(cast abi-encode "f(uint256,uint256)" 15 122)")
# Replace the first byte of the claim with the invalid vm status indicator
ROOT_CLAIM="0x01${ROOT_CLAIM:4}"

GAME_TYPE=255 "${SOURCE_DIR}"/../create_game.sh http://localhost:8545 "${DISPUTE_GAME_FACTORY}" "${ROOT_CLAIM}" --private-key "${DEVNET_SPONSOR}"
