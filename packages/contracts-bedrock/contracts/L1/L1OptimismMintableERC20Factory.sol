// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { OptimismMintableERC20Factory } from "../universal/OptimismMintableERC20Factory.sol";

/**
 * @custom:proxied
 * @title L1OptimismMintableERC20Factory
 * @notice L1OptimismMintableERC20Factory is the OptimismMintableERC20Factory that is deployed on
 *         L1. It allows for L2 native tokens to be withdrawn to L1.
 */
contract L1OptimismMintableERC20Factory is OptimismMintableERC20Factory {
    /**
     * @param _bridge Address of the StandardBridge on this chain.
     */
    constructor(address _bridge) OptimismMintableERC20Factory(_bridge) {}
}
