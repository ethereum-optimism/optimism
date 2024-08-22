
export OLD_DEPLOY_CONFIG_PATH=./deploy-config/polymer-mainnet.json
export DEPLOY_CONFIG_PATH=./deploy-config/polymer-mainnet-1.json
export IMPL_SALT="polymer-deploy-1"

cat $DEPLOY_CONFIG_PATH

export blockNumber=$(cast block --rpc-url $LOCAL_RPC| grep "number" | grep -Eo '[0-9]+' | sed 's/\.$//')

echo $blockNumber
# Deploy l2output oracle address
jq --arg new_value $blockNumber '.systemConfigStartBlock= $new_value' $OLD_DEPLOY_CONFIG_PATH > $DEPLOY_CONFIG_PATH

# forge script \
#     scripts/deploy/Deploy.s.sol:Deploy \
#     --sig deployPolymerL1Contracts \
#     --broadcast \
#     --private-key $DUMMY_PRIVATE_KEY \
#     --rpc-url $LOCAL_RPC

forge script \
    scripts/deploy/Deploy.s.sol:Deploy \
    --broadcast \
    --private-key $DUMMY_PRIVATE_KEY \
    --rpc-url $LOCAL_RPC


export FORK=latest
export CONTRACT_ADDRESSES_PATH=./deployments/1-deploy.json
export STATE_DUMP_PATH=peptide-allocs.json

forge script scripts/L2Genesis.s.sol:L2Genesis \
    --sig 'runWithStateDump()'
