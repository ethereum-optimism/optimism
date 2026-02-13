// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "interfaces/universal/ISemver.sol";

interface INativeAssetLiquidity is ISemver {
    error NativeAssetLiquidity_Unauthorized();
    error NativeAssetLiquidity_InsufficientBalance();

    event LiquidityDeposited(address indexed caller, uint256 value);
    event LiquidityWithdrawn(address indexed caller, uint256 value);

    function deposit() external payable;
    function withdraw(uint256 _amount) external;
}
