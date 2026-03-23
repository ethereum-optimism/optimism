// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @notice Error returns when a non-depositor account tries to set L1 block values.
error NotDepositor();

/// @notice Error when attempting to enable a feature that is already enabled.
error L1Block_FeatureAlreadyEnabled();

/// @notice Error when a caller is not authorized to set a feature.
error L1Block_NotAuthorizedToSetFeature();
