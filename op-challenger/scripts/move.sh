#!/bin/bash
set -euo pipefail

RPC=${1:?Must specify RPC URL}
GAME_ADDR=${2:?Must specify game address}
ACTION=${3:?Must specify attack or defend}
CLAIM=${4:?Must specify claim hash}
SIGNER_ARGS="${@:5}"

# Respond to the last claim that was made
CLAIM_IDX=$(cast call --rpc-url "${RPC}" "${GAME_ADDR}" 'claimDataLen() returns(uint256)')
((CLAIM_IDX=CLAIM_IDX-1))

cast send --rpc-url "${RPC}" ${SIGNER_ARGS} "${GAME_ADDR}" "$ACTION(uint256,bytes32)" "${CLAIM_IDX}" "${CLAIM}"
