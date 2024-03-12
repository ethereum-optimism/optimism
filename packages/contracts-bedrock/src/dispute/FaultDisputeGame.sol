// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { FixedPointMathLib } from "solady/utils/FixedPointMathLib.sol";

import { IDelayedWETH } from "src/dispute/interfaces/IDelayedWETH.sol";
import { IDisputeGame } from "src/dispute/interfaces/IDisputeGame.sol";
import { IFaultDisputeGame } from "src/dispute/interfaces/IFaultDisputeGame.sol";
import { IInitializable } from "src/dispute/interfaces/IInitializable.sol";
import { IBigStepper, IPreimageOracle } from "src/dispute/interfaces/IBigStepper.sol";
import { IAnchorStateRegistry } from "src/dispute/interfaces/IAnchorStateRegistry.sol";

import { Clone } from "src/libraries/Clone.sol";
import { Types } from "src/libraries/Types.sol";
import { ISemver } from "src/universal/ISemver.sol";
import { LibClock } from "src/dispute/lib/LibUDT.sol";

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

    /// @notice The game type ID.
    GameType internal immutable GAME_TYPE;

    /// @notice WETH contract for holding ETH.
    IDelayedWETH internal immutable WETH;

    /// @notice The anchor state registry.
    IAnchorStateRegistry internal immutable ANCHOR_STATE_REGISTRY;

    /// @notice The chain ID of the L2 network this contract argues about.
    uint256 internal immutable L2_CHAIN_ID;

    /// @notice The global root claim's position is always at gindex 1.
    Position internal constant ROOT_POSITION = Position.wrap(1);

    /// @notice The flag set in the `bond` field of a `ClaimData` struct to indicate that the bond has been claimed.
    uint128 internal constant CLAIMED_BOND_FLAG = type(uint128).max;

    /// @notice The starting timestamp of the game
    Timestamp public createdAt;

    /// @notice The timestamp of the game's global resolution.
    Timestamp public resolvedAt;

    /// @inheritdoc IDisputeGame
    GameStatus public status;

    /// @notice An append-only array of all claims made during the dispute game.
    ClaimData[] public claimData;

    /// @notice Credited balances for winning participants.
    mapping(address => uint256) public credit;

    /// @notice An internal mapping to allow for constant-time lookups of existing claims.
    mapping(ClaimHash => bool) internal claims;

    /// @notice An internal mapping of subgames rooted at a claim index to other claim indices in the subgame.
    mapping(uint256 => uint256[]) internal subgames;

    /// @notice Indicates whether the subgame rooted at the root claim has been resolved.
    bool internal subgameAtRootResolved;

    /// @notice Flag for the `initialize` function to prevent re-initialization.
    bool internal initialized;

    /// @notice The latest finalized output root, serving as the anchor for output bisection.
    OutputRoot public startingOutputRoot;

    /// @notice Semantic version.
    /// @custom:semver 0.8.1
    string public constant version = "0.8.1";

    /// @param _gameType The type ID of the game.
    /// @param _absolutePrestate The absolute prestate of the instruction trace.
    /// @param _maxGameDepth The maximum depth of bisection.
    /// @param _splitDepth The final depth of the output bisection portion of the game.
    /// @param _gameDuration The duration of the game.
    /// @param _vm An onchain VM that performs single instruction steps on an FPP trace.
    /// @param _weth WETH contract for holding ETH.
    /// @param _anchorStateRegistry The contract that stores the anchor state for each game type.
    /// @param _l2ChainId Chain ID of the L2 network this contract argues about.
    constructor(
        GameType _gameType,
        Claim _absolutePrestate,
        uint256 _maxGameDepth,
        uint256 _splitDepth,
        Duration _gameDuration,
        IBigStepper _vm,
        IDelayedWETH _weth,
        IAnchorStateRegistry _anchorStateRegistry,
        uint256 _l2ChainId
    ) {
        // The split depth cannot be greater than or equal to the max game depth.
        if (_splitDepth >= _maxGameDepth) revert InvalidSplitDepth();

        GAME_TYPE = _gameType;
        ABSOLUTE_PRESTATE = _absolutePrestate;
        MAX_GAME_DEPTH = _maxGameDepth;
        SPLIT_DEPTH = _splitDepth;
        GAME_DURATION = _gameDuration;
        VM = _vm;
        WETH = _weth;
        ANCHOR_STATE_REGISTRY = _anchorStateRegistry;
        L2_CHAIN_ID = _l2ChainId;
    }

    /// @notice Receive function to allow the contract to receive ETH.
    receive() external payable { }

    /// @notice Fallback function to allow the contract to receive ETH.
    fallback() external payable { }

    ////////////////////////////////////////////////////////////////
    //                  `IFaultDisputeGame` impl                  //
    ////////////////////////////////////////////////////////////////

    /// @inheritdoc IFaultDisputeGame
    function step(
        uint256 _claimIndex,
        bool _isAttack,
        bytes calldata _stateData,
        bytes calldata _proof
    )
        public
        virtual
    {
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
                : _findTraceAncestor(Position.wrap(parentPos.raw() - 1), parent.parentIndex, false).claim;
            // For all attacks, the poststate is the parent claim.
            postState = parent;
        } else {
            // If the step is a defense, the poststate exists elsewhere in the game state,
            // and the parent claim is the expected pre-state.
            preStateClaim = parent.claim;
            postState = _findTraceAncestor(Position.wrap(parentPos.raw() + 1), parent.parentIndex, false);
        }

        // INVARIANT: The prestate is always invalid if the passed `_stateData` is not the
        //            preimage of the prestate claim hash.
        //            We ignore the highest order byte of the digest because it is used to
        //            indicate the VM Status and is added after the digest is computed.
        if (keccak256(_stateData) << 8 != preStateClaim.raw() << 8) revert InvalidPrestate();

        // Compute the local preimage context for the step.
        Hash uuid = _findLocalContext(_claimIndex);

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
        bool validStep = VM.step(_stateData, _proof, uuid.raw()) == postState.claim.raw();
        bool parentPostAgree = (parentPos.depth() - postState.position.depth()) % 2 == 0;
        if (parentPostAgree == validStep) revert ValidStep();

        // INVARIANT: A step cannot be made against a claim for a second time.
        if (parent.counteredBy != address(0)) revert DuplicateStep();

        // Set the parent claim as countered. We do not need to append a new claim to the game;
        // instead, we can just set the existing parent as countered.
        parent.counteredBy = msg.sender;
    }

    /// @notice Generic move function, used for both `attack` and `defend` moves.
    /// @param _challengeIndex The index of the claim being moved against.
    /// @param _claim The claim at the next logical position in the game.
    /// @param _isAttack Whether or not the move is an attack or defense.
    function move(uint256 _challengeIndex, Claim _claim, bool _isAttack) public payable virtual {
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
            _verifyExecBisectionRoot(_claim, _challengeIndex, parentPos, _isAttack);
        }

        // INVARIANT: The `msg.value` must be sufficient to cover the required bond.
        if (getRequiredBond(nextPosition) > msg.value) revert InsufficientBond();

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
                grandparentClock.duration().raw()
                // Second, add the difference between the current block timestamp and the
                // parent's clock timestamp.
                + block.timestamp - parent.clock.timestamp().raw()
            )
        );

        // INVARIANT: A move can never be made once its clock has exceeded `GAME_DURATION / 2`
        //            seconds of time.
        if (nextDuration.raw() > GAME_DURATION.raw() >> 1) revert ClockTimeExceeded();

        // Construct the next clock with the new duration and the current block timestamp.
        Clock nextClock = LibClock.wrap(nextDuration, Timestamp.wrap(uint64(block.timestamp)));

        // INVARIANT: There cannot be multiple identical claims with identical moves on the same challengeIndex. Multiple
        //            claims at the same position may dispute the same challengeIndex. However, they must have different
        //            values.
        ClaimHash claimHash = _claim.hashClaimPos(nextPosition, _challengeIndex);
        if (claims[claimHash]) revert ClaimAlreadyExists();
        claims[claimHash] = true;

        // Create the new claim.
        claimData.push(
            ClaimData({
                parentIndex: uint32(_challengeIndex),
                // This is updated during subgame resolution
                counteredBy: address(0),
                claimant: msg.sender,
                bond: uint128(msg.value),
                claim: _claim,
                position: nextPosition,
                clock: nextClock
            })
        );

        // Update the subgame rooted at the parent claim.
        subgames[_challengeIndex].push(claimData.length - 1);

        // Deposit the bond.
        WETH.deposit{ value: msg.value }();

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
    function addLocalData(uint256 _ident, uint256 _execLeafIdx, uint256 _partOffset) external {
        // INVARIANT: Local data can only be added if the game is currently in progress.
        if (status != GameStatus.IN_PROGRESS) revert GameNotInProgress();

        (Claim starting, Position startingPos, Claim disputed, Position disputedPos) =
            _findStartingAndDisputedOutputs(_execLeafIdx);
        Hash uuid = _computeLocalContext(starting, startingPos, disputed, disputedPos);

        IPreimageOracle oracle = VM.oracle();
        if (_ident == LocalPreimageKey.L1_HEAD_HASH) {
            // Load the L1 head hash
            oracle.loadLocalData(_ident, uuid.raw(), l1Head().raw(), 32, _partOffset);
        } else if (_ident == LocalPreimageKey.STARTING_OUTPUT_ROOT) {
            // Load the starting proposal's output root.
            oracle.loadLocalData(_ident, uuid.raw(), starting.raw(), 32, _partOffset);
        } else if (_ident == LocalPreimageKey.DISPUTED_OUTPUT_ROOT) {
            // Load the disputed proposal's output root
            oracle.loadLocalData(_ident, uuid.raw(), disputed.raw(), 32, _partOffset);
        } else if (_ident == LocalPreimageKey.DISPUTED_L2_BLOCK_NUMBER) {
            // Load the disputed proposal's L2 block number as a big-endian uint64 in the
            // high order 8 bytes of the word.

            // We add the index at depth + 1 to the starting block number to get the disputed L2
            // block number.
            uint256 l2Number = startingOutputRoot.l2BlockNumber + disputedPos.traceIndex(SPLIT_DEPTH) + 1;

            oracle.loadLocalData(_ident, uuid.raw(), bytes32(l2Number << 0xC0), 8, _partOffset);
        } else if (_ident == LocalPreimageKey.CHAIN_ID) {
            // Load the chain ID as a big-endian uint64 in the high order 8 bytes of the word.
            oracle.loadLocalData(_ident, uuid.raw(), bytes32(L2_CHAIN_ID << 0xC0), 8, _partOffset);
        } else {
            revert InvalidLocalIdent();
        }
    }

    /// @inheritdoc IFaultDisputeGame
    function l1Head() public pure returns (Hash l1Head_) {
        l1Head_ = Hash.wrap(_getArgFixedBytes(0x20));
    }

    /// @inheritdoc IFaultDisputeGame
    function l2BlockNumber() public pure returns (uint256 l2BlockNumber_) {
        l2BlockNumber_ = _getArgUint256(0x40);
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
        status_ = claimData[0].counteredBy == address(0) ? GameStatus.DEFENDER_WINS : GameStatus.CHALLENGER_WINS;
        resolvedAt = Timestamp.wrap(uint64(block.timestamp));

        // Update the status and emit the resolved event, note that we're performing an assignment here.
        emit Resolved(status = status_);

        // Try to update the anchor state, this should not revert.
        ANCHOR_STATE_REGISTRY.tryUpdateAnchorState();
    }

    /// @inheritdoc IFaultDisputeGame
    function resolveClaim(uint256 _claimIndex) external payable {
        // INVARIANT: Resolution cannot occur unless the game is currently in progress.
        if (status != GameStatus.IN_PROGRESS) revert GameNotInProgress();

        ClaimData storage parent = claimData[_claimIndex];

        // INVARIANT: Cannot resolve a subgame unless the clock of its root has expired
        uint64 parentClockDuration = parent.clock.duration().raw();
        uint64 timeSinceParentMove = uint64(block.timestamp) - parent.clock.timestamp().raw();
        if (parentClockDuration + timeSinceParentMove <= GAME_DURATION.raw() >> 1) {
            revert ClockNotExpired();
        }

        uint256[] storage challengeIndices = subgames[_claimIndex];
        uint256 challengeIndicesLen = challengeIndices.length;

        // INVARIANT: Cannot resolve subgames twice
        if (_claimIndex == 0 && subgameAtRootResolved) {
            revert ClaimAlreadyResolved();
        }

        // Uncontested claims are resolved implicitly unless they are the root claim. Pay out the bond to the claimant
        // and return early.
        if (challengeIndicesLen == 0 && _claimIndex != 0) {
            // In the event that the parent claim is at the max depth, there will always be 0 subgames. If the
            // `counteredBy` field is set and there are no subgames, this implies that the parent claim was successfully
            // stepped against. In this case, we pay out the bond to the party that stepped against the parent claim.
            // Otherwise, the parent claim is uncontested, and the bond is returned to the claimant.
            address counteredBy = parent.counteredBy;
            address recipient = counteredBy == address(0) ? parent.claimant : counteredBy;
            _distributeBond(recipient, parent);
            return;
        }

        // Assume parent is honest until proven otherwise
        address countered = address(0);
        Position leftmostCounter = Position.wrap(type(uint128).max);
        for (uint256 i = 0; i < challengeIndicesLen; ++i) {
            uint256 challengeIndex = challengeIndices[i];

            // INVARIANT: Cannot resolve a subgame containing an unresolved claim
            if (subgames[challengeIndex].length != 0) revert OutOfOrderResolution();

            ClaimData storage claim = claimData[challengeIndex];

            // If the child subgame is uncountered and further left than the current left-most counter,
            // update the parent subgame's `countered` address and the current `leftmostCounter`.
            // The left-most correct counter is preferred in bond payouts in order to discourage attackers
            // from countering invalid subgame roots via an invalid defense position. As such positions
            // cannot be correctly countered.
            // Note that correctly positioned defense, but invalid claimes can still be successfully countered.
            if (claim.counteredBy == address(0) && leftmostCounter.raw() > claim.position.raw()) {
                countered = claim.claimant;
                leftmostCounter = claim.position;
            }
        }

        // If the parent was not successfully countered, pay out the parent's bond to the claimant.
        // If the parent was successfully countered, pay out the parent's bond to the challenger.
        _distributeBond(countered == address(0) ? parent.claimant : countered, parent);

        // Once a subgame is resolved, we percolate the result up the DAG so subsequent calls to
        // resolveClaim will not need to traverse this subgame.
        parent.counteredBy = countered;

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
        extraData_ = _getArgDynBytes(0x40, 0x20);
    }

    /// @inheritdoc IDisputeGame
    function gameData() external view returns (GameType gameType_, Claim rootClaim_, bytes memory extraData_) {
        gameType_ = gameType();
        rootClaim_ = rootClaim();
        extraData_ = extraData();
    }

    /// @inheritdoc IFaultDisputeGame
    function startingBlockNumber() external view returns (uint256 startingBlockNumber_) {
        startingBlockNumber_ = startingOutputRoot.l2BlockNumber;
    }

    /// @inheritdoc IFaultDisputeGame
    function startingRootHash() external view returns (Hash startingRootHash_) {
        startingRootHash_ = startingOutputRoot.root;
    }

    ////////////////////////////////////////////////////////////////
    //                       MISC EXTERNAL                        //
    ////////////////////////////////////////////////////////////////

    /// @inheritdoc IInitializable
    function initialize() public payable virtual {
        // SAFETY: Any revert in this function will bubble up to the DisputeGameFactory and
        // prevent the game from being created.
        //
        // Implicit assumptions:
        // - The `gameStatus` state variable defaults to 0, which is `GameStatus.IN_PROGRESS`
        // - The dispute game factory will enforce the required bond to initialize the game.
        //
        // Explicit checks:
        // - The game must not have already been initialized.
        // - An output root cannot be proposed at or before the starting block number.

        // INVARIANT: The game must not have already been initialized.
        if (initialized) revert AlreadyInitialized();

        // Grab the latest anchor root.
        (Hash root, uint256 rootBlockNumber) = ANCHOR_STATE_REGISTRY.anchors(GAME_TYPE);

        // Should only happen if this is a new game type that hasn't been set up yet.
        if (root.raw() == bytes32(0)) revert AnchorRootNotFound();

        // Set the starting output root.
        startingOutputRoot = OutputRoot({ l2BlockNumber: rootBlockNumber, root: root });

        // Do not allow the game to be initialized if the root claim corresponds to a block at or before the
        // configured starting block number.
        if (l2BlockNumber() <= rootBlockNumber) revert UnexpectedRootClaim(rootClaim());

        // Revert if the calldata size is too large, which signals that the `extraData` contains more than expected.
        // This is to prevent adding extra bytes to the `extraData` that result in a different game UUID in the factory,
        // but are not used by the game, which would allow for multiple dispute games for the same output proposal to
        // be created.
        // Expected length: 0x66 (0x04 selector + 0x20 root claim + 0x20 l1 head + 0x20 extraData + 0x02 CWIA bytes)
        assembly {
            if gt(calldatasize(), 0x66) {
                // Store the selector for `ExtraDataTooLong()` & revert
                mstore(0x00, 0xc407e025)
                revert(0x1C, 0x04)
            }
        }

        // Set the root claim
        claimData.push(
            ClaimData({
                parentIndex: type(uint32).max,
                counteredBy: address(0),
                claimant: tx.origin,
                bond: uint128(msg.value),
                claim: rootClaim(),
                position: ROOT_POSITION,
                clock: LibClock.wrap(Duration.wrap(0), Timestamp.wrap(uint64(block.timestamp)))
            })
        );

        // Deposit the bond.
        WETH.deposit{ value: msg.value }();

        // Set the game's starting timestamp
        createdAt = Timestamp.wrap(uint64(block.timestamp));

        // Set the game as initialized.
        initialized = true;
    }

    /// @notice Returns the length of the `claimData` array.
    function claimDataLen() external view returns (uint256 len_) {
        len_ = claimData.length;
    }

    /// @notice Returns the required bond for a given move kind.
    /// @param _position The position of the bonded interaction.
    /// @return requiredBond_ The required ETH bond for the given move, in wei.
    function getRequiredBond(Position _position) public view returns (uint256 requiredBond_) {
        uint256 depth = uint256(_position.depth());
        if (depth > MAX_GAME_DEPTH) revert GameDepthExceeded();

        // Values taken from Big Bonds v1.5 (TM) spec.
        uint256 assumedBaseFee = 200 gwei;
        uint256 baseGasCharged = 400_000;
        uint256 highGasCharged = 200_000_000;

        // Goal here is to compute the fixed multiplier that will be applied to the base gas
        // charged to get the required gas amount for the given depth. We apply this multiplier
        // some `n` times where `n` is the depth of the position. We are looking for some number
        // that, when multiplied by itself `MAX_GAME_DEPTH` times and then multiplied by the base
        // gas charged, will give us the maximum gas that we want to charge.
        // We want to solve for (highGasCharged/baseGasCharged) ** (1/MAX_GAME_DEPTH).
        // We know that a ** (b/c) is equal to e ** (ln(a) * (b/c)).
        // We can compute e ** (ln(a) * (b/c)) quite easily with FixedPointMathLib.

        // Set up a, b, and c.
        uint256 a = highGasCharged / baseGasCharged;
        uint256 b = FixedPointMathLib.WAD;
        uint256 c = MAX_GAME_DEPTH * FixedPointMathLib.WAD;

        // Compute ln(a).
        // slither-disable-next-line divide-before-multiply
        uint256 lnA = uint256(FixedPointMathLib.lnWad(int256(a * FixedPointMathLib.WAD)));

        // Computes (b / c) with full precision using WAD = 1e18.
        uint256 bOverC = FixedPointMathLib.divWad(b, c);

        // Compute e ** (ln(a) * (b/c))
        // sMulWad can be used here since WAD = 1e18 maintains the same precision.
        uint256 numerator = FixedPointMathLib.mulWad(lnA, bOverC);
        int256 base = FixedPointMathLib.expWad(int256(numerator));

        // Compute the required gas amount.
        int256 rawGas = FixedPointMathLib.powWad(base, int256(depth * FixedPointMathLib.WAD));
        uint256 requiredGas = FixedPointMathLib.mulWad(baseGasCharged, uint256(rawGas));

        // Compute the required bond.
        requiredBond_ = assumedBaseFee * requiredGas;
    }

    /// @notice Claim the credit belonging to the recipient address.
    /// @param _recipient The owner and recipient of the credit.
    function claimCredit(address _recipient) external {
        // Remove the credit from the recipient prior to performing the external call.
        uint256 recipientCredit = credit[_recipient];
        credit[_recipient] = 0;

        // Revert if the recipient has no credit to claim.
        if (recipientCredit == 0) {
            revert NoCreditToClaim();
        }

        // Try to withdraw the WETH amount so it can be used here.
        WETH.withdraw(_recipient, recipientCredit);

        // Transfer the credit to the recipient.
        (bool success,) = _recipient.call{ value: recipientCredit }(hex"");
        if (!success) revert BondTransferFailed();
    }

    /// @notice Returns the flag set in the `bond` field of a `ClaimData` struct to indicate that the bond has been
    ///         claimed.
    function claimedBondFlag() external pure returns (uint128 claimedBondFlag_) {
        claimedBondFlag_ = CLAIMED_BOND_FLAG;
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

    /// @notice Returns the WETH contract for holding ETH.
    function weth() external view returns (IDelayedWETH weth_) {
        weth_ = WETH;
    }

    /// @notice Returns the chain ID of the L2 network this contract argues about.
    function l2ChainId() external view returns (uint256 l2ChainId_) {
        l2ChainId_ = L2_CHAIN_ID;
    }

    ////////////////////////////////////////////////////////////////
    //                          HELPERS                           //
    ////////////////////////////////////////////////////////////////

    /// @notice Pays out the bond of a claim to a given recipient.
    /// @param _recipient The recipient of the bond.
    /// @param _bonded The claim to pay out the bond of.
    function _distributeBond(address _recipient, ClaimData storage _bonded) internal {
        // Set all bits in the bond value to indicate that the bond has been paid out.
        uint256 bond = _bonded.bond;
        if (bond == CLAIMED_BOND_FLAG) revert ClaimAlreadyResolved();
        _bonded.bond = CLAIMED_BOND_FLAG;

        // Increase the recipient's credit.
        credit[_recipient] += bond;

        // Unlock the bond.
        WETH.unlock(_recipient, bond);
    }

    /// @notice Verifies the integrity of an execution bisection subgame's root claim. Reverts if the claim
    ///         is invalid.
    /// @param _rootClaim The root claim of the execution bisection subgame.
    function _verifyExecBisectionRoot(
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
        Position disputedLeafPos = Position.wrap(_parentPos.raw() + 1);
        ClaimData storage disputed = _findTraceAncestor({ _pos: disputedLeafPos, _start: _parentIdx, _global: true });
        uint8 vmStatus = uint8(_rootClaim.raw()[0]);

        if (_isAttack || disputed.position.depth() % 2 == SPLIT_DEPTH % 2) {
            // If the move is an attack, the parent output is always deemed to be disputed. In this case, we only need
            // to check that the root claim signals that the VM panicked or resulted in an invalid transition.
            // If the move is a defense, and the disputed output and creator of the execution trace subgame disagree,
            // the root claim should also signal that the VM panicked or resulted in an invalid transition.
            if (!(vmStatus == VMStatuses.INVALID.raw() || vmStatus == VMStatuses.PANIC.raw())) {
                revert UnexpectedRootClaim(_rootClaim);
            }
        } else if (vmStatus != VMStatuses.VALID.raw()) {
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
    function _findTraceAncestor(
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
        while (ancestor_.position.raw() != traceAncestorPos.raw()) {
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
    function _findStartingAndDisputedOutputs(uint256 _start)
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
        bool wasAttack = execRootPos.parent().raw() == outputPos.raw();

        // Determine the starting and disputed output root indices.
        // 1. If it was an attack, the disputed output root is `claim`, and the starting output root is
        //    elsewhere in the DAG (it must commit to the block # index at depth of `outputPos - 1`).
        // 2. If it was a defense, the starting output root is `claim`, and the disputed output root is
        //    elsewhere in the DAG (it must commit to the block # index at depth of `outputPos + 1`).
        if (wasAttack) {
            // If this is an attack on the first output root (the block directly after the starting
            // block number), the starting claim nor position exists in the tree. We leave these as
            // 0, which can be easily identified due to 0 being an invalid Gindex.
            if (outputPos.indexAtDepth() > 0) {
                ClaimData storage starting = _findTraceAncestor(Position.wrap(outputPos.raw() - 1), claimIdx, true);
                (startingClaim_, startingPos_) = (starting.claim, starting.position);
            } else {
                startingClaim_ = Claim.wrap(startingOutputRoot.root.raw());
            }
            (disputedClaim_, disputedPos_) = (claim.claim, claim.position);
        } else {
            ClaimData storage disputed = _findTraceAncestor(Position.wrap(outputPos.raw() + 1), claimIdx, true);
            (startingClaim_, startingPos_) = (claim.claim, claim.position);
            (disputedClaim_, disputedPos_) = (disputed.claim, disputed.position);
        }
    }

    /// @notice Finds the local context hash for a given claim index that is present in an execution trace subgame.
    /// @param _claimIndex The index of the claim to find the local context hash for.
    /// @return uuid_ The local context hash.
    function _findLocalContext(uint256 _claimIndex) internal view returns (Hash uuid_) {
        (Claim starting, Position startingPos, Claim disputed, Position disputedPos) =
            _findStartingAndDisputedOutputs(_claimIndex);
        uuid_ = _computeLocalContext(starting, startingPos, disputed, disputedPos);
    }

    /// @notice Computes the local context hash for a set of starting/disputed claim values and positions.
    /// @param _starting The starting claim.
    /// @param _startingPos The starting claim's position.
    /// @param _disputed The disputed claim.
    /// @param _disputedPos The disputed claim's position.
    /// @return uuid_ The local context hash.
    function _computeLocalContext(
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
        if (_startingPos.raw() == 0) {
            uuid_ = Hash.wrap(keccak256(abi.encode(_disputed, _disputedPos)));
        } else {
            uuid_ = Hash.wrap(keccak256(abi.encode(_starting, _startingPos, _disputed, _disputedPos)));
        }
    }
}
