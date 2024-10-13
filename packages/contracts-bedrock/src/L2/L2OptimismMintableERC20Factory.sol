// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { OptimismMintableERC20Factory } from "src/universal/OptimismMintableERC20Factory.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

/// @custom:proxied true
/// @custom:predeployed 0x4200000000000000000000000000000000000012
/// @title L2OptimismMintableERC20Factory
/// @notice L2OptimismMintableERC20Factory is a factory contract that generates OptimismMintableERC20
///         contracts on the network it's deployed to. Simplifies the deployment process for users
///         who may be less familiar with deploying smart contracts. Designed to be backwards
///         compatible with the older StandardL2ERC20Factory contract.
contract L2OptimismMintableERC20Factory is OptimismMintableERC20Factory {
    function bridge() public view virtual override returns (address) {
        return Predeploys.L2_STANDARD_BRIDGE;
    }
}
