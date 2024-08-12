#!/usr/bin/env bash
set -euo pipefail

RPC="${1:?Must specify RPC address}"
FAULT_GAME_ADDRESS="${2:?Must specify game address}"

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DIR="${DIR%/*}"
cd "$DIR"

forge script scripts/fpac/viz/FaultDisputeGameViz.s.sol \
  --sig "remote(address)" "$FAULT_GAME_ADDRESS" \
  --fork-url "$RPC"
