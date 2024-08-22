
export DEPLOY_CONFIG_PATH=./deploy-config/polymer-mainnet.json
export IMPL_SALT="polymer-deploy-1"

cat $DEPLOY_CONFIG_PATH

echo $(pwd)
# Deploy l2output oracle address
forge script \
    scripts/deploy/Deploy.s.sol:Deploy \
    --sig deployPolymerL1Contracts \
    --broadcast \
    --private-key $DUMMY_PRIVATE_KEY \
    --rpc-url $LOCAL_RPC