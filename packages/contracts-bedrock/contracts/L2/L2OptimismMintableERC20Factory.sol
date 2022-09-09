// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { OptimismMintableERC20Factory } from "../universal/OptimismMintableERC20Factory.sol";
import { Predeploys } from "../libraries/Predeploys.sol";

/**
 * @custom:proxied
 * @custom:predeployed 0x4200000000000000000000000000000000000012
 * @title L2OptimismMintableERC20Factory
 * @notice L2OptimismMintableERC20Factory is a factory contract that generates OptimismMintableERC20
 *         contracts on L2 that allows for deposits of L1 native tokens. Simplifies the deployment
 *         process for users who may be less familiar with deploying smart contracts. Designed to
           be backwards compatible with the legacy StandardL2ERC20Factory contract.
 */
contract L2OptimismMintableERC20Factory is OptimismMintableERC20Factory {
    constructor() OptimismMintableERC20Factory(Predeploys.L2_STANDARD_BRIDGE) {}
}
