#! /bin/bash

# Set up environment variables
source ./docker/.env.devnet

# Pre-funded devnet account
DEVNET_SPONSOR="ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

# Boot up the devnet
# Before booting, we make sure that we have a fresh devnet
(cd $MONOREPO_DIR && make devnet-down && make devnet-clean && L2OO_ADDRESS="0x6900000000000000000000000000000000000000" make devnet-up)

# Fetching balance of the sponsor
echo "----------------------------------------------------------------"
echo " - Fetching balance of the sponsor"
cast balance 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266 --rpc-url http://localhost:8545
echo "----------------------------------------------------------------"

# Send some ETH to the deployer
echo "----------------------------------------------------------------"
echo " - Sending some ETH to the deployer"
cast send --rpc-url http://localhost:8545 --private-key $DEVNET_SPONSOR 0xAF00b1C8A848BE9aFb28860BA0dC27b02fE3BD4d --value 100000000000000000000
echo "----------------------------------------------------------------"

# Deploy the mock dispute game contract
echo "----------------------------------------------------------------"
echo " - Deploying the mock dispute game contract"
(cd ./contracts/project && forge script script/DeployMocks.s.sol --rpc-url http://localhost:8545 --private-key $OP_CHALLENGER_PRIVATE_KEY --broadcast)
echo "----------------------------------------------------------------"

echo "----------------------------------------------------------------"
echo " All done! You can now run the \`op-challenger\`."
echo "----------------------------------------------------------------"
