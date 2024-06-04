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

/// @notice Thrown when data is submitted out of order in a large preimage proposal.
error WrongStartingBlock();

/// @notice Thrown when the pre and post states passed aren't contiguous.
error StatesNotContiguous();

/// @notice Thrown when the permutation yields the expected result.
error PostStateMatches();

/// @notice Thrown when the preimage is too large to fit in the tree.
error TreeSizeOverflow();

/// @notice Thrown when the preimage proposal has already been finalized.
error AlreadyFinalized();

/// @notice Thrown when the proposal has not matured past the challenge period.
error ActiveProposal();

/// @notice Thrown when attempting to finalize a proposal that has been challenged.
error BadProposal();

/// @notice Thrown when attempting to add leaves to a preimage proposal that has not been initialized.
error NotInitialized();

/// @notice Thrown when the caller of a function is not an EOA.
error NotEOA();

/// @notice Thrown when an insufficient bond is provided for a large preimage proposal.
error InsufficientBond();

/// @notice Thrown when a bond transfer fails.
error BondTransferFailed();
