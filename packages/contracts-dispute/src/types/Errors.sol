// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

////////////////////////////////////////////////////////////////
//               `DisputeGame_Fault.sol` Errors               //
////////////////////////////////////////////////////////////////

/// @notice Thrown when a supplied bond is too low to cover the cost of the next possible counter claim.
error BondTooLow();

/// @notice Thrown when a defense against the root claim is attempted.
error CannotDefendRootClaim();

/// @notice Thrown when a claim is attempting to be made that already exists.
error ClaimAlreadyExists();
