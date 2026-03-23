// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { DisputeGameFactory_TestInit } from "test/dispute/DisputeGameFactory.t.sol";

// Libraries
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { BondDistributionMode, Claim, Duration, GameStatus, GameType, Hash, Timestamp } from "src/dispute/lib/Types.sol";
import {
    IncorrectBondAmount,
    UnexpectedRootClaim,
    UnexpectedGameType,
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
    UnknownChainId,
    BondTransferFailed,
    AlreadyInitialized
} from "src/dispute/lib/Errors.sol";
import { GameTypes } from "src/dispute/lib/Types.sol";

// Contracts
import { ZKDisputeGame } from "src/dispute/zk/ZKDisputeGame.sol";

// Interfaces
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IZKVerifier } from "interfaces/dispute/zk/IZKVerifier.sol";

/// @title ZKDisputeGame_TestInit
/// @notice Base test contract with shared setup for ZKDisputeGame tests.
abstract contract ZKDisputeGame_TestInit is DisputeGameFactory_TestInit {
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

        // Create the child game before claiming parent credit, because claimCredit triggers
        // closeGame() which advances the anchor to parentL2SequenceNumber. After that, the
        // parent's seq num would equal the anchor, violating the "strictly above" invariant.
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
contract ZKDisputeGame_Initialize_Test is ZKDisputeGame_TestInit {
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
        assertEq(game.parentIndex(), parentGameIndex);
        assertEq(game.absolutePrestate(), bytes32(0));
        assertTrue(address(game.verifier()) != address(0));
        assertEq(game.challengerBond(), 1 ether);
        assertTrue(game.l1Head().raw() != bytes32(0));
        assertEq(game.rootClaimByChainId(l2ChainId).raw(), rootClaim.raw());

        // The game was created while its game type was respected.
        assertTrue(game.wasRespectedGameTypeWhenCreated());

        // extraData is 36 bytes: l2SequenceNumber (32) + parentIndex (4).
        bytes memory extra = game.extraData();
        assertEq(extra.length, 36);

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

    function test_initialize_parentRetired_reverts() public {
        // Retire all existing games by updating the retirement timestamp.
        vm.prank(superchainConfig.guardian());
        anchorStateRegistry.updateRetirementTimestamp();

        // Try to create a new game referencing the (now retired) child game as parent.
        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);
        vm.expectRevert(InvalidParentGame.selector);
        disputeGameFactory.create{ value: 1 ether }(
            gameType,
            Claim.wrap(keccak256("child-of-retired-parent")),
            abi.encodePacked(childL2SequenceNumber + grandchildOffset1, childGameIndex)
        );
        vm.stopPrank();
    }

    function test_initialize_parentChallengerWins_reverts() public {
        // Challenge the child game (our `game`) and let it expire without proof.
        vm.startPrank(challenger);
        vm.deal(challenger, 1 ether);
        game.challenge{ value: 1 ether }();
        vm.stopPrank();

        (,,,, Timestamp deadline,) = game.claimData();
        vm.warp(deadline.raw() + 1);
        game.resolve();
        assertEq(uint8(game.status()), uint8(GameStatus.CHALLENGER_WINS));

        // Trying to create a new game referencing the invalidated game should revert.
        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);
        vm.expectRevert(InvalidParentGame.selector);
        disputeGameFactory.create{ value: 1 ether }(
            gameType,
            Claim.wrap(keccak256("child-of-challenger-wins")),
            abi.encodePacked(childL2SequenceNumber + grandchildOffset1, childGameIndex)
        );
        vm.stopPrank();
    }

    function test_initialize_parentDifferentGameType_reverts() public {
        // Deploy a second ZK game impl with a different GameType id.
        IZKVerifier zkVerifier = IZKVerifier(address(new ZKMockVerifier()));
        address newImpl = address(new ZKDisputeGame());
        GameType differentGameType = GameType.wrap(201);

        bytes memory gameArgs = abi.encodePacked(
            bytes32(0),
            zkVerifier,
            maxChallengeDuration,
            maxProveDuration,
            uint256(1 ether),
            anchorStateRegistry,
            delayedWeth,
            l2ChainId
        );

        vm.prank(superchainConfig.guardian());
        anchorStateRegistry.setRespectedGameType(differentGameType);

        vm.startPrank(disputeGameFactory.owner());
        disputeGameFactory.setImplementation(differentGameType, IDisputeGame(newImpl), gameArgs);
        disputeGameFactory.setInitBond(differentGameType, 1 ether);
        vm.stopPrank();

        // Try to create a game of differentGameType referencing childGameIndex (which is gameType).
        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);
        vm.expectRevert(UnexpectedGameType.selector);
        disputeGameFactory.create{ value: 1 ether }(
            differentGameType,
            Claim.wrap(keccak256("different-type-claim")),
            abi.encodePacked(childL2SequenceNumber + grandchildOffset1, childGameIndex)
        );
        vm.stopPrank();
    }

    function test_initialize_parentBelowAnchor_reverts() public {
        // Resolve and finalize the child game so it becomes the new anchor.
        (,,,, Timestamp deadline,) = game.claimData();
        vm.warp(deadline.raw() + 1);
        game.resolve();

        uint256 finalityDelay = anchorStateRegistry.disputeGameFinalityDelaySeconds();
        vm.warp(game.resolvedAt().raw() + finalityDelay + 1 seconds);
        game.closeGame();

        // Now the anchor is at childL2SequenceNumber, which is above the parent's sequence number.
        // Trying to create a new game referencing the parent (whose seq num < anchor) should revert.
        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);
        vm.expectRevert(InvalidParentGame.selector);
        disputeGameFactory.create{ value: 1 ether }(
            gameType,
            Claim.wrap(keccak256("below-anchor-claim")),
            abi.encodePacked(childL2SequenceNumber + grandchildOffset1, parentGameIndex)
        );
        vm.stopPrank();
    }

    function test_initialize_parentEqualAnchor_reverts() public {
        // Resolve and finalize the child game so it becomes the new anchor.
        (,,,, Timestamp deadline,) = game.claimData();
        vm.warp(deadline.raw() + 1);
        game.resolve();

        uint256 finalityDelay = anchorStateRegistry.disputeGameFinalityDelaySeconds();
        vm.warp(game.resolvedAt().raw() + finalityDelay + 1 seconds);
        game.closeGame();

        // Anchor is now at childL2SequenceNumber.
        // The child game's l2SequenceNumber == anchor, so using it as parent should revert
        // because parent seq num must be strictly above anchor.
        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);
        vm.expectRevert(InvalidParentGame.selector);
        disputeGameFactory.create{ value: 1 ether }(
            gameType,
            Claim.wrap(keccak256("equal-anchor-claim")),
            abi.encodePacked(childL2SequenceNumber + grandchildOffset1, childGameIndex)
        );
        vm.stopPrank();
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

    function test_initialize_fromAnchorState_succeeds() public {
        // Create a first game (parentIndex = uint32.max) and verify it uses anchor state values.
        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);

        uint256 seqNum = anchorL2SequenceNumber + 5000;
        ZKDisputeGame anchorGame = ZKDisputeGame(
            payable(
                address(
                    disputeGameFactory.create{ value: 1 ether }(
                        gameType,
                        Claim.wrap(keccak256("anchor-start-claim")),
                        abi.encodePacked(seqNum, type(uint32).max)
                    )
                )
            )
        );
        vm.stopPrank();

        // The starting proposal should match the anchor state values.
        (Hash anchorRoot, uint256 anchorSeqNum) = anchorStateRegistry.getAnchorRoot();
        assertEq(anchorGame.startingRootHash().raw(), anchorRoot.raw());
        assertEq(anchorGame.startingBlockNumber(), anchorSeqNum);
        assertEq(anchorGame.parentIndex(), type(uint32).max);
    }

    function test_initialize_alreadyInitialized_reverts() public {
        // The game is already initialized in setUp. Calling initialize again should revert.
        vm.expectRevert(AlreadyInitialized.selector);
        game.initialize{ value: 1 ether }();
    }
}

/// @title ZKDisputeGame_Challenge_Test
/// @notice Tests for challenge functionality of ZKDisputeGame.
contract ZKDisputeGame_Challenge_Test is ZKDisputeGame_TestInit {
    function test_challenge_permissionless_succeeds() public {
        // Record deadline before challenge.
        (,,,, Timestamp deadlineBefore,) = game.claimData();

        // Any address can challenge (permissionless).
        address anyChallenger = address(0x9999);
        vm.startPrank(anyChallenger);
        vm.deal(anyChallenger, 1 ether);

        game.challenge{ value: 1 ether }();
        vm.stopPrank();

        (,, address challenger_,, Timestamp deadlineAfter,) = game.claimData();
        assertEq(challenger_, anyChallenger);

        // The deadline should be reset to block.timestamp + maxProveDuration after challenge.
        assertEq(deadlineAfter.raw(), block.timestamp + maxProveDuration.raw());
        assertTrue(deadlineAfter.raw() != deadlineBefore.raw());
    }

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

    function test_challenge_afterDeadline_reverts() public {
        // Warp past the challenge deadline so the game is over.
        (,,,, Timestamp deadline,) = game.claimData();
        vm.warp(deadline.raw() + 1);

        vm.startPrank(challenger);
        vm.deal(challenger, 1 ether);
        vm.expectRevert(GameOver.selector);
        game.challenge{ value: 1 ether }();
        vm.stopPrank();
    }

    function test_challenge_afterProve_reverts() public {
        // Prove the game so it is over (prover != address(0)).
        vm.prank(prover);
        game.prove(bytes(""));

        vm.startPrank(challenger);
        vm.deal(challenger, 1 ether);
        vm.expectRevert(GameOver.selector);
        game.challenge{ value: 1 ether }();
        vm.stopPrank();
    }
}

/// @title ZKDisputeGame_Prove_Test
/// @notice Tests for prove functionality of ZKDisputeGame.
contract ZKDisputeGame_Prove_Test is ZKDisputeGame_TestInit {
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

    function test_prove_emitsProvedEvent_succeeds() public {
        vm.expectEmit(true, false, false, false, address(game));
        emit Proved(prover);

        vm.prank(prover);
        game.prove(bytes(""));
    }

    function test_prove_invalidProof_reverts() public {
        // Deploy a rejecting verifier and create a game that uses it.
        ZKRejectingVerifier rejectingVerifier = new ZKRejectingVerifier();
        address newImpl = address(new ZKDisputeGame());
        GameType rejectGameType = GameType.wrap(202);

        bytes memory gameArgs = abi.encodePacked(
            bytes32(0),
            rejectingVerifier,
            maxChallengeDuration,
            maxProveDuration,
            uint256(1 ether),
            anchorStateRegistry,
            delayedWeth,
            l2ChainId
        );

        vm.prank(superchainConfig.guardian());
        anchorStateRegistry.setRespectedGameType(rejectGameType);

        vm.startPrank(disputeGameFactory.owner());
        disputeGameFactory.setImplementation(rejectGameType, IDisputeGame(newImpl), gameArgs);
        disputeGameFactory.setInitBond(rejectGameType, 1 ether);
        vm.stopPrank();

        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);
        ZKDisputeGame rejectGame = ZKDisputeGame(
            payable(
                address(
                    disputeGameFactory.create{ value: 1 ether }(
                        rejectGameType,
                        Claim.wrap(keccak256("reject-claim")),
                        abi.encodePacked(parentL2SequenceNumber, type(uint32).max)
                    )
                )
            )
        );
        vm.stopPrank();

        // Proving should revert because the verifier always rejects.
        vm.expectRevert("ZKRejectingVerifier: invalid proof");
        vm.prank(prover);
        rejectGame.prove(bytes(""));
    }
}

/// @title ZKDisputeGame_Resolve_Test
/// @notice Tests for resolve functionality of ZKDisputeGame.
contract ZKDisputeGame_Resolve_Test is ZKDisputeGame_TestInit {
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

        // Prover does not get any credit. First call closes the game and returns without
        // reverting; second call reverts with NoCreditToClaim.
        vm.warp(game.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);
        game.claimCredit(prover);
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

    function test_resolve_challengedProverIsCreator_succeeds() public {
        // Challenge the game.
        vm.startPrank(challenger);
        vm.deal(challenger, 1 ether);
        game.challenge{ value: 1 ether }();
        vm.stopPrank();

        // Proposer (game creator) proves their own claim.
        vm.prank(proposer);
        game.prove(bytes(""));

        // Resolve the game.
        game.resolve();

        // Proposer should get the entire totalBonds (2 ether: 1 proposer bond + 1 challenger bond).
        vm.warp(game.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);
        uint256 balanceBefore = proposer.balance;
        _claimCreditTwoPhase(game, proposer);

        assertEq(proposer.balance, balanceBefore + 2 ether);
        assertEq(uint8(game.status()), uint8(GameStatus.DEFENDER_WINS));

        // Challenger gets nothing. Game already closed, so second call reverts.
        vm.expectRevert(NoCreditToClaim.selector);
        game.claimCredit(challenger);
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

    function test_resolve_alreadyResolved_reverts() public {
        // Warp past deadline and resolve.
        (,,,, Timestamp deadline,) = game.claimData();
        vm.warp(deadline.raw() + 1);
        game.resolve();

        // Second resolve should revert.
        vm.expectRevert(ClaimAlreadyResolved.selector);
        game.resolve();
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

        // Challenger hasn't challenged the child game, so it gets nothing. First call closes
        // the game and returns without reverting; second call reverts with NoCreditToClaim.
        vm.warp(childGame.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);
        childGame.claimCredit(challenger);
        vm.expectRevert(NoCreditToClaim.selector);
        childGame.claimCredit(challenger);

        assertEq(uint8(childGame.status()), uint8(GameStatus.CHALLENGER_WINS));

        // Bond is in DelayedWETH, game ETH balance is 0.
        assertEq(address(childGame).balance, 0);
    }

    function test_resolve_parentInvalidChildChallenged_succeeds() public {
        // Create a child game referencing our game as parent.
        vm.startPrank(proposer);
        ZKDisputeGame childGame = ZKDisputeGame(
            payable(
                address(
                    disputeGameFactory.create{ value: 1 ether }(
                        gameType,
                        Claim.wrap(keccak256("child-of-invalid-parent")),
                        abi.encodePacked(childL2SequenceNumber + grandchildOffset4, childGameIndex)
                    )
                )
            )
        );
        vm.stopPrank();

        // Challenge the child game.
        vm.startPrank(challenger);
        vm.deal(challenger, 2 ether);
        childGame.challenge{ value: 1 ether }();
        vm.stopPrank();

        // Make the parent game invalid: challenge it and let it expire without proof.
        vm.startPrank(address(0xABC));
        vm.deal(address(0xABC), 1 ether);
        game.challenge{ value: 1 ether }();
        vm.stopPrank();

        (,,,, Timestamp gameDeadline,) = game.claimData();
        vm.warp(gameDeadline.raw() + 1);
        game.resolve();
        assertEq(uint8(game.status()), uint8(GameStatus.CHALLENGER_WINS));

        // Resolve the child game - parent is invalid so challenger wins everything.
        childGame.resolve();
        assertEq(uint8(childGame.status()), uint8(GameStatus.CHALLENGER_WINS));

        // Challenger should get totalBonds (2 ether: proposer bond + challenger bond).
        vm.warp(childGame.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);
        uint256 balanceBefore = challenger.balance;
        _claimCreditTwoPhase(childGame, challenger);
        assertEq(challenger.balance, balanceBefore + 2 ether);
    }
}

/// @title ZKDisputeGame_ClaimCredit_Test
/// @notice Tests for claimCredit functionality of ZKDisputeGame.
contract ZKDisputeGame_ClaimCredit_Test is ZKDisputeGame_TestInit {
    /// @notice Phase 1: claimCredit zeros the credit mapping and unlocks in DelayedWETH,
    ///         then returns early. The recipient's on-chain credit is zeroed immediately.
    function test_claimCredit_phaseOne_succeeds() public {
        // Resolve and finalize the game so it can be closed.
        (,,,, Timestamp deadline,) = game.claimData();
        vm.warp(deadline.raw() + 1);
        game.resolve();
        vm.warp(game.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);

        // Before phase 1: proposer has non-zero refund credit.
        assertGt(game.refundModeCredit(proposer), 0);

        // Phase 1: credit is zeroed and unlock is queued in DelayedWETH.
        game.claimCredit(proposer);

        // After phase 1: on-chain credit mappings are zeroed.
        assertEq(game.refundModeCredit(proposer), 0);
        assertEq(game.normalModeCredit(proposer), 0);

        // A pending withdrawal should exist in DelayedWETH.
        (uint256 amount,) = delayedWeth.withdrawals(address(game), proposer);
        assertGt(amount, 0);
    }

    /// @notice Phase 2: after the DelayedWETH delay, claimCredit withdraws and transfers ETH.
    function test_claimCredit_phaseTwo_succeeds() public {
        // Resolve and finalize the game.
        (,,,, Timestamp deadline,) = game.claimData();
        vm.warp(deadline.raw() + 1);
        game.resolve();
        vm.warp(game.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);

        uint256 expectedCredit = game.refundModeCredit(proposer);

        // Phase 1: unlock.
        game.claimCredit(proposer);

        // Warp past DelayedWETH delay.
        vm.warp(block.timestamp + delayedWeth.delay() + 1 seconds);

        uint256 balanceBefore = proposer.balance;

        // Phase 2: withdraw and transfer.
        game.claimCredit(proposer);

        // Proposer received the ETH.
        assertEq(proposer.balance, balanceBefore + expectedCredit);

        // No pending withdrawal remains.
        (uint256 remaining,) = delayedWeth.withdrawals(address(game), proposer);
        assertEq(remaining, 0);
    }

    /// @notice claimCredit can be used to close the game even when the recipient has no credit.
    ///         First call closes the game and returns without reverting; second call reverts.
    function test_claimCredit_noCredit_succeeds() public {
        // Resolve and finalize the game.
        (,,,, Timestamp deadline,) = game.claimData();
        vm.warp(deadline.raw() + 1);
        game.resolve();
        vm.warp(game.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);

        address noCredit = address(0xdead);

        assertTrue(game.bondDistributionMode() == BondDistributionMode.UNDECIDED);

        // Game is open (UNDECIDED). First call closes it and returns without reverting.
        game.claimCredit(noCredit);

        // Game is now closed.
        assertTrue(game.bondDistributionMode() != BondDistributionMode.UNDECIDED);

        // Second call: game already closed, no credit → revert.
        vm.expectRevert(NoCreditToClaim.selector);
        game.claimCredit(noCredit);
    }

    function test_claimCredit_refundMode_succeeds() public {
        // Challenge the game so both proposer and challenger have refund credits.
        vm.startPrank(challenger);
        vm.deal(challenger, 1 ether);
        game.challenge{ value: 1 ether }();
        vm.stopPrank();

        // Let the prove deadline expire and resolve.
        (,,,, Timestamp deadline,) = game.claimData();
        vm.warp(deadline.raw() + 1);
        game.resolve();

        // Wait for finality delay.
        vm.warp(game.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);

        // Retire all games to make isGameProper return false.
        vm.prank(superchainConfig.guardian());
        anchorStateRegistry.updateRetirementTimestamp();

        // Close the game - enters REFUND mode.
        game.closeGame();
        assertTrue(game.bondDistributionMode() == BondDistributionMode.REFUND);

        // Both proposer and challenger should get their original bonds back.
        uint256 proposerBalanceBefore = proposer.balance;
        _claimCreditTwoPhase(game, proposer);
        assertEq(proposer.balance, proposerBalanceBefore + 1 ether);

        uint256 challengerBalanceBefore = challenger.balance;
        _claimCreditTwoPhase(game, challenger);
        assertEq(challenger.balance, challengerBalanceBefore + 1 ether);
    }

    function test_claimCredit_notFinalized_reverts() public {
        (,,,, Timestamp deadline,) = game.claimData();
        vm.warp(deadline.raw() + 1);
        game.resolve();

        // Game is resolved but not finalized - closeGame should revert with GameNotFinalized
        vm.expectRevert(GameNotFinalized.selector);
        game.claimCredit(proposer);
    }

    function test_claimCredit_bondTransferFailed_reverts() public {
        // Deploy a contract that rejects ETH transfers.
        ZKDisputeGame_RevertOnReceive_Harness revertingRecipient = new ZKDisputeGame_RevertOnReceive_Harness();

        // Create a game where the proposer is the reverting contract.
        vm.startPrank(address(revertingRecipient));
        vm.deal(address(revertingRecipient), 1 ether);
        ZKDisputeGame revertGame = ZKDisputeGame(
            payable(
                address(
                    disputeGameFactory.create{ value: 1 ether }(
                        gameType,
                        Claim.wrap(keccak256("revert-recipient-claim")),
                        abi.encodePacked(childL2SequenceNumber + grandchildOffset1, childGameIndex)
                    )
                )
            )
        );
        vm.stopPrank();

        // Resolve the parent game first (required for child resolution).
        (,,,, Timestamp parentDeadline,) = game.claimData();
        vm.warp(parentDeadline.raw() + 1);
        game.resolve();

        // Let the revertGame expire unchallenged and resolve.
        (,,,, Timestamp deadline,) = revertGame.claimData();
        vm.warp(deadline.raw() + 1);
        revertGame.resolve();

        // Wait for finality delay.
        vm.warp(revertGame.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);

        // Phase 1: unlock.
        revertGame.claimCredit(address(revertingRecipient));

        // Wait for DelayedWETH delay.
        vm.warp(block.timestamp + delayedWeth.delay() + 1 seconds);

        // Phase 2: should revert because recipient rejects ETH.
        vm.expectRevert(BondTransferFailed.selector);
        revertGame.claimCredit(address(revertingRecipient));
    }
}

/// @title ZKDisputeGame_CloseGame_Test
/// @notice Tests for closeGame functionality of ZKDisputeGame.
contract ZKDisputeGame_CloseGame_Test is ZKDisputeGame_TestInit {
    function test_closeGame_updatesAnchorGame_succeeds() public {
        (,,,, Timestamp deadline,) = game.claimData();
        vm.warp(deadline.raw() + 1);
        game.resolve();

        vm.warp(game.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);
        game.closeGame();

        assertEq(address(anchorStateRegistry.anchorGame()), address(game));
    }

    function test_closeGame_refundModeRetired_succeeds() public {
        // Resolve the game.
        (,,,, Timestamp deadline,) = game.claimData();
        vm.warp(deadline.raw() + 1);
        game.resolve();

        // Wait for finality delay.
        vm.warp(game.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);

        // Retire all existing games by updating the retirement timestamp.
        vm.prank(superchainConfig.guardian());
        anchorStateRegistry.updateRetirementTimestamp();

        // Close the game - it should enter REFUND mode because isGameProper returns false.
        game.closeGame();
        assertTrue(game.bondDistributionMode() == BondDistributionMode.REFUND);
    }

    function test_closeGame_alreadyClosed_succeeds() public {
        (,,,, Timestamp deadline,) = game.claimData();
        vm.warp(deadline.raw() + 1);
        game.resolve();

        vm.warp(game.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);
        game.closeGame();
        assertTrue(game.bondDistributionMode() == BondDistributionMode.NORMAL);

        // Calling closeGame again should return early without reverting.
        game.closeGame();
        assertTrue(game.bondDistributionMode() == BondDistributionMode.NORMAL);
    }

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
}

/// @title ZKDisputeGame_GameOver_Test
/// @notice Tests for gameOver view function of ZKDisputeGame.
contract ZKDisputeGame_GameOver_Test is ZKDisputeGame_TestInit {
    function test_gameOver_beforeDeadline_succeeds() public view {
        assertFalse(game.gameOver());
    }

    function test_gameOver_afterDeadline_succeeds() public {
        (,,,, Timestamp deadline,) = game.claimData();
        vm.warp(deadline.raw() + 1);
        assertTrue(game.gameOver());
    }

    function test_gameOver_afterProve_succeeds() public {
        vm.prank(prover);
        game.prove(bytes(""));
        assertTrue(game.gameOver());
    }
}

/// @title ZKDisputeGame_Credit_Test
/// @notice Tests for credit view function of ZKDisputeGame.
contract ZKDisputeGame_Credit_Test is ZKDisputeGame_TestInit {
    function test_credit_normalMode_succeeds() public {
        // Let the game expire unchallenged.
        (,,,, Timestamp deadline,) = game.claimData();
        vm.warp(deadline.raw() + 1);
        game.resolve();
        vm.warp(game.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);

        // Close in NORMAL mode (game is proper).
        game.closeGame();
        assertTrue(game.bondDistributionMode() == BondDistributionMode.NORMAL);

        // credit() should return normal mode credits.
        assertEq(game.credit(proposer), 1 ether);
        assertEq(game.credit(challenger), 0);
    }

    function test_credit_refundMode_succeeds() public {
        // Challenge the game.
        vm.startPrank(challenger);
        vm.deal(challenger, 1 ether);
        game.challenge{ value: 1 ether }();
        vm.stopPrank();

        // Resolve.
        (,,,, Timestamp deadline,) = game.claimData();
        vm.warp(deadline.raw() + 1);
        game.resolve();
        vm.warp(game.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);

        // Retire and close in REFUND mode.
        vm.prank(superchainConfig.guardian());
        anchorStateRegistry.updateRetirementTimestamp();
        game.closeGame();

        // credit() should return refund mode credits.
        assertEq(game.credit(proposer), 1 ether);
        assertEq(game.credit(challenger), 1 ether);
    }

    function test_credit_defaultMode_succeeds() public view {
        // Before closeGame is called, bondDistributionMode is UNDECIDED.
        // credit() should default to returning normalModeCredit.
        assertEq(game.credit(proposer), 0);
    }
}

/// @title ZKDisputeGame_RevertOnReceive_Harness
/// @notice Helper contract that rejects ETH transfers.
contract ZKDisputeGame_RevertOnReceive_Harness {
    receive() external payable {
        revert BondTransferFailed();
    }
}

// Import needed for the l2ChainId=0 test
import { ZKMockVerifier } from "test/dispute/zk/ZKMockVerifier.sol";
import { ZKRejectingVerifier } from "test/dispute/zk/ZKRejectingVerifier.sol";
