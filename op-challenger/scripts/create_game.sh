#!/usr/bin/env bash
set -euo pipefail

SOURCE_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
CHALLENGER_DIR="${SOURCE_DIR%/*}"

# ./create_game.sh <rpc-addr> <dispute-game-factory-addr> <output-root> <l2-block-num> <cast signing args>
RPC=${1:?Must specify RPC address}
FACTORY_ADDR=${2:?Must specify factory address}
ROOT_CLAIM=${3:?Must specify claimed output root}
L2_BLOCK_NUM=${4:?Must specify L2 block number of claimed output root}
SIGNER_ARGS=("${@:5}")

# Default to Cannon Fault game type
GAME_TYPE=${GAME_TYPE:-0}

# Fault dispute game extra data is calculated as follows.
# abi.encode(uint256(l2_block_number), uint256(l1 checkpoint))
EXTRA_DATA=$(cast abi-encode "f(uint256)" "${L2_BLOCK_NUM}" )

echo "Initializing the game"
FAULT_GAME_DATA=$(cast send --rpc-url "${RPC}" "${SIGNER_ARGS[@]}" "${FACTORY_ADDR}" "create(uint8,bytes32,bytes) returns(address)" "${GAME_TYPE}" "${ROOT_CLAIM}" "${EXTRA_DATA}" --json)

# Extract the address of the newly created game from the receipt logs.
FAULT_GAME_ADDRESS=$(echo "${FAULT_GAME_DATA}" | jq -r '.logs[0].topics[1]' | cast parse-bytes32-address)
echo "Fault game address: ${FAULT_GAME_ADDRESS}"
echo "${FAULT_GAME_ADDRESS}" > "$CHALLENGER_DIR"/.fault-game-address
