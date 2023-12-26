// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { IDisputeGame } from "src/dispute/interfaces/IDisputeGame.sol";
import { IOutputBisectionGame } from "src/dispute/interfaces/IOutputBisectionGame.sol";
import { IInitializable } from "src/dispute/interfaces/IInitializable.sol";
import { IBondManager } from "src/dispute/interfaces/IBondManager.sol";
import { IBigStepper, IPreimageOracle } from "src/dispute/interfaces/IBigStepper.sol";

import { Clone } from "src/libraries/Clone.sol";
import { Types } from "src/libraries/Types.sol";
import { ISemver } from "src/universal/ISemver.sol";
import { LibHashing } from "src/dispute/lib/LibHashing.sol";
import { LibPosition } from "src/dispute/lib/LibPosition.sol";
import { LibClock } from "src/dispute/lib/LibClock.sol";

import "src/libraries/DisputeTypes.sol";
import "src/libraries/DisputeErrors.sol";

/// @title OutputBisectionGame
/// @notice An implementation of the `IOutputBisectionGame` interface.
contract OutputBisectionGame is IOutputBisectionGame, Clone, ISemver {
    ////////////////////////////////////////////////////////////////
    //                         State Vars                         //
    ////////////////////////////////////////////////////////////////

    /// @notice The absolute prestate of the instruction trace. This is a constant that is defined
    ///         by the program that is being used to execute the trace.
    Claim internal immutable ABSOLUTE_PRESTATE;

    /// @notice The max depth of the game.
    uint256 internal immutable MAX_GAME_DEPTH;

    /// @notice The max depth of the output bisection portion of the position tree. Immediately beneath
    ///         this depth, execution trace bisection begins.
    uint256 internal immutable SPLIT_DEPTH;

    /// @notice The duration of the game.
    Duration internal immutable GAME_DURATION;

    /// @notice An onchain VM that performs single instruction steps on a fault proof program trace.
    IBigStepper internal immutable VM;

    /// @notice The genesis block number
    uint256 internal immutable GENESIS_BLOCK_NUMBER;

    /// @notice The genesis output root
    Hash internal immutable GENESIS_OUTPUT_ROOT;

    /// @notice The game type ID
    GameType internal immutable GAME_TYPE;

    /// @notice The global root claim's position is always at gindex 1.
    Position internal constant ROOT_POSITION = Position.wrap(1);

    /// @notice The starting timestamp of the game
    Timestamp public createdAt;

    /// @notice The timestamp of the game's global resolution.
    Timestamp public resolvedAt;

    /// @inheritdoc IDisputeGame
    GameStatus public status;

    /// @inheritdoc IDisputeGame
    IBondManager public bondManager;

    /// @inheritdoc IOutputBisectionGame
    Hash public l1Head;

    /// @notice An append-only array of all claims made during the dispute game.
    ClaimData[] public claimData;

    /// @notice An internal mapping to allow for constant-time lookups of existing claims.
    mapping(ClaimHash => bool) internal claims;

    /// @notice An internal mapping of subgames rooted at a claim index to other claim indices in the subgame.
    mapping(uint256 => uint256[]) internal subgames;

    /// @notice Indicates whether the subgame rooted at the root claim has been resolved.
    bool internal subgameAtRootResolved;

    /// @notice Semantic version.
    /// @custom:semver 0.0.18
    string public constant version = "0.0.18";

    /// @param _gameType The type ID of the game.
    /// @param _absolutePrestate The absolute prestate of the instruction trace.
    /// @param _genesisBlockNumber The block number of the genesis block.
    /// @param _genesisOutputRoot The output root of the genesis block.
    /// @param _maxGameDepth The maximum depth of bisection.
    /// @param _splitDepth The final depth of the output bisection portion of the game.
    /// @param _gameDuration The duration of the game.
    /// @param _vm An onchain VM that performs single instruction steps on a fault proof program
    ///            trace.
    constructor(
        GameType _gameType,
        Claim _absolutePrestate,
        uint256 _genesisBlockNumber,
        Hash _genesisOutputRoot,
        uint256 _maxGameDepth,
        uint256 _splitDepth,
        Duration _gameDuration,
        IBigStepper _vm
    ) {
        // The split depth cannot be greater than or equal to the max game depth.
        if (_splitDepth >= _maxGameDepth) revert InvalidSplitDepth();

        GAME_TYPE = _gameType;
        ABSOLUTE_PRESTATE = _absolutePrestate;
        GENESIS_BLOCK_NUMBER = _genesisBlockNumber;
        GENESIS_OUTPUT_ROOT = _genesisOutputRoot;
        MAX_GAME_DEPTH = _maxGameDepth;
        SPLIT_DEPTH = _splitDepth;
        GAME_DURATION = _gameDuration;
        VM = _vm;
    }

    ////////////////////////////////////////////////////////////////
    //                  `IOutputBisectionGame` impl               //
    ////////////////////////////////////////////////////////////////

    /// @inheritdoc IOutputBisectionGame
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
            // NOTE: We localize the `indexAtDepth` for the current execution trace subgame by finding
            //       the remainder of the index at depth divided by 2 ** (MAX_GAME_DEPTH - SPLIT_DEPTH),
            //       which is the number of leaves in each execution trace subgame. This is so that we can
            //       determine whether or not the step position is represents the `ABSOLUTE_PRESTATE`.
            preStateClaim = (stepPos.indexAtDepth() % (1 << (MAX_GAME_DEPTH - SPLIT_DEPTH))) == 0
                ? ABSOLUTE_PRESTATE
                : findTraceAncestor(Position.wrap(Position.unwrap(parentPos) - 1), parent.parentIndex, false).claim;
            // For all attacks, the poststate is the parent claim.
            postState = parent;
        } else {
            // If the step is a defense, the poststate exists elsewhere in the game state,
            // and the parent claim is the expected pre-state.
            preStateClaim = parent.claim;
            postState = findTraceAncestor(Position.wrap(Position.unwrap(parentPos) + 1), parent.parentIndex, false);
        }

        // INVARIANT: The prestate is always invalid if the passed `_stateData` is not the
        //            preimage of the prestate claim hash.
        //            We ignore the highest order byte of the digest because it is used to
        //            indicate the VM Status and is added after the digest is computed.
        if (keccak256(_stateData) << 8 != Claim.unwrap(preStateClaim) << 8) revert InvalidPrestate();

        // Compute the local preimage context for the step.
        Hash uuid = findLocalContext(_claimIndex);

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
        bool validStep = VM.step(_stateData, _proof, Hash.unwrap(uuid)) == Claim.unwrap(postState.claim);
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

        // Get the parent. If it does not exist, the call will revert with OOB.
        ClaimData memory parent = claimData[_challengeIndex];

        // Compute the position that the claim commits to. Because the parent's position is already
        // known, we can compute the next position by moving left or right depending on whether
        // or not the move is an attack or defense.
        Position parentPos = parent.position;
        Position nextPosition = parentPos.move(_isAttack);
        uint256 nextPositionDepth = nextPosition.depth();

        // INVARIANT: A defense can never be made against the root claim of either the output root game or any
        //            of the execution trace bisection subgames. This is because the root claim commits to the
        //            entire state. Therefore, the only valid defense is to do nothing if it is agreed with.
        if ((_challengeIndex == 0 || nextPositionDepth == SPLIT_DEPTH + 2) && !_isAttack) {
            revert CannotDefendRootClaim();
        }

        // INVARIANT: A move can never surpass the `MAX_GAME_DEPTH`. The only option to counter a
        //            claim at this depth is to perform a single instruction step on-chain via
        //            the `step` function to prove that the state transition produces an unexpected
        //            post-state.
        if (nextPositionDepth > MAX_GAME_DEPTH) revert GameDepthExceeded();

        // When the next position surpasses the split depth (i.e., it is the root claim of an execution
        // trace bisection sub-game), we need to perform some extra verification steps.
        if (nextPositionDepth == SPLIT_DEPTH + 1) {
            verifyExecBisectionRoot(_claim, _challengeIndex, parentPos, _isAttack);
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
        // claims at the same position may dispute the same challengeIndex. However, they must have different values.
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

    /// @inheritdoc IOutputBisectionGame
    function attack(uint256 _parentIndex, Claim _claim) external payable {
        move(_parentIndex, _claim, true);
    }

    /// @inheritdoc IOutputBisectionGame
    function defend(uint256 _parentIndex, Claim _claim) external payable {
        move(_parentIndex, _claim, false);
    }

    /// @inheritdoc IOutputBisectionGame
    function addLocalData(uint256 _ident, uint256 _execLeafIdx, uint256 _partOffset) external {
        // INVARIANT: Local data can only be added if the game is currently in progress.
        if (status != GameStatus.IN_PROGRESS) revert GameNotInProgress();

        (Claim starting, Position startingPos, Claim disputed, Position disputedPos) =
            findStartingAndDisputedOutputs(_execLeafIdx);
        Hash uuid = computeLocalContext(starting, startingPos, disputed, disputedPos);

        IPreimageOracle oracle = VM.oracle();
        if (_ident == LocalPreimageKey.L1_HEAD_HASH) {
            // Load the L1 head hash
            oracle.loadLocalData(_ident, Hash.unwrap(uuid), Hash.unwrap(l1Head), 32, _partOffset);
        } else if (_ident == LocalPreimageKey.STARTING_OUTPUT_ROOT) {
            // Load the starting proposal's output root.
            oracle.loadLocalData(_ident, Hash.unwrap(uuid), Claim.unwrap(starting), 32, _partOffset);
        } else if (_ident == LocalPreimageKey.DISPUTED_OUTPUT_ROOT) {
            // Load the disputed proposal's output root
            oracle.loadLocalData(_ident, Hash.unwrap(uuid), Claim.unwrap(disputed), 32, _partOffset);
        } else if (_ident == LocalPreimageKey.STARTING_L2_BLOCK_NUMBER) {
            // Load the starting proposal's L2 block number as a big-endian uint64 in the
            // high order 8 bytes of the word.

            // If the starting position is 0 (invalid), the starting output root is genesis. Otherwise,
            // we add the index at depth + 1 to the genesis block number to get the L2 block number.
            uint256 l2Number = Position.unwrap(startingPos) == 0
                ? GENESIS_BLOCK_NUMBER
                : GENESIS_BLOCK_NUMBER + startingPos.traceIndex(SPLIT_DEPTH) + 1;

            oracle.loadLocalData(_ident, Hash.unwrap(uuid), bytes32(l2Number << 0xC0), 8, _partOffset);
        } else if (_ident == LocalPreimageKey.CHAIN_ID) {
            // Load the chain ID as a big-endian uint64 in the high order 8 bytes of the word.
            oracle.loadLocalData(_ident, Hash.unwrap(uuid), bytes32(block.chainid << 0xC0), 8, _partOffset);
        } else {
            revert InvalidLocalIdent();
        }
    }

    /// @inheritdoc IOutputBisectionGame
    function l2BlockNumber() public pure returns (uint256 l2BlockNumber_) {
        l2BlockNumber_ = _getArgUint256(0x20);
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

        // Update the global game status; The dispute has concluded.
        status_ = claimData[0].countered ? GameStatus.CHALLENGER_WINS : GameStatus.DEFENDER_WINS;
        resolvedAt = Timestamp.wrap(uint64(block.timestamp));

        emit Resolved(status = status_);
    }

    /// @inheritdoc IOutputBisectionGame
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

        // Indicate the game is ready to be resolved globally.
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
        // is 32 bytes long.
        extraData_ = _getArgDynBytes(0x20, 0x20);
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
        //
        // Explicit checks:
        // - An output root cannot be proposed at or before the genesis block.

        // Do not allow the game to be initialized if the root claim corresponds to a block at or before the
        // configured genesis block number.
        if (l2BlockNumber() <= GENESIS_BLOCK_NUMBER) revert UnexpectedRootClaim(rootClaim());

        // Set the game's starting timestamp
        createdAt = Timestamp.wrap(uint64(block.timestamp));

        // Revert if the calldata size is too large, which signals that the `extraData` contains more than expected.
        // This is to prevent adding extra bytes to the `extraData` that result in a different game UUID in the factory,
        // but are not used by the game, which would allow for multiple dispute games for the same output proposal to
        // be created.
        // Expected length: 0x46 (0x04 selector + 0x20 root claim + 0x20 extraData + 0x02 CWIA bytes)
        assembly {
            if gt(calldatasize(), 0x46) {
                // Store the selector for `ExtraDataTooLong()` & revert
                mstore(0x00, 0xc407e025)
                revert(0x1C, 0x04)
            }
        }

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

        // Persist the blockhash of the parent block.
        l1Head = Hash.wrap(blockhash(block.number - 1));
    }

    /// @notice Returns the length of the `claimData` array.
    function claimDataLen() external view returns (uint256 len_) {
        len_ = claimData.length;
    }

    ////////////////////////////////////////////////////////////////
    //                     IMMUTABLE GETTERS                      //
    ////////////////////////////////////////////////////////////////

    /// @notice Returns the absolute prestate of the instruction trace.
    function absolutePrestate() external view returns (Claim absolutePrestate_) {
        absolutePrestate_ = ABSOLUTE_PRESTATE;
    }

    /// @notice Returns the max game depth.
    function maxGameDepth() external view returns (uint256 maxGameDepth_) {
        maxGameDepth_ = MAX_GAME_DEPTH;
    }

    /// @notice Returns the split depth.
    function splitDepth() external view returns (uint256 splitDepth_) {
        splitDepth_ = SPLIT_DEPTH;
    }

    /// @notice Returns the game duration.
    function gameDuration() external view returns (Duration gameDuration_) {
        gameDuration_ = GAME_DURATION;
    }

    /// @notice Returns the address of the VM.
    function vm() external view returns (IBigStepper vm_) {
        vm_ = VM;
    }

    /// @notice Returns the genesis block number.
    function genesisBlockNumber() external view returns (uint256 genesisBlockNumber_) {
        genesisBlockNumber_ = GENESIS_BLOCK_NUMBER;
    }

    /// @notice Returns the genesis output root.
    function genesisOutputRoot() external view returns (Hash genesisOutputRoot_) {
        genesisOutputRoot_ = GENESIS_OUTPUT_ROOT;
    }

    ////////////////////////////////////////////////////////////////
    //                          HELPERS                           //
    ////////////////////////////////////////////////////////////////

    /// @notice Verifies the integrity of an execution bisection subgame's root claim. Reverts if the claim
    ///         is invalid.
    /// @param _rootClaim The root claim of the execution bisection subgame.
    function verifyExecBisectionRoot(
        Claim _rootClaim,
        uint256 _parentIdx,
        Position _parentPos,
        bool _isAttack
    )
        internal
        view
    {
        // The root claim of an execution trace bisection sub-game must:
        // 1. Signal that the VM panicked or resulted in an invalid transition if the disputed output root
        //    was made by the opposing party.
        // 2. Signal that the VM resulted in a valid transition if the disputed output root was made by the same party.

        // If the move is a defense, the disputed output could have been made by either party. In this case, we
        // need to search for the parent output to determine what the expected status byte should be.
        Position disputedLeafPos = Position.wrap(Position.unwrap(_parentPos) + 1);
        ClaimData storage disputed = findTraceAncestor({ _pos: disputedLeafPos, _start: _parentIdx, _global: true });
        uint8 vmStatus = uint8(Claim.unwrap(_rootClaim)[0]);

        if (_isAttack || disputed.position.depth() % 2 == SPLIT_DEPTH % 2) {
            // If the move is an attack, the parent output is always deemed to be disputed. In this case, we only need
            // to check that the root claim signals that the VM panicked or resulted in an invalid transition.
            // If the move is a defense, and the disputed output and creator of the execution trace subgame disagree,
            // the root claim should also signal that the VM panicked or resulted in an invalid transition.
            if (!(vmStatus == VMStatus.unwrap(VMStatuses.INVALID) || vmStatus == VMStatus.unwrap(VMStatuses.PANIC))) {
                revert UnexpectedRootClaim(_rootClaim);
            }
        } else if (vmStatus != VMStatus.unwrap(VMStatuses.VALID)) {
            // The disputed output and the creator of the execution trace subgame agree. The status byte should
            // have signaled that the VM succeeded.
            revert UnexpectedRootClaim(_rootClaim);
        }
    }

    /// @notice Finds the trace ancestor of a given position within the DAG.
    /// @param _pos The position to find the trace ancestor claim of.
    /// @param _start The index to start searching from.
    /// @param _global Whether or not to search the entire dag or just within an execution trace subgame. If set to
    ///                `true`, and `_pos` is at or above the split depth, this function will revert.
    /// @return ancestor_ The ancestor claim that commits to the same trace index as `_pos`.
    function findTraceAncestor(
        Position _pos,
        uint256 _start,
        bool _global
    )
        internal
        view
        returns (ClaimData storage ancestor_)
    {
        // Grab the trace ancestor's expected position.
        Position traceAncestorPos = _global ? _pos.traceAncestor() : _pos.traceAncestorBounded(SPLIT_DEPTH);

        // Walk up the DAG to find a claim that commits to the same trace index as `_pos`. It is
        // guaranteed that such a claim exists.
        ancestor_ = claimData[_start];
        while (Position.unwrap(ancestor_.position) != Position.unwrap(traceAncestorPos)) {
            ancestor_ = claimData[ancestor_.parentIndex];
        }
    }

    /// @notice Finds the starting and disputed output root for a given `ClaimData` within the DAG. This
    ///         `ClaimData` must be below the `SPLIT_DEPTH`.
    /// @param _start The index within `claimData` of the claim to start searching from.
    /// @return startingClaim_ The starting output root claim.
    /// @return startingPos_ The starting output root position.
    /// @return disputedClaim_ The disputed output root claim.
    /// @return disputedPos_ The disputed output root position.
    function findStartingAndDisputedOutputs(uint256 _start)
        internal
        view
        returns (Claim startingClaim_, Position startingPos_, Claim disputedClaim_, Position disputedPos_)
    {
        // Fatch the starting claim.
        uint256 claimIdx = _start;
        ClaimData storage claim = claimData[claimIdx];

        // If the starting claim's depth is less than or equal to the split depth, we revert as this is UB.
        if (claim.position.depth() <= SPLIT_DEPTH) revert ClaimAboveSplit();

        // We want to:
        // 1. Find the first claim at the split depth.
        // 2. Determine whether it was the starting or disputed output for the exec game.
        // 3. Find the complimentary claim depending on the info from #2 (pre or post).

        // Walk up the DAG until the ancestor's depth is equal to the split depth.
        uint256 currentDepth;
        ClaimData storage execRootClaim = claim;
        while ((currentDepth = claim.position.depth()) > SPLIT_DEPTH) {
            uint256 parentIndex = claim.parentIndex;

            // If we're currently at the split depth + 1, we're at the root of the execution sub-game.
            // We need to keep track of the root claim here to determine whether the execution sub-game was
            // started with an attack or defense against the output leaf claim.
            if (currentDepth == SPLIT_DEPTH + 1) execRootClaim = claim;

            claim = claimData[parentIndex];
            claimIdx = parentIndex;
        }

        // Determine whether the start of the execution sub-game was an attack or defense to the output root
        // above. This is important because it determines which claim is the starting output root and which
        // is the disputed output root.
        (Position execRootPos, Position outputPos) = (execRootClaim.position, claim.position);
        bool wasAttack = Position.unwrap(execRootPos.parent()) == Position.unwrap(outputPos);

        // Determine the starting and disputed output root indices.
        // 1. If it was an attack, the disputed output root is `claim`, and the starting output root is
        //    elsewhere in the DAG (it must commit to the block # index at depth of `outputPos - 1`).
        // 2. If it was a defense, the starting output root is `claim`, and the disputed output root is
        //    elsewhere in the DAG (it must commit to the block # index at depth of `outputPos + 1`).
        if (wasAttack) {
            // If this is an attack on the first output root (the block directly after genesis), the
            // starting claim nor position exists in the tree. We leave these as 0, which can be easily
            // identified due to 0 being an invalid Gindex.
            if (outputPos.indexAtDepth() > 0) {
                ClaimData storage starting =
                    findTraceAncestor(Position.wrap(Position.unwrap(outputPos) - 1), claimIdx, true);
                (startingClaim_, startingPos_) = (starting.claim, starting.position);
            } else {
                startingClaim_ = Claim.wrap(Hash.unwrap(GENESIS_OUTPUT_ROOT));
            }
            (disputedClaim_, disputedPos_) = (claim.claim, claim.position);
        } else {
            ClaimData storage disputed =
                findTraceAncestor(Position.wrap(Position.unwrap(outputPos) + 1), claimIdx, true);
            (startingClaim_, startingPos_) = (claim.claim, claim.position);
            (disputedClaim_, disputedPos_) = (disputed.claim, disputed.position);
        }
    }

    /// @notice Finds the local context hash for a given claim index that is present in an execution trace subgame.
    /// @param _claimIndex The index of the claim to find the local context hash for.
    /// @return uuid_ The local context hash.
    function findLocalContext(uint256 _claimIndex) internal view returns (Hash uuid_) {
        (Claim starting, Position startingPos, Claim disputed, Position disputedPos) =
            findStartingAndDisputedOutputs(_claimIndex);
        uuid_ = computeLocalContext(starting, startingPos, disputed, disputedPos);
    }

    /// @notice Computes the local context hash for a set of starting/disputed claim values and positions.
    /// @param _starting The starting claim.
    /// @param _startingPos The starting claim's position.
    /// @param _disputed The disputed claim.
    /// @param _disputedPos The disputed claim's position.
    /// @return uuid_ The local context hash.
    function computeLocalContext(
        Claim _starting,
        Position _startingPos,
        Claim _disputed,
        Position _disputedPos
    )
        internal
        pure
        returns (Hash uuid_)
    {
        // A position of 0 indicates that the starting claim is the absolute prestate. In this special case,
        // we do not include the starting claim within the local context hash.
        if (Position.unwrap(_startingPos) == 0) {
            uuid_ = Hash.wrap(keccak256(abi.encode(_disputed, _disputedPos)));
        } else {
            uuid_ = Hash.wrap(keccak256(abi.encode(_starting, _startingPos, _disputed, _disputedPos)));
        }
    }
}
