#!/usr/bin/env bash

set -euo pipefail

RPC=${1:?Must specify RPC address}
FACTORY_ADDR=${2:?Must specify dispute game factory address}

COUNT=$(cast call --rpc-url "${RPC}" "${FACTORY_ADDR}" 'gameCount() returns(uint256)')
echo "Game count: ${COUNT}"
if [[ "${COUNT}" == "0" ]]
then
  exit
fi
((COUNT=COUNT-1))
for i in $(seq 0 "${COUNT}")
do
  GAME=$(cast call --rpc-url "${RPC}" "${FACTORY_ADDR}" 'gameAtIndex(uint256) returns(uint8, uint64, address)' "${i}")
  SAVEIFS=$IFS   # Save current IFS (Internal Field Separator)
  IFS=$'\n'      # Change IFS to newline char
  # shellcheck disable=SC2206
  GAME=($GAME) # split the string into an array by the same name
  IFS=$SAVEIFS   # Restore original IFS

  GAME_ADDR="${GAME[2]}"
  CLAIMS=$(cast call --rpc-url "${RPC}" "${GAME_ADDR}" "claimDataLen() returns(uint256)")
  STATUS=$(cast call --rpc-url "${RPC}" "${GAME_ADDR}" "status() return(uint8)" | cast to-dec)
  if [[ "${STATUS}" == "0" ]]
  then
    STATUS="In Progress"
  elif [[ "${STATUS}" == "1" ]]
  then
    STATUS="Challenger Wins"
  elif [[ "${STATUS}" == "2" ]]
  then
    STATUS="Defender Wins"
  fi
  echo "${i}  Game: ${GAME_ADDR} Type: ${GAME[0]} Created: ${GAME[1]} Claims: ${CLAIMS} Status: ${STATUS}"
done
