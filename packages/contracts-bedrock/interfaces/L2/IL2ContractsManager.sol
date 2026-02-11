// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title IL2ContractsManager
/// @notice Interface for the L2ContractsManager contract.
interface IL2ContractsManager {
    /// @notice Executes the upgrade for all predeploys.
    /// @dev This function MUST be called via DELEGATECALL from the L2ProxyAdmin.
    function upgrade() external;
}
