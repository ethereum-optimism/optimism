// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";

// Libraries
import { L2ContractsManagerTypes } from "src/libraries/L2ContractsManagerTypes.sol";

/// @title IL2ContractsManager
/// @notice Interface for the L2ContractsManager contract.
interface IL2ContractsManager is ISemver {
    /// @notice Executes the upgrade for all predeploys.
    /// @dev This function MUST be called via DELEGATECALL from the L2ProxyAdmin.
    function upgrade() external;

    /// @notice Constructor for the L2ContractsManager contract.
    /// @param _implementations The implementation struct containing the new implementation addresses for the L2
    /// predeploys.
    function __constructor__(L2ContractsManagerTypes.Implementations memory _implementations) external;
}
