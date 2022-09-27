#!/bin/bash

# Step 1: deploy the Proxy to the predeploy address on L2
npx hardhat deploy --tags L2ERC721BridgeProxy --network ops-l2

# Step 2: deploy the Proxy for the L1ERC721Bridge to L1
npx hardhat deploy --tags L1ERC721BridgeProxy --network ops-l1

# Step 3: deploy the L2ERC721Bridge implementation
npx hardhat deploy --tags L2ERC721BridgeImplementation --network ops-l2

# Step 4: deploy the L1ERC721Bridge implementation to L1
npx hardhat deploy --tags L1ERC721BridgeImplementation --network ops-l1

# Step 5: deploy the Proxy for the OptimismMintableERC721Factory to L2
npx hardhat deploy --tags OptimismMintableERC721FactoryProxy --network ops-l2

# Step 5: deploy the OptimismMintableERC721Factory to L2
npx hardhat deploy --tags OptimismMintableERC721FactoryImplementation --network ops-l2
