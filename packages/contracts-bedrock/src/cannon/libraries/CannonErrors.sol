// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/// @notice Thrown when a passed part offset is out of bounds.
error PartOffsetOOB();

/// @notice Thrown when the input length to the keccak256 permutation is not a multiple of the block size.
error InvalidInputLength();

/// @notice Thrown when the claimed size of a large preimage is not equal to the actual size when the sponge
///         is squeezed.
error InvalidClaimedSize();
