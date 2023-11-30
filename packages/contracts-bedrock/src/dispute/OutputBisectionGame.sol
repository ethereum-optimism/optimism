// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { IDisputeGame } from "src/dispute/interfaces/IDisputeGame.sol";
import { IOutputBisectionGame } from "src/dispute/interfaces/IOutputBisectionGame.sol";
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

/// @title OutputBisectionGame
/// @notice An implementation of the `IOutputBisectionGame` interface.
contract OutputBisectionGame is IOutputBisectionGame, Clone, ISemver {
    ////////////////////////////////////////////////////////////////
    //                         State Vars                         //
    ////////////////////////////////////////////////////////////////

    /// @notice The absolute prestate of the instruction trace. This is a constant that is defined
    ///         by the program that is being used to execute the trace.
    Claim public immutable ABSOLUTE_PRESTATE;

    /// @notice The max depth of the game.
    uint256 public immutable MAX_GAME_DEPTH;

    /// @notice The max depth of the output bisection portion of the position tree. Immediately beneath
    ///         this depth, execution trace bisection begins.
    uint256 public immutable SPLIT_DEPTH;

    /// @notice The duration of the game.
    Duration public immutable GAME_DURATION;

    /// @notice An onchain VM that performs single instruction steps on a fault proof program trace.
    IBigStepper public immutable VM;

    /// @notice The genesis block number
    uint256 public immutable GENESIS_BLOCK_NUMBER;

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
    /// @custom:semver 0.0.13
    string public constant version = "0.0.13";

    /// @param _gameType The type ID of the game.
    /// @param _absolutePrestate The absolute prestate of the instruction trace.
    /// @param _genesisBlockNumber The block number of the genesis block.
    /// @param _maxGameDepth The maximum depth of bisection.
    /// @param _splitDepth The final depth of the output bisection portion of the game.
    /// @param _gameDuration The duration of the game.
    /// @param _vm An onchain VM that performs single instruction steps on a fault proof program
    ///            trace.
    constructor(
        GameType _gameType,
        Claim _absolutePrestate,
        uint256 _genesisBlockNumber,
        uint256 _maxGameDepth,
        uint256 _splitDepth,
        Duration _gameDuration,
        IBigStepper _vm
    ) {
        if (_splitDepth >= _maxGameDepth) revert InvalidSplitDepth();

        GAME_TYPE = _gameType;
        ABSOLUTE_PRESTATE = _absolutePrestate;
        GENESIS_BLOCK_NUMBER = _genesisBlockNumber;
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
        if (keccak256(_stateData) << 8 != Claim.unwrap(preStateClaim) << 8) revert InvalidPrestate();

        // TODO(clabby): Include less context. See Adrian's proposal for the local context salt.
        (ClaimData storage starting, ClaimData storage disputed) = findStartingAndDisputedOutputs(_claimIndex);
        bytes32 uuid = keccak256(abi.encode(starting.claim, starting.parentIndex, disputed.claim, disputed.parentIndex));

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
        bool validStep = VM.step(_stateData, _proof, uuid) == Claim.unwrap(postState.claim);
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

        // When the next position surpasses the split depth (i.e., it is the root claim of an execution
        // trace bisection sub-game), we need to perform some extra verification steps.
        if (nextPosition.depth() == SPLIT_DEPTH + 1) verifyExecBisectionRoot(_claim, _challengeIndex);

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

        // TODO(clabby): Include less context. See Adrian's proposal for the local context salt.
        (ClaimData storage starting, ClaimData storage disputed) = findStartingAndDisputedOutputs(_execLeafIdx);
        bytes32 uuid = keccak256(abi.encode(starting.claim, starting.parentIndex, disputed.claim, disputed.parentIndex));

        IPreimageOracle oracle = VM.oracle();
        if (_ident == 1) {
            // Load the L1 head hash
            oracle.loadLocalData(_ident, uuid, Hash.unwrap(l1Head), 32, _partOffset);
        } else if (_ident == 2) {
            // Load the starting proposal's output root.
            oracle.loadLocalData(_ident, uuid, Claim.unwrap(starting.claim), 32, _partOffset);
        } else if (_ident == 3) {
            // Load the disputed proposal's output root
            oracle.loadLocalData(_ident, uuid, Claim.unwrap(disputed.claim), 32, _partOffset);
        } else if (_ident == 4) {
            // Load the starting proposal's L2 block number as a big-endian uint64 in the
            // high order 8 bytes of the word.
            // TODO(clabby): +1?
            oracle.loadLocalData(
                _ident,
                uuid,
                bytes32(GENESIS_BLOCK_NUMBER + uint256(starting.position.indexAtDepth()) << 0xC0),
                8,
                _partOffset
            );
        } else if (_ident == 5) {
            // Load the chain ID as a big-endian uint64 in the high order 8 bytes of the word.
            oracle.loadLocalData(_ident, uuid, bytes32(block.chainid << 0xC0), 8, _partOffset);
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

        // Persist the L1 head hash of the parent block.
        // TODO(clabby): There may be a bug here - Do we just allow the dispute game to be invalid? We can
        //               always just create another, but it is possible to create a game where the data was not
        //               already available on L1.
        l1Head = Hash.wrap(blockhash(block.number - 1));
    }

    /// @notice Returns the length of the `claimData` array.
    function claimDataLen() external view returns (uint256 len_) {
        len_ = claimData.length;
    }

    ////////////////////////////////////////////////////////////////
    //                          HELPERS                           //
    ////////////////////////////////////////////////////////////////

    /// @notice Verifies the integrity of an execution bisection subgame's root claim. Reverts if the claim
    ///         is invalid.
    /// @param _rootClaim The root claim of the execution bisection subgame.
    function verifyExecBisectionRoot(Claim _rootClaim, uint256 /* _parentIndex */ ) internal pure {
        // The VMStatus must indicate 'invalid' (1), to argue that disputed thing is invalid.
        // Games that agree with the existing outcome are not allowed.
        // TODO(clabby): This assumption will change in Alpha Chad, and also depending on the split depth! Be careful
        //               about what we go with here.
        uint8 vmStatus = uint8(Claim.unwrap(_rootClaim)[0]);
        if (!(vmStatus == VMStatus.unwrap(VMStatuses.INVALID) || vmStatus == VMStatus.unwrap(VMStatuses.PANIC))) {
            revert UnexpectedRootClaim(_rootClaim);
        }

        // TODO(clabby): Other verification steps (?)
    }

    /// @notice Finds the trace ancestor of a given position within the DAG.
    /// @param _pos The position to find the trace ancestor claim of.
    /// @param _start The index to start searching from.
    /// @return ancestor_ The ancestor claim that commits to the same trace index as `_pos`.
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

    /// @notice Finds the starting and disputed output root for a given `ClaimData` within the DAG. This
    ///         `ClaimData` must be below the `SPLIT_DEPTH`.
    /// @param _start The index within `claimData` of the claim to start searching from.
    /// @return starting_ The starting, agreed upon output root claim.
    /// @return disputed_ The disputed output root claim.
    function findStartingAndDisputedOutputs(uint256 _start)
        internal
        view
        returns (ClaimData storage starting_, ClaimData storage disputed_)
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
        while ((currentDepth = claim.position.depth()) != SPLIT_DEPTH) {
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
        //    elsewhere in the dag (it must commit to the block # index at depth of `outputPos - 1`).
        // 2. If it was a defense, the starting output root is `claim`, and the disputed output root is
        //    elsewhere in the dag (it must commit to the block # index at depth of `outputPos + 1`).
        if (wasAttack) {
            starting_ = findTraceAncestor(Position.wrap(Position.unwrap(outputPos) - 1), claimIdx);
            disputed_ = claimData[claimIdx];
        } else {
            starting_ = claimData[claimIdx];
            disputed_ = findTraceAncestor(Position.wrap(Position.unwrap(outputPos) + 1), claimIdx);
        }
    }
}
