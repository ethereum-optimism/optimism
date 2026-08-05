// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { DisputeGameFactory_TestInit } from "test/dispute/DisputeGameFactory.t.sol";

// Libraries
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { Types } from "src/libraries/Types.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import {
    BondDistributionMode,
    Claim,
    Duration,
    GameStatus,
    GameType,
    GameTypes,
    Hash,
    Proposal
} from "src/dispute/lib/Types.sol";

// Contracts
import { ZKDisputeGame } from "src/dispute/zk/ZKDisputeGame.sol";

// Interfaces
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { ISuperFaultDisputeGame } from "interfaces/dispute/ISuperFaultDisputeGame.sol";
import { IOPContractsManagerV2 } from "interfaces/L1/opcm/IOPContractsManagerV2.sol";
import { IOPContractsManagerUtils } from "interfaces/L1/opcm/IOPContractsManagerUtils.sol";

/// @title ZKDisputeGameSuperMigration_Test
/// @notice Integration test exercising Path A (SFDG -> ZKDG) and Path B (SPDG -> ZKDG): migrating an
///         isolated chain's respected super game to the ZK dispute game via the real
///         `OPCMv2.upgrade()` flow (the same flow the `op-deployer manage` commands invoke). Asserts
///         (1) the upgrade succeeds (ZK registered, source super game cleared, respected type flipped
///         to ZK), (2) a ZK game can be created, proven, resolved, and finalized in place of the
///         super game (closeGame -> valid claim -> anchor advances), and (3) a super game created
///         just before the flip still finalizes afterwards.
contract ZKDisputeGameSuperMigration_Test is DisputeGameFactory_TestInit {
    /// @notice Actors used across the migration scenarios.
    address internal flipProposer = address(0xA11CE);
    address internal flipChallenger = address(0xB0B);
    address internal flipProver = address(0xC0FFEE);

    /// @notice Bond used for the games (init bond + challenger bond).
    uint256 internal flipBond = 1 ether;

    /// @notice ZK game durations injected via the OPCM upgrade.
    Duration internal zkMaxChallengeDuration = Duration.wrap(uint64(12 hours));
    Duration internal zkMaxProveDuration = Duration.wrap(uint64(3 days));

    /// @notice Storage-resident upgrade input (nested dynamic arrays must live in storage to push).
    IOPContractsManagerV2.UpgradeInput internal _zkUpgradeInput;

    function setUp() public override {
        super.setUp();
        skipIfDevFeatureDisabled(DevFeatures.ZK_DISPUTE_GAME);
        skipIfDevFeatureDisabled(DevFeatures.SUPER_ROOT_GAMES_MIGRATION);

        vm.deal(flipProposer, 100 ether);
        vm.deal(flipChallenger, 100 ether);
        vm.deal(flipProver, 100 ether);
    }

    /// @notice Path A: SuperFaultDisputeGame -> ZKDisputeGame.
    function test_flip_sfdgToZK_succeeds() public {
        _runFlipAndAssert(GameTypes.SUPER_CANNON_KONA);
    }

    /// @notice Path B: SuperPermissionedDisputeGame -> ZKDisputeGame (permissioned -> permissionless).
    function test_flip_spdgToZK_succeeds() public {
        _runFlipAndAssert(GameTypes.SUPER_PERMISSIONED);
    }

    /// @notice Drives the full migration for a given source super game type.
    function _runFlipAndAssert(GameType _sourceSuperType) internal {
        // ── 1. Precondition: register the source super game and make it the respected type ──
        _registerSourceSuperGame(_sourceSuperType);
        vm.prank(superchainConfig.guardian());
        anchorStateRegistry.setRespectedGameType(_sourceSuperType);

        // Warp past the retirement timestamp so the old super game is not "retired".
        vm.warp(anchorStateRegistry.retirementTimestamp() + 1);

        // ── 2. Create + resolve the old super game ──
        (, uint256 anchorSeqNum) = anchorStateRegistry.getAnchorRoot();
        IDisputeGame oldGame = _createSuperGame(_sourceSuperType, anchorSeqNum + 1);
        _resolveSuperGame(_sourceSuperType, oldGame);
        assertEq(uint8(oldGame.status()), uint8(GameStatus.DEFENDER_WINS), "old super game must resolve DEFENDER_WINS");

        // ── 3. Run the OPCMv2.upgrade() that disables the source super game, enables ZK, ──
        //       flips the respected type to ZK, re-seeds the anchor.
        _buildZKFlipUpgradeInput();
        _runUpgradeAsPAO();

        // ── 4. Assert the upgrade succeeds ──
        assertTrue(
            address(disputeGameFactory.gameImpls(GameTypes.ZK_DISPUTE_GAME)) != address(0),
            "ZK game impl must be registered"
        );
        assertEq(
            address(disputeGameFactory.gameImpls(_sourceSuperType)),
            address(0),
            "source super game impl must be cleared"
        );
        assertEq(
            anchorStateRegistry.respectedGameType().raw(),
            GameTypes.ZK_DISPUTE_GAME.raw(),
            "respected game type must be ZK"
        );

        // Assert that the flip re-seeded the anchor to the supplied honest root.
        (Hash reseededRoot, uint256 reseededSeq) = anchorStateRegistry.getAnchorRoot();
        assertEq(reseededRoot.raw(), keccak256("zkMigrationAnchor"), "anchor must be re-seeded to the supplied root");
        assertEq(reseededSeq, anchorSeqNum + 1, "anchor seq must be re-seeded");

        // ── 5. A ZK game can be created, used, and finalized in place of the super game ──
        _createAndFinalizeZKGame();

        // ── 6. The old super game (created just before the flip) still finalizes afterwards ──
        // Also checks wasRespectedGameTypeWhenCreated() stays true for the old super game.
        // Note: The ZK game updated the anchor so setAnchorState at finalization will not advance the anchor.
        assertTrue(anchorStateRegistry.isGameClaimValid(oldGame), "old super game must remain a valid claim post-flip");

        // SuperFaultDisputeGame additionally needs a closeGame() call to finalize.
        if (_sourceSuperType.raw() == GameTypes.SUPER_CANNON_KONA.raw()) {
            ISuperFaultDisputeGame sfdg = ISuperFaultDisputeGame(address(oldGame));
            sfdg.closeGame();
            // closeGame() finalizes the proper, valid game in NORMAL bond mode without altering its resolution.
            assertEq(
                uint8(sfdg.status()), uint8(GameStatus.DEFENDER_WINS), "old SFDG must remain DEFENDER_WINS after close"
            );
            assertEq(
                uint8(sfdg.bondDistributionMode()),
                uint8(BondDistributionMode.NORMAL),
                "old SFDG must close in NORMAL bond distribution mode"
            );
        }
    }

    /// @notice Registers the source super game implementation in the factory.
    function _registerSourceSuperGame(GameType _sourceSuperType) internal {
        if (_sourceSuperType.raw() == GameTypes.SUPER_PERMISSIONED.raw()) {
            // SuperPermissionedDisputeGame resolves DEFENDER_WINS at init. Registered with
            // `flipProposer` as the proposer so the old game can be created from that account.
            setupSuperPermissionedDisputeGame(flipProposer);
        } else {
            // SuperFaultDisputeGame registered at the OPCM-valid SUPER_CANNON_KONA slot. The shared
            // helper's default registers at the retired SUPER_CANNON slot, which is not one of the
            // OPCMv2 valid game types, so the game-type overload is used here.
            setupSuperFaultDisputeGame(Claim.wrap(bytes32(0)), GameTypes.SUPER_CANNON_KONA);
        }
    }

    /// @notice Creates a super game of the given type from the proposer account. The super root
    ///         commits a single dummy output root.
    function _createSuperGame(GameType _sourceSuperType, uint256 _seqNum) internal returns (IDisputeGame game_) {
        Types.OutputRootWithChainId[] memory roots = new Types.OutputRootWithChainId[](1);
        roots[0] = Types.OutputRootWithChainId({ chainId: systemConfig.l2ChainId(), root: keccak256("oldSuperRoot") });
        Types.SuperRootProof memory proof =
            Types.SuperRootProof({ version: bytes1(uint8(1)), timestamp: uint64(_seqNum), outputRoots: roots });
        Claim claim = Claim.wrap(Hashing.hashSuperRootProof(proof));
        bytes memory extra = Encoding.encodeSuperRootProof(proof);

        uint256 initBond = disputeGameFactory.initBonds(_sourceSuperType);
        vm.prank(flipProposer, flipProposer);
        game_ = disputeGameFactory.create{ value: initBond }(_sourceSuperType, claim, extra);
    }

    /// @notice Resolves a freshly-created, uncontested source super game to DEFENDER_WINS.
    function _resolveSuperGame(GameType _sourceSuperType, IDisputeGame _game) internal {
        // SuperPermissionedDisputeGame resolves DEFENDER_WINS at init so nothing to do.
        if (_sourceSuperType.raw() == GameTypes.SUPER_PERMISSIONED.raw()) {
            return;
        }
        // SuperFaultDisputeGame: warp past the chess clock and resolve the uncontested root.
        ISuperFaultDisputeGame sfdg = ISuperFaultDisputeGame(address(_game));
        vm.warp(block.timestamp + sfdg.maxClockDuration().raw() + 1 seconds);
        sfdg.resolveClaim(0, 0);
        sfdg.resolve();
    }

    /// @notice Builds the OPCMv2 upgrade input that flips the chain to the ZK dispute game.
    ///         The anchor override re-seeds a new super root.
    function _buildZKFlipUpgradeInput() internal {
        delete _zkUpgradeInput.disputeGameConfigs;
        delete _zkUpgradeInput.extraInstructions;

        _zkUpgradeInput.systemConfig = systemConfig;

        _pushDisabled(GameTypes.CANNON);
        _pushDisabled(GameTypes.PERMISSIONED_CANNON);
        _pushDisabled(GameTypes.CANNON_KONA);
        _pushDisabled(GameTypes.SUPER_PERMISSIONED);
        _pushDisabled(GameTypes.SUPER_CANNON_KONA);

        _zkUpgradeInput.disputeGameConfigs.push(
            IOPContractsManagerUtils.DisputeGameConfig({
                enabled: true,
                initBond: flipBond,
                gameType: GameTypes.ZK_DISPUTE_GAME,
                gameArgs: abi.encode(
                    IOPContractsManagerUtils.ZKDisputeGameConfig({
                        absolutePrestate: Claim.wrap(bytes32(0)),
                        maxChallengeDuration: zkMaxChallengeDuration,
                        maxProveDuration: zkMaxProveDuration,
                        challengerBond: flipBond
                    })
                )
            })
        );

        // Flip the respected game type to ZK (the previously-respected super game is now disabled).
        _zkUpgradeInput.extraInstructions.push(
            IOPContractsManagerUtils.ExtraInstruction({
                key: "overrides.cfg.startingRespectedGameType",
                data: abi.encode(GameTypes.ZK_DISPUTE_GAME)
            })
        );

        // Re-seed the anchor to a fresh honest super root, as the real migration does. No game has
        // advanced the anchor yet (anchorGame == 0), so this simply sets the new startingAnchorRoot.
        (, uint256 anchorSeqNum) = anchorStateRegistry.getAnchorRoot();
        _zkUpgradeInput.extraInstructions.push(
            IOPContractsManagerUtils.ExtraInstruction({
                key: "overrides.cfg.startingAnchorRoot",
                data: abi.encode(
                    Proposal({ root: Hash.wrap(keccak256("zkMigrationAnchor")), l2SequenceNumber: anchorSeqNum + 1 })
                )
            })
        );
    }

    /// @notice Pushes a disabled dispute game config for the given type.
    function _pushDisabled(GameType _gameType) internal {
        _zkUpgradeInput.disputeGameConfigs.push(
            IOPContractsManagerUtils.DisputeGameConfig({
                enabled: false,
                initBond: 0,
                gameType: _gameType,
                gameArgs: hex""
            })
        );
    }

    /// @notice Runs the OPCM upgrade as the chain ProxyAdmin owner
    function _runUpgradeAsPAO() internal {
        IOPContractsManagerV2.UpgradeInput memory input = _zkUpgradeInput;
        address chainPAO = proxyAdmin.owner();

        prankDelegateCall(chainPAO);
        (bool ok, bytes memory ret) =
            address(opcmV2).delegatecall(abi.encodeCall(IOPContractsManagerV2.upgrade, (input)));
        if (!ok) {
            assembly {
                revert(add(ret, 0x20), mload(ret))
            }
        }
    }

    /// @notice Creates a ZK game from the anchor, runs it through challenge -> prove -> resolve ->
    ///         close, and asserts it becomes a valid claim and advances the anchor — proving a ZK
    ///         game works in place of the migrated-away super game.
    function _createAndFinalizeZKGame() internal {
        (, uint256 anchorSeqNum) = anchorStateRegistry.getAnchorRoot();
        uint256 zkSeqNum = anchorSeqNum + 1000;

        (bytes memory ed, Claim zkClaim) =
            _makeZKExtraDataAndClaim(type(uint32).max, uint64(zkSeqNum), keccak256("zkOutputRoot"));

        vm.prank(flipProposer);
        ZKDisputeGame zkGame = ZKDisputeGame(
            payable(address(disputeGameFactory.create{ value: flipBond }(GameTypes.ZK_DISPUTE_GAME, zkClaim, ed)))
        );
        assertTrue(zkGame.wasRespectedGameTypeWhenCreated(), "ZK game must be respected at creation");

        // Challenge the new ZK game, then prove it. The mock verifier always accepts the proof.
        vm.prank(flipChallenger);
        zkGame.challenge{ value: flipBond }();
        vm.prank(flipProver);
        zkGame.prove(bytes(""));
        zkGame.resolve();
        assertEq(uint8(zkGame.status()), uint8(GameStatus.DEFENDER_WINS), "ZK game must resolve DEFENDER_WINS");

        // Warp past the finality delay (airgap from resolution), then close the game so it becomes a
        // valid claim and advances the anchor.
        vm.warp(zkGame.resolvedAt().raw() + anchorStateRegistry.disputeGameFinalityDelaySeconds() + 1 seconds);
        zkGame.closeGame();

        assertTrue(anchorStateRegistry.isGameClaimValid(IDisputeGame(address(zkGame))), "ZK game must be a valid claim");
        (Hash newAnchorRoot, uint256 newAnchorSeq) = anchorStateRegistry.getAnchorRoot();
        assertEq(newAnchorRoot.raw(), zkClaim.raw(), "anchor root must advance to the ZK game claim");
        assertEq(newAnchorSeq, zkSeqNum, "anchor sequence number must advance to the ZK game");
    }
}
