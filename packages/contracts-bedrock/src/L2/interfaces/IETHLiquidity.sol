// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

/// @title IETHLiquidity
/// @notice Interface for the ETHLiquidity contract.
interface IETHLiquidity {
    /// @notice Emitted when an address burns ETH liquidity.
    event LiquidityBurned(address indexed caller, uint256 value);

    /// @notice Emitted when an address mints ETH liquidity.
    event LiquidityMinted(address indexed caller, uint256 value);

    function burn() external payable;
    function mint(uint256 _amount) external;
}
