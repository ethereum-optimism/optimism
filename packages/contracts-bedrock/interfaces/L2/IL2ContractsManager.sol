// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";

// Libraries
import { L2ContractsManagerTypes } from "src/libraries/L2ContractsManagerTypes.sol";

/// @title IL2ContractsManager
/// @notice Interface for the L2ContractsManager contract.
interface IL2ContractsManager is ISemver {
    /// @notice Thrown when the upgrade function is called outside of a DELEGATECALL context.
    error L2ContractsManager_OnlyDelegatecall();

    /// @notice Thrown when a user attempts to downgrade a contract.
    /// @param _target The address of the contract that was attempted to be downgraded.
    error L2ContractsManager_DowngradeNotAllowed(address _target);

    /// @notice Error thrown when a semver string has less than 3 parts.
    error SemverComp_InvalidSemverParts();

    /// @notice Thrown when a contract is in the process of being initialized during an upgrade.
    error L2ContractsManager_InitializingDuringUpgrade();

    /// @notice Thrown when a feature flag mismatch is detected.
    error L2ContractsManager_FeatureFlagMismatch();

    /// @notice Thrown when a predeploy is not upgradeable.
    /// @param _target The address of the non-upgradeable predeploy.
    error L2ContractsManager_NotUpgradeable(address _target);

    /// @notice Thrown when a v5 slot is passed with a non-zero offset.
    error L2ContractsManager_InvalidV5Offset();

    /// @notice Executes the upgrade for all predeploys.
    /// @dev This function MUST be called via DELEGATECALL from the L2ProxyAdmin.
    function upgrade() external;

    /// @notice Returns the implementation addresses for each predeploy upgraded by the L2ContractsManager.
    /// @return implementations_ The implementation addresses for each predeploy upgraded by the L2ContractsManager.
    function getImplementations()
        external
        view
        returns (L2ContractsManagerTypes.Implementations memory implementations_);

    /// @notice Constructor for the L2ContractsManager contract.
    /// @param _implementations The implementation struct containing the new implementation addresses for the L2
    /// predeploys.
    function __constructor__(L2ContractsManagerTypes.Implementations memory _implementations) external;
}
