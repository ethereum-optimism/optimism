// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title IGovernanceTokenInterop
/// @notice Interface for the IGovernanceTokenInterop contract.
interface IGovernanceTokenInterop {
    /// @notice Migrates a delegator to the `GovernanceDelegation` contract.
    /// @param _delegator The account to migrate.
    function migrate(address _delegator) external;
}
