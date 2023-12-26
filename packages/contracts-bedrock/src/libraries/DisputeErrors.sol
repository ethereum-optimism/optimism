// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "./DisputeTypes.sol";

////////////////////////////////////////////////////////////////
//                `DisputeGameFactory` Errors                 //
////////////////////////////////////////////////////////////////

/// @notice Thrown when a dispute game is attempted to be created with an unsupported game type.
/// @param gameType The unsupported game type.
error NoImplementation(GameType gameType);

/// @notice Thrown when a dispute game that already exists is attempted to be created.
/// @param uuid The UUID of the dispute game that already exists.
error GameAlreadyExists(Hash uuid);

/// @notice Thrown when the root claim has an unexpected VM status.
///         Some games can only start with a root-claim with a specific status.
/// @param rootClaim is the claim that was unexpected.
error UnexpectedRootClaim(Claim rootClaim);

////////////////////////////////////////////////////////////////
//                 `FaultDisputeGame` Errors                  //
////////////////////////////////////////////////////////////////

/// @notice Thrown when a supplied bond is too low to cover the
///         cost of the next possible counter claim.
error BondTooLow();

/// @notice Thrown when the `extraData` passed to the CWIA proxy is too long for the `FaultDisputeGame`.
error ExtraDataTooLong();

/// @notice Thrown when a defense against the root claim is attempted.
error CannotDefendRootClaim();

/// @notice Thrown when a claim is attempting to be made that already exists.
error ClaimAlreadyExists();

/// @notice Thrown when a given claim is invalid (0).
error InvalidClaim();

/// @notice Thrown when an action that requires the game to be `IN_PROGRESS` is invoked when
///         the game is not in progress.
error GameNotInProgress();

/// @notice Thrown when a move is attempted to be made after the clock has timed out.
error ClockTimeExceeded();

/// @notice Thrown when the game is attempted to be resolved too early.
error ClockNotExpired();

/// @notice Thrown when a move is attempted to be made at or greater than the max depth of the game.
error GameDepthExceeded();

/// @notice Thrown when a step is attempted above the maximum game depth.
error InvalidParent();

/// @notice Thrown when an invalid prestate is supplied to `step`.
error InvalidPrestate();

/// @notice Thrown when a step is made that computes the expected post state correctly.
error ValidStep();

/// @notice Thrown when a game is attempted to be initialized with an L1 head that does
///         not contain the disputed output root.
error L1HeadTooOld();

/// @notice Thrown when an invalid local identifier is passed to the `addLocalData` function.
error InvalidLocalIdent();

/// @notice Thrown when resolving claims out of order.
error OutOfOrderResolution();

/// @notice Thrown when resolving a claim that has already been resolved.
error ClaimAlreadyResolved();

/// @notice Thrown when a parent output root is attempted to be found on a claim that is in
///         the output root portion of the tree.
error ClaimAboveSplit();

/// @notice Thrown on deployment if the split depth is greater than or equal to the max
///         depth of the game.
error InvalidSplitDepth();

////////////////////////////////////////////////////////////////
//              `AttestationDisputeGame` Errors               //
////////////////////////////////////////////////////////////////

/// @notice Thrown when an invalid signature is submitted to `challenge`.
error InvalidSignature();

/// @notice Thrown when a signature that has already been used to support the
///         `rootClaim` is submitted to `challenge`.
error AlreadyChallenged();

////////////////////////////////////////////////////////////////
//                      `Ownable` Errors                      //
////////////////////////////////////////////////////////////////

/// @notice Thrown when a function that is protected by the `onlyOwner` modifier
///          is called from an account other than the owner.
error NotOwner();

////////////////////////////////////////////////////////////////
//                    `BlockOracle` Errors                    //
////////////////////////////////////////////////////////////////

/// @notice Thrown when a block that is out of the range of the `BLOCKHASH` opcode
///         is attempted to be loaded.
error BlockNumberOOB();

/// @notice Thrown when a block hash is attempted to be loaded that has not been stored.
error BlockHashNotPresent();
