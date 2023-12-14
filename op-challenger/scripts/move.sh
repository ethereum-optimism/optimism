#!/bin/bash
set -euo pipefail

RPC=${1:?Must specify RPC URL}
GAME_ADDR=${2:?Must specify game address}
ACTION=${3:?Must specify attack or defend}
PARENT_INDEX=${4:?Must specify parent index. Use latest to counter the latest claim added to the game.}
CLAIM=${5:?Must specify claim hash}
SIGNER_ARGS=("${@:6}")


if [[ "${ACTION}" != "attack" && "${ACTION}" != "defend" ]]
then
  echo "Action must be either attack or defend"
  exit 1
fi

if [[ "${PARENT_INDEX}" == "latest" ]]
then
  # Fetch the index of the most recent claim made.
  PARENT_INDEX=$(cast call --rpc-url "${RPC}" "${GAME_ADDR}" 'claimDataLen() returns(uint256)')
  ((PARENT_INDEX=PARENT_INDEX-1))
fi

# Perform the move.
cast send --rpc-url "${RPC}" "${SIGNER_ARGS[@]}" "${GAME_ADDR}" "$ACTION(uint256,bytes32)" "${PARENT_INDEX}" "${CLAIM}"
