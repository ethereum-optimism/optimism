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
/// @notice Error for special gas estimation.
error GasEstimation();
/// @notice Error for when a method is being reentered.
error NonReentrant();
