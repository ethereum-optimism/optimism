// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "./DisputeTypes.sol";

////////////////////////////////////////////////////////////////
//                `DisputeGameFactory` Errors                 //
////////////////////////////////////////////////////////////////

/**
 * @notice Thrown when a dispute game is attempted to be created with an unsupported game type.
 * @param gameType The unsupported game type.
 */
error NoImplementation(GameType gameType);

/**
 * @notice Thrown when a dispute game that already exists is attempted to be created.
 * @param uuid The UUID of the dispute game that already exists.
 */
error GameAlreadyExists(Hash uuid);

////////////////////////////////////////////////////////////////
//               `DisputeGame_Fault.sol` Errors               //
////////////////////////////////////////////////////////////////

/**
 * @notice Thrown when a supplied bond is too low to cover the
 *         cost of the next possible counter claim.
 */
error BondTooLow();

/**
 * @notice Thrown when a defense against the root claim is attempted.
 */
error CannotDefendRootClaim();

/**
 * @notice Thrown when a claim is attempting to be made that already exists.
 */
error ClaimAlreadyExists();

/**
 * @notice Thrown when a given claim is invalid (0).
 */
error InvalidClaim();

/**
 * @notice Thrown when an action that requires the game to be `IN_PROGRESS` is invoked when
 *         the game is not in progress.
 */
error GameNotInProgress();

/**
 * @notice Thrown when a move is attempted to be made after the clock has timed out.
 */
error ClockTimeExceeded();

/**
 * @notice Thrown when a move is attempted to be made at or greater than the max depth of the game.
 */
error GameDepthExceeded();

////////////////////////////////////////////////////////////////
//              `AttestationDisputeGame` Errors               //
////////////////////////////////////////////////////////////////

/**
 * @notice Thrown when an invalid signature is submitted to `challenge`.
 */
error InvalidSignature();

/**
 * @notice Thrown when a signature that has already been used to support the
 *         `rootClaim` is submitted to `challenge`.
 */
error AlreadyChallenged();

////////////////////////////////////////////////////////////////
//                      `Ownable` Errors                      //
////////////////////////////////////////////////////////////////

/**
 * @notice Thrown when a function that is protected by the `onlyOwner` modifier
 *          is called from an account other than the owner.
 */
error NotOwner();
