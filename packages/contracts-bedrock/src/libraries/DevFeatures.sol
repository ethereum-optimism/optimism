// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @notice Library of constants representing development features. We use a 32 byte bitmap because
///         it's easier to integrate with op-deployer. Note that users should typically set a
///         single nibble to 1 and the rest to zero, which gives us 64 potential features, like:
///         0x0000000000000000000000000000000000000000000000000000000000000001
///         0x0000000000000000000000000000000000000000000000000000000000000010
///         0x0000000000000000000000000000000000000000000000000000000000000100
///         etc.
///         We'll expand to using all available bits if we need more than 64 concurrent features.
library DevFeatures {
    /// @notice The feature that enables the OptimismPortalInterop contract.
    bytes32 public constant OPTIMISM_PORTAL_INTEROP =
        bytes32(0x0000000000000000000000000000000000000000000000000000000000000001);

    /// @notice The feature that enables deployment of the CANNON_KONA fault dispute game.
    /// @custom:legacy
    /// This feature is no longer used, but is kept here for legacy reasons.
    bytes32 public constant CANNON_KONA = bytes32(0x0000000000000000000000000000000000000000000000000000000000000010);

    /// @notice The feature that enables deployment of V2 dispute game contracts.
    /// @custom:legacy
    /// This feature is no longer used, but is kept here for legacy reasons.
    bytes32 public constant DEPLOY_V2_DISPUTE_GAMES =
        bytes32(0x0000000000000000000000000000000000000000000000000000000000000100);

    /// @notice Checks if a feature is enabled in a bitmap. Note that this function does not check
    ///         that the input feature represents a single feature and the bitwise AND operation
    ///         allows for multiple features to be enabled at once. Users should generally check
    ///         for only a single feature at a time.
    /// @param _bitmap The bitmap to check.
    /// @param _feature The feature to check.
    /// @return True if the feature is enabled, false otherwise.
    function isDevFeatureEnabled(bytes32 _bitmap, bytes32 _feature) internal pure returns (bool) {
        return _feature != 0 && (_bitmap & _feature) == _feature;
    }
}
