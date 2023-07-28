// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { IDisputeGame } from "./interfaces/IDisputeGame.sol";
import { IFaultDisputeGame } from "./interfaces/IFaultDisputeGame.sol";
import { IInitializable } from "./interfaces/IInitializable.sol";
import { IBondManager } from "./interfaces/IBondManager.sol";
import { IBigStepper, IPreimageOracle } from "./interfaces/IBigStepper.sol";
import { L2OutputOracle } from "../L1/L2OutputOracle.sol";

import { Clone } from "../libraries/Clone.sol";
import { Types } from "../libraries/Types.sol";
import { Semver } from "../universal/Semver.sol";
import { LibHashing } from "./lib/LibHashing.sol";
import { LibPosition } from "./lib/LibPosition.sol";
import { LibClock } from "./lib/LibClock.sol";

import "../libraries/DisputeTypes.sol";
import "../libraries/DisputeErrors.sol";

/// @title FaultDisputeGame
/// @notice An implementation of the `IFaultDisputeGame` interface.
contract FaultDisputeGame is IFaultDisputeGame, Clone, Semver {
    ////////////////////////////////////////////////////////////////
    //                         State Vars                         //
    ////////////////////////////////////////////////////////////////

    /// @notice The absolute prestate of the instruction trace. This is a constant that is defined
    ///         by the program that is being used to execute the trace.
    Claim public immutable ABSOLUTE_PRESTATE;

    /// @notice The max depth of the game.
    uint256 public immutable MAX_GAME_DEPTH;

    /// @notice The duration of the game.
    Duration public immutable GAME_DURATION;

    /// @notice An onchain VM that performs single instruction steps on a fault proof program trace.
    IBigStepper public immutable VM;

    /// @notice The trusted L2OutputOracle contract.
    L2OutputOracle public immutable L2_OUTPUT_ORACLE;

    /// @notice The root claim's position is always at gindex 1.
    Position internal constant ROOT_POSITION = Position.wrap(1);

    /// @notice The starting timestamp of the game
    Timestamp public gameStart;

    /// @inheritdoc IDisputeGame
    GameStatus public status;

    /// @inheritdoc IDisputeGame
    IBondManager public bondManager;

    /// @inheritdoc IFaultDisputeGame
    Hash public l1Head;

    /// @notice An append-only array of all claims made during the dispute game.
    ClaimData[] public claimData;

    /// @notice An internal mapping to allow for constant-time lookups of existing claims.
    mapping(ClaimHash => bool) internal claims;

    /// @param _absolutePrestate The absolute prestate of the instruction trace.
    /// @param _maxGameDepth The maximum depth of bisection.
    /// @param _gameDuration The duration of the game.
    /// @param _vm An onchain VM that performs single instruction steps on a fault proof program
    ///            trace.
    /// @param _l2oo The trusted L2OutputOracle contract.
    /// @custom:semver 0.0.4
    constructor(
        Claim _absolutePrestate,
        uint256 _maxGameDepth,
        Duration _gameDuration,
        IBigStepper _vm,
        L2OutputOracle _l2oo
    ) Semver(0, 0, 4) {
        ABSOLUTE_PRESTATE = _absolutePrestate;
        MAX_GAME_DEPTH = _maxGameDepth;
        GAME_DURATION = _gameDuration;
        VM = _vm;
        L2_OUTPUT_ORACLE = _l2oo;
    }

    ////////////////////////////////////////////////////////////////
    //                  `IFaultDisputeGame` impl                  //
    ////////////////////////////////////////////////////////////////

    /// @inheritdoc IFaultDisputeGame
    function step(
        uint256 _claimIndex,
        bool _isAttack,
        bytes calldata _stateData,
        bytes calldata _proof
    ) external {
        // INVARIANT: Steps cannot be made unless the game is currently in progress.
        if (status != GameStatus.IN_PROGRESS) revert GameNotInProgress();

        // Get the parent. If it does not exist, the call will revert with OOB.
        ClaimData storage parent = claimData[_claimIndex];

        // Pull the parent position out of storage.
        Position parentPos = parent.position;
        // Determine the position of the step.
        Position stepPos = parentPos.move(_isAttack);

        // INVARIANT: A step cannot be made unless the move position is 1 below the `MAX_GAME_DEPTH`
        if (stepPos.depth() != MAX_GAME_DEPTH + 1) revert InvalidParent();

        // Determine the expected pre & post states of the step.
        Claim preStateClaim;
        ClaimData storage postState;
        if (_isAttack) {
            if (stepPos.indexAtDepth() == 0) {
                // If the step position's index at depth is 0, the prestate is the absolute
                // prestate.
                preStateClaim = ABSOLUTE_PRESTATE;
            } else {
                // If the step is an attack at a trace index > 0, the prestate exists elsewhere in
                // the game state.
                preStateClaim = findTraceAncestor(
                    Position.wrap(Position.unwrap(parentPos) - 1),
                    parent.parentIndex
                ).claim;
            }

            // For all attacks, the poststate is the parent claim.
            postState = parent;
        } else {
            // If the step is a defense, the poststate exists elsewhere in the game state,
            // and the parent claim is the expected pre-state.
            preStateClaim = parent.claim;
            postState = findTraceAncestor(
                Position.wrap(Position.unwrap(parentPos) + 1),
                parent.parentIndex
            );
        }

        // INVARIANT: The prestate is always invalid if the passed `_stateData` is not the
        //            preimage of the prestate claim hash.
        if (keccak256(_stateData) != Claim.unwrap(preStateClaim)) revert InvalidPrestate();

        // INVARIANT: If a step is an attack, the poststate is valid if the step produces
        //            the same poststate hash as the parent claim's value.
        //            If a step is a defense:
        //              1. If the parent claim and the found post state agree with each other
        //                 (depth diff % 2 == 0), the step is valid if it produces the same
        //                 state hash as the post state's claim.
        //              2. If the parent claim and the found post state disagree with each other
        //                 (depth diff % 2 != 0), the parent cannot be countered unless the step
        //                 produces the same state hash as `postState.claim`.
        // SAFETY:    While the `attack` path does not need an extra check for the post
        //            state's depth in relation to the parent, we don't need another
        //            branch because (n - n) % 2 == 0.
        bool validStep = VM.step(_stateData, _proof) == Claim.unwrap(postState.claim);
        bool parentPostAgree = (parentPos.depth() - postState.position.depth()) % 2 == 0;
        if (parentPostAgree == validStep) revert ValidStep();

        // Set the parent claim as countered. We do not need to append a new claim to the game;
        // instead, we can just set the existing parent as countered.
        parent.countered = true;
    }

    /// @notice Internal move function, used by both `attack` and `defend`.
    /// @param _challengeIndex The index of the claim being moved against.
    /// @param _claim The claim at the next logical position in the game.
    /// @param _isAttack Whether or not the move is an attack or defense.
    function move(
        uint256 _challengeIndex,
        Claim _claim,
        bool _isAttack
    ) public payable {
        // INVARIANT: Moves cannot be made unless the game is currently in progress.
        if (status != GameStatus.IN_PROGRESS) revert GameNotInProgress();

        // INVARIANT: A defense can never be made against the root claim. This is because the root
        //            claim commits to the entire state. Therefore, the only valid defense is to
        //            do nothing if it is agreed with.
        if (_challengeIndex == 0 && !_isAttack) revert CannotDefendRootClaim();

        // Get the parent. If it does not exist, the call will revert with OOB.
        ClaimData memory parent = claimData[_challengeIndex];

        // Set the parent claim as countered.
        claimData[_challengeIndex].countered = true;

        // Compute the position that the claim commits to. Because the parent's position is already
        // known, we can compute the next position by moving left or right depending on whether
        // or not the move is an attack or defense.
        Position nextPosition = parent.position.move(_isAttack);

        // INVARIANT: A move can never surpass the `MAX_GAME_DEPTH`. The only option to counter a
        //            claim at this depth is to perform a single instruction step on-chain via
        //            the `step` function to prove that the state transition produces an unexpected
        //            post-state.
        if (nextPosition.depth() > MAX_GAME_DEPTH) revert GameDepthExceeded();

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
                Duration.unwrap(grandparentClock.duration()) +
                    // Second, add the difference between the current block timestamp and the
                    // parent's clock timestamp.
                    block.timestamp -
                    Timestamp.unwrap(parent.clock.timestamp())
            )
        );

        // INVARIANT: A move can never be made once its clock has exceeded `GAME_DURATION / 2`
        //            seconds of time.
        if (Duration.unwrap(nextDuration) > Duration.unwrap(GAME_DURATION) >> 1) {
            revert ClockTimeExceeded();
        }

        // Construct the next clock with the new duration and the current block timestamp.
        Clock nextClock = LibClock.wrap(nextDuration, Timestamp.wrap(uint64(block.timestamp)));

        // INVARIANT: A claim may only exist at a given position once. Multiple claims may exist
        //            at the same position, however they must have different values.
        ClaimHash claimHash = _claim.hashClaimPos(nextPosition);
        if (claims[claimHash]) revert ClaimAlreadyExists();
        claims[claimHash] = true;

        // Create the new claim.
        claimData.push(
            ClaimData({
                parentIndex: uint32(_challengeIndex),
                claim: _claim,
                position: nextPosition,
                clock: nextClock,
                countered: false
            })
        );

        // Emit the appropriate event for the attack or defense.
        emit Move(_challengeIndex, _claim, msg.sender);
    }

    /// @inheritdoc IFaultDisputeGame
    function attack(uint256 _parentIndex, Claim _claim) external payable {
        move(_parentIndex, _claim, true);
    }

    /// @inheritdoc IFaultDisputeGame
    function defend(uint256 _parentIndex, Claim _claim) external payable {
        move(_parentIndex, _claim, false);
    }

    /// @inheritdoc IFaultDisputeGame
    function addLocalData(uint256 _ident, uint256 _partOffset) external {
        // INVARIANT: Local data can only be added if the game is currently in progress.
        if (status != GameStatus.IN_PROGRESS) revert GameNotInProgress();

        IPreimageOracle oracle = VM.oracle();
        if (_ident == 1) {
            // Load the L1 head hash into the game's local context in the preimage oracle.
            oracle.loadLocalData(_ident, Hash.unwrap(l1Head), 32, _partOffset);
        } else if (_ident == 2) {
            // Load the earliest output root that commits to the passed L2 block number
            // into the game's local context in the preimage oracle.
            Types.OutputProposal memory proposal = L2_OUTPUT_ORACLE.getL2OutputAfter(
                l2BlockNumber()
            );
            oracle.loadLocalData(_ident, proposal.outputRoot, 32, _partOffset);
        } else if (_ident == 3) {
            // Load the root claim into the game's local context in the preimage oracle.
            oracle.loadLocalData(_ident, Claim.unwrap(rootClaim()), 32, _partOffset);
        } else if (_ident == 4) {
            // Load the L2 block number into the game's local context in the preimage oracle.
            // The L2 block number is stored as a big-endian uint64 in the upper 8 bytes of the
            // passed word.
            oracle.loadLocalData(_ident, bytes32(l2BlockNumber() << 192), 8, _partOffset);
        } else if (_ident == 5) {
            // Load the chain ID into the game's local context in the preimage oracle.
            // The chain ID is stored as a big-endian uint64 in the upper 8 bytes of the
            // passed word.
            oracle.loadLocalData(_ident, bytes32(block.chainid << 192), 8, _partOffset);
        }
    }

    /// @inheritdoc IFaultDisputeGame
    function l2BlockNumber() public pure returns (uint256 l2BlockNumber_) {
        l2BlockNumber_ = _getArgUint256(0x20);
    }

    ////////////////////////////////////////////////////////////////
    //                    `IDisputeGame` impl                     //
    ////////////////////////////////////////////////////////////////

    /// @inheritdoc IDisputeGame
    function gameType() public pure override returns (GameType gameType_) {
        gameType_ = GameTypes.FAULT;
    }

    /// @inheritdoc IDisputeGame
    function createdAt() external view returns (Timestamp createdAt_) {
        createdAt_ = gameStart;
    }

    /// @inheritdoc IDisputeGame
    function resolve() external returns (GameStatus status_) {
        // INVARIANT: Resolution cannot occur unless the game is currently in progress.
        if (status != GameStatus.IN_PROGRESS) revert GameNotInProgress();

        // Search for the left-most dangling non-bottom node
        // The most recent claim is always a dangling, non-bottom node so we start with that
        uint256 leftMostIndex = claimData.length - 1;
        uint256 leftMostTraceIndex = type(uint128).max;
        for (uint256 i = leftMostIndex; i < type(uint64).max; ) {
            // Fetch the claim at the current index.
            ClaimData storage claim = claimData[i];

            // Decrement the loop counter; If it underflows, we've reached the root
            // claim and can stop searching.
            unchecked {
                --i;
            }

            // INVARIANT: A claim can never be considered as the leftMostIndex or leftMostTraceIndex
            //            if it has been countered.
            if (claim.countered) continue;

            // If the claim is a dangling node, we can check if it is the left-most
            // dangling node we've come across so far. If it is, we can update the
            // left-most trace index.
            uint256 traceIndex = claim.position.traceIndex(MAX_GAME_DEPTH);
            if (traceIndex < leftMostTraceIndex) {
                leftMostTraceIndex = traceIndex;
                unchecked {
                    leftMostIndex = i + 1;
                }
            }
        }

        // Create a reference to the left most uncontested claim and its parent.
        ClaimData storage leftMostUncontested = claimData[leftMostIndex];

        // INVARIANT: The game may never be resolved unless the clock of the left-most uncontested
        //            claim's parent has expired. If the left-most uncontested claim is the root
        //            claim, it is uncountered, and we check if 3.5 days has passed since its
        //            creation.
        uint256 parentIndex = leftMostUncontested.parentIndex;
        Clock opposingClock = parentIndex == type(uint32).max
            ? leftMostUncontested.clock
            : claimData[parentIndex].clock;
        if (
            Duration.unwrap(opposingClock.duration()) +
                (block.timestamp - Timestamp.unwrap(opposingClock.timestamp())) <=
            Duration.unwrap(GAME_DURATION) >> 1
        ) {
            revert ClockNotExpired();
        }

        // If the left-most dangling node is at an even depth, the defender wins.
        // Otherwise, the challenger wins and the root claim is deemed invalid.
        if (
            // slither-disable-next-line weak-prng
            leftMostUncontested.position.depth() % 2 == 0 && leftMostTraceIndex != type(uint128).max
        ) {
            status_ = GameStatus.DEFENDER_WINS;
        } else {
            status_ = GameStatus.CHALLENGER_WINS;
        }

        // Update the game status
        emit Resolved(status = status_);
    }

    /// @inheritdoc IDisputeGame
    function rootClaim() public pure returns (Claim rootClaim_) {
        rootClaim_ = Claim.wrap(_getArgFixedBytes(0x00));
    }

    /// @inheritdoc IDisputeGame
    function extraData() public pure returns (bytes memory extraData_) {
        // The extra data starts at the second word within the cwia calldata.
        extraData_ = _getArgDynBytes(0x20, 0x20);
    }

    /// @inheritdoc IDisputeGame
    function gameData()
        external
        pure
        returns (
            GameType gameType_,
            Claim rootClaim_,
            bytes memory extraData_
        )
    {
        gameType_ = gameType();
        rootClaim_ = rootClaim();
        extraData_ = extraData();
    }

    ////////////////////////////////////////////////////////////////
    //                       MISC EXTERNAL                        //
    ////////////////////////////////////////////////////////////////

    /// @inheritdoc IInitializable
    function initialize() external {
        // Set the game start
        gameStart = Timestamp.wrap(uint64(block.timestamp));
        // Set the game status
        status = GameStatus.IN_PROGRESS;

        // Set the root claim
        claimData.push(
            ClaimData({
                parentIndex: type(uint32).max,
                claim: rootClaim(),
                position: ROOT_POSITION,
                clock: LibClock.wrap(Duration.wrap(0), Timestamp.wrap(uint64(block.timestamp))),
                countered: false
            })
        );

        // Set the L1 head hash at the time of the game's creation.
        l1Head = Hash.wrap(blockhash(block.number - 1));
    }

    /// @notice Returns the length of the `claimData` array.
    function claimDataLen() external view returns (uint256 len_) {
        len_ = claimData.length;
    }

    ////////////////////////////////////////////////////////////////
    //                          HELPERS                           //
    ////////////////////////////////////////////////////////////////

    /// @notice Finds the trace ancestor of a given position within the DAG.
    /// @param _pos The position to find the trace ancestor claim of.
    /// @param _start The index to start searching from.
    /// @return ancestor_ The ancestor claim that commits to the same trace index as `_pos`.
    // TODO: Can we form a relationship between the trace path and the position to avoid looping?
    function findTraceAncestor(Position _pos, uint256 _start)
        internal
        view
        returns (ClaimData storage ancestor_)
    {
        // Grab the trace ancestor's expected position.
        Position preStateTraceAncestor = _pos.traceAncestor();

        // Walk up the DAG to find a claim that commits to the same trace index as `_pos`. It is
        // guaranteed that such a claim exists.
        ancestor_ = claimData[_start];
        while (Position.unwrap(ancestor_.position) != Position.unwrap(preStateTraceAncestor)) {
            ancestor_ = claimData[ancestor_.parentIndex];
        }
    }
}
