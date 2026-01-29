// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { DisputeGameFactory_TestInit } from "test/dispute/DisputeGameFactory.t.sol";

// Libraries
import { Claim, Duration, GameStatus, GameType, Timestamp } from "src/dispute/lib/Types.sol";
import {
    BadAuth,
    IncorrectBondAmount,
    UnexpectedRootClaim,
    NoCreditToClaim,
    GameNotFinalized,
    ParentGameNotResolved,
    InvalidParentGame,
    ClaimAlreadyChallenged,
    GameOver,
    GameNotOver,
    IncorrectDisputeGameFactory
} from "src/dispute/lib/Errors.sol";
import { GameTypes } from "src/dispute/lib/Types.sol";

// Contracts
import { DisputeGameFactory } from "src/dispute/DisputeGameFactory.sol";
import { OptimisticZkGame } from "src/dispute/zk/OptimisticZkGame.sol";
import { AccessManager } from "src/dispute/zk/AccessManager.sol";

// Interfaces
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { Proxy } from "src/universal/Proxy.sol";

/// @title OptimisticZkGame_TestInit
/// @notice Base test contract with shared setup for OptimisticZkGame tests.
abstract contract OptimisticZkGame_TestInit is DisputeGameFactory_TestInit {
    // Events
    event Challenged(address indexed challenger);
    event Proved(address indexed prover);
    event Resolved(GameStatus indexed status);

    OptimisticZkGame gameImpl;
    OptimisticZkGame parentGame;
    OptimisticZkGame game;
    AccessManager accessManager;

    address proposer = address(0x123);
    address challenger = address(0x456);
    address prover = address(0x789);

    // Fixed parameters.
    GameType gameType = GameTypes.OPTIMISTIC_ZK_GAME_TYPE;
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

    // For a new parent game that we manipulate separately in some tests.
    OptimisticZkGame separateParentGame;

    function setUp() public virtual override {
        super.setUp();
        skipIfForkTest("Skip not supported yet");

        // Get anchor state to calculate valid sequence numbers
        (, anchorL2SequenceNumber) = anchorStateRegistry.getAnchorRoot();
        parentL2SequenceNumber = anchorL2SequenceNumber + parentSequenceOffset;
        childL2SequenceNumber = anchorL2SequenceNumber + childSequenceOffset;

        // Setup game implementation using shared helper
        address impl;
        (impl, accessManager,) = setupOptimisticZkGame(
            OptimisticZkGameParams({
                maxChallengeDuration: maxChallengeDuration,
                maxProveDuration: maxProveDuration,
                proposer: proposer,
                challenger: challenger,
                rollupConfigHash: bytes32(0),
                aggregationVkey: bytes32(0),
                rangeVkeyCommitment: bytes32(0),
                challengerBond: 1 ether
            })
        );
        gameImpl = OptimisticZkGame(impl);

        // Create the first (parent) game – it uses uint32.max as parent index.
        vm.startPrank(proposer);
        vm.deal(proposer, 2 ether);

        // Warp time forward to ensure the parent game is created after the respectedGameTypeUpdatedAt timestamp.
        vm.warp(block.timestamp + 1000);

        // Create parent game (uses uint32.max to indicate first game in chain).
        parentGame = OptimisticZkGame(
            address(
                disputeGameFactory.create{ value: 1 ether }(
                    gameType,
                    Claim.wrap(keccak256("genesis")),
                    abi.encodePacked(parentL2SequenceNumber, type(uint32).max)
                )
            )
        );

        // Record actual index of parent game (on fork, existing games already occupy indices 0, 1, ...)
        parentGameIndex = uint32(disputeGameFactory.gameCount() - 1);

        // We want the parent game to finalize. We'll skip its challenge period.
        (,,,,, Timestamp parentGameDeadline) = parentGame.claimData();
        vm.warp(parentGameDeadline.raw() + 1 seconds);
        parentGame.resolve();

        uint256 finalityDelay = anchorStateRegistry.disputeGameFinalityDelaySeconds();
        vm.warp(parentGame.resolvedAt().raw() + finalityDelay + 1 seconds);
        parentGame.claimCredit(proposer);

        // Create the child game referencing actual parent game index.
        game = OptimisticZkGame(
            address(
                disputeGameFactory.create{ value: 1 ether }(
                    gameType, rootClaim, abi.encodePacked(childL2SequenceNumber, parentGameIndex)
                )
            )
        );

        // Record actual index of child game.
        childGameIndex = uint32(disputeGameFactory.gameCount() - 1);

        vm.stopPrank();
    }
}

/// @title OptimisticZkGame_Initialize_Test
/// @notice Tests for initialization of OptimisticZkGame.
contract OptimisticZkGame_Initialize_Test is OptimisticZkGame_TestInit {
    function test_initialize_succeeds() public view {
        // Test that the factory is correctly initialized.
        assertEq(address(disputeGameFactory.owner()), address(this));
        assertEq(address(disputeGameFactory.gameImpls(gameType)), address(gameImpl));
        // We expect games including parent and child (indices may vary on fork).
        assertEq(disputeGameFactory.gameCount(), childGameIndex + 1);

        // Check that our child game matches the game at childGameIndex.
        (,, IDisputeGame proxy_) = disputeGameFactory.gameAtIndex(childGameIndex);
        assertEq(address(game), address(proxy_));

        // Check the child game fields.
        assertEq(game.gameType().raw(), gameType.raw());
        assertEq(game.rootClaim().raw(), rootClaim.raw());
        assertEq(game.maxChallengeDuration().raw(), maxChallengeDuration.raw());
        assertEq(game.maxProveDuration().raw(), maxProveDuration.raw());
        assertEq(address(game.disputeGameFactory()), address(disputeGameFactory));
        assertEq(game.l2SequenceNumber(), childL2SequenceNumber);

        // The parent's sequence number (startingBlockNumber() returns l2SequenceNumber).
        assertEq(game.startingBlockNumber(), parentL2SequenceNumber);

        // The parent's root was keccak256("genesis").
        assertEq(game.startingRootHash().raw(), keccak256("genesis"));

        assertEq(address(game).balance, 1 ether);

        // Check the claimData.
        (
            uint32 parentIndex_,
            address counteredBy_,
            address prover_,
            Claim claim_,
            OptimisticZkGame.ProposalStatus status_,
            Timestamp deadline_
        ) = game.claimData();

        assertEq(parentIndex_, parentGameIndex);
        assertEq(counteredBy_, address(0));
        assertEq(game.gameCreator(), proposer);
        assertEq(prover_, address(0));
        assertEq(claim_.raw(), rootClaim.raw());

        // Initially, the status is Unchallenged.
        assertEq(uint8(status_), uint8(OptimisticZkGame.ProposalStatus.Unchallenged));

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

    function test_initialize_parentNotRespected_reverts() public {
        // Create a new game which will be the parent.
        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);
        OptimisticZkGame parentNotRespected = OptimisticZkGame(
            address(
                disputeGameFactory.create{ value: 1 ether }(
                    gameType,
                    Claim.wrap(keccak256("not-respected-parent-game")),
                    abi.encodePacked(childL2SequenceNumber + grandchildOffset1, childGameIndex)
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

    function test_initialize_noPermission_reverts() public {
        address maliciousProposer = address(0x1234);

        vm.startPrank(maliciousProposer);
        vm.deal(maliciousProposer, 1 ether);

        vm.expectRevert(BadAuth.selector);
        disputeGameFactory.create{ value: 1 ether }(
            gameType,
            Claim.wrap(keccak256("new-claim")),
            abi.encodePacked(childL2SequenceNumber + grandchildOffset1, childGameIndex)
        );

        vm.stopPrank();
    }

    function test_initialize_wrongFactory_reverts() public {
        // Deploy the implementation contract for new DisputeGameFactory.
        DisputeGameFactory newFactoryImpl = new DisputeGameFactory();

        // Deploy a proxy pointing to the new factory implementation.
        Proxy newFactoryProxyContract = new Proxy(address(this));
        newFactoryProxyContract.upgradeTo(address(newFactoryImpl));

        // Cast the proxy to the DisputeGameFactory interface and initialize it.
        DisputeGameFactory newFactory = DisputeGameFactory(address(newFactoryProxyContract));
        newFactory.initialize(address(this));

        // Set the implementation with the same implementation as the old disputeGameFactory.
        newFactory.setImplementation(gameType, IDisputeGame(address(gameImpl)));
        newFactory.setInitBond(gameType, 1 ether);

        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);

        vm.expectRevert(IncorrectDisputeGameFactory.selector);
        newFactory.create{ value: 1 ether }(
            gameType,
            Claim.wrap(keccak256("new-claim")),
            abi.encodePacked(childL2SequenceNumber + grandchildOffset1, childGameIndex)
        );

        vm.stopPrank();
    }
}

/// @title OptimisticZkGame_Resolve_Test
/// @notice Tests for resolve functionality of OptimisticZkGame.
contract OptimisticZkGame_Resolve_Test is OptimisticZkGame_TestInit {
    function test_resolve_unchallenged_succeeds() public {
        assertEq(uint8(game.status()), uint8(GameStatus.IN_PROGRESS));

        // Should revert if we try to resolve before deadline.
        vm.expectRevert(GameNotOver.selector);
        game.resolve();

        // Warp forward past the challenge deadline.
        (,,,,, Timestamp deadline) = game.claimData();
        vm.warp(deadline.raw() + 1);

        // Expect the Resolved event.
        vm.expectEmit(true, false, false, false, address(game));
        emit Resolved(GameStatus.DEFENDER_WINS);

        // Now we can resolve successfully.
        game.resolve();

        // Proposer gets the bond back.
        vm.warp(game.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);
        game.claimCredit(proposer);

        // Check final state
        assertEq(uint8(game.status()), uint8(GameStatus.DEFENDER_WINS));
        // The contract should have paid back the proposer.
        assertEq(address(game).balance, 0);
        // Proposer posted 1 ether, so they get it back.
        assertEq(proposer.balance, 2 ether);
        assertEq(challenger.balance, 0);
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
        vm.expectRevert(NoCreditToClaim.selector);
        game.claimCredit(prover);

        // Proposer gets the bond back.
        game.claimCredit(proposer);

        // Final status: DEFENDER_WINS.
        assertEq(uint8(game.status()), uint8(GameStatus.DEFENDER_WINS));
        assertEq(address(game).balance, 0);

        // Proposer gets their 1 ether back.
        assertEq(proposer.balance, 2 ether);
        // Prover does NOT get the reward because no challenger posted a bond.
        assertEq(prover.balance, 0 ether);
        assertEq(challenger.balance, 0);
    }

    function test_resolve_challengedWithProof_succeeds() public {
        assertEq(uint8(game.status()), uint8(GameStatus.IN_PROGRESS));
        assertEq(address(game).balance, 1 ether);

        // Try to resolve too early.
        vm.expectRevert(GameNotOver.selector);
        game.resolve();

        // Challenger posts the bond incorrectly.
        vm.startPrank(challenger);
        vm.deal(challenger, 1 ether);

        // Must pay exactly the required bond.
        vm.expectRevert(IncorrectBondAmount.selector);
        game.challenge{ value: 0.5 ether }();

        // Correctly challenge the game.
        game.challenge{ value: 1 ether }();
        vm.stopPrank();

        // Now the contract holds 2 ether total.
        assertEq(address(game).balance, 2 ether);

        // Confirm the proposal is in Challenged state.
        (, address counteredBy_,,, OptimisticZkGame.ProposalStatus challStatus,) = game.claimData();
        assertEq(counteredBy_, challenger);
        assertEq(uint8(challStatus), uint8(OptimisticZkGame.ProposalStatus.Challenged));

        // Prover proves the claim in time.
        vm.startPrank(prover);
        game.prove(bytes(""));
        vm.stopPrank();

        // Confirm the proposal is now ChallengedAndValidProofProvided.
        (,,,, challStatus,) = game.claimData();
        assertEq(uint8(challStatus), uint8(OptimisticZkGame.ProposalStatus.ChallengedAndValidProofProvided));
        assertEq(uint8(game.status()), uint8(GameStatus.IN_PROGRESS));

        // Resolve the game.
        game.resolve();

        // Prover gets the proof reward.
        vm.warp(game.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);
        game.claimCredit(prover);

        // Proposer gets the bond back.
        game.claimCredit(proposer);

        assertEq(uint8(game.status()), uint8(GameStatus.DEFENDER_WINS));
        assertEq(address(game).balance, 0);

        // Final balances:
        // - The proposer recovers their 1 ether stake.
        // - The prover gets 1 ether reward.
        // - The challenger gets nothing.
        assertEq(proposer.balance, 2 ether);
        assertEq(prover.balance, 1 ether);
        assertEq(challenger.balance, 0);
    }

    function test_resolve_challengedNoProof_succeeds() public {
        // Challenge the game.
        vm.startPrank(challenger);
        vm.deal(challenger, 2 ether);
        game.challenge{ value: 1 ether }();
        vm.stopPrank();

        // The contract now has 2 ether total.
        assertEq(address(game).balance, 2 ether);

        // We must wait for the prove deadline to pass.
        (,,,,, Timestamp deadline) = game.claimData();
        vm.warp(deadline.raw() + 1);

        // Now we can resolve, resulting in CHALLENGER_WINS.
        game.resolve();

        // Challenger gets the bond back and wins proposer's bond.
        vm.warp(game.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);
        game.claimCredit(challenger);

        assertEq(uint8(game.status()), uint8(GameStatus.CHALLENGER_WINS));

        // The challenger receives the entire 3 ether.
        assertEq(challenger.balance, 3 ether); // started with 2, spent 1, got 2 from the game.

        // The proposer loses their 1 ether stake.
        assertEq(proposer.balance, 1 ether); // started with 2, lost 1.
        // The contract balance is zero.
        assertEq(address(game).balance, 0);
    }

    function test_resolve_parentGameInProgress_reverts() public {
        vm.startPrank(proposer);

        // Create a new game referencing the child game as parent.
        OptimisticZkGame childGame = OptimisticZkGame(
            address(
                disputeGameFactory.create{ value: 1 ether }(
                    gameType,
                    Claim.wrap(keccak256("new-claim")),
                    abi.encodePacked(childL2SequenceNumber + grandchildOffset1, childGameIndex)
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
        OptimisticZkGame childGame = OptimisticZkGame(
            address(
                disputeGameFactory.create{ value: 1 ether }(
                    gameType,
                    Claim.wrap(keccak256("child-of-loser")),
                    abi.encodePacked(childL2SequenceNumber + grandchildOffset4, childGameIndex)
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
        (,,,,, Timestamp gameDeadline) = game.claimData();
        vm.warp(gameDeadline.raw() + 1);

        // 4) The game resolves as CHALLENGER_WINS.
        game.resolve();

        // Challenger gets the bond back and wins proposer's bond.
        vm.warp(game.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);
        game.claimCredit(challenger);

        assertEq(uint8(game.status()), uint8(GameStatus.CHALLENGER_WINS));

        // 5) If we try to resolve the child game, it should be resolved as CHALLENGER_WINS
        // because parent's claim is invalid.
        // The child's bond is lost since there is no challenger for the child game.
        childGame.resolve();

        // Challenger hasn't challenged the child game, so it gets nothing.
        vm.warp(childGame.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);

        vm.expectRevert(NoCreditToClaim.selector);
        childGame.claimCredit(challenger);

        assertEq(uint8(childGame.status()), uint8(GameStatus.CHALLENGER_WINS));

        assertEq(address(childGame).balance, 1 ether);
        assertEq(address(challenger).balance, 3 ether);
        assertEq(address(proposer).balance, 0 ether);
    }
}

/// @title OptimisticZkGame_Challenge_Test
/// @notice Tests for challenge functionality of OptimisticZkGame.
contract OptimisticZkGame_Challenge_Test is OptimisticZkGame_TestInit {
    function test_challenge_alreadyChallenged_reverts() public {
        // Initially unchallenged.
        (, address counteredBy_,,, OptimisticZkGame.ProposalStatus status_,) = game.claimData();
        assertEq(counteredBy_, address(0));
        assertEq(uint8(status_), uint8(OptimisticZkGame.ProposalStatus.Unchallenged));

        // The first challenge is valid.
        vm.startPrank(challenger);
        vm.deal(challenger, 2 ether);
        game.challenge{ value: 1 ether }();

        // A second challenge from any party should revert because the proposal is no longer "Unchallenged".
        vm.expectRevert(ClaimAlreadyChallenged.selector);
        game.challenge{ value: 1 ether }();
        vm.stopPrank();
    }

    function test_challenge_noPermission_reverts() public {
        address maliciousChallenger = address(0x1234);

        vm.startPrank(maliciousChallenger);
        vm.deal(maliciousChallenger, 1 ether);

        vm.expectRevert(BadAuth.selector);
        game.challenge{ value: 1 ether }();

        vm.stopPrank();
    }
}

/// @title OptimisticZkGame_Prove_Test
/// @notice Tests for prove functionality of OptimisticZkGame.
contract OptimisticZkGame_Prove_Test is OptimisticZkGame_TestInit {
    function test_prove_afterDeadline_reverts() public {
        // Challenge first.
        vm.startPrank(challenger);
        vm.deal(challenger, 1 ether);
        game.challenge{ value: 1 ether }();
        vm.stopPrank();

        // Move time forward beyond the prove period.
        (,,,,, Timestamp deadline) = game.claimData();
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
}

/// @title OptimisticZkGame_ClaimCredit_Test
/// @notice Tests for claimCredit functionality of OptimisticZkGame.
contract OptimisticZkGame_ClaimCredit_Test is OptimisticZkGame_TestInit {
    function test_claimCredit_notFinalized_reverts() public {
        (,,,,, Timestamp deadline) = game.claimData();
        vm.warp(deadline.raw() + 1);
        game.resolve();

        vm.expectRevert(GameNotFinalized.selector);
        game.claimCredit(proposer);
    }
}

/// @title OptimisticZkGame_CloseGame_Test
/// @notice Tests for closeGame functionality of OptimisticZkGame.
contract OptimisticZkGame_CloseGame_Test is OptimisticZkGame_TestInit {
    function test_closeGame_notResolved_reverts() public {
        vm.expectRevert(GameNotFinalized.selector);
        game.closeGame();
    }

    function test_closeGame_updatesAnchorGame_succeeds() public {
        (,,,,, Timestamp deadline) = game.claimData();
        vm.warp(deadline.raw() + 1);
        game.resolve();

        vm.warp(game.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);
        game.closeGame();

        assertEq(address(anchorStateRegistry.anchorGame()), address(game));
    }
}

/// @title OptimisticZkGame_AccessManager_Test
/// @notice Tests for AccessManager permissionless fallback functionality.
contract OptimisticZkGame_AccessManager_Test is OptimisticZkGame_TestInit {
    function test_accessManager_permissionlessAfterTimeout_succeeds() public {
        // Initially, unauthorized user should not be allowed
        address unauthorizedUser = address(0x9999);

        // Try to create a game as unauthorized user - should fail
        vm.prank(unauthorizedUser);
        vm.deal(unauthorizedUser, 1 ether);
        vm.expectRevert(BadAuth.selector);
        disputeGameFactory.create{ value: 1 ether }(
            gameType,
            Claim.wrap(keccak256("new-claim-1")),
            abi.encodePacked(childL2SequenceNumber + grandchildOffset1, childGameIndex)
        );

        vm.prank(proposer);
        vm.deal(proposer, 1 ether);
        disputeGameFactory.create{ value: 1 ether }(
            gameType, Claim.wrap(keccak256("new-claim-2")), abi.encodePacked(childL2SequenceNumber, parentGameIndex)
        );

        // Warp time forward past the timeout
        vm.warp(block.timestamp + 2 weeks + 1);

        // Now unauthorized user should be allowed due to timeout
        vm.prank(unauthorizedUser);
        vm.deal(unauthorizedUser, 1 ether);
        disputeGameFactory.create{ value: 1 ether }(
            gameType,
            Claim.wrap(keccak256("new-claim-3")),
            abi.encodePacked(childL2SequenceNumber + grandchildOffset2, childGameIndex)
        );

        // After the new game, timeout resets - unauthorized user should not be allowed immediately
        vm.prank(unauthorizedUser);
        vm.deal(unauthorizedUser, 1 ether);
        vm.expectRevert(BadAuth.selector);
        disputeGameFactory.create{ value: 1 ether }(
            gameType,
            Claim.wrap(keccak256("new-claim-4")),
            abi.encodePacked(childL2SequenceNumber + grandchildOffset3, childGameIndex)
        );
    }

    function test_accessManager_permissionlessNoGamesAfterTimeout_succeeds() public {
        // Initially, unauthorized user should not be allowed
        address unauthorizedUser = address(0x9999);

        // Try to create a game as unauthorized user - should fail
        vm.prank(unauthorizedUser);
        vm.deal(unauthorizedUser, 1 ether);
        vm.expectRevert(BadAuth.selector);
        disputeGameFactory.create{ value: 1 ether }(
            gameType,
            Claim.wrap(keccak256("new-claim-1")),
            abi.encodePacked(childL2SequenceNumber + grandchildOffset1, childGameIndex)
        );

        // Warp time forward past the timeout
        vm.warp(block.timestamp + 2 weeks + 1 hours);

        // Now unauthorized user should be allowed due to timeout
        vm.prank(unauthorizedUser);
        vm.deal(unauthorizedUser, 1 ether);
        disputeGameFactory.create{ value: 1 ether }(
            gameType,
            Claim.wrap(keccak256("new-claim-3")),
            abi.encodePacked(childL2SequenceNumber + grandchildOffset2, childGameIndex)
        );

        // After the new game, timeout resets - unauthorized user should not be allowed immediately
        vm.prank(unauthorizedUser);
        vm.deal(unauthorizedUser, 1 ether);
        vm.expectRevert(BadAuth.selector);
        disputeGameFactory.create{ value: 1 ether }(
            gameType,
            Claim.wrap(keccak256("new-claim-4")),
            abi.encodePacked(childL2SequenceNumber + grandchildOffset3, childGameIndex)
        );
    }
}
