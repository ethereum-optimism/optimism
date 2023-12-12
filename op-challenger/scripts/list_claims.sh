#!/usr/bin/env bash

set -euo pipefail

RPC=${1:?Must specify RPC address}
GAME_ADDR=${2:?Must specify fault dispute game address}

COUNT=$(cast call --rpc-url "${RPC}" "${GAME_ADDR}" 'claimDataLen() returns(uint256)')
echo "Claim count: ${COUNT}"
((COUNT=COUNT-1))
for i in $(seq 0 "${COUNT}")
do
  CLAIM=$(cast call --rpc-url "${RPC}" "${GAME_ADDR}" 'claimData(uint256) returns(uint32 parentIndex, bool countered, bytes32 claim, uint128 position, uint128 clock)' "${i}")
  # Use read -ra to safely split the string into an array named CLAIM, assuming
  # data is newline-separated.
  IFS=$'\n' read -ra CLAIM <<< "$CLAIM"

  echo "${i}  Parent: ${CLAIM[0]} Countered: ${CLAIM[1]} Claim: ${CLAIM[2]} Position: ${CLAIM[3]} Clock ${CLAIM[4]}"
done
