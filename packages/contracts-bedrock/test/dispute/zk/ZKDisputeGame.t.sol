// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { DisputeGameFactory_TestInit } from "test/dispute/DisputeGameFactory.t.sol";

// Libraries
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { BondDistributionMode, Claim, Duration, GameStatus, GameType, Hash, Timestamp } from "src/dispute/lib/Types.sol";
import { Types } from "src/libraries/Types.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import {
    AnchorRootNotFound,
    BadExtraData,
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
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
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

    // Per-chain output roots packed into each game's SuperRootProof.
    // Parent game only uses one output root for a single chain.
    // Child game uses multiple output roots for multiple chains.
    bytes32 parentOutputRoot = keccak256("parent-output-root");
    bytes32 childOutputRoot = keccak256("child-output-root");

    // The child game's multi-chain super root pairs (populated in setUp). The `l2ChainId` pair
    // carries `childOutputRoot` while the others test the `rootClaimByChainId` implementation.
    Types.OutputRootWithChainId[] internal childPairs;

    // Game rootClaims, computed in setUp() once the per-chain pairs are known.
    Claim parentRootClaim;
    Claim rootClaim;

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
        // The parent only uses one output root for a single chain.
        parentGame = _createZKGame(type(uint32).max, uint64(parentL2SequenceNumber), parentOutputRoot);
        parentRootClaim = parentGame.rootClaim();

        // Record actual index of parent game (on fork, existing games already occupy indices 0, 1, ...)
        parentGameIndex = uint32(disputeGameFactory.gameCount() - 1);

        // We want the parent game to finalize. We'll skip its challenge period.
        (,,,, Timestamp parentGameDeadline,) = parentGame.claimData();
        vm.warp(parentGameDeadline.raw() + 1 seconds);
        parentGame.resolve();

        // Create the child game before claiming parent credit, because claimCredit triggers
        // closeGame() which advances the anchor to parentL2SequenceNumber. After that, the
        // parent's seq num would equal the anchor, violating the "strictly above" invariant.
        // The child carries a multi-chain super root so all the tests check for multi-root validity.
        // In practice, the absolute prestate would also change here to match the new ZK program.
        childPairs.push(Types.OutputRootWithChainId({ chainId: 10, root: keccak256("op-mainnet") }));
        childPairs.push(Types.OutputRootWithChainId({ chainId: l2ChainId, root: childOutputRoot }));
        childPairs.push(Types.OutputRootWithChainId({ chainId: 130, root: keccak256("unichain") }));
        game = _createZKGame(parentGameIndex, uint64(childL2SequenceNumber), childPairs);
        rootClaim = game.rootClaim();

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

    /// @notice Single-chain variant of `_createZKGame` — delegates to the multi-chain variant with
    ///         a one-pair set bound to the fixture's `l2ChainId`.
    function _createZKGame(
        uint32 _parentIndex,
        uint64 _timestamp,
        bytes32 _outputRoot
    )
        internal
        returns (ZKDisputeGame game_)
    {
        Types.OutputRootWithChainId[] memory pairs = new Types.OutputRootWithChainId[](1);
        pairs[0] = Types.OutputRootWithChainId({ chainId: l2ChainId, root: _outputRoot });
        game_ = _createZKGame(_parentIndex, _timestamp, pairs);
    }

    /// @notice Build the SuperRootProof from `_pairs`, derive its rootClaim, and create the ZK
    ///         dispute game via `disputeGameFactory.create`. Returns the deployed game.
    function _createZKGame(
        uint32 _parentIndex,
        uint64 _timestamp,
        Types.OutputRootWithChainId[] memory _pairs
    )
        internal
        returns (ZKDisputeGame game_)
    {
        (bytes memory ed, Claim rc) = _makeZKExtraDataAndClaim(_parentIndex, _timestamp, _pairs);
        game_ = ZKDisputeGame(
            payable(
                address(disputeGameFactory.create{ value: disputeGameFactory.initBonds(gameType) }(gameType, rc, ed))
            )
        );
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
        assertEq(address(game.weth()), address(delayedWeth));
        assertEq(address(game.anchorStateRegistry()), address(anchorStateRegistry));
        assertEq(game.parentIndex(), parentGameIndex);
        assertEq(game.absolutePrestate(), bytes32(0));
        assertTrue(address(game.verifier()) != address(0));
        assertEq(game.challengerBond(), 1 ether);
        assertTrue(game.l1Head().raw() != bytes32(0));
        // Check all chains in the SuperRootProof return their packed output roots.
        for (uint256 i; i < childPairs.length; i++) {
            assertEq(game.rootClaimByChainId(childPairs[i].chainId).raw(), childPairs[i].root);
        }

        // The game was created while its game type was respected.
        assertTrue(game.wasRespectedGameTypeWhenCreated());

        // extraData layout: 4 byte parentIndex + 1 byte version + 8 byte timestamp + 3 pairs (3*64B) = 205 bytes.
        bytes memory extra = game.extraData();
        assertEq(extra.length, 205);

        // The parent's sequence number (timestamp).
        assertEq(game.startingSequenceNumber(), parentL2SequenceNumber);

        // The parent's rootClaim is the hash of its own SuperRootProof.
        assertEq(game.startingRootHash().raw(), parentRootClaim.raw());

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

        ZKDisputeGame newGame = _createZKGame(
            childGameIndex, uint64(childL2SequenceNumber + grandchildOffset1), keccak256("permissionless-claim")
        );
        vm.stopPrank();

        assertEq(newGame.gameCreator(), anyUser);
    }

    function test_initialize_sequenceNumberTooSmall_reverts() public {
        // Try to create a child game whose super-root timestamp is below the parent's.
        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);

        (bytes memory ed, Claim rc) = _makeZKExtraDataAndClaim(parentGameIndex, uint64(1), keccak256("rootClaim"));

        // We expect revert because timestamp (1) <= parent's sequence number.
        vm.expectRevert(abi.encodeWithSelector(UnexpectedRootClaim.selector, rc));
        disputeGameFactory.create{ value: 1 ether }(gameType, rc, ed);
        vm.stopPrank();
    }

    /// @notice Fuzz over every timestamp in `[0, parentL2SequenceNumber]`. All must revert with
    ///         `UnexpectedRootClaim` because the new claim must be strictly above the parent's
    ///         sequence number.
    function testFuzz_initialize_timestampAtOrBeforeParent_reverts(uint64 _timestamp) public {
        _timestamp = uint64(bound(_timestamp, 0, parentL2SequenceNumber));

        (bytes memory ed, Claim rc) =
            _makeZKExtraDataAndClaim(parentGameIndex, _timestamp, keccak256("any-output-root"));

        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);
        vm.expectRevert(abi.encodeWithSelector(UnexpectedRootClaim.selector, rc));
        disputeGameFactory.create{ value: 1 ether }(gameType, rc, ed);
        vm.stopPrank();
    }

    function test_initialize_parentBlacklisted_reverts() public {
        // Blacklist the game on the anchor state registry (which is what's actually used for validation).
        vm.prank(superchainConfig.guardian());
        anchorStateRegistry.blacklistDisputeGame(IDisputeGame(address(game)));

        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);
        (bytes memory ed, Claim rc) = _makeZKExtraDataAndClaim(
            childGameIndex, uint64(childL2SequenceNumber + grandchildOffset1), keccak256("blacklisted-parent-game")
        );
        vm.expectRevert(InvalidParentGame.selector);
        disputeGameFactory.create{ value: 1 ether }(gameType, rc, ed);
        vm.stopPrank();
    }

    function test_initialize_parentBlacklistedAfterCreation_reverts() public {
        // Create a new game which will be the parent.
        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);
        ZKDisputeGame parentNotRespected = _createZKGame(
            childGameIndex, uint64(childL2SequenceNumber + grandchildOffset1), keccak256("not-respected-parent-game")
        );
        uint32 parentNotRespectedIndex = uint32(disputeGameFactory.gameCount() - 1);
        vm.stopPrank();

        // Blacklist the parent game to make it invalid.
        vm.prank(superchainConfig.guardian());
        anchorStateRegistry.blacklistDisputeGame(IDisputeGame(address(parentNotRespected)));

        // Try to create a game with a parent game that is not valid.
        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);
        (bytes memory ed2, Claim rc2) = _makeZKExtraDataAndClaim(
            parentNotRespectedIndex,
            uint64(childL2SequenceNumber + grandchildOffset2),
            keccak256("child-with-not-respected-parent")
        );
        vm.expectRevert(InvalidParentGame.selector);
        disputeGameFactory.create{ value: 1 ether }(gameType, rc2, ed2);
        vm.stopPrank();
    }

    function test_initialize_parentRetired_reverts() public {
        // Retire all existing games by updating the retirement timestamp.
        vm.prank(superchainConfig.guardian());
        anchorStateRegistry.updateRetirementTimestamp();

        // Try to create a new game referencing the (now retired) child game as parent.
        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);
        (bytes memory ed, Claim rc) = _makeZKExtraDataAndClaim(
            childGameIndex, uint64(childL2SequenceNumber + grandchildOffset1), keccak256("child-of-retired-parent")
        );
        vm.expectRevert(InvalidParentGame.selector);
        disputeGameFactory.create{ value: 1 ether }(gameType, rc, ed);
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
        (bytes memory ed, Claim rc) = _makeZKExtraDataAndClaim(
            childGameIndex, uint64(childL2SequenceNumber + grandchildOffset1), keccak256("child-of-challenger-wins")
        );
        vm.expectRevert(InvalidParentGame.selector);
        disputeGameFactory.create{ value: 1 ether }(gameType, rc, ed);
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
            delayedWeth
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
        (bytes memory ed, Claim rc) = _makeZKExtraDataAndClaim(
            childGameIndex, uint64(childL2SequenceNumber + grandchildOffset1), keccak256("different-type-claim")
        );
        vm.expectRevert(UnexpectedGameType.selector);
        disputeGameFactory.create{ value: 1 ether }(differentGameType, rc, ed);
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
        (bytes memory ed, Claim rc) = _makeZKExtraDataAndClaim(
            parentGameIndex, uint64(childL2SequenceNumber + grandchildOffset1), keccak256("below-anchor-claim")
        );
        vm.expectRevert(InvalidParentGame.selector);
        disputeGameFactory.create{ value: 1 ether }(gameType, rc, ed);
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
        (bytes memory ed, Claim rc) = _makeZKExtraDataAndClaim(
            childGameIndex, uint64(childL2SequenceNumber + grandchildOffset1), keccak256("equal-anchor-claim")
        );
        vm.expectRevert(InvalidParentGame.selector);
        disputeGameFactory.create{ value: 1 ether }(gameType, rc, ed);
        vm.stopPrank();
    }

    function test_initialize_fromAnchorState_succeeds() public {
        // Create a first game (parentIndex = uint32.max) and verify it uses anchor state values.
        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);

        ZKDisputeGame anchorGame =
            _createZKGame(type(uint32).max, uint64(anchorL2SequenceNumber + 5000), keccak256("anchor-start-claim"));
        vm.stopPrank();

        // The starting proposal should match the anchor state values.
        (Hash anchorRoot, uint256 anchorSeqNum) = anchorStateRegistry.getAnchorRoot();
        assertEq(anchorGame.startingRootHash().raw(), anchorRoot.raw());
        assertEq(anchorGame.startingSequenceNumber(), anchorSeqNum);
        assertEq(anchorGame.parentIndex(), type(uint32).max);
    }

    function test_initialize_zeroAnchorRoot_reverts() public {
        // Mock the anchor state registry to return a zero root hash, simulating a misconfigured
        // or uninitialized anchor state. Only genesis games (parentIndex == uint32.max) read
        // from the anchor directly, so we target that path.
        vm.mockCall(
            address(anchorStateRegistry),
            abi.encodeCall(IAnchorStateRegistry.getAnchorRoot, ()),
            abi.encode(Hash.wrap(bytes32(0)), anchorL2SequenceNumber)
        );

        (bytes memory ed, Claim rc) = _makeZKExtraDataAndClaim(
            type(uint32).max, uint64(anchorL2SequenceNumber + 5000), keccak256("zero-anchor-claim")
        );

        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);
        vm.expectRevert(AnchorRootNotFound.selector);
        disputeGameFactory.create{ value: 1 ether }(gameType, rc, ed);
        vm.stopPrank();
    }

    function test_initialize_notRespectedGameType_succeeds() public {
        // Change the respected game type so our game type is no longer respected.
        vm.prank(superchainConfig.guardian());
        anchorStateRegistry.setRespectedGameType(GameType.wrap(250));

        // Create a game after the respected type has changed.
        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);
        ZKDisputeGame notRespectedGame = _createZKGame(
            childGameIndex, uint64(childL2SequenceNumber + grandchildOffset1), keccak256("not-respected-claim")
        );
        vm.stopPrank();

        // The game was created while its game type was NOT respected.
        assertFalse(notRespectedGame.wasRespectedGameTypeWhenCreated());
    }

    function test_initialize_invalidCalldataSize_reverts() public {
        // initialize() validates the extraData shape via `_verifyInitCallDataLength`. The minimum
        // valid length is 4 (parentIndex) + 1 (version) + 8 (timestamp) + 64 (one pair) = 77 bytes;
        // any larger size must add full 64-byte pairs. Anything else reverts BadExtraData.
        vm.startPrank(proposer);
        vm.deal(proposer, 5 ether);

        // Case 1: 78-byte extraData — one byte over the 77-byte minimum, doesn't extend to a full
        // additional pair (would need 77 + 64 = 141). Hits `rem % 64 != 0`.
        bytes memory tooBig = new bytes(78);
        vm.expectRevert(BadExtraData.selector);
        disputeGameFactory.create{ value: 1 ether }(gameType, rootClaim, tooBig);

        // Case 2: 76-byte extraData — one byte under the 77-byte minimum. Hits `rem % 64 != 0`.
        bytes memory tooSmall = new bytes(76);
        vm.expectRevert(BadExtraData.selector);
        disputeGameFactory.create{ value: 1 ether }(gameType, rootClaim, tooSmall);

        // Case 3: 13-byte extraData — header bytes only, no chain pairs. Hits `rem == 0`.
        bytes memory noPairs = new bytes(13);
        vm.expectRevert(BadExtraData.selector);
        disputeGameFactory.create{ value: 1 ether }(gameType, rootClaim, noPairs);

        // Case 4: 12-byte extraData — header incomplete. Hits `extraLen < 13`.
        bytes memory headerTooShort = new bytes(12);
        vm.expectRevert(BadExtraData.selector);
        disputeGameFactory.create{ value: 1 ether }(gameType, rootClaim, headerTooShort);

        vm.stopPrank();
    }

    function test_initialize_calledDirectlyOnImpl_reverts() public {
        // Calling initialize() directly on the impl has no CWIA-appended immutable args, so
        // msg.data.length is just the 4-byte selector — below `preExtraDataLen` (94). Hits the
        // `msg.data.length < preExtraDataLen` branch in `_verifyInitCallDataLength` which is
        // unreachable through the factory.
        vm.expectRevert(BadExtraData.selector);
        gameImpl.initialize();
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

    function test_challenge_atExactDeadline_succeeds() public {
        // Warp to exactly the challenge deadline. gameOver() uses strict `<`, so at
        // exactly deadline the game is NOT over and challenge should succeed.
        (,,,, Timestamp deadline,) = game.claimData();
        vm.warp(deadline.raw());

        vm.startPrank(challenger);
        vm.deal(challenger, 1 ether);
        game.challenge{ value: 1 ether }();
        vm.stopPrank();

        (,, address challenger_,,,) = game.claimData();
        assertEq(challenger_, challenger);
    }

    function test_challenge_afterDeadline_reverts() public {
        // Warp one second past the challenge deadline so the game is over.
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
        ZKDisputeGame childGame =
            _createZKGame(childGameIndex, uint64(childL2SequenceNumber + grandchildOffset1), keccak256("child-claim"));
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

    function test_prove_publicValuesEncoding_succeeds() public {
        // Build the expected public values that prove() should pass to the verifier. The super-root
        // migration removed `l2ChainId` from the encoding.
        bytes memory expectedPublicValues = abi.encode(
            game.l1Head(),
            game.startingRootHash(),
            game.rootClaim(),
            game.l2SequenceNumber(),
            prover // msg.sender inside prove()
        );

        // Expect the verifier to be called with exactly these arguments.
        vm.expectCall(
            address(game.verifier()),
            abi.encodeCall(IZKVerifier.verify, (game.absolutePrestate(), expectedPublicValues, bytes("")))
        );

        vm.prank(prover);
        game.prove(bytes(""));
    }

    function test_prove_emitsProvedEvent_succeeds() public {
        vm.expectEmit(true, false, false, false, address(game));
        emit Proved(prover);

        vm.prank(prover);
        game.prove(bytes(""));
    }

    function test_prove_nonProposerCanProve_succeeds() public {
        // Verify the prover is not the game creator.
        assertFalse(prover == game.gameCreator());

        vm.prank(prover);
        game.prove(bytes(""));

        // The claimData.prover should be set to the actual caller, not the proposer.
        (,,, address prover_,,) = game.claimData();
        assertEq(prover_, prover);
        assertFalse(prover_ == game.gameCreator());
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
            delayedWeth
        );

        vm.prank(superchainConfig.guardian());
        anchorStateRegistry.setRespectedGameType(rejectGameType);

        vm.startPrank(disputeGameFactory.owner());
        disputeGameFactory.setImplementation(rejectGameType, IDisputeGame(newImpl), gameArgs);
        disputeGameFactory.setInitBond(rejectGameType, 1 ether);
        vm.stopPrank();

        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);
        (bytes memory ed, Claim rc) =
            _makeZKExtraDataAndClaim(type(uint32).max, uint64(parentL2SequenceNumber), keccak256("reject-claim"));
        ZKDisputeGame rejectGame = ZKDisputeGame(
            payable(
                address(
                    disputeGameFactory.create{ value: disputeGameFactory.initBonds(rejectGameType) }(
                        rejectGameType, rc, ed
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
        ZKDisputeGame childGame =
            _createZKGame(childGameIndex, uint64(childL2SequenceNumber + grandchildOffset1), keccak256("new-claim"));

        vm.stopPrank();

        // The parent game is still in progress, not resolved.
        // So, if we try to resolve the childGame, it should revert with ParentGameNotResolved.
        vm.expectRevert(ParentGameNotResolved.selector);
        childGame.resolve();
    }

    function test_resolve_parentGameInvalid_succeeds() public {
        // 1) Now create a child game referencing that losing parent at index 1.
        vm.startPrank(proposer);
        ZKDisputeGame childGame = _createZKGame(
            childGameIndex, uint64(childL2SequenceNumber + grandchildOffset4), keccak256("child-of-loser")
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
        ZKDisputeGame childGame = _createZKGame(
            childGameIndex, uint64(childL2SequenceNumber + grandchildOffset4), keccak256("child-of-invalid-parent")
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
        ZKDisputeGame revertGame = _createZKGame(
            childGameIndex, uint64(childL2SequenceNumber + grandchildOffset1), keccak256("revert-recipient-claim")
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

    function test_closeGame_setAnchorStateReverts_succeeds() public {
        // Resolve and finalize the game.
        (,,,, Timestamp deadline,) = game.claimData();
        vm.warp(deadline.raw() + 1);
        game.resolve();
        vm.warp(game.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);

        // Force `setAnchorState` to revert. The try/catch inside `closeGame` swallows the failure
        // and continues; `isGameProper` is independently consulted to determine bond distribution.
        vm.mockCallRevert(
            address(anchorStateRegistry),
            abi.encodeCall(IAnchorStateRegistry.setAnchorState, (IDisputeGame(address(game)))),
            bytes("setAnchorState failed")
        );

        game.closeGame();

        // Despite the anchor update failing, the game is still proper → NORMAL distribution.
        assertTrue(game.bondDistributionMode() == BondDistributionMode.NORMAL);
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

/// @title ZKDisputeGame_RootClaim_Test
/// @notice Tests the `rootClaimByChainId` function of `ZKDisputeGame`.
contract ZKDisputeGame_RootClaim_Test is ZKDisputeGame_TestInit {
    /// @notice Tests that rootClaimByChainId returns the per-chain output root packed in the
    ///         SuperRootProof, not the super-root hash itself.
    function test_rootClaimByChainId_succeeds() public view {
        // setUp() packed (l2ChainId, childOutputRoot) into the child game's SuperRootProof.
        assertEq(game.rootClaimByChainId(l2ChainId).raw(), childOutputRoot);
        // The returned value is NOT the super-root hash.
        assertTrue(game.rootClaimByChainId(l2ChainId).raw() != game.rootClaim().raw());
    }

    /// @notice Tests that rootClaimByChainId reverts when called with a chain ID that is not
    ///         present in the SuperRootProof's pair list.
    function testFuzz_rootClaimByChainId_wrongChainId_reverts(uint256 _chainId) public {
        // Exclude every chain present in the default child super root (see setUp).
        for (uint256 i; i < childPairs.length; i++) {
            vm.assume(_chainId != childPairs[i].chainId);
        }
        vm.expectRevert(UnknownChainId.selector);
        game.rootClaimByChainId(_chainId);
    }

    /// @notice Tests that the contract's hash-binding invariant rejects mismatched extraData ↔
    ///         rootClaim pairs.
    function test_initialize_rootClaimMismatch_reverts() public {
        // Build valid extraData but pair it with the wrong rootClaim (any unrelated hash).
        (bytes memory ed, /* Claim */ ) = _makeZKExtraDataAndClaim(
            childGameIndex, uint64(childL2SequenceNumber + grandchildOffset1), keccak256("genuine")
        );
        Claim wrongRootClaim = Claim.wrap(keccak256("not-the-binding-hash"));

        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);
        vm.expectRevert(BadExtraData.selector);
        disputeGameFactory.create{ value: 1 ether }(gameType, wrongRootClaim, ed);
        vm.stopPrank();
    }

    /// @notice Verifies that `superRootProof()` returns exactly the bytes packed after the 4-byte
    ///         parentIndex header and that those bytes decode to a well-formed `SuperRootProof`
    ///         whose fields match what setUp packed.
    function test_superRootProof_returnsExpectedBytes_succeeds() public view {
        bytes memory full = game.extraData();
        bytes memory proofBytes = game.superRootProof();

        // `superRootProof()` returns extraData[4:].
        assertEq(proofBytes.length, full.length - 4);
        for (uint256 i = 0; i < proofBytes.length; i++) {
            assertEq(proofBytes[i], full[i + 4]);
        }

        // The returned bytes decode to a SuperRootProof matching the values setUp packed.
        Types.SuperRootProof memory decoded = Encoding.decodeSuperRootProof(proofBytes);
        assertEq(decoded.version, bytes1(0x01));
        assertEq(decoded.timestamp, uint64(childL2SequenceNumber));
        assertEq(decoded.outputRoots.length, childPairs.length);
        for (uint256 i = 0; i < childPairs.length; i++) {
            assertEq(decoded.outputRoots[i].chainId, childPairs[i].chainId);
            assertEq(decoded.outputRoots[i].root, childPairs[i].root);
        }
    }
}

/// @title ZKDisputeGame_RevertOnReceive_Harness
/// @notice Helper contract that rejects ETH transfers.
contract ZKDisputeGame_RevertOnReceive_Harness {
    receive() external payable {
        revert BondTransferFailed();
    }
}

// Verifier test doubles used by the prove() tests.
import { ZKMockVerifier } from "test/dispute/zk/ZKMockVerifier.sol";
import { ZKRejectingVerifier } from "test/dispute/zk/ZKRejectingVerifier.sol";
