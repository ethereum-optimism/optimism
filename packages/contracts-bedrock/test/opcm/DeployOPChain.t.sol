// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { FeatureFlags } from "test/setup/FeatureFlags.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";

import { DeploySuperchain } from "scripts/deploy/DeploySuperchain.s.sol";
import { DeployImplementations } from "scripts/deploy/DeployImplementations.s.sol";
import { DeployOPChain } from "scripts/deploy/DeployOPChain.s.sol";
import { StandardConstants } from "scripts/deploy/StandardConstants.sol";
import { Types } from "scripts/libraries/Types.sol";

import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { Claim, Duration, GameType, GameTypes } from "src/dispute/lib/Types.sol";
import { IPermissionedDisputeGame } from "interfaces/dispute/IPermissionedDisputeGame.sol";

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
    IOPContractsManager opcm;

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
        opcm = dio.opcm;
        vm.label(address(opcm), "opcm");

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
            opcm: address(opcm),
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
            operatorFeeConstant: 0
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

        IPermissionedDisputeGame pdg = getPermissionedDisputeGame(doo);
        assertEq(pdg.splitDepth(), disputeSplitDepth, "PDG splitDepth");
        assertEq(pdg.maxGameDepth(), disputeMaxGameDepth, "PDG maxGameDepth");
        assertEq(Duration.unwrap(pdg.clockExtension()), Duration.unwrap(disputeClockExtension), "PDG clockExtension");
        assertEq(
            Duration.unwrap(pdg.maxClockDuration()), Duration.unwrap(disputeMaxClockDuration), "PDG maxClockDuration"
        );

        if (isDevFeatureEnabled(DevFeatures.DEPLOY_V2_DISPUTE_GAMES)) {
            // For v2 contracts, some immutable args are passed in at game creation time from DGF.gameArgs
            assertEq(address(pdg.proposer()), address(0), "PDG proposer");
            assertEq(address(pdg.challenger()), address(0), "PDG challenger");
            assertEq(Claim.unwrap(pdg.absolutePrestate()), bytes32(0), "PDG absolutePrestate");
        } else {
            assertEq(address(pdg.proposer()), proposer, "PDG proposer");
            assertEq(address(pdg.challenger()), challenger, "PDG challenger");
            assertEq(
                Claim.unwrap(pdg.absolutePrestate()), Claim.unwrap(disputeAbsolutePrestate), "PDG absolutePrestate"
            );
        }
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

        DeployOPChain.Output memory doo = deployOPChain.run(deployOPChainInput);

        // Verify that the initial bonds are zero.
        assertEq(doo.disputeGameFactoryProxy.initBonds(GameTypes.CANNON), 0, "2700");
        assertEq(doo.disputeGameFactoryProxy.initBonds(GameTypes.PERMISSIONED_CANNON), 0, "2800");

        // Check dispute game deployments
        // Validate permissionedDisputeGame (PDG) address
        bool isDeployV2Games = isDevFeatureEnabled(DevFeatures.DEPLOY_V2_DISPUTE_GAMES);
        IOPContractsManager.Implementations memory impls = opcm.implementations();
        address expectedPDGAddress =
            isDeployV2Games ? impls.permissionedDisputeGameV2Impl : address(doo.permissionedDisputeGame);
        address actualPDGAddress = address(doo.disputeGameFactoryProxy.gameImpls(GameTypes.PERMISSIONED_CANNON));
        assertNotEq(actualPDGAddress, address(0), "PDG address should be non-zero");
        assertEq(actualPDGAddress, expectedPDGAddress, "PDG address should match expected address");

        // Check PDG getters
        IPermissionedDisputeGame pdg = IPermissionedDisputeGame(actualPDGAddress);
        bytes32 expectedPrestate =
            isDeployV2Games ? bytes32(0) : bytes32(0x038512e02c4c3f7bdaec27d00edf55b7155e0905301e1a88083e4e0a6764d54c);
        assertEq(pdg.l2BlockNumber(), 0, "3000");
        assertEq(Claim.unwrap(pdg.absolutePrestate()), expectedPrestate, "3100");
        assertEq(Duration.unwrap(pdg.clockExtension()), 10800, "3200");
        assertEq(Duration.unwrap(pdg.maxClockDuration()), 302400, "3300");
        assertEq(pdg.splitDepth(), 30, "3400");
        assertEq(pdg.maxGameDepth(), 73, "3500");
    }

    function test_customDisputeGame_customEnabled_succeeds() public {
        // For v2 games, these parameters have already been configured at OPCM deploy time
        skipIfDevFeatureEnabled(DevFeatures.DEPLOY_V2_DISPUTE_GAMES);

        deployOPChainInput.allowCustomDisputeParameters = true;
        deployOPChainInput.disputeSplitDepth = disputeSplitDepth + 1;
        DeployOPChain.Output memory doo = deployOPChain.run(deployOPChainInput);

        IPermissionedDisputeGame pdg = getPermissionedDisputeGame(doo);
        assertEq(pdg.splitDepth(), disputeSplitDepth + 1);
    }

    function getPermissionedDisputeGame(DeployOPChain.Output memory doo)
        internal
        view
        returns (IPermissionedDisputeGame)
    {
        return IPermissionedDisputeGame(address(doo.disputeGameFactoryProxy.gameImpls(GameTypes.PERMISSIONED_CANNON)));
    }
}
