#生成genesis和rollup
op-deployer inspect genesis --workdir .deployer 42069 >.deployer/genesis.json
op-deployer inspect rollup --workdir .deployer 42069 >.deployer/rollup.json

#这三个指令可选，用于查看部署结果
# outputs all L1 contract addresses for an L2 chain
op-deployer inspect l1 --workdir .deployer 42069 >.deployer/l1.json
# outputs the deploy config for an L2 chain
op-deployer inspect deploy-config --workdir .deployer 42069 >.deployer/config.json
# outputs the semvers for all L2 chains
op-deployer inspect l2-semvers --workdir .deployer 42069 >.deployer/l2-sem.json
