// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @notice Library of constants representing development features.
library DevFeatures {
    /// @notice The feature that enables the OptimismPortalInterop contract.
    bytes32 public constant OPTIMISM_PORTAL_INTEROP =
        bytes32(0x0000000000000000000000000000000000000000000000000000000000000001);

    /// @notice Checks if a feature is enabled in a bitmap. Note that this function does not check
    ///         that the input feature represents a single feature and the bitwise AND operation
    ///         allows for multiple features to be enabled at once. Users should generally check
    ///         for only a single feature at a time.
    /// @param _bitmap The bitmap to check.
    /// @param _feature The feature to check.
    /// @return True if the feature is enabled, false otherwise.
    function isDevFeatureEnabled(bytes32 _bitmap, bytes32 _feature) internal pure returns (bool) {
        return (_bitmap & _feature) != 0;
    }
}
