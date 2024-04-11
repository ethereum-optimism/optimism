// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @notice Error for when a deposit or withdrawal is to a bad target.
error BadTarget();
/// @notice Error for when a deposit has too much calldata.
error LargeCalldata();
/// @notice Error for when a deposit has too small of a gas limit.
error SmallGasLimit();
/// @notice Error for when a withdrawal transfer fails.
error TransferFailed();
/// @notice Error for when a method is called that only works when using a custom gas token.
error OnlyCustomGasToken();
/// @notice Error for when a method cannot be called with non zero CALLVALUE.
error NoValue();
/// @notice Error for an unauthorized CALLER.
error Unauthorized();
/// @notice Error for when a method cannot be called when paused. This could be renamed
///         to `Paused` in the future, but it collides with the `Paused` event.
error CallPaused();
/// @notice Error for invalid proof.
error InvalidOutputRootProof();
/// @notice Error for invalid proof.
error InvalidInclusionProof();
/// @notice Error when attempting to prove the same withdrawal more than once.
error AlreadyProven();
/// @notice Reentrancy guard.
error NonReentrant();
/// @notice Error for attempting to finalize a withdrawal that is not yet proven.
error NotProven();
/// @notice Error for when the withdrawal proof timestamp makes no sense.
error BadTimestamp();
/// @notice Error for when trying to finalize a withdrawal too early.
error TooEarly();
/// @notice Error for when the output root down't match the proof.
error BadOutputRoot();
/// @notice Error for when the withdrawal has already been processed.
error AlreadyFinalized();
/// @notice Error for when the game type is invalid.
error InvalidGameType();
/// @notice Error for when the dispute game is created early.
error DisputeGameCreatedEarly();
/// @notice Error for when the dispute game has been blacklisted.
error Blacklisted();
/// @notice Error for when the proposal isn't valid.
error InvalidProposal();
/// @notice Error for when the proposal is in air gap.
error AirGapped();
/// @notice Error for special gas estimation.
error GasEstimation();
/// @notice Error for when the dispute game is invalid.
error InvalidDisputeGame();
