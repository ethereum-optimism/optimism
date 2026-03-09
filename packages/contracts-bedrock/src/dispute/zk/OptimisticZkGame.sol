// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Libraries
import { Clone } from "@solady/utils/Clone.sol";
import {
    BondDistributionMode,
    Claim,
    Duration,
    GameStatus,
    GameType,
    Hash,
    Timestamp,
    Proposal
} from "src/dispute/lib/Types.sol";
import { AggregationOutputs, GameTypes } from "src/dispute/lib/Types.sol";
import {
    AlreadyInitialized,
    BadAuth,
    BondTransferFailed,
    ClaimAlreadyResolved,
    GameNotFinalized,
    IncorrectBondAmount,
    InvalidBondDistributionMode,
    NoCreditToClaim,
    UnexpectedRootClaim,
    ClaimAlreadyChallenged,
    InvalidParentGame,
    ParentGameNotResolved,
    GameOver,
    GameNotOver,
    InvalidProposalStatus,
    IncorrectDisputeGameFactory
} from "src/dispute/lib/Errors.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { ISP1Verifier } from "src/dispute/zk/ISP1Verifier.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";

// Contracts
import { AccessManager } from "src/dispute/zk/AccessManager.sol";

/// @title OptimisticZkGame
/// @notice An implementation of the `IFaultDisputeGame` interface.
/// @dev Derived from https://github.com/succinctlabs/op-succinct (at commit c13844a9bbc330cca69eef2538d8f8ec123e1653)
contract OptimisticZkGame is Clone, ISemver, IDisputeGame {
    ////////////////////////////////////////////////////////////////
    //                         Enums                              //
    ////////////////////////////////////////////////////////////////

    enum ProposalStatus {
        // The initial state of a new proposal.
        Unchallenged,
        // A proposal that has been challenged but not yet proven.
        Challenged,
        // An unchallenged proposal that has been proven valid with a verified proof.
        UnchallengedAndValidProofProvided,
        // A challenged proposal that has been proven valid with a verified proof.
        ChallengedAndValidProofProvided,
        // The final state after resolution, either GameStatus.CHALLENGER_WINS or GameStatus.DEFENDER_WINS.
        Resolved
    }

    ////////////////////////////////////////////////////////////////
    //                         Structs                            //
    ////////////////////////////////////////////////////////////////

    /// @notice The `ClaimData` struct represents the data associated with a Claim.
    struct ClaimData {
        uint32 parentIndex;
        address counteredBy;
        address prover;
        Claim claim;
        ProposalStatus status;
        Timestamp deadline;
    }

    ////////////////////////////////////////////////////////////////
    //                         Events                             //
    ////////////////////////////////////////////////////////////////

    /// @notice Emitted when the game is challenged.
    /// @param challenger The address of the challenger.
    event Challenged(address indexed challenger);

    /// @notice Emitted when the game is proved.
    /// @param prover The address of the prover.
    event Proved(address indexed prover);

    /// @notice Emitted when the game is closed.
    event GameClosed(BondDistributionMode bondDistributionMode);

    ////////////////////////////////////////////////////////////////
    //                         State Vars                         //
    ////////////////////////////////////////////////////////////////

    /// @notice The maximum duration allowed for a challenger to challenge a game.
    // nosemgrep: sol-safety-no-immutable-variables
    Duration internal immutable MAX_CHALLENGE_DURATION;

    /// @notice The maximum duration allowed for a proposer to prove against a challenge.
    // nosemgrep: sol-safety-no-immutable-variables
    Duration internal immutable MAX_PROVE_DURATION;

    /// @notice The game type ID.
    // nosemgrep: sol-safety-no-immutable-variables
    GameType internal immutable GAME_TYPE;

    /// @notice The dispute game factory.
    // nosemgrep: sol-safety-no-immutable-variables
    IDisputeGameFactory internal immutable DISPUTE_GAME_FACTORY;

    /// @notice The SP1 verifier.
    // nosemgrep: sol-safety-no-immutable-variables
    ISP1Verifier internal immutable SP1_VERIFIER;

    /// @notice The rollup config hash.
    // nosemgrep: sol-safety-no-immutable-variables
    bytes32 internal immutable ROLLUP_CONFIG_HASH;

    /// @notice The vkey for the aggregation program.
    // nosemgrep: sol-safety-no-immutable-variables
    bytes32 internal immutable AGGREGATION_VKEY;

    /// @notice The 32 byte commitment to the BabyBear representation of the verification key of the range SP1 program.
    /// Specifically,
    /// this verification is the output of converting the [u32; 8] range BabyBear verification key to a [u8; 32] array.
    // nosemgrep: sol-safety-no-immutable-variables
    bytes32 internal immutable RANGE_VKEY_COMMITMENT;

    /// @notice The challenger bond for the game. This is the amount of the bond that the
    ///         challenger has to bond to challenge. The prover will receive this bond if they
    ///         provide a valid proof in response to a challenge.
    // nosemgrep: sol-safety-no-immutable-variables
    uint256 internal immutable CHALLENGER_BOND;

    /// @notice The anchor state registry.
    // nosemgrep: sol-safety-no-immutable-variables
    IAnchorStateRegistry internal immutable ANCHOR_STATE_REGISTRY;

    /// @notice The access manager.
    // nosemgrep: sol-safety-no-immutable-variables
    AccessManager internal immutable ACCESS_MANAGER;

    /// @notice Semantic version.
    /// @custom:semver 0.2.0
    string public constant version = "0.2.0";

    /// @notice The starting timestamp of the game.
    Timestamp public createdAt;

    /// @notice The timestamp of the game's global resolution.
    Timestamp public resolvedAt;

    /// @notice The current status of the game.
    GameStatus public status;

    /// @notice Flag for the `initialize` function to prevent re-initialization.
    bool internal initialized;

    /// @notice The claim made by the proposer.
    ClaimData public claimData;

    /// @notice Credited balances for winning participants.
    mapping(address => uint256) public normalModeCredit;

    /// @notice A mapping of each claimant's refund mode credit.
    mapping(address => uint256) public refundModeCredit;

    /// @notice The starting output root of the game that is proven from in case of a challenge.
    /// @dev This should match the claim root of the parent game.
    Proposal public startingProposal;

    /// @notice A boolean for whether or not the game type was respected when the game was created.
    bool public wasRespectedGameTypeWhenCreated;

    /// @notice The bond distribution mode of the game.
    BondDistributionMode public bondDistributionMode;

    /// @param _maxChallengeDuration The maximum duration allowed for a challenger to challenge a game.
    /// @param _maxProveDuration The maximum duration allowed for a proposer to prove against a challenge.
    /// @param _disputeGameFactory The factory that creates the dispute games.
    /// @param _sp1Verifier The address of the SP1 verifier that verifies the proof for the aggregation program.
    /// @param _rollupConfigHash The rollup config hash for the L2 network.
    /// @param _aggregationVkey The vkey for the aggregation program.
    /// @param _rangeVkeyCommitment The commitment to the range vkey.
    /// @param _challengerBond The bond amount that must be submitted by the challenger.
    /// @param _anchorStateRegistry The anchor state registry for the L2 network.
    constructor(
        Duration _maxChallengeDuration,
        Duration _maxProveDuration,
        IDisputeGameFactory _disputeGameFactory,
        ISP1Verifier _sp1Verifier,
        bytes32 _rollupConfigHash,
        bytes32 _aggregationVkey,
        bytes32 _rangeVkeyCommitment,
        uint256 _challengerBond,
        IAnchorStateRegistry _anchorStateRegistry,
        AccessManager _accessManager
    ) {
        // Set up initial game state.
        GAME_TYPE = GameTypes.OPTIMISTIC_ZK_GAME_TYPE;
        MAX_CHALLENGE_DURATION = _maxChallengeDuration;
        MAX_PROVE_DURATION = _maxProveDuration;
        DISPUTE_GAME_FACTORY = _disputeGameFactory;
        SP1_VERIFIER = _sp1Verifier;
        ROLLUP_CONFIG_HASH = _rollupConfigHash;
        AGGREGATION_VKEY = _aggregationVkey;
        RANGE_VKEY_COMMITMENT = _rangeVkeyCommitment;
        CHALLENGER_BOND = _challengerBond;
        ANCHOR_STATE_REGISTRY = _anchorStateRegistry;
        ACCESS_MANAGER = _accessManager;
    }

    /// @notice Initializes the contract.
    /// @dev This function may only be called once.
    function initialize() external payable virtual {
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

        // INVARIANT: The game can only be initialized by the dispute game factory.
        if (address(DISPUTE_GAME_FACTORY) != msg.sender) revert IncorrectDisputeGameFactory();

        // INVARIANT: The proposer must be whitelisted.
        if (!ACCESS_MANAGER.isAllowedProposer(gameCreator())) revert BadAuth();

        // Revert if the calldata size is not the expected length.
        //
        // This is to prevent adding extra or omitting bytes from to `extraData` that result in a different game UUID
        // in the factory, but are not used by the game, which would allow for multiple dispute games for the same
        // output proposal to be created.
        //
        // Expected length: 0x7E
        // - 0x04 selector
        // - 0x14 creator address
        // - 0x20 root claim
        // - 0x20 l1 head
        // - 0x20 extraData (l2SequenceNumber)
        // - 0x04 extraData (parentIndex)
        // - 0x02 CWIA bytes
        assembly {
            if iszero(eq(calldatasize(), 0x7E)) {
                // Store the selector for `BadExtraData()` & revert
                mstore(0x00, 0x9824bdab)
                revert(0x1C, 0x04)
            }
        }

        // The first game is initialized with a parent index of uint32.max
        if (parentIndex() != type(uint32).max) {
            // For subsequent games, get the parent game's information
            (,, IDisputeGame proxy) = DISPUTE_GAME_FACTORY.gameAtIndex(parentIndex());

            // We perform a subset of AnchorStateRegistry.isGameProper() checks plus isGameRespected():
            // 1. isGameRespected(): Verifies the parent game was respected when it was created.
            //    There's only one respected game type in an AnchorStateRegistry at a time.
            // 2. isGameRetired(): Ensures the game hasn't been retroactively marked as retired.
            // 3. isGameBlacklisted(): Confirms the parent game isn't blacklisted.
            // Note: isGameRegistered() check is skipped since the parent game is coming directly from factory.
            if (
                !ANCHOR_STATE_REGISTRY.isGameRespected(proxy) || ANCHOR_STATE_REGISTRY.isGameBlacklisted(proxy)
                    || ANCHOR_STATE_REGISTRY.isGameRetired(proxy)
            ) {
                revert InvalidParentGame();
            }

            startingProposal = Proposal({
                l2SequenceNumber: OptimisticZkGame(address(proxy)).l2SequenceNumber(),
                root: Hash.wrap(OptimisticZkGame(address(proxy)).rootClaim().raw())
            });

            // INVARIANT: The parent game must be a valid game.
            if (proxy.status() == GameStatus.CHALLENGER_WINS) revert InvalidParentGame();
        } else {
            // When there is no parent game, the starting output root is the anchor state for the game type.
            (startingProposal.root, startingProposal.l2SequenceNumber) =
                IAnchorStateRegistry(ANCHOR_STATE_REGISTRY).anchors(GAME_TYPE);
        }

        // Do not allow the game to be initialized if the root claim corresponds to a block at or before the
        // configured starting block number.
        if (l2SequenceNumber() <= startingProposal.l2SequenceNumber) {
            revert UnexpectedRootClaim(rootClaim());
        }
        if (l2SequenceNumber() > type(uint64).max) {
            revert UnexpectedRootClaim(rootClaim());
        }

        // Set the root claim
        claimData = ClaimData({
            parentIndex: parentIndex(),
            counteredBy: address(0),
            prover: address(0),
            claim: rootClaim(),
            status: ProposalStatus.Unchallenged,
            deadline: Timestamp.wrap(uint64(block.timestamp + MAX_CHALLENGE_DURATION.raw()))
        });

        // Set the game as initialized.
        initialized = true;

        // Deposit the bond.
        refundModeCredit[gameCreator()] += msg.value;

        // Set the game's starting timestamp
        createdAt = Timestamp.wrap(uint64(block.timestamp));

        // Set whether the game type was respected when the game was created.
        wasRespectedGameTypeWhenCreated =
            GameType.unwrap(ANCHOR_STATE_REGISTRY.respectedGameType()) == GameType.unwrap(GAME_TYPE);
    }

    /// @notice The L2 block number for which this game is proposing an output root.
    function l2SequenceNumber() public pure returns (uint256 l2SequenceNumber_) {
        l2SequenceNumber_ = _getArgUint256(0x54);
    }

    /// @notice The parent index of the game.
    function parentIndex() public pure returns (uint32 parentIndex_) {
        parentIndex_ = _getArgUint32(0x74);
    }

    /// @notice Only the starting block number of the game.
    function startingBlockNumber() external view returns (uint256 startingBlockNumber_) {
        startingBlockNumber_ = startingProposal.l2SequenceNumber;
    }

    /// @notice Starting output root of the game.
    function startingRootHash() external view returns (Hash startingRootHash_) {
        startingRootHash_ = startingProposal.root;
    }

    ////////////////////////////////////////////////////////////////
    //                    `IDisputeGame` impl                     //
    ////////////////////////////////////////////////////////////////

    /// @notice Challenges the game.
    function challenge() external payable returns (ProposalStatus) {
        // INVARIANT: Can only challenge a game that has not been challenged yet.
        if (claimData.status != ProposalStatus.Unchallenged) revert ClaimAlreadyChallenged();

        // INVARIANT: The challenger must be whitelisted.
        if (!ACCESS_MANAGER.isAllowedChallenger(msg.sender)) revert BadAuth();

        // INVARIANT: Cannot challenge if the game is over.
        if (gameOver()) revert GameOver();

        // If the required bond is not met, revert.
        if (msg.value != CHALLENGER_BOND) revert IncorrectBondAmount();

        // Update the counteredBy address
        claimData.counteredBy = msg.sender;

        // Update the status of the proposal
        claimData.status = ProposalStatus.Challenged;

        // Update the clock to the current block timestamp, which marks the start of the challenge.
        claimData.deadline = Timestamp.wrap(uint64(block.timestamp + MAX_PROVE_DURATION.raw()));

        // Deposit the bond.
        refundModeCredit[msg.sender] += msg.value;

        emit Challenged(claimData.counteredBy);

        return claimData.status;
    }

    /// @notice Proves the game.
    /// @param _proofBytes The proof bytes to validate the claim.
    function prove(bytes calldata _proofBytes) external returns (ProposalStatus) {
        // INVARIANT: Cannot prove if the game is over.
        if (gameOver()) revert GameOver();

        // Decode the public values to check the claim root
        AggregationOutputs memory publicValues = AggregationOutputs({
            l1Head: Hash.unwrap(l1Head()),
            l2PreRoot: Hash.unwrap(startingProposal.root),
            claimRoot: rootClaim().raw(),
            claimBlockNum: l2SequenceNumber(),
            rollupConfigHash: ROLLUP_CONFIG_HASH,
            rangeVkeyCommitment: RANGE_VKEY_COMMITMENT,
            proverAddress: msg.sender
        });

        // Verify the proof. Reverts if the proof is invalid.
        SP1_VERIFIER.verifyProof(AGGREGATION_VKEY, abi.encode(publicValues), _proofBytes);

        // Update the prover address
        claimData.prover = msg.sender;

        // Update the status of the proposal
        if (claimData.counteredBy == address(0)) {
            claimData.status = ProposalStatus.UnchallengedAndValidProofProvided;
        } else {
            claimData.status = ProposalStatus.ChallengedAndValidProofProvided;
        }

        emit Proved(claimData.prover);

        return claimData.status;
    }

    /// @notice Returns the status of the parent game.
    /// @dev If the parent game index is `uint32.max`, then the parent game's status is considered as `DEFENDER_WINS`.
    function getParentGameStatus() private view returns (GameStatus) {
        if (parentIndex() != type(uint32).max) {
            (,, IDisputeGame parentGame) = DISPUTE_GAME_FACTORY.gameAtIndex(parentIndex());
            return parentGame.status();
        } else {
            // If this is the first dispute game (i.e. parent game index is `uint32.max`), then the
            // parent game's status is considered as `DEFENDER_WINS`.
            return GameStatus.DEFENDER_WINS;
        }
    }

    /// @notice Resolves the game after the clock expires.
    ///         `DEFENDER_WINS` when no one has challenged the proposer's claim and `MAX_CHALLENGE_DURATION` has passed
    ///         or there is a challenge but the prover has provided a valid proof within the `MAX_PROVE_DURATION`.
    ///         `CHALLENGER_WINS` when the proposer's claim has been challenged, but the proposer has not proven
    ///         its claim within the `MAX_PROVE_DURATION`.
    function resolve() external returns (GameStatus) {
        // INVARIANT: Resolution cannot occur unless the game has already been resolved.
        if (status != GameStatus.IN_PROGRESS) revert ClaimAlreadyResolved();

        // INVARIANT: Cannot resolve a game if the parent game has not been resolved.
        GameStatus parentGameStatus = getParentGameStatus();
        if (parentGameStatus == GameStatus.IN_PROGRESS) revert ParentGameNotResolved();

        // INVARIANT: If the parent game's claim is invalid, then the current game's claim is invalid.
        if (parentGameStatus == GameStatus.CHALLENGER_WINS) {
            // Parent game is invalid so this game is invalid too. Therefore the challenger wins and gets all bonds.
            // If the game has not been challenged then there will not be any challenger address and the bond is burned.
            status = GameStatus.CHALLENGER_WINS;
            normalModeCredit[claimData.counteredBy] = address(this).balance;
        } else {
            // INVARIANT: Game must be completed either by clock expiration or valid proof.
            if (!gameOver()) revert GameNotOver();

            // Determine status based on claim status.
            if (claimData.status == ProposalStatus.Unchallenged) {
                // Claim is unchallenged, defender wins, game creator gets everything.
                status = GameStatus.DEFENDER_WINS;
                normalModeCredit[gameCreator()] = address(this).balance;
            } else if (claimData.status == ProposalStatus.Challenged) {
                // Claim is challenged, challenger wins, challenger wins everything
                status = GameStatus.CHALLENGER_WINS;
                normalModeCredit[claimData.counteredBy] = address(this).balance;
            } else if (claimData.status == ProposalStatus.UnchallengedAndValidProofProvided) {
                // Claim is unchallenged but a valid proof was provided, defender wins, game
                // creator gets everything. Note that the prover does not receive any reward in
                // this particular case.
                status = GameStatus.DEFENDER_WINS;
                normalModeCredit[gameCreator()] = address(this).balance;
            } else if (claimData.status == ProposalStatus.ChallengedAndValidProofProvided) {
                // Claim is challenged but a valid proof was provided, defender wins, prover gets
                // the challenger's bond and the game creator gets everything else.
                status = GameStatus.DEFENDER_WINS;

                // If the prover is same as the proposer, the proposer takes the entire bond.
                if (claimData.prover == gameCreator()) {
                    normalModeCredit[claimData.prover] = address(this).balance;
                }
                // If the prover is different from the proposer, the proposer gets the initial bond back,
                // and the prover gets the challenger's bond.
                else {
                    normalModeCredit[claimData.prover] = CHALLENGER_BOND;
                    normalModeCredit[gameCreator()] = address(this).balance - CHALLENGER_BOND;
                }
            } else {
                // This edge case shouldn't be reached, sanity check just in case.
                revert InvalidProposalStatus();
            }
        }

        // Mark the game as resolved.
        claimData.status = ProposalStatus.Resolved;
        resolvedAt = Timestamp.wrap(uint64(block.timestamp));
        emit Resolved(status);

        return status;
    }

    /// @notice Claim the credit belonging to the recipient address. Reverts if the game isn't
    ///         finalized, if the recipient has no credit to claim, or if the bond transfer
    ///         fails. If the game is finalized but no bond has been paid out yet, this method
    ///         will determine the bond distribution mode and also try to update anchor game.
    /// @param _recipient The owner and recipient of the credit.
    function claimCredit(address _recipient) external {
        // Close out the game and determine the bond distribution mode if not already set.
        // We call this as part of claim credit to reduce the number of additional calls that a
        // Challenger needs to make to this contract.
        closeGame();

        // Fetch the recipient's credit balance based on the bond distribution mode.
        uint256 recipientCredit;
        if (bondDistributionMode == BondDistributionMode.REFUND) {
            recipientCredit = refundModeCredit[_recipient];
        } else if (bondDistributionMode == BondDistributionMode.NORMAL) {
            recipientCredit = normalModeCredit[_recipient];
        } else {
            // We shouldn't get here, but sanity check just in case.
            revert InvalidBondDistributionMode();
        }

        // Revert if the recipient has no credit to claim.
        if (recipientCredit == 0) revert NoCreditToClaim();

        // Set the recipient's credit balances to 0.
        refundModeCredit[_recipient] = 0;
        normalModeCredit[_recipient] = 0;

        // Transfer the credit to the recipient.
        (bool success,) = _recipient.call{ value: recipientCredit }(hex"");
        if (!success) revert BondTransferFailed();
    }

    /// @notice Closes out the game, determines the bond distribution mode, attempts to register
    ///         the game as the anchor game, and emits an event.
    function closeGame() public {
        // If the bond distribution mode has already been determined, we can return early.
        if (bondDistributionMode == BondDistributionMode.REFUND || bondDistributionMode == BondDistributionMode.NORMAL)
        {
            // We can't revert or we'd break claimCredit().
            return;
        } else if (bondDistributionMode != BondDistributionMode.UNDECIDED) {
            // We shouldn't get here, but sanity check just in case.
            revert InvalidBondDistributionMode();
        }

        // Game must be finalized according to the AnchorStateRegistry.
        bool finalized = ANCHOR_STATE_REGISTRY.isGameFinalized(IDisputeGame(address(this)));
        if (!finalized) {
            revert GameNotFinalized();
        }

        // Try to update the anchor game first. Won't always succeed because delays can lead
        // to situations in which this game might not be eligible to be a new anchor game.
        // nosemgrep: sol-safety-trycatch-eip150
        try ANCHOR_STATE_REGISTRY.setAnchorState(IDisputeGame(address(this))) { } catch { }

        // Check if the game is a proper game, which will determine the bond distribution mode.
        bool properGame = ANCHOR_STATE_REGISTRY.isGameProper(IDisputeGame(address(this)));

        // If the game is a proper game, the bonds should be distributed normally. Otherwise, go
        // into refund mode and distribute bonds back to their original depositors.
        if (properGame) {
            bondDistributionMode = BondDistributionMode.NORMAL;
        } else {
            bondDistributionMode = BondDistributionMode.REFUND;
        }

        // Emit an event to signal that the game has been closed.
        emit GameClosed(bondDistributionMode);
    }

    /// @notice Determines if the game is finished.
    /// @return gameOver_ True if the game is either expired or proven.
    function gameOver() public view returns (bool gameOver_) {
        gameOver_ = claimData.deadline.raw() < uint64(block.timestamp) || claimData.prover != address(0);
    }

    /// @notice Getter for the game type.
    /// @dev The reference impl should be entirely different depending on the type (fault, validity)
    ///      i.e. The game type should indicate the security model.
    /// @return gameType_ The type of proof system being used.
    function gameType() public view returns (GameType gameType_) {
        gameType_ = GAME_TYPE;
    }

    /// @notice Getter for the creator of the dispute game.
    /// @dev `clones-with-immutable-args` argument #1
    /// @return creator_ The creator of the dispute game.
    function gameCreator() public pure returns (address creator_) {
        creator_ = _getArgAddress(0x00);
    }

    /// @notice Getter for the root claim.
    /// @dev `clones-with-immutable-args` argument #2
    /// @return rootClaim_ The root claim of the DisputeGame.
    function rootClaim() public pure returns (Claim rootClaim_) {
        rootClaim_ = Claim.wrap(_getArgBytes32(0x14));
    }

    /// @notice Getter for the root claim for a given L2 chain ID.
    /// @dev For pre-interop games, returns the root claim regardless of chain ID.
    /// @return rootClaim_ The root claim of the DisputeGame.
    function rootClaimByChainId(uint256) public pure returns (Claim rootClaim_) {
        rootClaim_ = rootClaim();
    }

    /// @notice Getter for the parent hash of the L1 block when the dispute game was created.
    /// @dev `clones-with-immutable-args` argument #3
    /// @return l1Head_ The parent hash of the L1 block when the dispute game was created.
    function l1Head() public pure returns (Hash l1Head_) {
        l1Head_ = Hash.wrap(_getArgBytes32(0x34));
    }

    /// @notice Getter for the extra data.
    /// @dev `clones-with-immutable-args` argument #4
    /// @return extraData_ Any extra data supplied to the dispute game contract by the creator.
    function extraData() public pure returns (bytes memory extraData_) {
        // The extra data starts at the second word within the cwia calldata and
        // is 36 bytes long. 32 bytes are for the l2SequenceNumber, 4 bytes are for the parentIndex.
        extraData_ = _getArgBytes(0x54, 0x24);
    }

    /// @notice A compliant implementation of this interface should return the components of the
    ///         game UUID's preimage provided in the cwia payload. The preimage of the UUID is
    ///         constructed as `keccak256(gameType . rootClaim . extraData)` where `.` denotes
    ///         concatenation.
    /// @return gameType_ The type of proof system being used.
    /// @return rootClaim_ The root claim of the DisputeGame.
    /// @return extraData_ Any extra data supplied to the dispute game contract by the creator.
    function gameData() external view returns (GameType gameType_, Claim rootClaim_, bytes memory extraData_) {
        gameType_ = gameType();
        rootClaim_ = rootClaim();
        extraData_ = extraData();
    }

    ////////////////////////////////////////////////////////////////
    //                       MISC EXTERNAL                        //
    ////////////////////////////////////////////////////////////////

    /// @notice Returns the credit balance of a given recipient.
    /// @param _recipient The recipient of the credit.
    /// @return credit_ The credit balance of the recipient.
    function credit(address _recipient) external view returns (uint256 credit_) {
        if (bondDistributionMode == BondDistributionMode.REFUND) {
            credit_ = refundModeCredit[_recipient];
        } else {
            // Always return normal credit balance by default unless in refund mode.
            credit_ = normalModeCredit[_recipient];
        }
    }

    ////////////////////////////////////////////////////////////////
    //                     IMMUTABLE GETTERS                      //
    ////////////////////////////////////////////////////////////////

    /// @notice Returns the max challenge duration.
    function maxChallengeDuration() external view returns (Duration maxChallengeDuration_) {
        maxChallengeDuration_ = MAX_CHALLENGE_DURATION;
    }

    /// @notice Returns the max prove duration.
    function maxProveDuration() external view returns (Duration maxProveDuration_) {
        maxProveDuration_ = MAX_PROVE_DURATION;
    }

    /// @notice Returns the dispute game factory.
    function disputeGameFactory() external view returns (IDisputeGameFactory disputeGameFactory_) {
        disputeGameFactory_ = DISPUTE_GAME_FACTORY;
    }

    /// @notice Returns the SP1 verifier contract.
    function sp1Verifier() external view returns (ISP1Verifier verifier_) {
        verifier_ = SP1_VERIFIER;
    }

    /// @notice Returns the rollup config hash.
    function rollupConfigHash() external view returns (bytes32 rollupConfigHash_) {
        rollupConfigHash_ = ROLLUP_CONFIG_HASH;
    }

    /// @notice Returns the aggregation vkey.
    function aggregationVkey() external view returns (bytes32 aggregationVkey_) {
        aggregationVkey_ = AGGREGATION_VKEY;
    }

    /// @notice Returns the range vkey commitment.
    function rangeVkeyCommitment() external view returns (bytes32 rangeVkeyCommitment_) {
        rangeVkeyCommitment_ = RANGE_VKEY_COMMITMENT;
    }

    /// @notice Returns the challenger bond amount.
    function challengerBond() external view returns (uint256 challengerBond_) {
        challengerBond_ = CHALLENGER_BOND;
    }

    /// @notice Returns the anchor state registry contract.
    function anchorStateRegistry() external view returns (IAnchorStateRegistry registry_) {
        registry_ = ANCHOR_STATE_REGISTRY;
    }

    /// @notice Returns the access manager contract.
    function accessManager() external view returns (AccessManager accessManager_) {
        accessManager_ = ACCESS_MANAGER;
    }
}
