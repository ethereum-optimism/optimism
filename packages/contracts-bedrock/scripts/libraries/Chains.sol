// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @notice Chain IDs for the various networks.
library Chains {
    uint256 internal constant Mainnet = 1;
    uint256 internal constant OPMainnet = 10;
    uint256 internal constant Goerli = 5;
    uint256 internal constant OPGoerli = 420;
    uint256 internal constant Sepolia = 11155111;
    uint256 internal constant OPSepolia = 11155420;
    uint256 internal constant LocalDevnet = 900;
    uint256 internal constant OPLocalDevnet = 901;
    uint256 internal constant GethDevnet = 1337;
    uint256 internal constant Hardhat = 31337;
    /// @notice L1 chain ID of op-deployer's test anvil (op-service/testutils/devnet.DefaultChainID).
    uint256 internal constant AnvilDevnet = 77799777;
    /// @notice L1 chain ID of the interopgen dev recipes.
    uint256 internal constant InteropDevnet = 900100;
}
