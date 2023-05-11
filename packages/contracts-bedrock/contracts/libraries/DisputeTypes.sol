// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

/**
 * @notice A custom type for a generic hash.
 */
type Hash is bytes32;

/**
 * @notice A claim represents an MPT root representing the state of the fault proof program.
 */
type Claim is bytes32;

/**
 * @notice A claim hash represents a hash of a claim and a position within the game tree.
 * @dev Keccak hash of abi.encodePacked(Claim, Position);
 */
type ClaimHash is bytes32;

/**
 * @notice A bondamount represents the amount of collateral that a user has locked up in a claim.
 */
type BondAmount is uint256;

/**
 * @notice A dedicated timestamp type.
 */
type Timestamp is uint64;

/**
 * @notice A dedicated duration type.
 * @dev Unit: seconds
 */
type Duration is uint64;

/**
 * @notice A `Clock` represents a packed `Duration` and `Timestamp`
 * @dev The packed layout of this type is as follows:
 * ┌────────────┬────────────────┐
 * │    Bits    │     Value      │
 * ├────────────┼────────────────┤
 * │ [0, 128)   │ Duration       │
 * │ [128, 256) │ Timestamp      │
 * └────────────┴────────────────┘
 */
type Clock is uint256;

/**
 * @notice A `Position` represents a position of a claim within the game tree.
 * @dev The packed layout of this type is as follows:
 * ┌────────────┬────────────────┐
 * │    Bits    │     Value      │
 * ├────────────┼────────────────┤
 * │ [0, 128)   │ Depth          │
 * │ [128, 256) │ Index at depth │
 * └────────────┴────────────────┘
 */
type Position is uint256;

/**
 * @notice The current status of the dispute game.
 */
enum GameStatus {
    // The game is currently in progress, and has not been resolved.
    IN_PROGRESS,
    // The game has concluded, and the `rootClaim` was challenged successfully.
    CHALLENGER_WINS,
    // The game has concluded, and the `rootClaim` could not be contested.
    DEFENDER_WINS
}

/**
 * @notice The type of proof system being used.
 */
enum GameType {
    // The game will use a `IDisputeGame` implementation that utilizes fault proofs.
    FAULT,
    // The game will use a `IDisputeGame` implementation that utilizes validity proofs.
    VALIDITY,
    // The game will use a `IDisputeGame` implementation that utilizes attestation proofs.
    ATTESTATION
}
