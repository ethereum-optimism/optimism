// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "./CannonTypes.sol";

// PreImageOracle Errors

/// @notice Thrown when a preimage is attempted to be read from the oracle that does not exist.
/// @param key The key of the preimage.
/// @param offset The offset of the preimage.
error MissingPreimage(PreimageKey key, PreimageOffset offset);

/// @notice Thrown when the caller is not permitted to write to the preimage oracle.
/// @param caller The caller of the function.
error UnauthorizedCaller(address caller);
