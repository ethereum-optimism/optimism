// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { LibHashing } from "src/dispute/lib/LibHashing.sol";
import {
    LibClaim,
    LibHash,
    LibDuration,
    LibClock,
    LibTimestamp,
    LibVMStatus,
    LibGameType
} from "src/dispute/lib/LibUDT.sol";
import { LibPosition } from "src/dispute/lib/LibPosition.sol";
import { LibGameId } from "src/dispute/lib/LibGameId.sol";

using LibClaim for Claim global;
using LibHashing for Claim global;
using LibHash for Hash global;
using LibPosition for Position global;
using LibDuration for Duration global;
using LibClock for Clock global;
using LibGameId for GameId global;
using LibTimestamp for Timestamp global;
using LibVMStatus for VMStatus global;
using LibGameType for GameType global;

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

/// @notice A `GameId` represents a packed 1 byte game ID, an 11 byte timestamp, and a 20 byte address.
/// @dev The packed layout of this type is as follows:
/// ┌───────────┬───────────┐
/// │   Bits    │   Value   │
/// ├───────────┼───────────┤
/// │ [0, 8)    │ Game Type │
/// │ [8, 96)   │ Timestamp │
/// │ [96, 256) │ Address   │
/// └───────────┴───────────┘
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
type GameType is uint32;

/// @notice A `VMStatus` represents the status of a VM execution.
type VMStatus is uint8;

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
    /// @dev A dispute game type the uses the cannon vm.
    GameType internal constant CANNON = GameType.wrap(0);

    /// @notice A dispute game type that uses an alphabet vm.
    ///         Not intended for production use.
    GameType internal constant ALPHABET = GameType.wrap(255);
}

/// @title VMStatuses
/// @notice Named type aliases for the various valid VM status bytes.
library VMStatuses {
    /// @notice The VM has executed successfully and the outcome is valid.
    VMStatus internal constant VALID = VMStatus.wrap(0);

    /// @notice The VM has executed successfully and the outcome is invalid.
    VMStatus internal constant INVALID = VMStatus.wrap(1);

    /// @notice The VM has paniced.
    VMStatus internal constant PANIC = VMStatus.wrap(2);

    /// @notice The VM execution is still in progress.
    VMStatus internal constant UNFINISHED = VMStatus.wrap(3);
}

/// @title LocalPreimageKey
/// @notice Named type aliases for local `PreimageOracle` key identifiers.
library LocalPreimageKey {
    /// @notice The identifier for the L1 head hash.
    uint256 internal constant L1_HEAD_HASH = 0x01;

    /// @notice The identifier for the starting output root.
    uint256 internal constant STARTING_OUTPUT_ROOT = 0x02;

    /// @notice The identifier for the disputed output root.
    uint256 internal constant DISPUTED_OUTPUT_ROOT = 0x03;

    /// @notice The identifier for the starting L2 block number.
    uint256 internal constant STARTING_L2_BLOCK_NUMBER = 0x04;

    /// @notice The identifier for the chain ID.
    uint256 internal constant CHAIN_ID = 0x05;
}
