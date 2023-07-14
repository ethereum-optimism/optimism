#!/bin/bash

set -euo pipefail

DISPUTE_GAME_PROXY="0xB7f8BC63BbcaD18155201308C8f3540b07f84F5e"

FAULT_GAME_ADDRESS="0x8daf17a20c9dba35f005b6324f493785d239719d"

dir=$(cd $(dirname "${BASH_SOURCE[0]}") && pwd)
cd "$dir"
cd ../packages/contracts-bedrock

forge script scripts/FaultDisputeGameViz.s.sol --sig "remote(address)" $FAULT_GAME_ADDRESS --fork-url http://localhost:8545
mv dispute_game.svg "$dir"
