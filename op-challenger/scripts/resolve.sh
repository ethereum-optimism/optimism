#!/bin/bash
set -euo pipefail

RPC=${1:?Must specify RPC URL}
GAME_ADDR=${2:?Must specify game address}
SIGNER_ARGS=("${@:3}")

# Perform the move.
# shellcheck disable=SC2086
RESULT_DATA=$(cast send --rpc-url "${RPC}" "${SIGNER_ARGS[@]}" "${GAME_ADDR}" "resolve()" --json)
RESULT=$(echo "${RESULT_DATA}" | jq -r '.logs[0].topics[1]' | cast to-dec)

if [[ "${RESULT}" == "0" ]]
then
  RESULT="In Progress"
elif [[ "${RESULT}" == "1" ]]
then
  RESULT="Challenger Wins"
elif [[ "${RESULT}" == "2" ]]
then
  RESULT="Defender Wins"
fi

echo "Result: $RESULT"
