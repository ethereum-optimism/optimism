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

# Charlie's Address: 0xF45B7537828CB2fffBC69996B054c2Aaf36DC778
CHARLIE_KEY="74feb147d72bfae943e6b4e483410933d9e447d5dc47d52432dcc2c1454dabb7"

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
  --alphabet "abcdefgh" \
  --datadir "${DATADIR}" \
  --game-factory-address "$DISPUTE_GAME_PROXY" \
  --game-allowlist "$FAULT_GAME_ADDRESS" \
  --private-key "$CHARLIE_KEY" \
  --num-confirmations 1 \
  --metrics.enabled --metrics.port=7304 \
  --pprof.enabled --pprof.port=6064 \
  --agree-with-proposed-output=true
