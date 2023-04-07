/// @title Types
/// @author clabby <https://github.com/clabby>
/// @notice This library contains all of the types used in the Cannon contracts.

// SPDX-License-Identifier: MIT
pragma solidity ^0.8.17;

/// @notice The current status of the dispute game.
enum GameStatus
/// @dev The game is currently in progress, and has not been resolved.
{
    IN_PROGRESS,
    /// @dev The game has concluded, and the `rootClaim` was challenged successfully.
    CHALLENGER_WINS,
    /// @dev The game has concluded, and the `rootClaim` could not be contested.
    DEFENDER_WINS
}

/// @notice A claim represents an MPT root representing the state of the fault proof program.
type Claim is bytes32;

/// @notice A generalized index represents a position in the MPT.
type Gindex is uint256;

/// @notice A game type represents a type of dispute game.
type GameType is bytes32;

/// Keccak hash of abi.encodePacked(Claim, Gindex);
type GindexClaim is bytes32;

/// @notice A bond represents the amount of collateral that a user has locked up in a claim.
type Bond is uint256;

/// @notice A dedicated timestamp type.
type Timestamp is uint64;

/// @notice A dedicated duration type.
/// @dev Unit: seconds
type Duration is uint64;
