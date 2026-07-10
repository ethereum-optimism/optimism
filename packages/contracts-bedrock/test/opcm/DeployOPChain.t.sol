// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";
import { FeatureFlags } from "test/setup/FeatureFlags.sol";

// Scripts
import { DeploySuperchain } from "scripts/deploy/DeploySuperchain.s.sol";
import { DeployImplementations } from "scripts/deploy/DeployImplementations.s.sol";
import { DeployOPChain } from "scripts/deploy/DeployOPChain.s.sol";
import { StandardConstants } from "scripts/deploy/StandardConstants.sol";
import { Types } from "scripts/libraries/Types.sol";

// Libraries
import { Constants } from "src/libraries/Constants.sol";
import { Features } from "src/libraries/Features.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { LibGameArgs } from "src/dispute/lib/LibGameArgs.sol";

// Interfaces
import { IOPContractsManagerV2 } from "interfaces/L1/opcm/IOPContractsManagerV2.sol";
import { IOPContractsManagerStandardValidator } from "interfaces/L1/IOPContractsManagerStandardValidator.sol";
import { IOPContractsManagerContainer } from "interfaces/L1/opcm/IOPContractsManagerContainer.sol";
import { IResourceMetering } from "interfaces/L1/IResourceMetering.sol";
import { Claim, Duration, GameType, GameTypes, Hash, Proposal } from "src/dispute/lib/Types.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IPermissionedDisputeGame } from "interfaces/dispute/IPermissionedDisputeGame.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";

contract DeployOPChain_TestBase is Test, FeatureFlags {
    DeploySuperchain deploySuperchain;
    DeployImplementations deployImplementations;
    DeployOPChain deployOPChain;
    Types.DeployOPChainInput deployOPChainInput;

    // DeploySuperchain default inputs.
    address superchainProxyAdminOwner = makeAddr("superchainProxyAdminOwner");
    address guardian = makeAddr("guardian");
    bool paused = false;

    // DeployImplementations default inputs.
    // - superchainConfigProxy is set during `setUp` since it is an output of DeploySuperchain.
    uint256 withdrawalDelaySeconds = 100;
    uint256 minProposalSizeBytes = 126_000;
    uint256 challengePeriodSeconds = 86_400;
    uint256 proofMaturityDelaySeconds = 400;
    uint256 disputeGameFinalityDelaySeconds = 500;

    // DeployOPChain default inputs.
    // - opcm is set during `setUp` since it is an output of DeployImplementations.
    address opChainProxyAdminOwner = makeAddr("opChainProxyAdminOwner");
    address systemConfigOwner = makeAddr("systemConfigOwner");
    address batcher = makeAddr("batcher");
    address unsafeBlockSigner = makeAddr("unsafeBlockSigner");
    address proposer = makeAddr("proposer");
    address challenger = makeAddr("challenger");
    uint32 basefeeScalar = 100;
    uint32 blobBaseFeeScalar = 200;
    uint256 l2ChainId = 300;
    string saltMixer = "saltMixer";
    uint64 gasLimit = 60_000_000;
    GameType disputeGameType = GameTypes.PERMISSIONED_CANNON;
    // Prestates are real release hashes from the superchain registry's standard-prestates.toml;
    // the tests only need them to be non-zero and distinct from each other.
    // cannon32 v1.3.1 (op-program).
    Claim disputeAbsolutePrestate = Claim.wrap(0x038512e02c4c3f7bdaec27d00edf55b7155e0905301e1a88083e4e0a6764d54c);
    Hash startingAnchorRoot = Hash.wrap(Constants.PLACEHOLDER_STARTING_ANCHOR_ROOT);
    // cannon64 v1.6.1 (op-program).
    Claim cannonAbsolutePrestate = Claim.wrap(0x03eb07101fbdeaf3f04d9fb76526362c1eea2824e4c6e970bdb19675b72e4fc8);
    // cannon64-kona-interop v1.2.13 (Kona).
    Claim cannonKonaAbsolutePrestate = Claim.wrap(0x035ef680a6fa34c50d8d8169075b5d133ecd7b38fe2b2a83cc76fc81ae5d7c52);
    // Arbitrary non-placeholder anchor root for the permissionless deploy tests.
    Hash permissionlessAnchorRoot = Hash.wrap(0x02f4397b2de6fce03b3f9982378c2b4c4deff9c92c662dcc6f9643267aeb5e47);
    uint256 disputeMaxGameDepth = 73;
    uint256 disputeSplitDepth = 30;
    Duration disputeClockExtension = Duration.wrap(3 hours);
    Duration disputeMaxClockDuration = Duration.wrap(3.5 days);
    address opcmAddr;
    ISuperchainConfig superchainConfig;
    bool useCustomGasToken = false;

    event Deployed(uint256 indexed l2ChainId, address indexed deployer, bytes deployOutput);

    function setUp() public virtual {
        resolveFeaturesFromEnv();
        deploySuperchain = new DeploySuperchain();
        deployImplementations = new DeployImplementations();
        deployOPChain = new DeployOPChain();

        // 1) DeploySuperchain
        DeploySuperchain.Output memory dso = deploySuperchain.run(
            DeploySuperchain.Input({
                superchainProxyAdminOwner: superchainProxyAdminOwner,
                guardian: guardian,
                paused: paused
            })
        );

        // 2) DeployImplementations (produces OPCM)
        DeployImplementations.Output memory dio = deployImplementations.run(
            DeployImplementations.Input({
                withdrawalDelaySeconds: withdrawalDelaySeconds,
                minProposalSizeBytes: minProposalSizeBytes,
                challengePeriodSeconds: challengePeriodSeconds,
                proofMaturityDelaySeconds: proofMaturityDelaySeconds,
                disputeGameFinalityDelaySeconds: disputeGameFinalityDelaySeconds,
                mipsVersion: StandardConstants.MIPS_VERSION,
                faultGameV2MaxGameDepth: 73,
                faultGameV2SplitDepth: 30,
                faultGameV2ClockExtension: 10800,
                faultGameV2MaxClockDuration: 302400,
                superchainConfigProxy: dso.superchainConfigProxy,
                superchainProxyAdmin: dso.superchainProxyAdmin,
                l1ProxyAdminOwner: dso.superchainProxyAdmin.owner(),
                challenger: challenger,
                devFeatureBitmap: devFeatureBitmap
            })
        );
        opcmAddr = address(dio.opcmV2);
        vm.label(address(dio.opcmV2), "opcmV2");

        // Set superchainConfig from deployment
        superchainConfig = dso.superchainConfigProxy;

        // 3) Build DeployOPChainInput struct
        deployOPChainInput = Types.DeployOPChainInput({
            opChainProxyAdminOwner: opChainProxyAdminOwner,
            systemConfigOwner: systemConfigOwner,
            batcher: batcher,
            unsafeBlockSigner: unsafeBlockSigner,
            proposer: proposer,
            challenger: challenger,
            basefeeScalar: basefeeScalar,
            blobBaseFeeScalar: blobBaseFeeScalar,
            l2ChainId: l2ChainId,
            opcm: opcmAddr,
            saltMixer: saltMixer,
            gasLimit: gasLimit,
            disputeGameType: disputeGameType,
            disputeAbsolutePrestate: disputeAbsolutePrestate,
            startingAnchorRoot: startingAnchorRoot,
            cannonAbsolutePrestate: cannonAbsolutePrestate,
            disputeMaxGameDepth: disputeMaxGameDepth,
            disputeSplitDepth: disputeSplitDepth,
            disputeClockExtension: disputeClockExtension,
            disputeMaxClockDuration: disputeMaxClockDuration,
            allowCustomDisputeParameters: false,
            operatorFeeScalar: 0,
            operatorFeeConstant: 0,
            superchainConfig: superchainConfig,
            useCustomGasToken: useCustomGasToken
        });
    }
}

contract DeployOPChain_Test is DeployOPChain_TestBase {
    function hash(bytes32 _seed, uint256 _i) internal pure returns (bytes32) {
        return keccak256(abi.encode(_seed, _i));
    }

    function test_run_succeeds() public {
        DeployOPChain.Output memory doo = deployOPChain.run(deployOPChainInput);
        // Basic non-zero and code checks are covered inside run->checkOutput.
        // Additional targeted assertions added below.
        _checkDeploymentAssertions(doo);
    }

    function testFuzz_run_memory_succeeds(bytes32 _seed) public {
        deployOPChainInput.opChainProxyAdminOwner = address(uint160(uint256(hash(_seed, 0))));
        deployOPChainInput.systemConfigOwner = address(uint160(uint256(hash(_seed, 1))));
        deployOPChainInput.batcher = address(uint160(uint256(hash(_seed, 2))));
        deployOPChainInput.unsafeBlockSigner = address(uint160(uint256(hash(_seed, 3))));
        deployOPChainInput.proposer = address(uint160(uint256(hash(_seed, 4))));
        deployOPChainInput.challenger = address(uint160(uint256(hash(_seed, 5))));
        deployOPChainInput.basefeeScalar = uint32(uint256(hash(_seed, 6)));
        deployOPChainInput.blobBaseFeeScalar = uint32(uint256(hash(_seed, 7)));
        deployOPChainInput.l2ChainId = uint256(hash(_seed, 8));
        deployOPChainInput.useCustomGasToken = uint256(hash(_seed, 9)) % 2 == 1;

        DeployOPChain.Output memory doo = deployOPChain.run(deployOPChainInput);

        // Check dispute game deployments
        // Validate permissionedDisputeGame (PDG) address
        GameType permGameType = isDevFeatureEnabled(DevFeatures.SUPER_ROOT_GAMES_MIGRATION)
            ? GameTypes.SUPER_PERMISSIONED
            : GameTypes.PERMISSIONED_CANNON;
        IOPContractsManagerContainer.Implementations memory impls = IOPContractsManagerV2(opcmAddr).implementations();
        address expectedPDGAddress = isDevFeatureEnabled(DevFeatures.SUPER_ROOT_GAMES_MIGRATION)
            ? impls.superPermissionedDisputeGameImpl
            : impls.permissionedDisputeGameImpl;
        address actualPDGAddress = address(doo.disputeGameFactoryProxy.gameImpls(permGameType));
        assertNotEq(actualPDGAddress, address(0), "PDG address should be non-zero");
        assertEq(actualPDGAddress, expectedPDGAddress, "PDG address should match expected address");

        // Verify custom gas token feature is set as seeded
        assertEq(
            doo.systemConfigProxy.isCustomGasToken(),
            deployOPChainInput.useCustomGasToken,
            "SystemConfig isCustomGasToken (fuzz)"
        );
        assertEq(
            doo.systemConfigProxy.isFeatureEnabled(Features.CUSTOM_GAS_TOKEN),
            deployOPChainInput.useCustomGasToken,
            "SystemConfig CUSTOM_GAS_TOKEN feature (fuzz)"
        );
    }

    function test_customGasToken_enabled_succeeds() public {
        deployOPChainInput.useCustomGasToken = true;
        DeployOPChain.Output memory doo = deployOPChain.run(deployOPChainInput);

        assertEq(doo.systemConfigProxy.isCustomGasToken(), true, "SystemConfig isCustomGasToken should be true");
        assertEq(
            doo.systemConfigProxy.isFeatureEnabled(Features.CUSTOM_GAS_TOKEN),
            true,
            "SystemConfig CUSTOM_GAS_TOKEN feature should be true"
        );
    }

    function getPermissionedDisputeGame(DeployOPChain.Output memory doo)
        internal
        view
        returns (IPermissionedDisputeGame)
    {
        GameType permGameType = isDevFeatureEnabled(DevFeatures.SUPER_ROOT_GAMES_MIGRATION)
            ? GameTypes.SUPER_PERMISSIONED
            : GameTypes.PERMISSIONED_CANNON;
        return IPermissionedDisputeGame(address(doo.disputeGameFactoryProxy.gameImpls(permGameType)));
    }

    function test_runWithBytes_succeeds() public {
        bytes memory inputBytes = abi.encode(deployOPChainInput);
        bytes memory outputBytes = deployOPChain.runWithBytes(inputBytes);
        DeployOPChain.Output memory doo = abi.decode(outputBytes, (DeployOPChain.Output));

        // covers basic non-zero and code checks are covered inside run->checkOutput.
        _checkDeploymentAssertions(doo);
    }

    /// @notice Legacy CANNON is rejected as an initial deployment game type.
    function test_run_cannonGameType_reverts() public {
        deployOPChainInput.disputeGameType = GameTypes.CANNON;

        vm.expectRevert("DeployOPChain: unsupported dispute game type");
        deployOPChain.run(deployOPChainInput);
    }

    /// @notice Non-super-root CANNON_KONA deploys respect CANNON_KONA and register
    ///         PERMISSIONED_CANNON for guardian fallback.
    function test_run_cannonKonaGameType_succeeds() public {
        skipIfDevFeatureEnabled(DevFeatures.SUPER_ROOT_GAMES_MIGRATION);
        deployOPChainInput.disputeGameType = GameTypes.CANNON_KONA;
        deployOPChainInput.disputeAbsolutePrestate = cannonKonaAbsolutePrestate;
        deployOPChainInput.startingAnchorRoot = permissionlessAnchorRoot;

        DeployOPChain.Output memory doo = deployOPChain.run(deployOPChainInput);
        _checkCannonKonaPermissionlessDeployment(doo);
        _validateCannonKonaPermissionlessDeployment(doo);
    }

    /// @notice Verifies the guardian can switch a CANNON_KONA deploy to PERMISSIONED_CANNON
    ///         and the trusted proposer can create a respected fallback game.
    function test_run_cannonKonaGameTypeFallback_succeeds() public {
        skipIfDevFeatureEnabled(DevFeatures.SUPER_ROOT_GAMES_MIGRATION);
        deployOPChainInput.disputeGameType = GameTypes.CANNON_KONA;
        deployOPChainInput.disputeAbsolutePrestate = cannonKonaAbsolutePrestate;
        deployOPChainInput.startingAnchorRoot = permissionlessAnchorRoot;
        DeployOPChain.Output memory doo = deployOPChain.run(deployOPChainInput);

        IAnchorStateRegistry asr = doo.anchorStateRegistryProxy;
        vm.prank(doo.systemConfigProxy.guardian());
        asr.setRespectedGameType(GameTypes.PERMISSIONED_CANNON);

        uint256 bond = doo.disputeGameFactoryProxy.initBonds(GameTypes.PERMISSIONED_CANNON);
        vm.deal(proposer, bond);
        vm.prank(proposer, proposer);
        IDisputeGame game = doo.disputeGameFactoryProxy.create{ value: bond }(
            GameTypes.PERMISSIONED_CANNON, Claim.wrap(keccak256("fallback proposal")), abi.encode(uint256(1))
        );
        assertTrue(asr.isGameRespected(game), "fallback game must be respected");
    }

    /// @notice checkOutput rejects a permissionless deployment output missing the permissioned fallback.
    function test_checkOutput_missingPermissionedFallback_reverts() public {
        skipIfDevFeatureEnabled(DevFeatures.SUPER_ROOT_GAMES_MIGRATION);
        deployOPChainInput.disputeGameType = GameTypes.CANNON_KONA;
        deployOPChainInput.disputeAbsolutePrestate = cannonKonaAbsolutePrestate;
        deployOPChainInput.startingAnchorRoot = permissionlessAnchorRoot;
        DeployOPChain.Output memory doo = deployOPChain.run(deployOPChainInput);

        doo.permissionedDisputeGame = IPermissionedDisputeGame(address(0));
        vm.expectRevert("DeployOPChain: permissionedDisputeGame output mismatch");
        deployOPChain.checkOutput(deployOPChainInput, doo);
    }

    /// @notice Permissionless game types are rejected when super roots are enabled.
    function test_run_permissionlessGameTypeWithSuperRoot_reverts() public {
        skipIfDevFeatureDisabled(DevFeatures.SUPER_ROOT_GAMES_MIGRATION);
        deployOPChainInput.disputeGameType = GameTypes.CANNON_KONA;
        deployOPChainInput.disputeAbsolutePrestate = cannonKonaAbsolutePrestate;
        deployOPChainInput.startingAnchorRoot = permissionlessAnchorRoot;
        vm.expectRevert("DeployOPChain: permissionless game type not supported with super roots");
        deployOPChain.run(deployOPChainInput);
    }

    /// @notice Asserts non-super-root CANNON_KONA deploys register the permissioned fallback
    ///         with matching bond, prestate, proposer, and challenger.
    ///
    /// @param doo The deployment output.
    function _checkCannonKonaPermissionlessDeployment(DeployOPChain.Output memory doo) internal view {
        IOPContractsManagerContainer.Implementations memory impls = IOPContractsManagerV2(opcmAddr).implementations();
        assertEq(
            doo.disputeGameFactoryProxy.initBonds(GameTypes.CANNON_KONA),
            deployOPChain.DEFAULT_INIT_BOND(),
            "selected init bond"
        );
        assertEq(
            address(doo.disputeGameFactoryProxy.gameImpls(GameTypes.CANNON_KONA)),
            impls.faultDisputeGameImpl,
            "selected impl"
        );
        assertEq(address(doo.faultDisputeGame), impls.faultDisputeGameImpl, "output faultDisputeGame");
        assertEq(doo.disputeGameFactoryProxy.initBonds(GameTypes.CANNON), 0, "unselected init bond");
        assertEq(address(doo.disputeGameFactoryProxy.gameImpls(GameTypes.CANNON)), address(0), "unselected impl");
        assertEq(doo.disputeGameFactoryProxy.gameArgs(GameTypes.CANNON).length, 0, "unselected args");
        assertEq(
            doo.disputeGameFactoryProxy.initBonds(GameTypes.PERMISSIONED_CANNON),
            deployOPChain.DEFAULT_INIT_BOND(),
            "fallback init bond"
        );
        assertEq(
            address(doo.disputeGameFactoryProxy.gameImpls(GameTypes.PERMISSIONED_CANNON)),
            impls.permissionedDisputeGameImpl,
            "fallback impl"
        );
        assertEq(address(doo.permissionedDisputeGame), impls.permissionedDisputeGameImpl, "output fallback");

        assertEq(
            LibGameArgs.decode(doo.disputeGameFactoryProxy.gameArgs(GameTypes.CANNON_KONA)).absolutePrestate,
            deployOPChainInput.disputeAbsolutePrestate.raw(),
            "selected prestate wiring"
        );
        LibGameArgs.GameArgs memory pdgArgs =
            LibGameArgs.decode(doo.disputeGameFactoryProxy.gameArgs(GameTypes.PERMISSIONED_CANNON));
        assertEq(pdgArgs.absolutePrestate, deployOPChainInput.cannonAbsolutePrestate.raw(), "fallback prestate");
        assertEq(pdgArgs.proposer, proposer, "fallback proposer");
        assertEq(pdgArgs.challenger, challenger, "fallback challenger");

        IAnchorStateRegistry asr = doo.anchorStateRegistryProxy;
        assertEq(asr.respectedGameType().raw(), GameTypes.CANNON_KONA.raw(), "respected game type");
        Proposal memory anchor = asr.getStartingAnchorRoot();
        assertEq(anchor.root.raw(), deployOPChainInput.startingAnchorRoot.raw(), "anchor root");
        assertEq(anchor.l2SequenceNumber, 0, "anchor seq");
    }

    /// @notice Validates the full standard shape of a CANNON_KONA deployment with its permissioned fallback.
    /// @param doo The deployment output.
    function _validateCannonKonaPermissionlessDeployment(DeployOPChain.Output memory doo) internal view {
        IOPContractsManagerStandardValidator validator = IOPContractsManagerV2(opcmAddr).opcmStandardValidator();
        validator.validateWithOverrides(
            IOPContractsManagerStandardValidator.ValidationInputDev({
                sysCfg: doo.systemConfigProxy,
                cannonPrestate: cannonAbsolutePrestate.raw(),
                cannonKonaPrestate: cannonKonaAbsolutePrestate.raw(),
                l2ChainID: l2ChainId,
                proposer: proposer
            }),
            false,
            IOPContractsManagerStandardValidator.ValidationOverrides({
                l1PAOMultisig: opChainProxyAdminOwner,
                challenger: challenger
            })
        );
    }

    /// @notice Tests that faultDisputeGame is set to address(0) and permissionedDisputeGame is set to the correct
    /// implementation for GameTypes.PERMISSIONED_CANNON.
    function test_run_faultDisputeGamePermissionedCannon_succeeds() public {
        deployOPChainInput.disputeGameType = GameTypes.PERMISSIONED_CANNON;
        DeployOPChain.Output memory doo = deployOPChain.run(deployOPChainInput);

        GameType permType = isDevFeatureEnabled(DevFeatures.SUPER_ROOT_GAMES_MIGRATION)
            ? GameTypes.SUPER_PERMISSIONED
            : GameTypes.PERMISSIONED_CANNON;
        address expectedPermissioned = address(doo.disputeGameFactoryProxy.gameImpls(permType));
        assertEq(address(doo.permissionedDisputeGame), expectedPermissioned, "PDG impl");
        assertEq(address(doo.faultDisputeGame), address(0), "FDG should be set to address(0)");
    }

    /// @notice Checks for additional assertions that are not covered by the basic non-zero and code checks in
    /// `DeployOPChain.checkOutput`.
    /// @param doo The output of the deployment.
    function _checkDeploymentAssertions(DeployOPChain.Output memory doo) internal view {
        IPermissionedDisputeGame pdg = getPermissionedDisputeGame(doo);
        if (!isDevFeatureEnabled(DevFeatures.SUPER_ROOT_GAMES_MIGRATION)) {
            assertEq(pdg.splitDepth(), disputeSplitDepth, "PDG splitDepth");
            assertEq(pdg.maxGameDepth(), disputeMaxGameDepth, "PDG maxGameDepth");
            assertEq(
                Duration.unwrap(pdg.clockExtension()), Duration.unwrap(disputeClockExtension), "PDG clockExtension"
            );
            assertEq(
                Duration.unwrap(pdg.maxClockDuration()),
                Duration.unwrap(disputeMaxClockDuration),
                "PDG maxClockDuration"
            );
            assertEq(address(pdg.proposer()), address(0), "PDG proposer");
            assertEq(address(pdg.challenger()), address(0), "PDG challenger");
            assertEq(Claim.unwrap(pdg.absolutePrestate()), bytes32(0), "PDG absolutePrestate");
        }

        // Custom gas token feature should reflect input
        assertEq(doo.systemConfigProxy.isCustomGasToken(), useCustomGasToken, "SystemConfig isCustomGasToken");
        assertEq(
            doo.systemConfigProxy.isFeatureEnabled(Features.CUSTOM_GAS_TOKEN),
            useCustomGasToken,
            "SystemConfig CUSTOM_GAS_TOKEN feature"
        );

        // Verify superchainConfig is set correctly
        assertEq(
            address(doo.systemConfigProxy.superchainConfig()),
            address(deployOPChainInput.superchainConfig),
            "superchainConfig mismatch"
        );

        bool isSuperRoot = isDevFeatureEnabled(DevFeatures.SUPER_ROOT_GAMES_MIGRATION);
        GameType permType = isSuperRoot ? GameTypes.SUPER_PERMISSIONED : GameTypes.PERMISSIONED_CANNON;
        GameType konaType = isSuperRoot ? GameTypes.SUPER_CANNON_KONA : GameTypes.CANNON_KONA;

        // The legacy permissioned game keeps the default bond. The super permissioned game
        // has no bonded participation path, so its init bond must be zero.
        uint256 expectedInitBond = isSuperRoot ? 0 : deployOPChain.DEFAULT_INIT_BOND();
        assertEq(doo.disputeGameFactoryProxy.initBonds(permType), expectedInitBond);
        assertNotEq(address(doo.disputeGameFactoryProxy.gameImpls(permType)), address(0));

        // CANNON must be disabled for the default permissioned initial deployment.
        if (!isSuperRoot) {
            assertEq(doo.disputeGameFactoryProxy.initBonds(GameTypes.CANNON), 0, "CANNON init bond should be 0");
            assertEq(
                address(doo.disputeGameFactoryProxy.gameImpls(GameTypes.CANNON)),
                address(0),
                "CANNON impl should be the zero address"
            );
        }

        // Kona must be disabled for the default permissioned initial deployment.
        assertEq(doo.disputeGameFactoryProxy.initBonds(konaType), 0, "CANNON_KONA init bond should be 0");
        assertEq(
            address(doo.disputeGameFactoryProxy.gameImpls(konaType)),
            address(0),
            "CANNON_KONA impl should be the zero address"
        );

        IAnchorStateRegistry asr = doo.anchorStateRegistryProxy;
        assertEq(asr.respectedGameType().raw(), permType.raw(), "ASR respected game type");
        Proposal memory anchor = asr.getStartingAnchorRoot();
        assertEq(anchor.root.raw(), deployOPChainInput.startingAnchorRoot.raw(), "ASR anchor root");
        assertEq(anchor.l2SequenceNumber, 0, "ASR anchor seq");
    }
}

contract DeployOPChain_TestFail is DeployOPChain_TestBase {
    function test_run_zeroOpChainProxyAdminOwner_reverts() public {
        deployOPChainInput.opChainProxyAdminOwner = address(0);
        vm.expectRevert("DeployOPChainInput: opChainProxyAdminOwner not set");
        deployOPChain.run(deployOPChainInput);
    }

    function test_run_zeroSystemConfigOwner_reverts() public {
        deployOPChainInput.systemConfigOwner = address(0);
        vm.expectRevert("DeployOPChainInput: systemConfigOwner not set");
        deployOPChain.run(deployOPChainInput);
    }

    function test_run_zeroBatcher_reverts() public {
        deployOPChainInput.batcher = address(0);
        vm.expectRevert("DeployOPChainInput: batcher not set");
        deployOPChain.run(deployOPChainInput);
    }

    function test_run_zeroUnsafeBlockSigner_reverts() public {
        deployOPChainInput.unsafeBlockSigner = address(0);
        vm.expectRevert("DeployOPChainInput: unsafeBlockSigner not set");
        deployOPChain.run(deployOPChainInput);
    }

    function test_run_zeroProposer_reverts() public {
        deployOPChainInput.proposer = address(0);
        vm.expectRevert("DeployOPChainInput: proposer not set");
        deployOPChain.run(deployOPChainInput);
    }

    function test_run_zeroChallenger_reverts() public {
        deployOPChainInput.challenger = address(0);
        vm.expectRevert("DeployOPChainInput: challenger not set");
        deployOPChain.run(deployOPChainInput);
    }

    function test_run_zeroBasefeeScalar_reverts() public {
        deployOPChainInput.basefeeScalar = 0;
        vm.expectRevert("DeployOPChainInput: basefeeScalar not set");
        deployOPChain.run(deployOPChainInput);
    }

    function test_run_zeroBlobBaseFeeScalar_reverts() public {
        deployOPChainInput.blobBaseFeeScalar = 0;
        vm.expectRevert("DeployOPChainInput: blobBaseFeeScalar not set");
        deployOPChain.run(deployOPChainInput);
    }

    function test_run_zeroGasLimit_reverts() public {
        deployOPChainInput.gasLimit = 0;
        vm.expectRevert("DeployOPChainInput: gasLimit not set");
        deployOPChain.run(deployOPChainInput);
    }

    function test_run_zeroL2ChainId_reverts() public {
        deployOPChainInput.l2ChainId = 0;
        vm.expectRevert("DeployOPChainInput: l2ChainId not set");
        deployOPChain.run(deployOPChainInput);
    }

    function test_run_l2ChainIdMatchesBlockChainId_reverts() public {
        deployOPChainInput.l2ChainId = block.chainid;
        vm.expectRevert("DeployOPChainInput: l2ChainId matches block.chainid");
        deployOPChain.run(deployOPChainInput);
    }

    function test_run_zeroOpcm_reverts() public {
        deployOPChainInput.opcm = address(0);
        vm.expectRevert("DeployOPChainInput: opcm not set");
        deployOPChain.run(deployOPChainInput);
    }

    function test_run_invalidOpcmAddress_reverts() public {
        // It should revert if the opcm address is not a contract.
        address eoaAddress = makeAddr("EOA");
        deployOPChainInput.opcm = eoaAddress;
        // nosemgrep: sol-safety-expectrevert-no-args
        vm.expectRevert();
        deployOPChain.run(deployOPChainInput);
    }

    function test_run_zeroDisputeMaxGameDepth_reverts() public {
        deployOPChainInput.disputeMaxGameDepth = 0;
        vm.expectRevert("DeployOPChainInput: disputeMaxGameDepth not set");
        deployOPChain.run(deployOPChainInput);
    }

    function test_run_zeroDisputeSplitDepth_reverts() public {
        deployOPChainInput.disputeSplitDepth = 0;
        vm.expectRevert("DeployOPChainInput: disputeSplitDepth not set");
        deployOPChain.run(deployOPChainInput);
    }

    function test_run_zeroDisputeMaxClockDuration_reverts() public {
        deployOPChainInput.disputeMaxClockDuration = Duration.wrap(0);
        vm.expectRevert("DeployOPChainInput: disputeMaxClockDuration not set");
        deployOPChain.run(deployOPChainInput);
    }

    function test_run_zeroDisputeAbsolutePrestate_reverts() public {
        deployOPChainInput.disputeAbsolutePrestate = Claim.wrap(bytes32(0));
        vm.expectRevert("DeployOPChainInput: disputeAbsolutePrestate not set");
        deployOPChain.run(deployOPChainInput);
    }

    function test_run_cannonKonaZeroCannonAbsolutePrestate_reverts() public {
        skipIfDevFeatureEnabled(DevFeatures.SUPER_ROOT_GAMES_MIGRATION);
        deployOPChainInput.disputeGameType = GameTypes.CANNON_KONA;
        deployOPChainInput.disputeAbsolutePrestate = cannonKonaAbsolutePrestate;
        deployOPChainInput.startingAnchorRoot = permissionlessAnchorRoot;
        deployOPChainInput.cannonAbsolutePrestate = Claim.wrap(bytes32(0));
        vm.expectRevert("DeployOPChainInput: cannonAbsolutePrestate not set");
        deployOPChain.run(deployOPChainInput);
    }

    /// @notice The Cannon fallback prestate and the selected Kona prestate can never legitimately
    ///         be equal (they commit to different fault-proof programs).
    function test_run_cannonKonaEqualPrestates_reverts() public {
        skipIfDevFeatureEnabled(DevFeatures.SUPER_ROOT_GAMES_MIGRATION);
        deployOPChainInput.disputeGameType = GameTypes.CANNON_KONA;
        deployOPChainInput.disputeAbsolutePrestate = cannonKonaAbsolutePrestate;
        deployOPChainInput.startingAnchorRoot = permissionlessAnchorRoot;
        deployOPChainInput.cannonAbsolutePrestate = cannonKonaAbsolutePrestate;
        vm.expectRevert("DeployOPChainInput: cannonAbsolutePrestate must differ from disputeAbsolutePrestate");
        deployOPChain.run(deployOPChainInput);
    }

    function test_run_zeroStartingAnchorRoot_reverts() public {
        deployOPChainInput.startingAnchorRoot = Hash.wrap(bytes32(0));
        vm.expectRevert("DeployOPChainInput: startingAnchorRoot not set");
        deployOPChain.run(deployOPChainInput);
    }

    function test_run_permissionlessPlaceholderStartingAnchorRoot_reverts() public {
        skipIfDevFeatureEnabled(DevFeatures.SUPER_ROOT_GAMES_MIGRATION);
        deployOPChainInput.disputeGameType = GameTypes.CANNON_KONA;
        deployOPChainInput.disputeAbsolutePrestate = cannonKonaAbsolutePrestate;
        // startingAnchorRoot stays at the 0xdead placeholder default.
        vm.expectRevert("DeployOPChainInput: permissionless startingAnchorRoot cannot be placeholder");
        deployOPChain.run(deployOPChainInput);
    }

    function test_runWithBytes_invalidInput_reverts() public {
        // It should revert if the input bytes cannot be decoded.
        bytes memory invalidInput = "invalid";
        // nosemgrep: sol-safety-expectrevert-no-args
        vm.expectRevert();
        deployOPChain.runWithBytes(invalidInput);
    }

    function test_runWithBytes_emptyInput_reverts() public {
        bytes memory emptyInput = "";
        vm.expectRevert("DeployOPChain: input cannot be empty");
        deployOPChain.runWithBytes(emptyInput);
    }
}

contract DeployOPChain_GasLimit_Test is DeployOPChain_TestBase {
    /// @notice A gasLimit large enough to fit the default reserved gas should produce the
    ///         unchanged DEFAULT_RESOURCE_CONFIG. The boundary value is the sum of
    ///         default maxResourceLimit and systemTxMaxGas (currently 21M).
    function test_run_gasLimitAtDefaultThreshold_succeeds() public {
        IResourceMetering.ResourceConfig memory expected = Constants.DEFAULT_RESOURCE_CONFIG();
        deployOPChainInput.gasLimit = uint64(expected.maxResourceLimit) + uint64(expected.systemTxMaxGas);

        DeployOPChain.Output memory doo = deployOPChain.run(deployOPChainInput);
        IResourceMetering.ResourceConfig memory actual = doo.systemConfigProxy.resourceConfig();

        assertEq(actual.maxResourceLimit, expected.maxResourceLimit, "maxResourceLimit");
        assertEq(actual.systemTxMaxGas, expected.systemTxMaxGas, "systemTxMaxGas");
        assertEq(actual.elasticityMultiplier, expected.elasticityMultiplier, "elasticityMultiplier");
        assertEq(
            actual.baseFeeMaxChangeDenominator, expected.baseFeeMaxChangeDenominator, "baseFeeMaxChangeDenominator"
        );
        assertEq(actual.minimumBaseFee, expected.minimumBaseFee, "minimumBaseFee");
        assertEq(actual.maximumBaseFee, expected.maximumBaseFee, "maximumBaseFee");
    }

    /// @notice A 5M gasLimit (below the 21M default-reserved threshold) should produce a
    ///         scaled-down ResourceConfig where maxResourceLimit + systemTxMaxGas == gasLimit.
    function test_run_gasLimitFiveMillion_succeeds() public {
        deployOPChainInput.gasLimit = 5_000_000;

        DeployOPChain.Output memory doo = deployOPChain.run(deployOPChainInput);
        IResourceMetering.ResourceConfig memory actual = doo.systemConfigProxy.resourceConfig();
        IResourceMetering.ResourceConfig memory defaults = Constants.DEFAULT_RESOURCE_CONFIG();

        assertEq(actual.maxResourceLimit, 4_000_000, "maxResourceLimit scaled");
        assertEq(actual.systemTxMaxGas, defaults.systemTxMaxGas, "systemTxMaxGas preserved");
        assertEq(actual.elasticityMultiplier, defaults.elasticityMultiplier, "elasticityMultiplier preserved");
        assertEq(
            actual.baseFeeMaxChangeDenominator,
            defaults.baseFeeMaxChangeDenominator,
            "baseFeeMaxChangeDenominator preserved"
        );
        assertEq(actual.minimumBaseFee, defaults.minimumBaseFee, "minimumBaseFee preserved");
        assertEq(actual.maximumBaseFee, defaults.maximumBaseFee, "maximumBaseFee preserved");
        assertEq(doo.systemConfigProxy.gasLimit(), 5_000_000, "SystemConfig gasLimit");
        // Sanity: reserved gas exactly equals the requested gasLimit at the small-chain floor.
        assertEq(
            uint64(actual.maxResourceLimit) + uint64(actual.systemTxMaxGas),
            deployOPChainInput.gasLimit,
            "reserved gas == gasLimit"
        );
    }
}

contract DeployOPChain_GasLimit_TestFail is DeployOPChain_TestBase {
    /// @notice A gasLimit at or below the default systemTxMaxGas leaves no room for any
    ///         deposit budget and must revert at the deploy script with a clear message,
    ///         rather than failing deeper inside SystemConfig.
    function test_run_gasLimitBelowSystemTxMaxGas_reverts() public {
        IResourceMetering.ResourceConfig memory defaults = Constants.DEFAULT_RESOURCE_CONFIG();
        deployOPChainInput.gasLimit = uint64(defaults.systemTxMaxGas);
        vm.expectRevert("DeployOPChain: gasLimit must exceed systemTxMaxGas");
        deployOPChain.run(deployOPChainInput);
    }

    /// @notice A gasLimit only marginally above systemTxMaxGas rounds maxResourceLimit
    ///         down to zero (because of the elasticityMultiplier divisibility constraint)
    ///         and must revert before reaching SystemConfig.
    function test_run_gasLimitTooSmallForDeposits_reverts() public {
        IResourceMetering.ResourceConfig memory defaults = Constants.DEFAULT_RESOURCE_CONFIG();
        // available = 5 gas, which rounds down to 0 under elasticityMultiplier = 10.
        deployOPChainInput.gasLimit = uint64(defaults.systemTxMaxGas) + 5;
        vm.expectRevert("DeployOPChain: gasLimit too small for any deposit budget");
        deployOPChain.run(deployOPChainInput);
    }
}
