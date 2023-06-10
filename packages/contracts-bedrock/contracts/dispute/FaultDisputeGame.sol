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
     * @notice The current status of the game.
     */
    GameStatus public status;

    /**
     * @notice The DisputeGame's bond manager.
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
     * @notice Performs a VM step via an on-chain fault proof processor
     * @dev This function should point to a fault proof processor in order to execute
     * a step in the fault proof program on-chain. The interface of the fault proof processor
     * contract should be generic enough such that we can use different fault proof VMs (MIPS, RiscV5, etc.)
     * @param disagreement The GindexClaim of the disagreement
     */
    function step(ClaimHash disagreement) public {
        // TODO - Call the VM to perform the execution step.
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

        // Compute the clock for the next claim. The clock's duration is computed by taking the
        // difference between the current block timestamp and the parent's clock timestamp.
        Clock nextClock = LibClock.wrap(
            Duration.wrap(
                uint64(block.timestamp - Timestamp.unwrap(LibClock.timestamp(parent.clock)))
            ),
            Timestamp.wrap(uint64(block.timestamp))
        );

        // Enforce the clock time. If the new clock duration is greater than half of the game
        // duration, then the move is invalid and cannot be made.
        if (Duration.unwrap(LibClock.duration(nextClock)) > Duration.unwrap(GAME_DURATION) >> 1) {
            revert ClockTimeExceeded();
        }

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
        for (; i > 0; i--) {
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
                    // If the claim here is deeper than the current left most, deepest claim,
                    // update `leftMost`.
                    // If the claim here is at the same depth, but further left, update `leftMost`.
                    if (
                        depth > LibPosition.depth(leftMost) ||
                        (depth == LibPosition.depth(leftMost) &&
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
                    claimData[claim.parentIndex].rc -= 1;
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
