#!/usr/bin/env bash
set -euo pipefail

SOURCE_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
CHALLENGER_DIR="${SOURCE_DIR%/*/*}"
MONOREPO_DIR="${SOURCE_DIR%/*/*/*}"

# Check that the fault game address file exists
FAULT_GAME_ADDR_FILE="$CHALLENGER_DIR/.fault-game-address"
if [[ ! -f "$FAULT_GAME_ADDR_FILE" ]]; then
    echo "Game not initialized, exiting..."
    exit 1
fi

# Mallory's Address: 0x4641c704a6c743f73ee1f36C7568Fbf4b80681e4
MALLORY_KEY="28d7045146193f5f4eeb151c4843544b1b0d30a7ac1680c845a416fac65a7715"

DISPUTE_GAME_PROXY=$(jq -r .DisputeGameFactoryProxy "$MONOREPO_DIR"/.devnet/addresses.json)
FAULT_GAME_ADDRESS=$(cat "$FAULT_GAME_ADDR_FILE")
echo "Fault dispute game address: $FAULT_GAME_ADDRESS"

DATADIR=$(mktemp -d)
trap cleanup SIGINT
cleanup(){
  rm -rf "${DATADIR}"
}

"$CHALLENGER_DIR"/bin/op-challenger \
  --l1-eth-rpc http://localhost:8545 \
  --trace-type="alphabet" \
  --alphabet "abcdexyz" \
  --datadir "${DATADIR}" \
  --game-factory-address "$DISPUTE_GAME_PROXY" \
  --game-allowlist "$FAULT_GAME_ADDRESS" \
  --private-key "$MALLORY_KEY" \
  --num-confirmations 1 \
  --metrics.enabled --metrics.port=7305 \
  --pprof.enabled --pprof.port=6065 \
  --agree-with-proposed-output=false
