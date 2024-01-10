// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/// @notice Thrown when a passed part offset is out of bounds.
error PartOffsetOOB();

/// @notice Thrown when a merkle proof fails to verify.
error InvalidProof();

/// @notice Thrown when the prestate preimage doesn't match the claimed preimage.
error InvalidPreimage();

/// @notice Thrown when a leaf with an invalid input size is added.
error InvalidInputSize();

/// @notice Thrown when the pre and post states passed aren't contiguous.
error StatesNotContiguous();

/// @notice Thrown when the permutation yields the expected result.
error PostStateMatches();
