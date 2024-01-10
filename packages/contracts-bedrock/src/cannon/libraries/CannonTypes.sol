// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/// @notice Packed LPP metadata.
/// ┌────────────┬───────────┐
/// │  [0, 255)  │ Timestamp │
/// ├────────────┼───────────┤
/// │ [255, 256) │ Countered │
/// └────────────┴───────────┘
type LPPMetaData is bytes32;
