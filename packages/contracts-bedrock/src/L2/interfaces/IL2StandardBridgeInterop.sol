// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title IL2StandardBridgeInterop
/// @notice Interface for the L2StandardBridgeInterop contract.
interface IL2StandardBridgeInterop {
    function convert(address _from, address _to, uint256 _amount) external;
}
