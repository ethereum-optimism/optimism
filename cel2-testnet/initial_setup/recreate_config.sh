#!/bin/bash
set -euo pipefail
set -x

TESTNET_DIR=$(readlink -f $(dirname $0))/..
cd "$TESTNET_DIR/.."

(
  cd packages/contracts-bedrock/ &&
  GS_ADMIN_ADDRESS='$ADMIN_ADDR' GS_BATCHER_ADDRESS='$BATCHER_ADDR' GS_PROPOSER_ADDRESS='$PROPOSER_ADDR' GS_SEQUENCER_ADDRESS='$SEQUENCER_ADDR' L1_RPC_URL=$L1_RPC scripts/getting-started/config.sh
)
cp packages/contracts-bedrock/deploy-config/getting-started.json cel2-testnet/cel2-testnet.json
