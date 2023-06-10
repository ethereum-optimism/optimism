// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { IDisputeGame } from "./interfaces/IDisputeGame.sol";
import { IVersioned } from "./interfaces/IVersioned.sol";
import { IFaultDisputeGame } from "./interfaces/IFaultDisputeGame.sol";
import { IInitializable } from "./interfaces/IInitializable.sol";
import { IBondManager } from "./interfaces/IBondManager.sol";

import { Clone } from "../libraries/Clone.sol";
import { LibHashing } from "./lib/LibHashing.sol";
import { LibPosition } from "./lib/LibPosition.sol";
import { LibClock } from "./lib/LibClock.sol";

import "../libraries/DisputeTypes.sol";
import "../libraries/DisputeErrors.sol";

/**
 * @title FaultDisputeGame
 * @author clabby <https://github.com/clabby>
 * @author protolambda <https://github.com/protolambda>
 * @notice An implementation of the `IFaultDisputeGame` interface.
 */
contract FaultDisputeGame is IFaultDisputeGame, Clone {
    ////////////////////////////////////////////////////////////////
    //                         State Vars                         //
    ////////////////////////////////////////////////////////////////

    /**
     * @notice The current Semver of the FaultDisputeGame implementation.
     */
    string internal constant VERSION = "0.0.1";

    /**
     * @notice The max depth of the game.
     * @dev TODO: Update this to the value that we will use in prod. Do we want to have the factory
     *            set this value? Should it be a constant?
     */
    uint256 internal constant MAX_GAME_DEPTH = 4;

    /**
     * @notice The duration of the game.
     * @dev TODO: Account for resolution buffer. (?)
     */
    Duration internal constant GAME_DURATION = Duration.wrap(7 days);

    /**
     * @notice The root claim's position is always at depth 0; index 0.
     */
    Position internal constant ROOT_POSITION = Position.wrap(0);

    /**
     * @notice The starting timestamp of the game
     */
    Timestamp public gameStart;

    /**
     * @inheritdoc IDisputeGame
     */
    GameStatus public status;

    /**
     * @inheritdoc IDisputeGame
     */
    IBondManager public bondManager;

    /**
     * @notice The left most, deepest position found during the resolution phase.
     * @dev Defaults to the position of the root claim, but will be set during the resolution
     *      phase to the left most, deepest position found (if any qualify.)
     * @dev TODO: Consider removing this if games can be resolved within a single block reliably.
     */
    Position public leftMostPosition;

    /**
     * @notice An append-only array of all claims made during the dispute game.
     */
    ClaimData[] public claimData;

    ////////////////////////////////////////////////////////////////
    //                       External Logic                       //
    ////////////////////////////////////////////////////////////////

    /**
     * @inheritdoc IFaultDisputeGame
     */
    function attack(uint256 parentIndex, Claim pivot) external payable {
        _move(parentIndex, pivot, true);
    }

    /**
     * @inheritdoc IFaultDisputeGame
     */
    function defend(uint256 parentIndex, Claim pivot) external payable {
        _move(parentIndex, pivot, false);
    }

    /**
     * @inheritdoc IFaultDisputeGame
     */
    function step(
        uint256 parentIndex,
        bytes32 stateHash,
        bytes calldata stateData,
        bytes calldata
    ) public {
        // TODO - Call the VM to perform the execution step.

        // Mock a state transition
        // NOTE: This mock lacks several necessary checks. For testing only.
        uint256 inp = abi.decode(stateData, (uint256));
        bytes32 nextStateHash = bytes32(uint256(stateHash) + inp);

        ClaimData memory parent = claimData[parentIndex];

        if (nextStateHash != Claim.unwrap(parent.claim)) {
            revert("Invalid state transition");
        }

        // If the state transition was successful, append a new claim to the game at the
        // `MAX_GAME_DEPTH`
        Position nextPosition = LibPosition.attack(parent.position);
        claimData.push(
            ClaimData({
                parentIndex: uint32(parentIndex),
                claim: Claim.wrap(nextStateHash),
                position: nextPosition,
                clock: Clock.wrap(0),
                rc: 0,
                countered: false
            })
        );
    }

    ////////////////////////////////////////////////////////////////
    //                       Internal Logic                       //
    ////////////////////////////////////////////////////////////////

    /**
     * @notice Internal move function, used by both `attack` and `defend`.
     * @param challengeIndex The index of the claim being moved against.
     * @param pivot The claim at the next logical position in the game.
     * @param isAttack Whether or not the move is an attack or defense.
     */
    function _move(
        uint256 challengeIndex,
        Claim pivot,
        bool isAttack
    ) internal {
        // Moves cannot be made unless the game is currently in progress.
        if (status != GameStatus.IN_PROGRESS) {
            revert GameNotInProgress();
        }

        // The only move that can be made against a root claim is an attack. This is because the
        // root claim commits to the entire state; Therefore, the only valid defense is to do
        // nothing if it is agreed with.
        if (challengeIndex == 0 && !isAttack) {
            revert CannotDefendRootClaim();
        }

        // Get the parent
        ClaimData memory parent = claimData[challengeIndex];

        // The parent must exist.
        if (Claim.unwrap(parent.claim) == bytes32(0)) {
            revert ParentDoesNotExist();
        }

        // Bump the parent's reference counter.
        claimData[challengeIndex].rc += 1;
        // Set the parent claim as countered.
        claimData[challengeIndex].countered = true;

        // Compute the position that the claim commits to. Because the parent's position is already
        // known, we can compute the next position by moving left or right depending on whether
        // or not the move is an attack or defense.
        Position nextPosition = isAttack
            ? LibPosition.attack(parent.position)
            : LibPosition.defend(parent.position);

        // At the leaf nodes of the game, the only option is to run a step to prove or disprove
        // the above claim. At this depth, the parent claim commits to the state after a single
        // instruction step.
        if (LibPosition.depth(nextPosition) >= MAX_GAME_DEPTH) {
            revert GameDepthExceeded();
        }

        // Fetch the grandparent clock, if it exists.
        // The grandparent clock should always exist unless the parent is the root claim.
        Clock grandparentClock;
        if (parent.parentIndex != type(uint32).max) {
            grandparentClock = claimData[parent.parentIndex].clock;
        }

        // Compute the duration of the next clock. This is done by adding the duration of the
        // grandparent claim to the difference between the current block timestamp and the
        // parent's clock timestamp.
        Duration nextDuration = Duration.wrap(
            uint64(
                // First, fetch the duration of the grandparent claim.
                Duration.unwrap(LibClock.duration(grandparentClock)) +
                    // Second, add the difference between the current block timestamp and the
                    // parent's clock timestamp.
                    block.timestamp -
                    Timestamp.unwrap(LibClock.timestamp(parent.clock))
            )
        );

        // Enforce the clock time rules. If the new clock duration is greater than half of the game
        // duration, then the move is invalid and cannot be made.
        if (Duration.unwrap(nextDuration) > Duration.unwrap(GAME_DURATION) >> 1) {
            revert ClockTimeExceeded();
        }

        // Construct the next clock with the new duration and the current block timestamp.
        Clock nextClock = LibClock.wrap(nextDuration, Timestamp.wrap(uint64(block.timestamp)));

        // Do not allow for a duplicate claim to be made.
        // TODO.
        // Maybe map the claimHash? There's no efficient way to check for this with the flat DAG.

        // Create the new claim.
        claimData.push(
            ClaimData({
                parentIndex: uint32(challengeIndex),
                claim: pivot,
                position: nextPosition,
                clock: nextClock,
                rc: 0,
                countered: false
            })
        );

        // Emit the appropriate event for the attack or defense.
        if (isAttack) {
            emit Attack(challengeIndex, pivot, msg.sender);
        } else {
            emit Defend(challengeIndex, pivot, msg.sender);
        }
    }

    ////////////////////////////////////////////////////////////////
    //                    `IDisputeGame` impl                     //
    ////////////////////////////////////////////////////////////////

    /**
     * @inheritdoc IDisputeGame
     */
    function gameType() public pure override returns (GameType _gameType) {
        _gameType = GameType.FAULT;
    }

    /**
     * @inheritdoc IDisputeGame
     */
    function createdAt() external view returns (Timestamp _createdAt) {
        return gameStart;
    }

    /**
     * @inheritdoc IDisputeGame
     */
    function resolve() external returns (GameStatus _status) {
        // The game may only be resolved if it is currently in progress.
        if (status != GameStatus.IN_PROGRESS) {
            revert GameNotInProgress();
        }

        // TODO: Block the game from being resolved if the preconditions for resolution have not
        //       been met.

        // Fetch the final index of the claim data DAG.
        uint256 i = claimData.length - 1;
        // Store a variable on the stack to keep track of the left most, deepest claim found during
        // the search.
        Position leftMost;

        // Run an exhaustive search (`O(n)`) over the DAG to find the left most, deepest
        // uncontested claim.
        for (; i > 0; --i) {
            ClaimData memory claim = claimData[i];

            // If the claim has no refereces, we can virtually prune it.
            if (claim.rc == 0) {
                Position position = claim.position;
                uint128 depth = LibPosition.depth(position);

                // 1. Do not count nodes at the max game depth. These can be truthy, but they do not
                //    give us any intuition about the final outcome of the game.
                // 2. Any node that has been countered is not a dangling claim, which is all that
                //    we're concerned about.
                // All claims that pass this check qualify for pruning.
                if (depth != MAX_GAME_DEPTH && !claim.countered) {
                    uint128 leftMostDepth = LibPosition.depth(leftMost);

                    // If the claim here is deeper than the current left most, deepest claim,
                    // update `leftMost`.
                    // If the claim here is at the same depth, but further left, update `leftMost`.
                    if (
                        depth > leftMostDepth ||
                        (depth == leftMostDepth &&
                            LibPosition.indexAtDepth(position) <=
                            LibPosition.indexAtDepth(leftMost))
                    ) {
                        leftMost = position;
                    }
                }

                // If the claim has a parent, decrement the reference count of the parent. This
                // effectively "prunes" the claim from the DAG without spending extra gas on
                // deleting it from storage.
                if (claim.parentIndex != type(uint32).max) {
                    --claimData[claim.parentIndex].rc;
                }
            }
        }

        // If the depth of the left most, deepest dangling claim is odd, the root was attacked
        // successfully and the defender wins. Otherwise, the challenger wins.
        if (LibPosition.depth(leftMost) % 2 == 0) {
            _status = GameStatus.DEFENDER_WINS;
        } else {
            _status = GameStatus.CHALLENGER_WINS;
        }

        // Emit the `Resolved` event.
        emit Resolved(_status);

        // Store the resolved status of the game.
        status = _status;
    }

    /**
     * @inheritdoc IDisputeGame
     */
    function rootClaim() public pure returns (Claim _rootClaim) {
        _rootClaim = Claim.wrap(_getArgFixedBytes(0x00));
    }

    /**
     * @inheritdoc IDisputeGame
     */
    function extraData() public pure returns (bytes memory _extraData) {
        // The extra data starts at the second word within the cwia calldata.
        // TODO: What data do we need to pass along to this contract from the factory?
        //       Block hash, preimage data, etc.?
        _extraData = _getArgDynBytes(0x20, 0x20);
    }

    /**
     * @inheritdoc IDisputeGame
     */
    function gameData()
        external
        pure
        returns (
            GameType _gameType,
            Claim _rootClaim,
            bytes memory _extraData
        )
    {
        _gameType = gameType();
        _rootClaim = rootClaim();
        _extraData = extraData();
    }

    /**
     * @inheritdoc IInitializable
     */
    function initialize() external {
        // Set the game start
        gameStart = Timestamp.wrap(uint64(block.timestamp));

        // Set the root claim
        claimData.push(
            ClaimData({
                parentIndex: type(uint32).max,
                claim: rootClaim(),
                position: ROOT_POSITION,
                clock: LibClock.wrap(Duration.wrap(0), Timestamp.wrap(uint64(block.timestamp))),
                rc: 0,
                countered: false
            })
        );
    }

    /**
     * @inheritdoc IVersioned
     */
    function version() external pure override returns (string memory) {
        return VERSION;
    }
}
