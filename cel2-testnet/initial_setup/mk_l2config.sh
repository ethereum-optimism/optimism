#!/bin/bash
set -euo pipefail

# Following the OP-stack tutorial, steps
# https://stack.optimism.io/docs/build/getting-started/#generate-the-l2-config-files

export PATH=$(go1.19 env GOROOT)/bin:$PATH

TESTNET_DIR=$(readlink -f $(dirname $0))/..

cd $TESTNET_DIR/../op-node
go run cmd/main.go genesis l2 \
    --deploy-config ../packages/contracts-bedrock/deploy-config/$DEPLOYMENT_CONTEXT.json \
    --deployment-dir ../packages/contracts-bedrock/deployments/$DEPLOYMENT_CONTEXT/ \
    --outfile.l2 $TESTNET_DIR/generated/genesis-l2.json \
    --outfile.rollup $TESTNET_DIR/generated/rollup.json \
    --l1-rpc $L1_RPC
