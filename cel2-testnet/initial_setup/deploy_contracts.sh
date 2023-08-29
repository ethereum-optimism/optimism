#!/bin/bash
set -euo pipefail

# Following the OP-stack tutorial, steps
# https://stack.optimism.io/docs/build/getting-started/#configure-your-network
# https://stack.optimism.io/docs/build/getting-started/#deploy-the-l1-contracts

TESTNET_DIR=$(readlink -f $(dirname $0))/..
cd $TESTNET_DIR
<$DEPLOYMENT_CONTEXT.json envsubst > ../packages/contracts-bedrock/deploy-config/$DEPLOYMENT_CONTEXT.json
cd ../packages/contracts-bedrock

forge script scripts/Deploy.s.sol:Deploy --private-key $ADMIN_PRIVKEY --broadcast --rpc-url $L1_RPC && \
forge script scripts/Deploy.s.sol:Deploy --sig 'sync()' --private-key $ADMIN_PRIVKEY --broadcast --rpc-url $L1_RPC \
| tee deploy.log
