// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";
import { FeatureFlags } from "test/setup/FeatureFlags.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";

// Scripts
import { DeploySuperchain } from "scripts/deploy/DeploySuperchain.s.sol";
import { DeployImplementations } from "scripts/deploy/DeployImplementations.s.sol";
import { DeployOPChain } from "scripts/deploy/DeployOPChain.s.sol";
import { StandardConstants } from "scripts/deploy/StandardConstants.sol";
import { Types } from "scripts/libraries/Types.sol";

// Libraries
import { Features } from "src/libraries/Features.sol";

// Interfaces
import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { Claim, Duration, GameType, GameTypes } from "src/dispute/lib/Types.sol";
import { IPermissionedDisputeGame } from "interfaces/dispute/IPermissionedDisputeGame.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";

contract DeployOPChain_TestBase is Test, FeatureFlags {
    DeploySuperchain deploySuperchain;
    DeployImplementations deployImplementations;
    DeployOPChain deployOPChain;
    Types.DeployOPChainInput deployOPChainInput;

    // DeploySuperchain default inputs.
    address superchainProxyAdminOwner = makeAddr("superchainProxyAdminOwner");
    address protocolVersionsOwner = makeAddr("protocolVersionsOwner");
    address guardian = makeAddr("guardian");
    bool paused = false;
    bytes32 requiredProtocolVersion = bytes32(uint256(1));
    bytes32 recommendedProtocolVersion = bytes32(uint256(2));

    // DeployImplementations default inputs.
    // - superchainConfigProxy and protocolVersionsProxy are set during `setUp` since they are
    //   outputs of DeploySuperchain.
    uint256 withdrawalDelaySeconds = 100;
    uint256 minProposalSizeBytes = 200;
    uint256 challengePeriodSeconds = 300;
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
    Claim disputeAbsolutePrestate = Claim.wrap(0x038512e02c4c3f7bdaec27d00edf55b7155e0905301e1a88083e4e0a6764d54c);
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
                protocolVersionsOwner: protocolVersionsOwner,
                guardian: guardian,
                paused: paused,
                requiredProtocolVersion: requiredProtocolVersion,
                recommendedProtocolVersion: recommendedProtocolVersion
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
                protocolVersionsProxy: dso.protocolVersionsProxy,
                superchainProxyAdmin: dso.superchainProxyAdmin,
                l1ProxyAdminOwner: dso.superchainProxyAdmin.owner(),
                challenger: challenger,
                devFeatureBitmap: devFeatureBitmap
            })
        );
        // Select OPCM v1 or v2 based on feature flag
        opcmAddr = isDevFeatureEnabled(DevFeatures.OPCM_V2) ? address(dio.opcmV2) : address(dio.opcm);
        vm.label(address(dio.opcm), "opcm");
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
        // Additonal targeted assertions added below.
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

        // Skip init bond checks for OPCM v2 (bonds are set during deployment, not zero)
        if (!isDevFeatureEnabled(DevFeatures.OPCM_V2)) {
            // Verify that the initial bonds are zero for OPCM v1.
            assertEq(doo.disputeGameFactoryProxy.initBonds(GameTypes.CANNON), 0, "2700");
            assertEq(doo.disputeGameFactoryProxy.initBonds(GameTypes.PERMISSIONED_CANNON), 0, "2800");
        }

        // Check dispute game deployments
        // Validate permissionedDisputeGame (PDG) address
        IOPContractsManager.Implementations memory impls = IOPContractsManager(opcmAddr).implementations();
        address expectedPDGAddress = impls.permissionedDisputeGameImpl;
        address actualPDGAddress = address(doo.disputeGameFactoryProxy.gameImpls(GameTypes.PERMISSIONED_CANNON));
        assertNotEq(actualPDGAddress, address(0), "PDG address should be non-zero");
        assertEq(actualPDGAddress, expectedPDGAddress, "PDG address should match expected address");

        // Skip PDG getter checks for OPCM v2 (game args are passed at creation time)
        if (!isDevFeatureEnabled(DevFeatures.OPCM_V2)) {
            // Check PDG getters
            IPermissionedDisputeGame pdg = IPermissionedDisputeGame(actualPDGAddress);
            bytes32 expectedPrestate = bytes32(0);
            assertEq(pdg.l2BlockNumber(), 0, "3000");
            assertEq(Claim.unwrap(pdg.absolutePrestate()), expectedPrestate, "3100");
            assertEq(Duration.unwrap(pdg.clockExtension()), 10800, "3200");
            assertEq(Duration.unwrap(pdg.maxClockDuration()), 302400, "3300");
            assertEq(pdg.splitDepth(), 30, "3400");
            assertEq(pdg.maxGameDepth(), 73, "3500");
        }

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
        return IPermissionedDisputeGame(address(doo.disputeGameFactoryProxy.gameImpls(GameTypes.PERMISSIONED_CANNON)));
    }

    function test_runWithBytes_succeeds() public {
        bytes memory inputBytes = abi.encode(deployOPChainInput);
        bytes memory outputBytes = deployOPChain.runWithBytes(inputBytes);
        DeployOPChain.Output memory doo = abi.decode(outputBytes, (DeployOPChain.Output));

        // covers basic non-zero and code checks are covered inside run->checkOutput.
        _checkDeploymentAssertions(doo);
    }

    function test_run_cannonGameType_reverts() public {
        skipIfDevFeatureDisabled(DevFeatures.OPCM_V2);

        deployOPChainInput.disputeGameType = GameTypes.CANNON;
        vm.expectRevert("DeployOPChain: only PERMISSIONED_CANNON game type is supported for initial deployment");
        deployOPChain.run(deployOPChainInput);
    }

    function test_run_cannonKonaGameType_reverts() public {
        skipIfDevFeatureDisabled(DevFeatures.OPCM_V2);

        deployOPChainInput.disputeGameType = GameTypes.CANNON_KONA;
        vm.expectRevert("DeployOPChain: only PERMISSIONED_CANNON game type is supported for initial deployment");
        deployOPChain.run(deployOPChainInput);
    }

    /// @notice Tests that faultDisputeGame is set to address(0) and permissionedDisputeGame is set to the correct
    /// implementation for GameTypes.PERMISSIONED_CANNON.
    function test_run_faultDisputeGamePermissionedCannon_succeeds() public {
        skipIfDevFeatureDisabled(DevFeatures.OPCM_V2);

        deployOPChainInput.disputeGameType = GameTypes.PERMISSIONED_CANNON;
        DeployOPChain.Output memory doo = deployOPChain.run(deployOPChainInput);

        address expectedPermissioned = address(doo.disputeGameFactoryProxy.gameImpls(GameTypes.PERMISSIONED_CANNON));
        assertEq(address(doo.permissionedDisputeGame), expectedPermissioned, "PDG impl");
        assertEq(address(doo.faultDisputeGame), address(0), "FDG should be set to address(0)");
    }

    /// @notice Checks for additional assertions that are not covered by the basic non-zero and code checks in
    /// `DeployOPChain.checkOutput`.
    /// @param doo The output of the deployment.
    function _checkDeploymentAssertions(DeployOPChain.Output memory doo) internal view {
        IPermissionedDisputeGame pdg = getPermissionedDisputeGame(doo);
        assertEq(pdg.splitDepth(), disputeSplitDepth, "PDG splitDepth");
        assertEq(pdg.maxGameDepth(), disputeMaxGameDepth, "PDG maxGameDepth");
        assertEq(Duration.unwrap(pdg.clockExtension()), Duration.unwrap(disputeClockExtension), "PDG clockExtension");
        assertEq(
            Duration.unwrap(pdg.maxClockDuration()), Duration.unwrap(disputeMaxClockDuration), "PDG maxClockDuration"
        );

        // For v2 contracts, some immutable args are passed in at game creation time from DGF.gameArgs
        assertEq(address(pdg.proposer()), address(0), "PDG proposer");
        assertEq(address(pdg.challenger()), address(0), "PDG challenger");
        assertEq(Claim.unwrap(pdg.absolutePrestate()), bytes32(0), "PDG absolutePrestate");

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

        // OPCM v2 specific assertions
        if (isDevFeatureEnabled(DevFeatures.OPCM_V2)) {
            // PERMISSIONED_CANNON must always be enabled with DEFAULT_INIT_BOND init bond
            assertEq(
                doo.disputeGameFactoryProxy.initBonds(GameTypes.PERMISSIONED_CANNON), deployOPChain.DEFAULT_INIT_BOND()
            );
            assertNotEq(address(doo.disputeGameFactoryProxy.gameImpls(GameTypes.PERMISSIONED_CANNON)), address(0));

            // CANNON must be disabled for initial deployment
            assertEq(doo.disputeGameFactoryProxy.initBonds(GameTypes.CANNON), 0, "CANNON init bond should be 0");
            assertEq(
                address(doo.disputeGameFactoryProxy.gameImpls(GameTypes.CANNON)),
                address(0),
                "CANNON impl should be the zero address"
            );

            // CANNON_KONA must be disabled for initial deployment
            assertEq(
                doo.disputeGameFactoryProxy.initBonds(GameTypes.CANNON_KONA), 0, "CANNON_KONA init bond should be 0"
            );
            assertEq(
                address(doo.disputeGameFactoryProxy.gameImpls(GameTypes.CANNON_KONA)),
                address(0),
                "CANNON_KONA impl should be the zero address"
            );
        }
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
