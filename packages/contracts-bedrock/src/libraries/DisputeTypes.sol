// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { LibHashing } from "../dispute/lib/LibHashing.sol";
import { LibPosition } from "../dispute/lib/LibPosition.sol";
import { LibClock } from "../dispute/lib/LibClock.sol";

using LibHashing for Claim global;
using LibPosition for Position global;
using LibClock for Clock global;

/// @notice A custom type for a generic hash.
type Hash is bytes32;

/// @notice A claim represents an MPT root representing the state of the fault proof program.
type Claim is bytes32;

/// @notice A claim hash represents a hash of a claim and a position within the game tree.
/// @dev Keccak hash of abi.encodePacked(Claim, Position);
type ClaimHash is bytes32;

/// @notice A bondamount represents the amount of collateral that a user has locked up in a claim.
type BondAmount is uint256;

/// @notice A dedicated timestamp type.
type Timestamp is uint64;

/// @notice A dedicated duration type.
/// @dev Unit: seconds
type Duration is uint64;

/// @notice A `GameId` represents a packed 12 byte timestamp and a 20 byte address.
/// @dev The packed layout of this type is as follows:
/// ┌────────────┬────────────────┐
/// │    Bits    │     Value      │
/// ├────────────┼────────────────┤
/// │ [0, 96)    │ Timestamp      │
/// │ [96, 256)  │ Address        │
/// └────────────┴────────────────┘
type GameId is bytes32;

/// @notice A `Clock` represents a packed `Duration` and `Timestamp`
/// @dev The packed layout of this type is as follows:
/// ┌────────────┬────────────────┐
/// │    Bits    │     Value      │
/// ├────────────┼────────────────┤
/// │ [0, 64)    │ Duration       │
/// │ [64, 128)  │ Timestamp      │
/// └────────────┴────────────────┘
type Clock is uint128;

/// @notice A `Position` represents a position of a claim within the game tree.
/// @dev This is represented as a "generalized index" where the high-order bit
/// is the level in the tree and the remaining bits is a unique bit pattern, allowing
/// a unique identifier for each node in the tree. Mathematically, it is calculated
/// as 2^{depth} + indexAtDepth.
type Position is uint128;

/// @notice A `GameType` represents the type of game being played.
type GameType is uint8;

/// @notice The current status of the dispute game.
enum GameStatus {
    // The game is currently in progress, and has not been resolved.
    IN_PROGRESS,
    // The game has concluded, and the `rootClaim` was challenged successfully.
    CHALLENGER_WINS,
    // The game has concluded, and the `rootClaim` could not be contested.
    DEFENDER_WINS
}

/// @title GameTypes
/// @notice A library that defines the IDs of games that can be played.
library GameTypes {
    /// @dev The game will use a `IDisputeGame` implementation that utilizes fault proofs.
    GameType internal constant FAULT = GameType.wrap(0);

    /// @dev The game will use a `IDisputeGame` implementation that utilizes validity proofs.
    GameType internal constant VALIDITY = GameType.wrap(1);

    /// @dev The game will use a `IDisputeGame` implementation that utilizes attestation proofs.
    GameType internal constant ATTESTATION = GameType.wrap(2);
}
