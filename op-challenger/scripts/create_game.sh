#!/usr/bin/env bash
set -euo pipefail

SOURCE_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
CHALLENGER_DIR="${SOURCE_DIR%/*}"

# ./create_game.sh <rpc-addr> <dispute-game-factory-addr> <cast signing args>
RPC=${1:?Must specify RPC address}
FACTORY_ADDR=${2:?Must specify factory address}
ROOT_CLAIM=${3:?Must specify root claim}
SIGNER_ARGS=("${@:4}")

# Default to Cannon Fault game type
GAME_TYPE=${GAME_TYPE:-0}

# Get the fault dispute game implementation addr
GAME_IMPL_ADDR=$(cast call --rpc-url "${RPC}" "${FACTORY_ADDR}" 'gameImpls(uint8) returns(address)' "${GAME_TYPE}")
echo "Fault dispute game impl: ${GAME_IMPL_ADDR}"

# Get the L2 output oracle address
L2OO_ADDR=$(cast call --rpc-url "${RPC}" "${GAME_IMPL_ADDR}" 'L2_OUTPUT_ORACLE() returns(address)')
echo "L2OO: ${L2OO_ADDR}"

# Get the block oracle address
BLOCK_ORACLE_ADDR=$(cast call --rpc-url "${RPC}" "${GAME_IMPL_ADDR}" 'BLOCK_ORACLE() returns(address)')
echo "Block Oracle: ${BLOCK_ORACLE_ADDR}"

# Get the L2 block number of the latest output proposal. This is the proposal that will be disputed by the created game.
L2_BLOCK_NUM=$(cast call --rpc-url "${RPC}" "${L2OO_ADDR}" 'latestBlockNumber() returns(uint256)')
echo "L2 Block Number: ${L2_BLOCK_NUM}"

# Create a checkpoint in the block oracle to commit to the current L1 head.
# This defines the L1 head that will be used in the dispute game.
echo "Checkpointing the block oracle..."
L1_CHECKPOINT=$(cast send --rpc-url "${RPC}" "${SIGNER_ARGS[@]}" "${BLOCK_ORACLE_ADDR}" "checkpoint()" --json | jq -r '.logs[0].topics[1]' | cast to-dec)
echo "L1 Checkpoint: $L1_CHECKPOINT"

# Fault dispute game extra data is calculated as follows.
# abi.encode(uint256(l2_block_number), uint256(l1 checkpoint))
EXTRA_DATA=$(cast abi-encode "f(uint256,uint256)" "${L2_BLOCK_NUM}" "${L1_CHECKPOINT}")

echo "Initializing the game"
FAULT_GAME_DATA=$(cast send --rpc-url "${RPC}" "${SIGNER_ARGS[@]}" "${FACTORY_ADDR}" "create(uint8,bytes32,bytes) returns(address)" "${GAME_TYPE}" "${ROOT_CLAIM}" "${EXTRA_DATA}" --json)

# Extract the address of the newly created game from the receipt logs.
FAULT_GAME_ADDRESS=$(echo "${FAULT_GAME_DATA}" | jq -r '.logs[0].topics[1]' | cast parse-bytes32-address)
echo "Fault game address: ${FAULT_GAME_ADDRESS}"
echo "${FAULT_GAME_ADDRESS}" > "$CHALLENGER_DIR"/.fault-game-address
