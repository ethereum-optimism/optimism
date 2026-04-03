// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title IL2DevFeatureFlags
/// @notice Interface for the L2DevFeatureFlags contract.
interface IL2DevFeatureFlags is ISemver {
    /// @notice Thrown when the caller is not the depositor account.
    error L2DevFeatureFlags_Unauthorized();

    /// @notice Sets the development feature bitmap. Only callable by the DEPOSITOR_ACCOUNT.
    /// @param _bitmap The new development feature bitmap.
    function setDevFeatureBitmap(bytes32 _bitmap) external;

    /// @notice Returns the development feature bitmap.
    /// @return bitmap_ The current development feature bitmap.
    function devFeatureBitmap() external view returns (bytes32 bitmap_);

    /// @notice Checks if a development feature is enabled.
    /// @param _feature The feature to check.
    /// @return enabled_ True if the feature is enabled, false otherwise.
    function isDevFeatureEnabled(bytes32 _feature) external view returns (bool enabled_);

    /// @notice Constructor placeholder for the interface.
    function __constructor__() external;
}
