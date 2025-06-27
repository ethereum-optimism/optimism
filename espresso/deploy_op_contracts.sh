set -e
# https://devdocs.optimism.io/op-deployer/index.html

ROOT_DIR=$PWD/..
L1_RPC_URL=http://localhost:8545
DEPLOYMENT_DIR=$ROOT_DIR/espresso/deployment
OPERATOR_PRIVATE_KEY=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
OPERATOR_ADDRESS=0x8943545177806ed17b9f23f0a21ee5948ecaa776
ARTIFACTS_LOCATOR_DIR=$ROOT_DIR/packages/contracts-bedrock/forge-artifacts
BOOTSTRAP_SUPERCHAIN_OUTPUT=$DEPLOYMENT_DIR/bootstrap_superchain.json
BOOTSTRAP_IMPLEMENTATION_OUTPUT=$DEPLOYMENT_DIR/bootstrap_implementations.json
L1_CHAIN_ID=11155111

mkdir -p $DEPLOYMENT_DIR

# Build the deployer binary
cd $ROOT_DIR/op-deployer
just build

# Compile the contracts
cd $ROOT_DIR/packages/contracts-bedrock
just build

cd $ROOT_DIR/espresso


echo "\nDeploying superchain contracts..."
op-deployer bootstrap superchain \
--l1-rpc-url="$L1_RPC_URL" \
--private-key="$OPERATOR_PRIVATE_KEY" \
--artifacts-locator="file://$ARTIFACTS_LOCATOR_DIR" \
--outfile=$BOOTSTRAP_SUPERCHAIN_OUTPUT \
--superchain-proxy-admin-owner="$OPERATOR_ADDRESS" \
--protocol-versions-owner="$OPERATOR_ADDRESS" \
--guardian="$OPERATOR_ADDRESS"


PROXY_ADMIN_ADDRESS=`jq -r '.proxyAdminAddress' $BOOTSTRAP_SUPERCHAIN_OUTPUT`
SUPERCHAIN_CONFIG_PROXY_ADDRESS=`jq -r '.superchainConfigProxyAddress' $BOOTSTRAP_SUPERCHAIN_OUTPUT`
PROTOCOL_VERSIONS_PROXY_ADDRESS=`jq -r '.protocolVersionsProxyAddress' $BOOTSTRAP_SUPERCHAIN_OUTPUT`


echo "\nBootstraping implementations contracts..."
op-deployer bootstrap implementations \
--artifacts-locator="file://$ARTIFACTS_LOCATOR_DIR" \
--l1-rpc-url="$L1_RPC_URL" \
--outfile="$BOOTSTRAP_IMPLEMENTATION_OUTPUT" \
--mips-version="1" \
--private-key="$OPERATOR_PRIVATE_KEY" \
--superchain-config-proxy="$SUPERCHAIN_CONFIG_PROXY_ADDRESS" \
--protocol-versions-proxy="$PROTOCOL_VERSIONS_PROXY_ADDRESS" \
--upgrade-controller="$OPERATOR_ADDRESS"


echo "\nInitializing the intents..."
op-deployer init \
--l1-chain-id $L1_CHAIN_ID \
--l2-chain-ids 1 \
--outdir $ROOT_DIR/deployment \
--intent-type standard


echo "Deploy the contracts for real!"

op-deployer apply \
  --workdir $DEPLOYMENT_DIR \
  --l1-rpc-url $L1_RPC_URL \
  --private-key $OPERATOR_PRIVATE_KEY
