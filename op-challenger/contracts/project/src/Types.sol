// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/// @notice The type of proof system being used.
enum GameType {
    FAULT,
    VALIDITY,
    ATTESTATION
}

/// @notice A `Claim` type represents a 32 byte hash or other unique identifier.
/// @dev For the `FAULT` `GameType`, this will be a root of the merklized state of the fault proof
///      program at the end of the state transition.
///      For the `ATTESTATION` `GameType`, this will be an output root.
type Claim is bytes32;
