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
     * @dev TODO: Update this to the value that we will use in prod.
     */
    uint256 internal constant MAX_GAME_DEPTH = 4;

    /**
     * @notice The duration of the game.
     * @dev TODO: Account for resolution buffer.
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
     */
    Position public leftMostPosition;

    /**
     * @notice Maps a unique ClaimHash to a Claim.
     */
    mapping(ClaimHash => Claim) public claims;
    /**
     * @notice Maps a unique ClaimHash to its parent.
     */
    mapping(ClaimHash => ClaimHash) public parents;
    /**
     * @notice Maps a unique ClaimHash to its position in the game tree.
     */
    mapping(ClaimHash => Position) public positions;
    /**
     * @notice Maps a unique ClaimHash to a Bond.
     */
    mapping(ClaimHash => BondAmount) public bonds;
    /**
     * @notice Maps a unique ClaimHash its chess clock.
     */
    mapping(ClaimHash => Clock) public clocks;
    /**
     * @notice Maps a unique ClaimHash to its reference counter.
     */
    mapping(ClaimHash => uint64) public rc;
    /**
     * @notice Tracks whether or not a unique ClaimHash has been countered.
     */
    mapping(ClaimHash => bool) public countered;

    ////////////////////////////////////////////////////////////////
    //                       External Logic                       //
    ////////////////////////////////////////////////////////////////

    /**
     * Attack a disagreed upon ClaimHash.
     * @param disagreement Disagreed upon ClaimHash
     * @param pivot The supplied pivot to the disagreement.
     */
    function attack(ClaimHash disagreement, Claim pivot) external {
        _move(disagreement, pivot, true);
    }

    // TODO: Go right instead of left
    // The pivot goes into the right subtree rather than the left subtree
    function defend(ClaimHash agreement, Claim pivot) external {
        _move(agreement, pivot, false);
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
     * @param claimHash The claim hash that the move is being made against.
     * @param pivot The pivot point claim provided in response to `claimHash`.
     * @param isAttack Whether or not the move is an attack or defense.
     */
    function _move(ClaimHash claimHash, Claim pivot, bool isAttack) internal {
        // TODO: Require & store bond for the pivot point claim

        // Get the position of the claimHash
        Position claimHashPos = positions[claimHash];

        // If the current depth of the claimHash is 0, revert. The root claim cannot be defended, only
        // attacked.
        if (LibPosition.depth(claimHashPos) == 0 && !isAttack) {
            revert CannotDefendRootClaim();
        }

        // If the `claimHash` is at max depth - 1, we can perform a step.
        if (LibPosition.depth(claimHashPos) == MAX_GAME_DEPTH - 1) {
            // TODO: Step
            revert("todo: unimplemented");
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
        // TODO: Good lord, this is a lot of storage usage. Devs do something

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
            // TODO: Correct this
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

    /**
     * @notice Initializes the `DisputeGame_Fault` contract.
     */
    function initialize() external {
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

        // TODO: Init bond (possibly should be done in the factory.)
    }

    /**
     * @inheritdoc IVersioned
     */
    function version() external pure override returns (string memory) {
        // TODO: Alias to constant is unnecessary.
        return VERSION;
    }

    /**
     * @notice Fetches the game type for the implementation of `IDisputeGame`.
     * @dev The reference impl should be entirely different depending on the type (fault, validity)
     *      i.e. The game type should indicate the security model.
     * @return _gameType The type of proof system being used.
     */
    function gameType() public pure override returns (GameType _gameType) {
        _gameType = GameType.FAULT;
    }

    /**
     * @notice Returns the timestamp that the DisputeGame contract was created at.
     */
    function createdAt() external view returns (Timestamp _createdAt) {
        return gameStart;
    }

    /**
     * @notice If all necessary information has been gathered, this function should mark the game
     *         status as either `CHALLENGER_WINS` or `DEFENDER_WINS` and return the status of
     *         the resolved game. It is at this stage that the bonds should be awarded to the
     *         necessary parties.
     * @dev May only be called if the `status` is `IN_PROGRESS`.
     */
    function resolve() external view returns (GameStatus _status) {
        // TODO
        return status;
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
        // TODO: What data do we need to pass along to this contract from the factory? Block hash, preimage data, etc.?
        _extraData = _getArgDynBytes(0x20, 0x20);
    }

    /**
     * @inheritdoc IDisputeGame
     */
    function gameData() external pure returns (GameType _gameType, Claim _rootClaim, bytes memory _extraData) {
        _gameType = gameType();
        _rootClaim = rootClaim();
        _extraData = extraData();
    }
}
