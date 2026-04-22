// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { DisputeGameFactory_TestInit } from "test/dispute/DisputeGameFactory.t.sol";

// Libraries
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { Claim, Duration, GameStatus, GameType, Hash, Timestamp } from "src/dispute/lib/Types.sol";
import { GameTypes } from "src/dispute/lib/Types.sol";
import { NoCreditToClaim } from "src/dispute/lib/Errors.sol";

// Contracts
import { ZKDisputeGame } from "src/dispute/zk/ZKDisputeGame.sol";

// Interfaces
import { IPermissionedDisputeGame } from "interfaces/dispute/IPermissionedDisputeGame.sol";

/// @title ZKDisputeGame_Integration_Test
/// @notice Integration tests that exercise the full ZK dispute game lifecycle.
///
/// Scenario 1 — Full happy-path chain with anchor advancement
/// ───────────────────────────────────────────────────────────
///  1. Proposer creates Game A from the anchor state (parentIndex = uint32.max).
///  2. Nobody challenges Game A. The challenge deadline expires → DEFENDER_WINS.
///  3. Game B is created referencing Game A as parent (before A is closed).
///  4. Challenger challenges Game B, third-party prover proves it → DEFENDER_WINS (immediate).
///  5. Game C is created referencing Game B as parent (before B is closed).
///  6. Proposer self-proves Game C without a challenge → DEFENDER_WINS (immediate).
///  7. All games are finalized: credits are claimed, anchor advances three times (A → B → C).
///
/// This covers all three DEFENDER_WINS resolution paths:
///  - Unchallenged (Game A)
///  - Challenged + proved by third party (Game B)
///  - Unchallenged + self-proved by proposer (Game C)
///
/// Scenario 2 — Challenger wins, child games auto-invalidated, chain recovers
/// ───────────────────────────────────────────────────────────────────────────
///  1. A base game is created and resolved to establish the anchor.
///  2. Game D is created referencing the base game. Game E is created referencing Game D.
///  3. Game D is challenged and the prove deadline passes → CHALLENGER_WINS.
///  4. Game E auto-resolves as CHALLENGER_WINS because parent D is invalid.
///     The proposer bond in Game E is burned (no challenger to collect it).
///  5. Anchor has NOT advanced past the base game (D and E were invalid).
///  6. Proposer recovers: Game F is created from anchor, goes unchallenged → DEFENDER_WINS.
///  7. Credits are claimed, anchor advances to F.
///
/// Scenario 3 — Upgrade from a different game type to ZK dispute game
/// ────────────────────────────────────────────────────────────────────
///  1. Chain starts with PermissionedDisputeGame (PDG) as the respected game type.
///  2. A PDG game is created, resolved unchallenged, and closed. The anchor advances under PDG.
///  3. The respected game type is switched to ZK (simulating the OPCM upgrade result on ASR).
///  4. A new ZK game is created from the anchor (parentIndex = uint32.max). It must inherit the
///     PDG-established anchor root and sequence number.
///  5. The ZK game runs through a full challenge → prove → resolve cycle.
///  6. Credits are claimed and the anchor advances to the ZK game, crossing the game-type boundary.
contract ZKDisputeGame_Integration_Test is DisputeGameFactory_TestInit {
    // Events
    event Challenged(address indexed challenger);
    event Proved(address indexed prover);
    event Resolved(GameStatus indexed status);

    // Actors
    address proposer = address(0x1001);
    address challenger = address(0x2002);
    address prover = address(0x3003);

    // Game parameters
    GameType gameType = GameTypes.ZK_DISPUTE_GAME;
    Duration maxChallengeDuration = Duration.wrap(12 hours);
    Duration maxProveDuration = Duration.wrap(3 days);
    uint256 bond = 1 ether;

    function setUp() public virtual override {
        super.setUp();
        skipIfDevFeatureDisabled(DevFeatures.ZK_DISPUTE_GAME);

        // Register the ZK dispute game implementation.
        setupZKDisputeGame(
            ZKDisputeGameParams({
                maxChallengeDuration: maxChallengeDuration,
                maxProveDuration: maxProveDuration,
                absolutePrestate: bytes32(0),
                challengerBond: bond
            })
        );

        // Fund actors.
        vm.deal(proposer, 100 ether);
        vm.deal(challenger, 100 ether);
        vm.deal(prover, 100 ether);

        // Warp forward to ensure games are created after the respectedGameTypeUpdatedAt timestamp.
        vm.warp(block.timestamp + 1000);
    }

    // ─────────────────────────────────────────────────────────────────────────
    //  Scenario 1 — Full happy-path chain with anchor advancement
    // ─────────────────────────────────────────────────────────────────────────

    function test_integration_fullLifecycleWithAnchorAdvancement_succeeds() public {
        (Hash anchorHashBefore, uint256 anchorSeqNum) = anchorStateRegistry.getAnchorRoot();
        Claim anchorRootBefore = Claim.wrap(anchorHashBefore.raw());

        // ── Game A: Created from anchor, goes unchallenged ──
        uint256 seqNumA = anchorSeqNum + 1000;
        Claim claimA = Claim.wrap(keccak256("claimA"));
        (ZKDisputeGame gameA, uint32 indexA) = _createGame(claimA, seqNumA, type(uint32).max);

        assertEq(uint8(gameA.status()), uint8(GameStatus.IN_PROGRESS));
        assertEq(gameA.gameCreator(), proposer);
        assertTrue(gameA.wasRespectedGameTypeWhenCreated());

        // Warp past challenge deadline → resolve as DEFENDER_WINS.
        _resolveUnchallenged(gameA);
        assertEq(uint8(gameA.status()), uint8(GameStatus.DEFENDER_WINS));

        // ── Game B: Created referencing A (must happen before closing A) ──
        uint256 seqNumB = seqNumA + 1000;
        Claim claimB = Claim.wrap(keccak256("claimB"));
        (ZKDisputeGame gameB, uint32 indexB) = _createGame(claimB, seqNumB, indexA);

        assertEq(gameB.parentIndex(), indexA);
        assertEq(gameB.startingRootHash().raw(), claimA.raw());
        assertEq(gameB.startingBlockNumber(), seqNumA);

        // Challenge, prove (third party), and resolve immediately.
        vm.prank(challenger);
        gameB.challenge{ value: bond }();
        _assertProposalStatus(gameB, ZKDisputeGame.ProposalStatus.Challenged);

        vm.prank(prover);
        gameB.prove(bytes(""));
        _assertProposalStatus(gameB, ZKDisputeGame.ProposalStatus.ChallengedAndValidProofProvided);

        gameB.resolve();
        assertEq(uint8(gameB.status()), uint8(GameStatus.DEFENDER_WINS));

        // ── Game C: Created referencing B (must happen before closing B) ──
        uint256 seqNumC = seqNumB + 1000;
        Claim claimC = Claim.wrap(keccak256("claimC"));
        (ZKDisputeGame gameC,) = _createGame(claimC, seqNumC, indexB);

        // Proposer self-proves without challenge → resolve immediately.
        vm.prank(proposer);
        gameC.prove(bytes(""));
        _assertProposalStatus(gameC, ZKDisputeGame.ProposalStatus.UnchallengedAndValidProofProvided);

        gameC.resolve();
        assertEq(uint8(gameC.status()), uint8(GameStatus.DEFENDER_WINS));

        // ── Finalization: claim credits and verify anchor advancement ──
        // Warp once past the latest resolution + finality delay (covers all games).
        _waitForFinality(gameC);

        // Sanity check: anchor has NOT advanced yet (no credits claimed / games closed).
        _assertAnchor(anchorRootBefore, anchorSeqNum);

        // Game A: proposer gets bond back → closing A advances anchor to A.
        _claimCreditAndAssert(gameA, proposer, bond);
        _assertAnchor(claimA, seqNumA);

        // Game B: prover gets challenger's bond, proposer gets own bond back.
        // Closing B advances anchor to B.
        _claimCreditTwoPhase(gameB, prover);
        _claimCreditTwoPhase(gameB, proposer);
        assertEq(gameB.credit(challenger), 0);
        _assertAnchor(claimB, seqNumB);

        // Game C: proposer gets bond back. Third-party prover has no credit.
        // Closing C advances anchor to C.
        _claimCreditAndAssert(gameC, proposer, bond);
        vm.expectRevert(NoCreditToClaim.selector);
        gameC.claimCredit(prover);
        _assertAnchor(claimC, seqNumC);
    }

    // ─────────────────────────────────────────────────────────────────────────
    //  Scenario 2 — Challenger wins, child auto-invalidated, chain recovers
    // ─────────────────────────────────────────────────────────────────────────

    // Storage slots used by scenario 2 to avoid stack-too-deep.
    ZKDisputeGame private s2_gameBase;
    ZKDisputeGame private s2_gameD;
    ZKDisputeGame private s2_gameE;
    ZKDisputeGame private s2_gameF;
    Claim private s2_claimBase;
    Claim private s2_claimF;
    uint256 private s2_seqNumBase;
    uint256 private s2_seqNumF;

    function test_integration_challengerWinsRecovery_succeeds() public {
        (, uint256 anchorSeqNum) = anchorStateRegistry.getAnchorRoot();

        // ── Base game: established as the anchor ──
        s2_seqNumBase = anchorSeqNum + 1000;
        s2_claimBase = Claim.wrap(keccak256("anchorClaimBase"));
        uint32 indexBase;
        (s2_gameBase, indexBase) = _createGame(s2_claimBase, s2_seqNumBase, type(uint32).max);

        _resolveUnchallenged(s2_gameBase);

        // ── Game D: referencing base game (created before closing base) ──
        uint32 indexD;
        (s2_gameD, indexD) = _createGame(Claim.wrap(keccak256("claimD")), s2_seqNumBase + 1000, indexBase);

        // ── Game E: referencing D (created before D is resolved) ──
        (s2_gameE,) = _createGame(Claim.wrap(keccak256("claimE")), s2_seqNumBase + 2000, indexD);

        // Challenge D and let prove deadline expire.
        vm.prank(challenger);
        s2_gameD.challenge{ value: bond }();

        (,,,, Timestamp deadlineD,) = s2_gameD.claimData();
        vm.warp(deadlineD.raw() + 1);

        // ── Resolve D as CHALLENGER_WINS ──
        s2_gameD.resolve();
        assertEq(uint8(s2_gameD.status()), uint8(GameStatus.CHALLENGER_WINS));

        // ── Game E auto-resolves as CHALLENGER_WINS (parent D is invalid) ──
        s2_gameE.resolve();
        assertEq(uint8(s2_gameE.status()), uint8(GameStatus.CHALLENGER_WINS));

        // ── Game F: proposer recovers from anchor (parentIndex = uint32.max) ──
        s2_seqNumF = s2_seqNumBase + 500;
        s2_claimF = Claim.wrap(keccak256("claimF"));
        (s2_gameF,) = _createGame(s2_claimF, s2_seqNumF, type(uint32).max);

        _resolveUnchallenged(s2_gameF);

        // ── Finalization ──
        _waitForFinality(s2_gameF);

        // Base game: proposer gets bond back → anchor advances to base.
        _claimCreditAndAssert(s2_gameBase, proposer, bond);
        _assertAnchor(s2_claimBase, s2_seqNumBase);

        // D: challenger claims all bonds (proposer's + own).
        uint256 challengerBal = challenger.balance;
        _claimCreditTwoPhase(s2_gameD, challenger);
        assertEq(challenger.balance, challengerBal + 2 * bond);

        // E: no challenger, so credit is assigned to address(0). Claim it to burn the bond.
        // In practice, op-challenger is expected to trigger this claim since no one else
        // has an economic incentive to do so.
        uint256 burnBalanceBefore = address(0).balance;
        _claimCreditTwoPhase(s2_gameE, address(0));
        assertEq(address(0).balance, burnBalanceBefore + bond);

        // Anchor should NOT have advanced to D or E (they were invalid).
        _assertAnchor(s2_claimBase, s2_seqNumBase);

        // F: proposer gets bond back → anchor advances to F.
        _claimCreditAndAssert(s2_gameF, proposer, bond);
        _assertAnchor(s2_claimF, s2_seqNumF);
    }

    // ─────────────────────────────────────────────────────────────────────────
    //  Scenario 3 — Upgrade from a different game type to ZK
    // ─────────────────────────────────────────────────────────────────────────

    /// @notice Simulates the post-OPCM-upgrade dispute-game behavior: a chain running on
    ///         PermissionedDisputeGame transitions to ZKDisputeGame. The anchor established
    ///         under PDG must carry over to the first ZK game created from anchor state.
    function test_integration_upgradeToZKGameType_succeeds() public {
        GameType pdgType = GameTypes.PERMISSIONED_CANNON;

        // ── Pre-upgrade state: register PDG and make it the respected game type ──
        // setUp() wired up ZK as the respected game type. Flip the respected game type
        // back to PDG so a PDG game created next will have wasRespectedGameTypeWhenCreated=true.
        vm.prank(superchainConfig.guardian());
        anchorStateRegistry.setRespectedGameType(pdgType);

        Claim pdgPrestate = Claim.wrap(keccak256("pdgPrestate"));
        setupPermissionedDisputeGame(pdgPrestate, proposer, challenger);

        // Warp forward so any freshly-set retirement timestamp is in the past.
        vm.warp(block.timestamp + 1000);

        // ── Create and resolve a PDG game unchallenged ──
        (, uint256 anchorSeqNumBefore) = anchorStateRegistry.getAnchorRoot();
        uint256 pdgSeqNum = anchorSeqNumBefore + 1;
        Claim pdgClaim = Claim.wrap(keccak256("pdgRootClaim"));
        uint256 pdgBond = disputeGameFactory.initBonds(pdgType);

        vm.prank(proposer, proposer);
        IPermissionedDisputeGame pdg = IPermissionedDisputeGame(
            payable(address(disputeGameFactory.create{ value: pdgBond }(pdgType, pdgClaim, abi.encode(pdgSeqNum))))
        );
        assertTrue(pdg.wasRespectedGameTypeWhenCreated());

        // Unchallenged → defender wins once the game clock expires.
        vm.warp(block.timestamp + pdg.maxClockDuration().raw() + 1);
        pdg.resolveClaim(0, 0);
        pdg.resolve();
        assertEq(uint8(pdg.status()), uint8(GameStatus.DEFENDER_WINS));

        // Wait for finality and close the game to advance the anchor under PDG.
        vm.warp(pdg.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);
        pdg.closeGame();
        _assertAnchor(pdgClaim, pdgSeqNum);

        // ── Simulate OPCM upgrade: switch the respected game type to ZK ──
        vm.prank(superchainConfig.guardian());
        anchorStateRegistry.setRespectedGameType(gameType);
        assertEq(anchorStateRegistry.respectedGameType().raw(), gameType.raw());

        // ── Create the first ZK game from anchor (parentIndex = uint32.max) ──
        // It must inherit the PDG-established anchor, not the original starting anchor.
        uint256 zkSeqNum = pdgSeqNum + 1000;
        Claim zkClaim = Claim.wrap(keccak256("zkRootClaimAfterUpgrade"));
        (ZKDisputeGame zkGame,) = _createGame(zkClaim, zkSeqNum, type(uint32).max);

        assertEq(zkGame.parentIndex(), type(uint32).max);
        assertEq(zkGame.startingRootHash().raw(), pdgClaim.raw());
        assertEq(zkGame.startingBlockNumber(), pdgSeqNum);
        assertTrue(zkGame.wasRespectedGameTypeWhenCreated());

        // ── Full challenge-prove-resolve cycle on the new ZK game ──
        vm.prank(challenger);
        zkGame.challenge{ value: bond }();
        _assertProposalStatus(zkGame, ZKDisputeGame.ProposalStatus.Challenged);

        vm.prank(prover);
        zkGame.prove(bytes(""));
        _assertProposalStatus(zkGame, ZKDisputeGame.ProposalStatus.ChallengedAndValidProofProvided);

        zkGame.resolve();
        assertEq(uint8(zkGame.status()), uint8(GameStatus.DEFENDER_WINS));

        // ── Finalization: claim credit → anchor advances to the ZK game ──
        _waitForFinality(zkGame);
        _claimCreditTwoPhase(zkGame, prover);
        _claimCreditAndAssert(zkGame, proposer, bond);
        _assertAnchor(zkClaim, zkSeqNum);
    }

    // ─────────────────────────────────────────────────────────────────────────
    //  Helpers
    // ─────────────────────────────────────────────────────────────────────────

    /// @notice Creates a ZK dispute game via the factory.
    function _createGame(
        Claim _claim,
        uint256 _seqNum,
        uint32 _parentIndex
    )
        internal
        returns (ZKDisputeGame game_, uint32 index_)
    {
        vm.prank(proposer);
        game_ = ZKDisputeGame(
            payable(
                address(
                    disputeGameFactory.create{ value: bond }(gameType, _claim, abi.encodePacked(_seqNum, _parentIndex))
                )
            )
        );
        index_ = uint32(disputeGameFactory.gameCount() - 1);
    }

    /// @notice Warps past the challenge deadline and resolves the game.
    function _resolveUnchallenged(ZKDisputeGame _game) internal {
        (,,,, Timestamp deadline,) = _game.claimData();
        vm.warp(deadline.raw() + 1);
        _game.resolve();
    }

    /// @notice Warps to the finality delay after resolution (only moves forward).
    function _waitForFinality(ZKDisputeGame _game) internal {
        uint256 target = _game.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds;
        if (target > block.timestamp) {
            vm.warp(target);
        }
    }

    /// @notice Two-phase credit claim: phase 1 (unlock) + warp + phase 2 (withdraw).
    function _claimCreditTwoPhase(ZKDisputeGame _game, address _recipient) internal {
        _game.claimCredit(_recipient);
        vm.warp(block.timestamp + delayedWeth.delay() + 1 seconds);
        _game.claimCredit(_recipient);
    }

    /// @notice Claims credit via two-phase and asserts the expected payout.
    function _claimCreditAndAssert(ZKDisputeGame _game, address _recipient, uint256 _expected) internal {
        uint256 balanceBefore = _recipient.balance;
        _claimCreditTwoPhase(_game, _recipient);
        assertEq(_recipient.balance, balanceBefore + _expected);
    }

    /// @notice Asserts the current anchor state matches expected values.
    function _assertAnchor(Claim _expectedRoot, uint256 _expectedSeqNum) internal view {
        (Hash root, uint256 seqNum) = anchorStateRegistry.getAnchorRoot();
        assertEq(seqNum, _expectedSeqNum);
        assertEq(root.raw(), _expectedRoot.raw());
    }

    /// @notice Asserts the proposal status of a game.
    function _assertProposalStatus(ZKDisputeGame _game, ZKDisputeGame.ProposalStatus _expected) internal view {
        (, ZKDisputeGame.ProposalStatus status,,,,) = _game.claimData();
        assertEq(uint8(status), uint8(_expected));
    }
}
