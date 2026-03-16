// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { DisputeGameFactory_TestInit } from "test/dispute/DisputeGameFactory.t.sol";

// Libraries
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { Claim, Duration, GameStatus, GameType, Timestamp } from "src/dispute/lib/Types.sol";
import {
    IncorrectBondAmount,
    UnexpectedRootClaim,
    NoCreditToClaim,
    GameNotFinalized,
    GameNotResolved,
    GamePaused,
    ParentGameNotResolved,
    InvalidParentGame,
    ClaimAlreadyChallenged,
    ClaimAlreadyResolved,
    GameOver,
    GameNotOver,
    UnknownChainId
} from "src/dispute/lib/Errors.sol";
import { GameTypes } from "src/dispute/lib/Types.sol";

// Contracts
import { DisputeGameFactory } from "src/dispute/DisputeGameFactory.sol";
import { ZKDisputeGame } from "src/dispute/zk/ZKDisputeGame.sol";

// Interfaces
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IZKVerifier } from "interfaces/dispute/zk/IZKVerifier.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { Proxy } from "src/universal/Proxy.sol";

/// @title ZKDisputeGame_Init
/// @notice Base test contract with shared setup for ZKDisputeGame tests.
abstract contract ZKDisputeGame_Init is DisputeGameFactory_TestInit {
    // Events
    event Challenged(address indexed challenger);
    event Proved(address indexed prover);
    event Resolved(GameStatus indexed status);

    ZKDisputeGame gameImpl;
    ZKDisputeGame parentGame;
    ZKDisputeGame game;

    address proposer = address(0x123);
    address challenger = address(0x456);
    address prover = address(0x789);

    // Fixed parameters.
    GameType gameType = GameTypes.ZK_DISPUTE_GAME;
    Duration maxChallengeDuration = Duration.wrap(12 hours);
    Duration maxProveDuration = Duration.wrap(3 days);
    Claim rootClaim = Claim.wrap(keccak256("rootClaim"));

    // Sequence number offsets from anchor state (for parent and child games).
    uint256 parentSequenceOffset = 1000;
    uint256 childSequenceOffset = 2000;

    // Game indices are set dynamically in setUp (on fork, existing games already exist)
    uint32 parentGameIndex;
    uint32 childGameIndex;

    // Offsets from child sequence number for grandchild games.
    uint256 grandchildOffset1 = 1000;
    uint256 grandchildOffset2 = 2000;
    uint256 grandchildOffset3 = 3000;
    uint256 grandchildOffset4 = 8000;

    // Actual sequence numbers (set in setUp based on anchor state)
    uint256 anchorL2SequenceNumber;
    uint256 parentL2SequenceNumber;
    uint256 childL2SequenceNumber;

    function setUp() public virtual override {
        super.setUp();
        skipIfDevFeatureDisabled(DevFeatures.ZK_DISPUTE_GAME);
        skipIfForkTest("Skip not supported yet");

        // Get anchor state to calculate valid sequence numbers
        (, anchorL2SequenceNumber) = anchorStateRegistry.getAnchorRoot();
        parentL2SequenceNumber = anchorL2SequenceNumber + parentSequenceOffset;
        childL2SequenceNumber = anchorL2SequenceNumber + childSequenceOffset;

        // Setup game implementation using shared helper
        address impl;
        (impl,) = setupZKDisputeGame(
            ZKDisputeGameParams({
                maxChallengeDuration: maxChallengeDuration,
                maxProveDuration: maxProveDuration,
                absolutePrestate: bytes32(0),
                challengerBond: 1 ether
            })
        );
        gameImpl = ZKDisputeGame(payable(impl));

        // Create the first (parent) game - it uses uint32.max as parent index.
        vm.startPrank(proposer);
        vm.deal(proposer, 10 ether);

        // Warp time forward to ensure the parent game is created after the respectedGameTypeUpdatedAt timestamp.
        vm.warp(block.timestamp + 1000);

        // Create parent game (uses uint32.max to indicate first game in chain).
        parentGame = ZKDisputeGame(
            payable(
                address(
                    disputeGameFactory.create{ value: 1 ether }(
                        gameType,
                        Claim.wrap(keccak256("genesis")),
                        abi.encodePacked(parentL2SequenceNumber, type(uint32).max)
                    )
                )
            )
        );

        // Record actual index of parent game (on fork, existing games already occupy indices 0, 1, ...)
        parentGameIndex = uint32(disputeGameFactory.gameCount() - 1);

        // We want the parent game to finalize. We'll skip its challenge period.
        (,,,, Timestamp parentGameDeadline,) = parentGame.claimData();
        vm.warp(parentGameDeadline.raw() + 1 seconds);
        parentGame.resolve();

        // Claim credit (two-phase: unlock then withdraw)
        uint256 finalityDelay = anchorStateRegistry.disputeGameFinalityDelaySeconds();
        vm.warp(parentGame.resolvedAt().raw() + finalityDelay + 1 seconds);
        parentGame.claimCredit(proposer); // Phase 1: unlock

        vm.warp(block.timestamp + delayedWeth.delay() + 1 seconds);
        parentGame.claimCredit(proposer); // Phase 2: withdraw

        // Create the child game referencing actual parent game index.
        game = ZKDisputeGame(
            payable(
                address(
                    disputeGameFactory.create{ value: 1 ether }(
                        gameType, rootClaim, abi.encodePacked(childL2SequenceNumber, parentGameIndex)
                    )
                )
            )
        );

        // Record actual index of child game.
        childGameIndex = uint32(disputeGameFactory.gameCount() - 1);

        vm.stopPrank();
    }

    /// @notice Helper to perform two-phase credit claim (unlock + withdraw).
    function _claimCreditTwoPhase(ZKDisputeGame _game, address _recipient) internal {
        _game.claimCredit(_recipient); // Phase 1: unlock
        vm.warp(block.timestamp + delayedWeth.delay() + 1 seconds);
        _game.claimCredit(_recipient); // Phase 2: withdraw
    }
}

/// @title ZKDisputeGame_Initialize_Test
/// @notice Tests for initialization of ZKDisputeGame.
contract ZKDisputeGame_Initialize_Test is ZKDisputeGame_Init {
    function test_initialize_succeeds() public view {
        // Test that the factory is correctly initialized.
        assertEq(address(disputeGameFactory.owner()), address(this));
        assertEq(address(disputeGameFactory.gameImpls(gameType)), address(gameImpl));
        // We expect games including parent and child (indices may vary on fork).
        assertEq(disputeGameFactory.gameCount(), childGameIndex + 1);

        // Check that our child game matches the game at childGameIndex.
        (,, IDisputeGame proxy_) = disputeGameFactory.gameAtIndex(childGameIndex);
        assertEq(address(game), address(proxy_));

        // Check the child game fields via CWIA getters.
        assertEq(game.gameType().raw(), gameType.raw());
        assertEq(game.rootClaim().raw(), rootClaim.raw());
        assertEq(game.maxChallengeDuration().raw(), maxChallengeDuration.raw());
        assertEq(game.maxProveDuration().raw(), maxProveDuration.raw());
        assertEq(address(game.disputeGameFactory()), address(disputeGameFactory));
        assertEq(game.l2SequenceNumber(), childL2SequenceNumber);
        assertEq(game.l2ChainId(), l2ChainId);
        assertEq(address(game.weth()), address(delayedWeth));
        assertEq(address(game.anchorStateRegistry()), address(anchorStateRegistry));

        // The parent's sequence number (startingBlockNumber() returns l2SequenceNumber).
        assertEq(game.startingBlockNumber(), parentL2SequenceNumber);

        // The parent's root was keccak256("genesis").
        assertEq(game.startingRootHash().raw(), keccak256("genesis"));

        // ETH is deposited into DelayedWETH, so game balance is 0.
        assertEq(address(game).balance, 0);

        // Check the claimData.
        (
            uint32 parentIndex_,
            ZKDisputeGame.ProposalStatus status_,
            address challenger_,
            address prover_,
            Timestamp deadline_,
            Claim claim_
        ) = game.claimData();

        assertEq(parentIndex_, parentGameIndex);
        assertEq(challenger_, address(0));
        assertEq(game.gameCreator(), proposer);
        assertEq(prover_, address(0));
        assertEq(claim_.raw(), rootClaim.raw());

        // Initially, the status is Unchallenged.
        assertEq(uint8(status_), uint8(ZKDisputeGame.ProposalStatus.Unchallenged));

        // The child's initial deadline is block.timestamp + maxChallengeDuration.
        uint256 currentTime = block.timestamp;
        uint256 expectedDeadline = currentTime + maxChallengeDuration.raw();
        assertEq(deadline_.raw(), expectedDeadline);
    }

    function test_initialize_blockNumberTooSmall_reverts() public {
        // Try to create a child game that references a block number smaller than parent's.
        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);

        // We expect revert because l2BlockNumber (1) < parent's block number
        vm.expectRevert(
            abi.encodeWithSelector(
                UnexpectedRootClaim.selector,
                Claim.wrap(keccak256("rootClaim")) // The rootClaim we pass.
            )
        );

        disputeGameFactory.create{ value: 1 ether }(
            gameType,
            rootClaim,
            abi.encodePacked(uint256(1), parentGameIndex) // L2 block is smaller than parent's block.
        );
        vm.stopPrank();
    }

    function testFuzz_initialize_blockNumberTooLarge_reverts(uint256 _l2SequenceNumber) public {
        _l2SequenceNumber = bound(_l2SequenceNumber, uint256(type(uint64).max) + 1, type(uint256).max);

        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);
        vm.expectRevert(abi.encodeWithSelector(UnexpectedRootClaim.selector, rootClaim));
        disputeGameFactory.create{ value: 1 ether }(
            gameType, rootClaim, abi.encodePacked(_l2SequenceNumber, parentGameIndex)
        );
        vm.stopPrank();
    }

    function test_initialize_parentBlacklisted_reverts() public {
        // Blacklist the game on the anchor state registry (which is what's actually used for validation).
        vm.prank(superchainConfig.guardian());
        anchorStateRegistry.blacklistDisputeGame(IDisputeGame(address(game)));

        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);
        vm.expectRevert(InvalidParentGame.selector);
        disputeGameFactory.create{ value: 1 ether }(
            gameType,
            Claim.wrap(keccak256("blacklisted-parent-game")),
            abi.encodePacked(childL2SequenceNumber + grandchildOffset1, childGameIndex)
        );
        vm.stopPrank();
    }

    function test_initialize_parentBlacklistedAfterCreation_reverts() public {
        // Create a new game which will be the parent.
        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);
        ZKDisputeGame parentNotRespected = ZKDisputeGame(
            payable(
                address(
                    disputeGameFactory.create{ value: 1 ether }(
                        gameType,
                        Claim.wrap(keccak256("not-respected-parent-game")),
                        abi.encodePacked(childL2SequenceNumber + grandchildOffset1, childGameIndex)
                    )
                )
            )
        );
        uint32 parentNotRespectedIndex = uint32(disputeGameFactory.gameCount() - 1);
        vm.stopPrank();

        // Blacklist the parent game to make it invalid.
        vm.prank(superchainConfig.guardian());
        anchorStateRegistry.blacklistDisputeGame(IDisputeGame(address(parentNotRespected)));

        // Try to create a game with a parent game that is not valid.
        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);
        vm.expectRevert(InvalidParentGame.selector);
        disputeGameFactory.create{ value: 1 ether }(
            gameType,
            Claim.wrap(keccak256("child-with-not-respected-parent")),
            abi.encodePacked(childL2SequenceNumber + grandchildOffset2, parentNotRespectedIndex)
        );
        vm.stopPrank();
    }

    function test_initialize_permissionless_succeeds() public {
        // Any address can propose (permissionless).
        address anyUser = address(0x9999);
        vm.startPrank(anyUser);
        vm.deal(anyUser, 1 ether);

        ZKDisputeGame newGame = ZKDisputeGame(
            payable(
                address(
                    disputeGameFactory.create{ value: 1 ether }(
                        gameType,
                        Claim.wrap(keccak256("permissionless-claim")),
                        abi.encodePacked(childL2SequenceNumber + grandchildOffset1, childGameIndex)
                    )
                )
            )
        );
        vm.stopPrank();

        assertEq(newGame.gameCreator(), anyUser);
    }

    function test_initialize_l2ChainIdZero_reverts() public {
        // Deploy a new game impl with l2ChainId = 0 in gameArgs
        IZKVerifier zkVerifier = IZKVerifier(address(new ZKMockVerifier()));
        address newImpl = address(new ZKDisputeGame());

        bytes memory gameArgs = abi.encodePacked(
            bytes32(0), // absolutePrestate
            zkVerifier, // verifier
            maxChallengeDuration, // maxChallengeDuration
            maxProveDuration, // maxProveDuration
            uint256(1 ether), // challengerBond
            anchorStateRegistry, // anchorStateRegistry
            delayedWeth, // weth
            uint256(0) // l2ChainId = 0
        );

        GameType zeroChainGameType = GameType.wrap(200);

        vm.prank(superchainConfig.guardian());
        anchorStateRegistry.setRespectedGameType(zeroChainGameType);

        vm.startPrank(disputeGameFactory.owner());
        disputeGameFactory.setImplementation(zeroChainGameType, IDisputeGame(newImpl), gameArgs);
        disputeGameFactory.setInitBond(zeroChainGameType, 1 ether);
        vm.stopPrank();

        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);
        vm.expectRevert(UnknownChainId.selector);
        disputeGameFactory.create{ value: 1 ether }(
            zeroChainGameType,
            Claim.wrap(keccak256("zero-chain-claim")),
            abi.encodePacked(parentL2SequenceNumber, type(uint32).max)
        );
        vm.stopPrank();
    }
}

/// @title ZKDisputeGame_Resolve_Test
/// @notice Tests for resolve functionality of ZKDisputeGame.
contract ZKDisputeGame_Resolve_Test is ZKDisputeGame_Init {
    function test_resolve_unchallenged_succeeds() public {
        assertEq(uint8(game.status()), uint8(GameStatus.IN_PROGRESS));

        // Should revert if we try to resolve before deadline.
        vm.expectRevert(GameNotOver.selector);
        game.resolve();

        // Warp forward past the challenge deadline.
        (,,,, Timestamp deadline,) = game.claimData();
        vm.warp(deadline.raw() + 1);

        // Expect the Resolved event.
        vm.expectEmit(true, false, false, false, address(game));
        emit Resolved(GameStatus.DEFENDER_WINS);

        // Now we can resolve successfully.
        game.resolve();

        // Proposer gets the bond back (two-phase).
        vm.warp(game.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);
        vm.prank(proposer);
        _claimCreditTwoPhase(game, proposer);

        // Check final state
        assertEq(uint8(game.status()), uint8(GameStatus.DEFENDER_WINS));
        // After withdrawal, game balance is 0.
        assertEq(address(game).balance, 0);
    }

    function test_resolve_unchallengedWithProof_succeeds() public {
        assertEq(uint8(game.status()), uint8(GameStatus.IN_PROGRESS));

        // Should revert if we try to resolve before the first challenge deadline.
        vm.expectRevert(GameNotOver.selector);
        game.resolve();

        // Prover proves the claim while unchallenged.
        vm.startPrank(prover);
        game.prove(bytes(""));
        vm.stopPrank();

        // Now the proposal is UnchallengedAndValidProofProvided; we can resolve immediately.
        game.resolve();

        // Prover does not get any credit.
        vm.warp(game.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);

        // Phase 1 of prover claim should unlock 0, then phase 2 should revert.
        game.claimCredit(prover); // unlock with 0 credit
        vm.warp(block.timestamp + delayedWeth.delay() + 1 seconds);
        vm.expectRevert(NoCreditToClaim.selector);
        game.claimCredit(prover);

        // Proposer gets the bond back (two-phase).
        _claimCreditTwoPhase(game, proposer);

        // Final status: DEFENDER_WINS.
        assertEq(uint8(game.status()), uint8(GameStatus.DEFENDER_WINS));
        assertEq(address(game).balance, 0);
    }

    function test_resolve_challengedWithProof_succeeds() public {
        assertEq(uint8(game.status()), uint8(GameStatus.IN_PROGRESS));

        // Try to resolve too early.
        vm.expectRevert(GameNotOver.selector);
        game.resolve();

        // Challenger posts the bond incorrectly.
        vm.startPrank(challenger);
        vm.deal(challenger, 2 ether);

        // Must pay exactly the required bond.
        vm.expectRevert(IncorrectBondAmount.selector);
        game.challenge{ value: 0.5 ether }();

        // Correctly challenge the game.
        game.challenge{ value: 1 ether }();
        vm.stopPrank();

        // Confirm the proposal is in Challenged state.
        (, ZKDisputeGame.ProposalStatus challStatus, address challenger_,,,) = game.claimData();
        assertEq(challenger_, challenger);
        assertEq(uint8(challStatus), uint8(ZKDisputeGame.ProposalStatus.Challenged));

        // Prover proves the claim in time.
        vm.startPrank(prover);
        game.prove(bytes(""));
        vm.stopPrank();

        // Confirm the proposal is now ChallengedAndValidProofProvided.
        (, challStatus,,,,) = game.claimData();
        assertEq(uint8(challStatus), uint8(ZKDisputeGame.ProposalStatus.ChallengedAndValidProofProvided));
        assertEq(uint8(game.status()), uint8(GameStatus.IN_PROGRESS));

        // Resolve the game.
        game.resolve();

        // Prover gets the proof reward (two-phase).
        vm.warp(game.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);
        _claimCreditTwoPhase(game, prover);

        // Proposer gets the bond back (two-phase).
        _claimCreditTwoPhase(game, proposer);

        assertEq(uint8(game.status()), uint8(GameStatus.DEFENDER_WINS));
        assertEq(address(game).balance, 0);

        // Final balances:
        // - The prover gets 1 ether reward (challenger's bond).
        // - The challenger gets nothing.
        assertEq(prover.balance, 1 ether);
        assertEq(challenger.balance, 1 ether); // started with 2, spent 1
    }

    function test_resolve_challengedNoProof_succeeds() public {
        // Challenge the game.
        vm.startPrank(challenger);
        vm.deal(challenger, 2 ether);
        game.challenge{ value: 1 ether }();
        vm.stopPrank();

        // We must wait for the prove deadline to pass.
        (,,,, Timestamp deadline,) = game.claimData();
        vm.warp(deadline.raw() + 1);

        // Now we can resolve, resulting in CHALLENGER_WINS.
        game.resolve();

        // Challenger gets the bond back and wins proposer's bond (two-phase).
        vm.warp(game.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);
        _claimCreditTwoPhase(game, challenger);

        assertEq(uint8(game.status()), uint8(GameStatus.CHALLENGER_WINS));

        // The challenger receives the entire 2 ether (proposer bond + challenger bond).
        assertEq(challenger.balance, 3 ether); // started with 2, spent 1, got 2 from the game.

        // The contract balance is zero.
        assertEq(address(game).balance, 0);
    }

    function test_resolve_parentGameInProgress_reverts() public {
        vm.startPrank(proposer);

        // Create a new game referencing the child game as parent.
        ZKDisputeGame childGame = ZKDisputeGame(
            payable(
                address(
                    disputeGameFactory.create{ value: 1 ether }(
                        gameType,
                        Claim.wrap(keccak256("new-claim")),
                        abi.encodePacked(childL2SequenceNumber + grandchildOffset1, childGameIndex)
                    )
                )
            )
        );

        vm.stopPrank();

        // The parent game is still in progress, not resolved.
        // So, if we try to resolve the childGame, it should revert with ParentGameNotResolved.
        vm.expectRevert(ParentGameNotResolved.selector);
        childGame.resolve();
    }

    function test_resolve_parentGameInvalid_succeeds() public {
        // 1) Now create a child game referencing that losing parent at index 1.
        vm.startPrank(proposer);
        ZKDisputeGame childGame = ZKDisputeGame(
            payable(
                address(
                    disputeGameFactory.create{ value: 1 ether }(
                        gameType,
                        Claim.wrap(keccak256("child-of-loser")),
                        abi.encodePacked(childL2SequenceNumber + grandchildOffset4, childGameIndex)
                    )
                )
            )
        );
        vm.stopPrank();

        // 2) Challenge the parent game so that it ends up CHALLENGER_WINS when proof is not provided within the prove
        // deadline.
        vm.startPrank(challenger);
        vm.deal(challenger, 2 ether);
        game.challenge{ value: 1 ether }();
        vm.stopPrank();

        // 3) Warp past the prove deadline.
        (,,,, Timestamp gameDeadline,) = game.claimData();
        vm.warp(gameDeadline.raw() + 1);

        // 4) The game resolves as CHALLENGER_WINS.
        game.resolve();

        // Challenger gets the bond back and wins proposer's bond (two-phase).
        vm.warp(game.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);
        _claimCreditTwoPhase(game, challenger);

        assertEq(uint8(game.status()), uint8(GameStatus.CHALLENGER_WINS));

        // 5) If we try to resolve the child game, it should be resolved as CHALLENGER_WINS
        // because parent's claim is invalid.
        // The child's bond is lost since there is no challenger for the child game.
        childGame.resolve();

        // Challenger hasn't challenged the child game, so it gets nothing.
        vm.warp(childGame.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);

        // Phase 1: unlock with 0 credit for challenger, phase 2: revert NoCreditToClaim
        childGame.claimCredit(challenger); // unlock 0
        vm.warp(block.timestamp + delayedWeth.delay() + 1 seconds);
        vm.expectRevert(NoCreditToClaim.selector);
        childGame.claimCredit(challenger);

        assertEq(uint8(childGame.status()), uint8(GameStatus.CHALLENGER_WINS));

        // Bond is in DelayedWETH, game ETH balance is 0.
        assertEq(address(childGame).balance, 0);
    }
}

/// @title ZKDisputeGame_Challenge_Test
/// @notice Tests for challenge functionality of ZKDisputeGame.
contract ZKDisputeGame_Challenge_Test is ZKDisputeGame_Init {
    function test_challenge_alreadyChallenged_reverts() public {
        // Initially unchallenged.
        (, ZKDisputeGame.ProposalStatus status_, address challenger_,,,) = game.claimData();
        assertEq(challenger_, address(0));
        assertEq(uint8(status_), uint8(ZKDisputeGame.ProposalStatus.Unchallenged));

        // The first challenge is valid.
        vm.startPrank(challenger);
        vm.deal(challenger, 2 ether);
        game.challenge{ value: 1 ether }();

        // A second challenge from any party should revert because the proposal is no longer "Unchallenged".
        vm.expectRevert(ClaimAlreadyChallenged.selector);
        game.challenge{ value: 1 ether }();
        vm.stopPrank();
    }

    function test_challenge_permissionless_succeeds() public {
        // Any address can challenge (permissionless).
        address anyChallenger = address(0x9999);
        vm.startPrank(anyChallenger);
        vm.deal(anyChallenger, 1 ether);

        game.challenge{ value: 1 ether }();
        vm.stopPrank();

        (,, address challenger_,,,) = game.claimData();
        assertEq(challenger_, anyChallenger);
    }
}

/// @title ZKDisputeGame_Prove_Test
/// @notice Tests for prove functionality of ZKDisputeGame.
contract ZKDisputeGame_Prove_Test is ZKDisputeGame_Init {
    function test_prove_afterDeadline_reverts() public {
        // Challenge first.
        vm.startPrank(challenger);
        vm.deal(challenger, 1 ether);
        game.challenge{ value: 1 ether }();
        vm.stopPrank();

        // Move time forward beyond the prove period.
        (,,,, Timestamp deadline,) = game.claimData();
        vm.warp(deadline.raw() + 1);

        vm.startPrank(prover);
        // Attempting to prove after the deadline is exceeded.
        vm.expectRevert(GameOver.selector);
        game.prove(bytes(""));
        vm.stopPrank();
    }

    function test_prove_alreadyProved_reverts() public {
        vm.startPrank(prover);
        game.prove(bytes(""));
        vm.expectRevert(GameOver.selector);
        game.prove(bytes(""));
        vm.stopPrank();
    }

    function test_prove_alreadyResolved_reverts() public {
        // Warp past the challenge deadline so the game is over.
        (,,,, Timestamp deadline,) = game.claimData();
        vm.warp(deadline.raw() + 1);

        // Resolve the game.
        game.resolve();

        // Attempting to prove after resolution should revert.
        vm.expectRevert(ClaimAlreadyResolved.selector);
        game.prove(bytes(""));
    }

    function test_prove_parentChallengerWins_reverts() public {
        // Create a child game referencing our game as parent.
        vm.startPrank(proposer);
        ZKDisputeGame childGame = ZKDisputeGame(
            payable(
                address(
                    disputeGameFactory.create{ value: 1 ether }(
                        gameType,
                        Claim.wrap(keccak256("child-claim")),
                        abi.encodePacked(childL2SequenceNumber + grandchildOffset1, childGameIndex)
                    )
                )
            )
        );
        vm.stopPrank();

        // Challenge the parent game so it resolves as CHALLENGER_WINS.
        vm.startPrank(challenger);
        vm.deal(challenger, 1 ether);
        game.challenge{ value: 1 ether }();
        vm.stopPrank();

        (,,,, Timestamp gameDeadline,) = game.claimData();
        vm.warp(gameDeadline.raw() + 1);
        game.resolve();

        // Attempting to prove the child game should revert because parent is invalid.
        vm.expectRevert(InvalidParentGame.selector);
        childGame.prove(bytes(""));
    }
}

/// @title ZKDisputeGame_ClaimCredit_Test
/// @notice Tests for claimCredit functionality of ZKDisputeGame.
contract ZKDisputeGame_ClaimCredit_Test is ZKDisputeGame_Init {
    function test_claimCredit_notFinalized_reverts() public {
        (,,,, Timestamp deadline,) = game.claimData();
        vm.warp(deadline.raw() + 1);
        game.resolve();

        // Game is resolved but not finalized - closeGame should revert with GameNotFinalized
        vm.expectRevert(GameNotFinalized.selector);
        game.claimCredit(proposer);
    }
}

/// @title ZKDisputeGame_CloseGame_Test
/// @notice Tests for closeGame functionality of ZKDisputeGame.
contract ZKDisputeGame_CloseGame_Test is ZKDisputeGame_Init {
    function test_closeGame_notResolved_reverts() public {
        // Game is not resolved, so closeGame should revert with GameNotResolved
        vm.expectRevert(GameNotResolved.selector);
        game.closeGame();
    }

    function test_closeGame_paused_reverts() public {
        // Resolve the game first
        (,,,, Timestamp deadline,) = game.claimData();
        vm.warp(deadline.raw() + 1);
        game.resolve();

        // Pause the system
        vm.prank(superchainConfig.guardian());
        superchainConfig.pause(address(0));

        // closeGame should revert with GamePaused
        vm.expectRevert(GamePaused.selector);
        game.closeGame();

        // Unpause
        vm.prank(superchainConfig.guardian());
        superchainConfig.unpause(address(0));
    }

    function test_closeGame_updatesAnchorGame_succeeds() public {
        (,,,, Timestamp deadline,) = game.claimData();
        vm.warp(deadline.raw() + 1);
        game.resolve();

        vm.warp(game.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);
        game.closeGame();

        assertEq(address(anchorStateRegistry.anchorGame()), address(game));
    }
}

// Import needed for the l2ChainId=0 test
import { ZKMockVerifier } from "test/dispute/zk/ZKMockVerifier.sol";
