// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title ISystemConfigInterop
/// @notice Interface for the SystemConfigInterop contract.
interface ISystemConfigInterop {
    function addDependency(uint256 _chainId) external;
    function removeDependency(uint256 _chainId) external;
}
