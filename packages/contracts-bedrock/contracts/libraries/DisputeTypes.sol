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
 * │ [0, 64)    │ Duration       │
 * │ [64, 128)  │ Timestamp      │
 * └────────────┴────────────────┘
 */
type Clock is uint128;

/**
 * @notice A `Position` represents a position of a claim within the game tree.
 * @dev The packed layout of this type is as follows:
 * ┌────────────┬────────────────┐
 * │    Bits    │     Value      │
 * ├────────────┼────────────────┤
 * │ [0, 64)    │ Depth          │
 * │ [64, 128)  │ Index at depth │
 * └────────────┴────────────────┘
 */
type Position is uint128;

/**
 * @notice A `GameType` represents the type of game being played.
 */
type GameType is uint8;

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
 * @title GameTypes
 * @notice A library that defines the IDs of games that can be played.
 */
library GameTypes {
    /**
     * @dev The game will use a `IDisputeGame` implementation that utilizes fault proofs.
     */
    GameType internal constant FAULT = GameType.wrap(0);

    /**
     * @dev The game will use a `IDisputeGame` implementation that utilizes validity proofs.
     */
    GameType internal constant VALIDITY = GameType.wrap(1);

    /**
     * @dev The game will use a `IDisputeGame` implementation that utilizes attestation proofs.
     */
    GameType internal constant ATTESTATION = GameType.wrap(2);
}
