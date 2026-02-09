// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { XForkL2CMTypes } from "src/libraries/XForkL2CMTypes.sol";

/// @title IXForkL2ContractsManager
/// @notice Interface for the XForkL2ContractsManager contract.
interface IXForkL2ContractsManager is ISemver {
    /// @notice Executes the upgrade for all predeploys.
    /// @dev This function MUST be called via DELEGATECALL from the L2ProxyAdmin.
    function upgrade() external;

    /// @notice Constructor for the XForkL2ContractsManager contract.
    /// @param _implementations The implementation struct containing the new implementation addresses for the L2
    /// predeploys.
    function __constructor__(XForkL2CMTypes.Implementations memory _implementations) external;
}
