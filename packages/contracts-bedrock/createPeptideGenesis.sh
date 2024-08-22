export FORK=latest
export CONTRACT_ADDRESSES_PATH=./deployments/1-deploy.json
export DEPLOY_CONFIG_PATH=./deploy-config/polymer-mainnet.json
export STATE_DUMP_PATH./peptide-allocs.json

forge script scripts/L2Genesis.s.sol:L2Genesis \
    --sig 'runWithStateDump()'