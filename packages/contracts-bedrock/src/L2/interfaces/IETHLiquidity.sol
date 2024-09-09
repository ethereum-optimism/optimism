// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title IETHLiquidity
/// @notice Interface for the ETHLiquidity contract.
interface IETHLiquidity {
    function burn() external payable;
    function mint(uint256 _amount) external;
}
