#!/usr/bin/env bash

set -euo pipefail

RPC=${1:?Must specify RPC address}
GAME_ADDR=${2:?Must specify fault dispute game address}

COUNT=$(cast call --rpc-url "${RPC}" "${GAME_ADDR}" 'claimDataLen() returns(uint256)')
echo "Claim count: ${COUNT}"
((COUNT=COUNT-1))
for i in $(seq 0 "${COUNT}")
do
  CLAIM=$(cast call --rpc-url "${RPC}" "${GAME_ADDR}" 'claimData(uint256) returns(uint32 parentIndex, address counteredBy, address claimant, uint128 bond, bytes32 claim, uint128 position, uint128 clock)' "${i}")
  SAVEIFS=$IFS   # Save current IFS (Internal Field Separator)
  IFS=$'\n'      # Change IFS to newline char
  # shellcheck disable=SC2206
  CLAIM=($CLAIM) # split the string into an array by the same name
  IFS=$SAVEIFS # Restore original IFS

  echo "${i}  Parent: ${CLAIM[0]} Countered By: ${CLAIM[1]} Claimant: ${CLAIM[2]} Bond: ${CLAIM[3]} Claim: ${CLAIM[4]} Position: ${CLAIM[5]} Clock ${CLAIM[6]}"
done
