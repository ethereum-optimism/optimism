// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "./types/Errors.sol";

import { Bond } from "src/types/Types.sol";
import { Clock } from "src/types/Types.sol";
import { Claim } from "src/types/Types.sol";
import { Duration } from "src/types/Types.sol";
import { Position } from "src/types/Types.sol";
import { GameType } from "src/types/Types.sol";
import { ClaimHash } from "src/types/Types.sol";
import { Timestamp } from "src/types/Types.sol";
import { GameStatus } from "src/types/Types.sol";

import { LibClock } from "src/lib/LibClock.sol";
import { LibHashing } from "src/lib/LibHashing.sol";
import { LibPosition } from "src/lib/LibPosition.sol";

import { Clone } from "src/util/Clone.sol";
import { Initializable } from "src/util/Initializable.sol";
import { IBondManager } from "src/interfaces/IBondManager.sol";
import { IFaultDisputeGame } from "src/interfaces/IFaultDisputeGame.sol";

/// @title FaultDisputeGame
/// @author clabby <https://github.com/clabby>
/// @author protolambda <https://github.com/protolambda>
/// @notice An implementation of the `IFaultDisputeGame` interface.
contract FaultDisputeGame is IFaultDisputeGame, Clone, Initializable {
    ////////////////////////////////////////////////////////////////
    //                         State Vars                         //
    ////////////////////////////////////////////////////////////////

    /// @notice The max depth of the game.
    /// @dev TODO: Update this to the value that we will use in prod.
    uint256 internal constant MAX_GAME_DEPTH = 4;

    /// @notice The duration of the game.
    /// @dev TODO: Account for resolution buffer.
    Duration internal constant GAME_DURATION = Duration.wrap(7 days);

    /// @notice The root claim's position is always at depth 0; index 0.
    Position internal constant ROOT_POSITION = Position.wrap(0);

    /// @notice The starting timestamp of the game
    Timestamp public gameStart;

    /// @notice The DisputeGame's bond manager.
    IBondManager public bondManager;

    /// @notice The left most, deepest position found during the resolution phase.
    /// @dev Defaults to the position of the root claim, but will be set during the resolution
    ///      phase to the left most, deepest position found (if any qualify.)
    Position public leftMostPosition;

    /// @notice Maps a unique ClaimHash to a Claim.
    mapping(ClaimHash => Claim) public claims;
    /// @notice Maps a unique ClaimHash to its parent.
    mapping(ClaimHash => ClaimHash) public parents;
    /// @notice Maps a unique ClaimHash to its position in the game tree.
    mapping(ClaimHash => Position) public positions;
    /// @notice Maps a unique ClaimHash to a Bond.
    mapping(ClaimHash => Bond) public bonds;
    /// @notice Maps a unique ClaimHash its chess clock.
    mapping(ClaimHash => Clock) public clocks;
    /// @notice Maps a unique ClaimHash to its reference counter.
    mapping(ClaimHash => uint64) public rc;
    /// @notice Tracks whether or not a unique ClaimHash has been countered.
    mapping(ClaimHash => bool) public countered;

    ////////////////////////////////////////////////////////////////
    //                       External Logic                       //
    ////////////////////////////////////////////////////////////////

    /// Attack a disagreed upon ClaimHash.
    /// @param disagreement Disagreed upon ClaimHash
    /// @param pivot The supplied pivot to the disagreement.
    function attack(ClaimHash disagreement, Claim pivot) external {
        _move(disagreement, pivot, true);
    }

    // TODO: Go right instead of left
    // The pivot goes into the right subtree rather than the left subtree
    function defend(ClaimHash agreement, Claim pivot) external {
        _move(agreement, pivot, false);
    }

    // TODO: Create a separate function to get a Gindex to claim

    /// @notice Performs a VM step
    function step(ClaimHash disagreement) public {
        // TODO
    }

    /// @notice Performs a VM step via an on-chain fault proof processor
    /// @dev This function should point to a fault proof processor in order to execute
    /// a step in the fault proof program on-chain. The interface of the fault proof processor
    /// contract should be generic enough such that we can use different fault proof VMs (MIPS, RiscV5, etc.)
    /// @param disagreement The GindexClaim of the disagreement
    // function step(GindexClaim disagreement) public {
    //     uint256 maxDepth = 40;
    //     uint256 depthCheckMask = 1 << maxDepth;

    //     // Retrieve the Gindex of the current disagreement.
    //     Gindex disagreementGindex = gIndicies[disagreement];

    //     // Check that the `disagreementGindex` is 1
    //     // If the max depth has been reached, we do not want to persist a subclaim
    //     require(Gindex.unwrap(disagreementGindex) >> maxDepth == 1, "The maximum depth has been reached");
    // }

    ////////////////////////////////////////////////////////////////
    //                       Internal Logic                       //
    ////////////////////////////////////////////////////////////////

    /// @notice Internal move function, used by both `attack` and `defend`.
    /// @param claimHash The claim hash that the move is being made against.
    /// @param pivot The pivot point claim provided in response to `claimHash`.
    /// @param isAttack Whether or not the move is an attack or defense.
    function _move(ClaimHash claimHash, Claim pivot, bool isAttack) internal {
        // TODO: Require & store bond for the pivot point claim

        // Get the position of the claimHash
        Position claimHashPos = positions[claimHash];

        if (LibPosition.depth(claimHashPos) == 0 && !isAttack) {
            revert CannotDefendRootClaim();
        }

        // If the `claimHash` is at max depth - 1, we can perform a step.
        if (LibPosition.depth(claimHashPos) == MAX_GAME_DEPTH - 1) {
            // TODO: Step
            revert("unimplemented");
        }

        // Get the position of the move.
        Position pivotPos = isAttack ? LibPosition.attack(claimHashPos) : LibPosition.defend(claimHashPos);

        // Compute the claim hash for the pivot point claim
        ClaimHash pivotClaimHash = LibHashing.hashClaimPos(pivot, pivotPos);

        // Revert if the same claim has already been made.
        // Note: We assume no one will ever claim the zero hash here.
        if (Claim.unwrap(claims[pivotClaimHash]) != bytes32(0)) {
            revert ClaimAlreadyExists();
        }

        // Store information about the counterclaim
        // TODO: Good lord, this is a lot of storage reading & writing. Devs do something

        // Map `pivotClaimHash` to `pivot` in the `claims` mapping.
        claims[pivotClaimHash] = pivot;

        // Map the `pivotClaimHash` to `pivotPos` in the `positions` mapping.
        positions[pivotClaimHash] = pivotPos;

        // Map the `pivotClaimHash` to `claimHash` in the `parents` mapping.
        parents[pivotClaimHash] = claimHash;

        // Increment the reference counter for the `claimHash` claim.
        rc[claimHash] += 1;

        // Mark `claimHash` as countered.
        countered[claimHash] = true;

        // Attempt to inherit grandparent's clock.
        ClaimHash claimHashParent = parents[claimHash];

        // If the grandparent claim doesn't exist, the disagreed upon claim is the root claim.
        // In this case, the mover's clock starts at half the game duration minus the time elapsed since the game started.
        if (ClaimHash.unwrap(claimHashParent) == bytes32(0)) {
            // Calculate the time since the game started
            Duration timeSinceGameStart = Duration.wrap(uint64(block.timestamp - Timestamp.unwrap(gameStart)));

            // Set the clock for the pivot point claim.
            clocks[pivotClaimHash] = LibClock.wrap(
                Duration.wrap((Duration.unwrap(GAME_DURATION) >> 1) - Duration.unwrap(timeSinceGameStart)),
                Timestamp.wrap(uint64(block.timestamp))
            );
        } else {
            Clock grandparentClock = clocks[claimHashParent];
            Clock parentClock = clocks[claimHash];

            // Calculate the remaining clock time for the pivot point claim.
            // Grandparent clock time - (block timestamp - parent clock timestamp)
            Clock newClock = LibClock.wrap(
                Duration.wrap(
                    uint64(
                        Duration.unwrap(LibClock.duration(grandparentClock))
                            - (block.timestamp - Timestamp.unwrap(LibClock.timestamp(parentClock)))
                    )
                ),
                Timestamp.wrap(uint64(block.timestamp))
            );

            // Store the remaining clock time for the pivot point claim.
            clocks[pivotClaimHash] = newClock;
        }

        // Emit the proper event for other challenge agents to pick up on.
        if (isAttack) {
            // Emit the `Attack` event for other challenge agents.
            emit Attack(pivotClaimHash, pivot, msg.sender);
        } else {
            // Emit the `Defend` event for other challenge agents.
            emit Defend(pivotClaimHash, pivot, msg.sender);
        }
    }

    ////////////////////////////////////////////////////////////////
    //                    `IDisputeGame` impl                     //
    ////////////////////////////////////////////////////////////////

    /// @notice Initializes the `DisputeGame_Fault` contract.
    function initialize() external initializer {
        // Grab the root claim from the CWIA calldata.
        Claim _rootClaim = rootClaim();

        // The root claim is hashed with the root gindex to create the root ClaimHash.
        ClaimHash rootClaimHash = LibHashing.hashClaimPos(_rootClaim, ROOT_POSITION);

        // The root claim is the first claim in the game.
        claims[rootClaimHash] = _rootClaim;

        // We do not need to set the position slot for the root claim; It is already zero.

        // The root claim's chess clock begins with half of the game duration.
        clocks[rootClaimHash] =
            LibClock.wrap(Duration.wrap(Duration.unwrap(GAME_DURATION) >> 1), Timestamp.wrap(uint64(block.timestamp)));

        // The game starts when the `init()` function is called.
        gameStart = Timestamp.wrap(uint64(block.timestamp));

        // TODO: Init bond
    }

    /// @notice Returns the semantic version of the DisputeGame contract.
    /// @dev Current version: 0.0.1
    function version() external pure override returns (string memory) {
        assembly {
            // Store the pointer to the string
            mstore(returndatasize(), 0x20)
            // Store the version ("0.0.1")
            // len |   "0.0.1"
            // 0x05|302E302E31
            mstore(0x25, 0x05302E302E31)
            // Return the semantic version of the contract
            return(returndatasize(), 0x60)
        }
    }

    ////////////////////////////////////////////////////////////////
    //                            CWIA                            //
    ////////////////////////////////////////////////////////////////

    /// @notice Fetches the game type from the calldata appended by the CWIA proxy.
    /// @dev `clones-with-immutable-args` argument #1
    /// @dev The reference impl should be entirely different depending on the type (fault, validity)
    ///      i.e. The game type should indicate the security model.
    /// @return _gameType The type of proof system being used.
    function gameType() public pure override returns (GameType _gameType) {
        _gameType = GameType.wrap(bytes32(_getArgUint256(0)));
    }

    /// @notice Fetches the root claim from the calldata appended by the CWIA proxy.
    /// @return _rootClaim The root claim of the DisputeGame.
    /// @dev `clones-with-immutable-args` argument #2
    function rootClaim() public pure returns (Claim _rootClaim) {
        _rootClaim = Claim.wrap(bytes32(_getArgUint256(0x20)));
    }

    /// @notice Returns the timestamp that the DisputeGame contract was created at.
    function createdAt() external view returns (Timestamp _createdAt) {
        return gameStart;
    }

    // TODO: get the correct game status
    /// @notice Returns the current status of the game.
    function status() external pure returns (GameStatus _status) {
        return GameStatus.IN_PROGRESS;
    }

    // TODO: read extrabytes correctly
    /// @notice Getter for the extra data.
    /// @dev `clones-with-immutable-args` argument #3
    /// @return _extraData Any extra data supplied to the dispute game contract by the creator.
    function extraData() external pure returns (bytes memory _extraData) {
        _extraData = bytes("");
    }

    // TODO: properly resolve the game
    /// @notice If all necessary information has been gathered, this function should mark the game
    ///         status as either `CHALLENGER_WINS` or `DEFENDER_WINS` and return the status of
    ///         the resolved game. It is at this stage that the bonds should be awarded to the
    ///         necessary parties.
    /// @dev May only be called if the `status` is `IN_PROGRESS`.
    function resolve() external pure returns (GameStatus _status) {
        return GameStatus.IN_PROGRESS;
    }
}
