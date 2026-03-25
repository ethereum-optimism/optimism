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
import {
    AlreadyInitialized,
    BondTransferFailed,
    ClaimAlreadyResolved,
    GameNotFinalized,
    GameNotResolved,
    GamePaused,
    IncorrectBondAmount,
    InvalidBondDistributionMode,
    NoCreditToClaim,
    UnexpectedRootClaim,
    UnexpectedGameType,
    UnknownChainId,
    ClaimAlreadyChallenged,
    InvalidParentGame,
    ParentGameNotResolved,
    GameOver,
    GameNotOver,
    InvalidProposalStatus
} from "src/dispute/lib/Errors.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IZKVerifier } from "interfaces/dispute/zk/IZKVerifier.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";

/// @title ZKDisputeGame
/// @notice A ZK proof-based dispute game using the MCP (Modular Clone Proxy) pattern
///         with Clone-With-Immutable-Args (CWIA). Spec-compliant, permissionless
///         design that uses a generic IZKVerifier and DelayedWETH for bond custody.
/// @dev Derived from https://github.com/succinctlabs/op-succinct (at commit c13844a9bbc330cca69eef2538d8f8ec123e1653)
contract ZKDisputeGame is Clone, ISemver, IDisputeGame {
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
        uint32 parentIndex; // 4 bytes  \
        ProposalStatus status; // 1 byte    |-- slot 1 (25 bytes)
        address challenger; // 20 bytes /
        address prover; // 20 bytes \
        Timestamp deadline; // 8 bytes  /-- slot 2 (28 bytes)
        Claim claim; // 32 bytes --- slot 3
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

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

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
    Proposal public startingProposal;

    /// @notice The bond distribution mode of the game.
    BondDistributionMode public bondDistributionMode;

    /// @notice A boolean for whether or not the game type was respected when the game was created.
    bool public wasRespectedGameTypeWhenCreated;

    /// @notice The dispute game factory that created this game.
    IDisputeGameFactory public disputeGameFactory;

    /// @notice The total bonds deposited into the game.
    uint256 public totalBonds;

    ////////////////////////////////////////////////////////////////
    //                     CWIA GETTERS                           //
    ////////////////////////////////////////////////////////////////

    /// @notice Getter for the creator of the dispute game.
    /// @return creator_ The creator of the dispute game.
    function gameCreator() public pure returns (address creator_) {
        creator_ = _getArgAddress(0x00);
    }

    /// @notice Getter for the root claim.
    /// @return rootClaim_ The root claim of the DisputeGame.
    function rootClaim() public pure returns (Claim rootClaim_) {
        rootClaim_ = Claim.wrap(_getArgBytes32(0x14));
    }

    /// @notice Getter for the parent hash of the L1 block when the dispute game was created.
    /// @return l1Head_ The parent hash of the L1 block when the dispute game was created.
    function l1Head() public pure returns (Hash l1Head_) {
        l1Head_ = Hash.wrap(_getArgBytes32(0x34));
    }

    /// @notice Getter for the game type.
    /// @return gameType_ The type of proof system being used.
    function gameType() public pure returns (GameType gameType_) {
        gameType_ = GameType.wrap(_getArgUint32(0x54));
    }

    /// @notice The L2 sequence number for which this game proposes an output root.
    /// @dev Per spec, this value must fit within a uint64.
    function l2SequenceNumber() public pure returns (uint256 l2SequenceNumber_) {
        l2SequenceNumber_ = _getArgUint256(0x58);
    }

    /// @notice The parent index of the game.
    function parentIndex() public pure returns (uint32 parentIndex_) {
        parentIndex_ = _getArgUint32(0x78);
    }

    /// @notice Returns the absolute prestate commitment (ZK circuit identity).
    function absolutePrestate() public pure returns (bytes32 absolutePrestate_) {
        absolutePrestate_ = _getArgBytes32(0x7C);
    }

    /// @notice Returns the ZK verifier contract.
    function verifier() public pure returns (IZKVerifier verifier_) {
        verifier_ = IZKVerifier(_getArgAddress(0x9C));
    }

    /// @notice Returns the max challenge duration.
    function maxChallengeDuration() public pure returns (Duration maxChallengeDuration_) {
        maxChallengeDuration_ = Duration.wrap(_getArgUint64(0xB0));
    }

    /// @notice Returns the max prove duration.
    function maxProveDuration() public pure returns (Duration maxProveDuration_) {
        maxProveDuration_ = Duration.wrap(_getArgUint64(0xB8));
    }

    /// @notice Returns the challenger bond amount.
    function challengerBond() public pure returns (uint256 challengerBond_) {
        challengerBond_ = _getArgUint256(0xC0);
    }

    /// @notice Returns the anchor state registry contract.
    function anchorStateRegistry() public pure returns (IAnchorStateRegistry registry_) {
        registry_ = IAnchorStateRegistry(_getArgAddress(0xE0));
    }

    /// @notice Returns the DelayedWETH contract used for bond custody.
    function weth() public pure returns (IDelayedWETH weth_) {
        weth_ = IDelayedWETH(payable(_getArgAddress(0xF4)));
    }

    /// @notice Returns the L2 chain ID.
    function l2ChainId() public pure returns (uint256 l2ChainId_) {
        l2ChainId_ = _getArgUint256(0x108);
    }

    /// @notice Getter for the extra data.
    /// @return extraData_ Any extra data supplied to the dispute game contract by the creator.
    function extraData() public pure returns (bytes memory extraData_) {
        // The extra data starts at the second word within the cwia calldata and
        // is 36 bytes long. 32 bytes are for the l2SequenceNumber, 4 bytes are for the parentIndex.
        extraData_ = _getArgBytes(0x58, 0x24);
    }

    /// @notice Only the starting block number of the game.
    function startingBlockNumber() external view returns (uint256 startingBlockNumber_) {
        startingBlockNumber_ = startingProposal.l2SequenceNumber;
    }

    /// @notice Starting output root of the game.
    function startingRootHash() external view returns (Hash startingRootHash_) {
        startingRootHash_ = startingProposal.root;
    }

    /// @notice Getter for the root claim for a given L2 chain ID.
    /// @return rootClaim_ The root claim of the DisputeGame.
    function rootClaimByChainId(uint256) public pure returns (Claim rootClaim_) {
        rootClaim_ = rootClaim();
    }

    /// @notice Returns the components of the game UUID's preimage provided in the cwia payload.
    /// @return gameType_ The type of proof system being used.
    /// @return rootClaim_ The root claim of the DisputeGame.
    /// @return extraData_ Any extra data supplied to the dispute game contract by the creator.
    function gameData() external pure returns (GameType gameType_, Claim rootClaim_, bytes memory extraData_) {
        gameType_ = gameType();
        rootClaim_ = rootClaim();
        extraData_ = extraData();
    }

    ////////////////////////////////////////////////////////////////
    //                    INITIALIZATION                          //
    ////////////////////////////////////////////////////////////////

    /// @notice Initializes the contract.
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

        // Revert if the calldata size is not the expected length.
        //
        // This is to prevent adding extra or omitting bytes from to `extraData` that result in a different game UUID
        // in the factory, but are not used by the game, which would allow for multiple dispute games for the same
        // output proposal to be created.
        //
        // Expected length: 0x12E
        // - 0x04 selector
        // - 0x14 creator address
        // - 0x20 root claim
        // - 0x20 l1 head
        // - 0x04 gameType (factory-inserted)
        // - 0x20 extraData (l2SequenceNumber)
        // - 0x04 extraData (parentIndex)
        // - 0xAC gameArgs (absolutePrestate + verifier + durations + bond + registry + weth + l2ChainId)
        // - 0x02 CWIA bytes
        assembly {
            if iszero(eq(calldatasize(), 0x12E)) {
                // Store the selector for `BadExtraData()` & revert
                mstore(0x00, 0x9824bdab)
                revert(0x1C, 0x04)
            }
        }

        // INVARIANT: The L2 chain ID must not be zero.
        if (l2ChainId() == 0) revert UnknownChainId();

        // Store the factory reference for parent game lookups.
        disputeGameFactory = IDisputeGameFactory(msg.sender);

        // The first game is initialized with a parent index of uint32.max
        if (parentIndex() != type(uint32).max) {
            // For subsequent games, get the parent game's information
            (,, IDisputeGame proxy) = disputeGameFactory.gameAtIndex(parentIndex());

            // Verify parent game is not blacklisted or retired.
            if (anchorStateRegistry().isGameBlacklisted(proxy) || anchorStateRegistry().isGameRetired(proxy)) {
                revert InvalidParentGame();
            }

            // INVARIANT: The parent game must be of the same game type.
            if (IDisputeGame(payable(address(proxy))).gameType().raw() != gameType().raw()) {
                revert UnexpectedGameType();
            }

            startingProposal = Proposal({
                l2SequenceNumber: IDisputeGame(payable(address(proxy))).l2SequenceNumber(),
                root: Hash.wrap(IDisputeGame(payable(address(proxy))).rootClaim().raw())
            });

            // INVARIANT: The parent game's sequence number must be strictly above the anchor state.
            (, uint256 anchorL2SeqNum) = anchorStateRegistry().getAnchorRoot();
            if (startingProposal.l2SequenceNumber <= anchorL2SeqNum) revert InvalidParentGame();

            // INVARIANT: The parent game must be a valid game.
            if (proxy.status() == GameStatus.CHALLENGER_WINS) revert InvalidParentGame();
        } else {
            // When there is no parent game, the starting output root is the anchor state for the game type.
            (startingProposal.root, startingProposal.l2SequenceNumber) = anchorStateRegistry().getAnchorRoot();
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
            status: ProposalStatus.Unchallenged,
            challenger: address(0),
            prover: address(0),
            deadline: Timestamp.wrap(uint64(block.timestamp + maxChallengeDuration().raw())),
            claim: rootClaim()
        });

        // Set the game as initialized.
        initialized = true;

        // Deposit the bond into DelayedWETH and track credits.
        refundModeCredit[gameCreator()] += msg.value;
        totalBonds += msg.value;
        weth().deposit{ value: msg.value }();

        // Set the game's starting timestamp
        createdAt = Timestamp.wrap(uint64(block.timestamp));

        // Set whether the game type was respected when the game was created.
        wasRespectedGameTypeWhenCreated =
            GameType.unwrap(anchorStateRegistry().respectedGameType()) == GameType.unwrap(gameType());
    }

    ////////////////////////////////////////////////////////////////
    //                    `IDisputeGame` impl                     //
    ////////////////////////////////////////////////////////////////

    /// @notice Challenges the game.
    function challenge() external payable returns (ProposalStatus) {
        // INVARIANT: Cannot challenge if the game is over.
        if (gameOver()) revert GameOver();

        // INVARIANT: Can only challenge a game that has not been challenged yet.
        if (claimData.status != ProposalStatus.Unchallenged) revert ClaimAlreadyChallenged();

        // If the required bond is not met, revert.
        if (msg.value != challengerBond()) revert IncorrectBondAmount();

        // Update the challenger address
        claimData.challenger = msg.sender;

        // Update the status of the proposal
        claimData.status = ProposalStatus.Challenged;

        // Update the clock to the current block timestamp, which marks the start of the challenge.
        claimData.deadline = Timestamp.wrap(uint64(block.timestamp + maxProveDuration().raw()));

        // Deposit the bond into DelayedWETH and track credits.
        refundModeCredit[msg.sender] += msg.value;
        totalBonds += msg.value;
        weth().deposit{ value: msg.value }();

        emit Challenged(claimData.challenger);

        return claimData.status;
    }

    /// @notice Proves the game.
    /// @param _proofBytes The proof bytes to validate the claim.
    function prove(bytes calldata _proofBytes) external returns (ProposalStatus) {
        // INVARIANT: Cannot prove if the game is already resolved.
        if (status != GameStatus.IN_PROGRESS) revert ClaimAlreadyResolved();

        // INVARIANT: Cannot prove if the parent game is invalid.
        if (getParentGameStatus() == GameStatus.CHALLENGER_WINS) revert InvalidParentGame();

        // INVARIANT: Cannot prove if the game is over.
        if (gameOver()) revert GameOver();

        // Construct the public values for verification.
        bytes memory publicValues =
            abi.encode(l1Head(), startingProposal.root, rootClaim(), l2SequenceNumber(), l2ChainId(), msg.sender);

        // Verify the proof. Reverts if the proof is invalid.
        verifier().verify(absolutePrestate(), publicValues, _proofBytes);

        // Update the prover address
        claimData.prover = msg.sender;

        // Update the status of the proposal
        if (claimData.challenger == address(0)) {
            claimData.status = ProposalStatus.UnchallengedAndValidProofProvided;
        } else {
            claimData.status = ProposalStatus.ChallengedAndValidProofProvided;
        }

        emit Proved(claimData.prover);

        return claimData.status;
    }

    /// @notice Returns the status of the parent game.
    function getParentGameStatus() private view returns (GameStatus) {
        if (parentIndex() != type(uint32).max) {
            (,, IDisputeGame parentGame) = disputeGameFactory.gameAtIndex(parentIndex());
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
            normalModeCredit[claimData.challenger] = totalBonds;
        } else {
            // INVARIANT: Game must be completed either by clock expiration or valid proof.
            if (!gameOver()) revert GameNotOver();

            // Determine status based on claim status.
            if (claimData.status == ProposalStatus.Unchallenged) {
                // Claim is unchallenged, defender wins, game creator gets everything.
                status = GameStatus.DEFENDER_WINS;
                normalModeCredit[gameCreator()] = totalBonds;
            } else if (claimData.status == ProposalStatus.Challenged) {
                // Claim is challenged, challenger wins, challenger wins everything
                status = GameStatus.CHALLENGER_WINS;
                normalModeCredit[claimData.challenger] = totalBonds;
            } else if (claimData.status == ProposalStatus.UnchallengedAndValidProofProvided) {
                // Claim is unchallenged but a valid proof was provided, defender wins, game
                // creator gets everything. Note that the prover does not receive any reward in
                // this particular case.
                status = GameStatus.DEFENDER_WINS;
                normalModeCredit[gameCreator()] = totalBonds;
            } else if (claimData.status == ProposalStatus.ChallengedAndValidProofProvided) {
                // Claim is challenged but a valid proof was provided, defender wins, prover gets
                // the challenger's bond and the game creator gets everything else.
                status = GameStatus.DEFENDER_WINS;

                // If the prover is same as the proposer, the proposer takes the entire bond.
                if (claimData.prover == gameCreator()) {
                    normalModeCredit[claimData.prover] = totalBonds;
                }
                // If the prover is different from the proposer, the proposer gets the initial bond back,
                // and the prover gets the challenger's bond.
                else {
                    normalModeCredit[claimData.prover] = challengerBond();
                    normalModeCredit[gameCreator()] = totalBonds - challengerBond();
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

    /// @notice Claim the credit belonging to the recipient address. Uses a two-phase
    ///         DelayedWETH withdrawal pattern: first call unlocks, second call withdraws.
    ///         Does not revert if the recipient has no credit (even if they have no credit at all).
    ///         This lets op-challenger use claimCredit to close the game.
    /// @param _recipient The owner and recipient of the credit.
    function claimCredit(address _recipient) external {
        // Track whether the game was already closed before this call. If closeGame() sets
        // bondDistributionMode in this transaction and we later find nothing to claim, we must
        // return instead of revert so that the closeGame() state changes are not rolled back.
        bool gameWasOpen = (bondDistributionMode == BondDistributionMode.UNDECIDED);

        // Close out the game and determine the bond distribution mode if not already set.
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

        // Phase 1: If the recipient still has credit, zero it out and unlock in DelayedWETH.
        // Credits are only assigned once per address, so a non-zero balance means the unlock
        // hasn't happened yet.
        if (recipientCredit > 0) {
            refundModeCredit[_recipient] = 0;
            normalModeCredit[_recipient] = 0;
            weth().unlock(_recipient, recipientCredit);
            return;
        }

        // Phase 2: Credits have been zeroed (phase 1 completed). Check DelayedWETH for a
        // pending withdrawal and finalize it.
        (uint256 amount,) = weth().withdrawals(address(this), _recipient);
        if (amount == 0) {
            // If the game was just closed in this transaction, return without reverting so
            // the closeGame() state changes persist. This is the intended path for callers
            // (e.g. op-challenger) that use claimCredit solely to close the game.
            if (gameWasOpen) return;
            revert NoCreditToClaim();
        }

        // Withdraw the WETH amount so it can be used here.
        weth().withdraw(_recipient, amount);

        // Transfer the credit to the recipient.
        (bool success,) = _recipient.call{ value: amount }(hex"");
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

        // We won't close the game if the system is currently paused. Paused games are temporarily
        // invalid which would cause the game to go into refund mode and potentially cause some
        // confusion for honest challengers. By blocking the game from being closed while the
        // system is paused, the game will only go into refund mode if it ends up being explicitly
        // invalidated in the AnchorStateRegistry. If the game has already been closed and a refund
        // mode has been selected, we'll already have returned and we won't hit this revert.
        if (anchorStateRegistry().paused()) {
            revert GamePaused();
        }

        // Make sure that the game is resolved.
        // AnchorStateRegistry should be checking this but we're being defensive here.
        if (resolvedAt.raw() == 0) {
            revert GameNotResolved();
        }

        // Game must be finalized according to the AnchorStateRegistry.
        bool finalized = anchorStateRegistry().isGameFinalized(IDisputeGame(address(this)));
        if (!finalized) {
            revert GameNotFinalized();
        }

        // Try to update the anchor game first. Won't always succeed because delays can lead
        // to situations in which this game might not be eligible to be a new anchor game.
        // nosemgrep: sol-safety-trycatch-eip150
        try anchorStateRegistry().setAnchorState(IDisputeGame(address(this))) { } catch { }

        // Check if the game is a proper game, which will determine the bond distribution mode.
        bool properGame = anchorStateRegistry().isGameProper(IDisputeGame(address(this)));

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

    ////////////////////////////////////////////////////////////////
    //                       MISC EXTERNAL                        //
    ////////////////////////////////////////////////////////////////

    /// @notice Determines if the game is finished.
    /// @return gameOver_ True if the game is either expired or proven.
    function gameOver() public view returns (bool gameOver_) {
        gameOver_ = claimData.deadline.raw() < uint64(block.timestamp) || claimData.prover != address(0);
    }

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
}
