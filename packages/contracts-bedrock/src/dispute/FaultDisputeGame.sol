// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { IDisputeGame } from "src/dispute/interfaces/IDisputeGame.sol";
import { IFaultDisputeGame } from "src/dispute/interfaces/IFaultDisputeGame.sol";
import { IInitializable } from "src/dispute/interfaces/IInitializable.sol";
import { IBondManager } from "src/dispute/interfaces/IBondManager.sol";
import { IBigStepper, IPreimageOracle } from "src/dispute/interfaces/IBigStepper.sol";
import { L2OutputOracle } from "src/L1/L2OutputOracle.sol";
import { BlockOracle } from "src/dispute/BlockOracle.sol";

import { Clone } from "src/libraries/Clone.sol";
import { Types } from "src/libraries/Types.sol";
import { ISemver } from "src/universal/ISemver.sol";
import { LibHashing } from "src/dispute/lib/LibHashing.sol";
import { LibPosition } from "src/dispute/lib/LibPosition.sol";
import { LibClock } from "src/dispute/lib/LibClock.sol";

import "src/libraries/DisputeTypes.sol";
import "src/libraries/DisputeErrors.sol";

/// @title FaultDisputeGame
/// @notice An implementation of the `IFaultDisputeGame` interface.
contract FaultDisputeGame is IFaultDisputeGame, Clone, ISemver {
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

    /// @notice The block hash oracle, used for loading block hashes further back
    ///         than the `BLOCKHASH` opcode allows as well as their child's timestamp.
    BlockOracle public immutable BLOCK_ORACLE;

    /// @notice The game type ID
    GameType internal immutable GAME_TYPE;

    /// @notice The root claim's position is always at gindex 1.
    Position internal constant ROOT_POSITION = Position.wrap(1);

    /// @notice The starting timestamp of the game
    Timestamp public createdAt;

    /// @inheritdoc IDisputeGame
    GameStatus public status;

    /// @inheritdoc IDisputeGame
    IBondManager public bondManager;

    /// @inheritdoc IFaultDisputeGame
    Hash public l1Head;

    /// @notice An append-only array of all claims made during the dispute game.
    ClaimData[] public claimData;

    /// @notice The starting and disputed output proposal for the game. Includes information about
    ///         the output indexes in the `L2OutputOracle` and the output roots at the time of
    ///         game creation.
    OutputProposals public proposals;

    /// @notice An internal mapping to allow for constant-time lookups of existing claims.
    mapping(ClaimHash => bool) internal claims;

    /// @notice An internal mapping of subgames rooted at a claim index to other claim indices in the subgame.
    mapping(uint256 => uint256[]) internal subgames;

    /// @notice Indicates whether the subgame rooted at the root claim has been resolved.
    bool internal subgameAtRootResolved;

    /// @notice Semantic version.
    /// @custom:semver 0.0.13
    string public constant version = "0.0.13";

    /// @param _gameType The type ID of the game.
    /// @param _absolutePrestate The absolute prestate of the instruction trace.
    /// @param _maxGameDepth The maximum depth of bisection.
    /// @param _gameDuration The duration of the game.
    /// @param _vm An onchain VM that performs single instruction steps on a fault proof program
    ///            trace.
    /// @param _l2oo The trusted L2OutputOracle contract.
    /// @param _blockOracle The block oracle, used for loading block hashes further back
    ///                     than the `BLOCKHASH` opcode allows as well as their estimated
    ///                     timestamps.
    constructor(
        GameType _gameType,
        Claim _absolutePrestate,
        uint256 _maxGameDepth,
        Duration _gameDuration,
        IBigStepper _vm,
        L2OutputOracle _l2oo,
        BlockOracle _blockOracle
    ) {
        GAME_TYPE = _gameType;
        ABSOLUTE_PRESTATE = _absolutePrestate;
        MAX_GAME_DEPTH = _maxGameDepth;
        GAME_DURATION = _gameDuration;
        VM = _vm;
        L2_OUTPUT_ORACLE = _l2oo;
        BLOCK_ORACLE = _blockOracle;
    }

    ////////////////////////////////////////////////////////////////
    //                  `IFaultDisputeGame` impl                  //
    ////////////////////////////////////////////////////////////////

    /// @inheritdoc IFaultDisputeGame
    function step(uint256 _claimIndex, bool _isAttack, bytes calldata _stateData, bytes calldata _proof) external {
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
            // If the step position's index at depth is 0, the prestate is the absolute
            // prestate.
            // If the step is an attack at a trace index > 0, the prestate exists elsewhere in
            // the game state.
            preStateClaim = stepPos.indexAtDepth() == 0
                ? ABSOLUTE_PRESTATE
                : findTraceAncestor(Position.wrap(Position.unwrap(parentPos) - 1), parent.parentIndex).claim;

            // For all attacks, the poststate is the parent claim.
            postState = parent;
        } else {
            // If the step is a defense, the poststate exists elsewhere in the game state,
            // and the parent claim is the expected pre-state.
            preStateClaim = parent.claim;
            postState = findTraceAncestor(Position.wrap(Position.unwrap(parentPos) + 1), parent.parentIndex);
        }

        // INVARIANT: The prestate is always invalid if the passed `_stateData` is not the
        //            preimage of the prestate claim hash.
        //            We ignore the highest order byte of the digest because it is used to
        //            indicate the VM Status and is added after the digest is computed.
        if (keccak256(_stateData) << 8 != Claim.unwrap(preStateClaim) << 8) {
            revert InvalidPrestate();
        }

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
        bool validStep = VM.step(_stateData, _proof, 0) == Claim.unwrap(postState.claim);
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
    function move(uint256 _challengeIndex, Claim _claim, bool _isAttack) public payable {
        // INVARIANT: Moves cannot be made unless the game is currently in progress.
        if (status != GameStatus.IN_PROGRESS) revert GameNotInProgress();

        // INVARIANT: A defense can never be made against the root claim. This is because the root
        //            claim commits to the entire state. Therefore, the only valid defense is to
        //            do nothing if it is agreed with.
        if (_challengeIndex == 0 && !_isAttack) revert CannotDefendRootClaim();

        // Get the parent. If it does not exist, the call will revert with OOB.
        ClaimData memory parent = claimData[_challengeIndex];

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
                Duration.unwrap(grandparentClock.duration())
                // Second, add the difference between the current block timestamp and the
                // parent's clock timestamp.
                + block.timestamp - Timestamp.unwrap(parent.clock.timestamp())
            )
        );

        // INVARIANT: A move can never be made once its clock has exceeded `GAME_DURATION / 2`
        //            seconds of time.
        if (Duration.unwrap(nextDuration) > Duration.unwrap(GAME_DURATION) >> 1) {
            revert ClockTimeExceeded();
        }

        // Construct the next clock with the new duration and the current block timestamp.
        Clock nextClock = LibClock.wrap(nextDuration, Timestamp.wrap(uint64(block.timestamp)));

        // INVARIANT: There cannot be multiple identical claims with identical moves on the same challengeIndex. Multiple
        // claims
        //            at the same position may dispute the same challengeIndex. However, the must have different values.
        ClaimHash claimHash = _claim.hashClaimPos(nextPosition, _challengeIndex);
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

        // Set the parent claim as countered.
        claimData[_challengeIndex].countered = true;

        // Update the subgame rooted at the parent claim.
        subgames[_challengeIndex].push(claimData.length - 1);

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
    function addLocalData(uint256 _ident, bytes32 _localContext, uint256 _partOffset) external {
        // INVARIANT: Local data can only be added if the game is currently in progress.
        if (status != GameStatus.IN_PROGRESS) revert GameNotInProgress();

        IPreimageOracle oracle = VM.oracle();
        bytes4 loadLocalDataSelector = IPreimageOracle.loadLocalData.selector;
        assembly {
            // Store the `loadLocalData(uint256,bytes32,uint256,uint256)` selector
            mstore(0x1C, loadLocalDataSelector)
            // Store the `_ident` argument
            mstore(0x20, _ident)
            // Store the `_localContext` argument
            mstore(0x40, _localContext)
            // Store the data to load
            let data
            switch _ident
            case 1 {
                // Load the L1 head hash
                data := sload(l1Head.slot)
            }
            case 2 {
                // Load the starting proposal's output root.
                data := sload(add(proposals.slot, 0x01))
            }
            case 3 {
                // Load the disputed proposal's output root
                data := sload(add(proposals.slot, 0x03))
            }
            case 4 {
                // Load the starting proposal's L2 block number as a big-endian uint64 in the
                // high order 8 bytes of the word.
                data := shl(0xC0, shr(0x80, sload(proposals.slot)))
            }
            case 5 {
                // Load the chain ID as a big-endian uint64 in the high order 8 bytes of the word.
                data := shl(0xC0, chainid())
            }
            default {
                // Store the `InvalidLocalIdent()` selector.
                mstore(0x00, 0xff137e65)
                // Revert with  `InvalidLocalIdent()`
                revert(0x1C, 0x04)
            }
            mstore(0x60, data)
            // Store the size of the data to load
            // _ident > 3 ? 8 : 32
            mstore(0x80, shl(sub(0x05, shl(0x01, gt(_ident, 0x03))), 0x01))
            // Store the part offset of the data
            mstore(0xA0, _partOffset)

            // Attempt to add the local data to the preimage oracle and bubble up the revert
            // if it fails.
            if iszero(call(gas(), oracle, 0x00, 0x1C, 0xA4, 0x00, 0x00)) {
                returndatacopy(0x00, 0x00, returndatasize())
                revert(0x00, returndatasize())
            }
        }
    }

    /// @inheritdoc IFaultDisputeGame
    function l2BlockNumber() public pure returns (uint256 l2BlockNumber_) {
        l2BlockNumber_ = _getArgUint256(0x20);
    }

    /// @inheritdoc IFaultDisputeGame
    function l1BlockNumber() public pure returns (uint256 l1BlockNumber_) {
        l1BlockNumber_ = _getArgUint256(0x40);
    }

    ////////////////////////////////////////////////////////////////
    //                    `IDisputeGame` impl                     //
    ////////////////////////////////////////////////////////////////

    /// @inheritdoc IDisputeGame
    function gameType() public view override returns (GameType gameType_) {
        gameType_ = GAME_TYPE;
    }

    /// @inheritdoc IDisputeGame
    function resolve() external returns (GameStatus status_) {
        // INVARIANT: Resolution cannot occur unless the game is currently in progress.
        if (status != GameStatus.IN_PROGRESS) revert GameNotInProgress();

        // INVARIANT: Resolution cannot occur unless the absolute root subgame has been resolved.
        if (!subgameAtRootResolved) revert OutOfOrderResolution();

        status_ = claimData[0].countered ? GameStatus.CHALLENGER_WINS : GameStatus.DEFENDER_WINS;
        emit Resolved(status = status_);
    }

    /// @inheritdoc IFaultDisputeGame
    function resolveClaim(uint256 _claimIndex) external payable {
        // INVARIANT: Resolution cannot occur unless the game is currently in progress.
        if (status != GameStatus.IN_PROGRESS) revert GameNotInProgress();

        ClaimData storage parent = claimData[_claimIndex];

        // INVARIANT: Cannot resolve a subgame unless the clock of its root has expired
        if (
            Duration.unwrap(parent.clock.duration()) + (block.timestamp - Timestamp.unwrap(parent.clock.timestamp()))
                <= Duration.unwrap(GAME_DURATION) >> 1
        ) {
            revert ClockNotExpired();
        }

        uint256[] storage challengeIndices = subgames[_claimIndex];

        // INVARIANT: Cannot resolve subgames twice
        // Uncontested claims are resolved implicitly unless they are the root claim
        if (_claimIndex == 0 && subgameAtRootResolved) revert ClaimAlreadyResolved();
        if (challengeIndices.length == 0 && _claimIndex != 0) revert ClaimAlreadyResolved();

        // Assume parent is honest until proven otherwise
        bool countered = false;

        for (uint256 i = 0; i < challengeIndices.length; ++i) {
            uint256 challengeIndex = challengeIndices[i];

            // INVARIANT: Cannot resolve a subgame containing an unresolved claim
            if (subgames[challengeIndex].length != 0) revert OutOfOrderResolution();

            ClaimData storage claim = claimData[challengeIndex];

            // Ignore false claims
            if (!claim.countered) {
                countered = true;
                break;
            }
        }

        // Once a subgame is resolved, we percolate the result up the DAG so subsequent calls to
        // resolveClaim will not need to traverse this subgame.
        parent.countered = countered;

        // Resolved subgames have no entries
        delete subgames[_claimIndex];

        // Indicate the game is ready to be resolved
        if (_claimIndex == 0) {
            subgameAtRootResolved = true;
        }
    }

    /// @inheritdoc IDisputeGame
    function rootClaim() public pure returns (Claim rootClaim_) {
        rootClaim_ = Claim.wrap(_getArgFixedBytes(0x00));
    }

    /// @inheritdoc IDisputeGame
    function extraData() public pure returns (bytes memory extraData_) {
        // The extra data starts at the second word within the cwia calldata and
        // is 64 bytes long.
        extraData_ = _getArgDynBytes(0x20, 0x40);
    }

    /// @inheritdoc IDisputeGame
    function gameData() external view returns (GameType gameType_, Claim rootClaim_, bytes memory extraData_) {
        gameType_ = gameType();
        rootClaim_ = rootClaim();
        extraData_ = extraData();
    }

    ////////////////////////////////////////////////////////////////
    //                       MISC EXTERNAL                        //
    ////////////////////////////////////////////////////////////////

    /// @inheritdoc IInitializable
    function initialize() external {
        // SAFETY: Any revert in this function will bubble up to the DisputeGameFactory and
        // prevent the game from being created.
        //
        // Implicit assumptions:
        // - The `gameStatus` state variable defaults to 0, which is `GameStatus.IN_PROGRESS`

        // The VMStatus must indicate (1) 'invalid', to argue that disputed thing is invalid.
        // Games that agree with the existing outcome are not allowed.
        // NOTE(clabby): This assumption will change in Alpha Chad.
        uint8 vmStatus = uint8(Claim.unwrap(rootClaim())[0]);
        if (!(vmStatus == VMStatus.unwrap(VMStatuses.INVALID) || vmStatus == VMStatus.unwrap(VMStatuses.PANIC))) {
            revert UnexpectedRootClaim(rootClaim());
        }

        // Set the game's starting timestamp
        createdAt = Timestamp.wrap(uint64(block.timestamp));

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

        // Grab the index of the output proposal that commits to the starting L2 head.
        // All outputs after this one are disputed.
        // TODO(clabby): This is 2 calls too many for the information we need. Maybe
        //               add a function to the L2OO?
        // TODO(clabby): The block hash bisection game will allow us to dispute the first output
        //               root by using genesis as the starting point. For now, it is critical that
        //               the first proposed output root of an OP stack chain is done so by an
        //               honest party.
        uint256 proposalIdx = L2_OUTPUT_ORACLE.getL2OutputIndexAfter(l2BlockNumber());
        Types.OutputProposal memory starting = L2_OUTPUT_ORACLE.getL2Output(proposalIdx - 1);
        Types.OutputProposal memory disputed = L2_OUTPUT_ORACLE.getL2Output(proposalIdx);

        // SAFETY: This call can revert if the block hash oracle does not have information
        // about the block number provided to it.
        BlockOracle.BlockInfo memory blockInfo = BLOCK_ORACLE.load(l1BlockNumber());

        // INVARIANT: The L1 head must contain the disputed output root. If it does not,
        //            the game cannot be played.
        // SAFETY: The block timestamp in the oracle records the timestamp of the
        //         block *after* the hash stored. This means that the timestamp
        //         is off by 1 block. This is known, and covered as follows:
        //         - The timestamp will always be less than the disputed timestamp
        //           if the checkpoint was made before the proposal. We must revert here.
        //         - The timestamp will be equal to the disputed timestamp if the
        //           checkpoint was made in the same block as the proposal, and the
        //           hash will be the parent block, which does not contain the proposal.
        //           We must revert here.
        //         - The timestamp will always be greater than the disputed timestamp
        //           if the checkpoint was made any block after the proposal. This is
        //           the only case where we can continue, since we must have the L1
        //           head contain the disputed output root to play the game.
        if (Timestamp.unwrap(blockInfo.childTimestamp) <= disputed.timestamp) revert L1HeadTooOld();

        // Persist the output proposals fetched from the oracle. These outputs will be referenced
        // for loading local data into the preimage oracle as well as to authenticate the game's
        // resolution. If the disputed output has changed in the oracle, the game cannot be
        // resolved.
        proposals = OutputProposals({
            starting: OutputProposal({
                index: uint128(proposalIdx - 1),
                l2BlockNumber: starting.l2BlockNumber,
                outputRoot: Hash.wrap(starting.outputRoot)
            }),
            disputed: OutputProposal({
                index: uint128(proposalIdx),
                l2BlockNumber: disputed.l2BlockNumber,
                outputRoot: Hash.wrap(disputed.outputRoot)
            })
        });

        // Persist the L1 head hash of the L1 block number provided.
        l1Head = blockInfo.hash;
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
    // TODO(clabby): Can we form a relationship between the trace path and the position to avoid
    //               looping?
    function findTraceAncestor(Position _pos, uint256 _start) internal view returns (ClaimData storage ancestor_) {
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
