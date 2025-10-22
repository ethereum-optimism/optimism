// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test, stdStorage, StdStorage } from "forge-std/Test.sol";

// Libraries
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { Chains } from "scripts/libraries/Chains.sol";
import { StandardConstants } from "scripts/deploy/StandardConstants.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";

// Interfaces
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IProtocolVersions } from "interfaces/L1/IProtocolVersions.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";

import { DeployImplementations } from "scripts/deploy/DeployImplementations.s.sol";

contract DeployImplementations_Test is Test {
    using stdStorage for StdStorage;

    DeployImplementations deployImplementations;

    // Define default inputs for testing.
    uint256 withdrawalDelaySeconds = 100;
    uint256 minProposalSizeBytes = 200;
    uint256 challengePeriodSeconds = 300;
    uint256 proofMaturityDelaySeconds = 400;
    uint256 disputeGameFinalityDelaySeconds = 500;
    ISuperchainConfig superchainConfigProxy = ISuperchainConfig(makeAddr("superchainConfigProxy"));
    IProtocolVersions protocolVersionsProxy = IProtocolVersions(makeAddr("protocolVersionsProxy"));
    IProxyAdmin superchainProxyAdmin = IProxyAdmin(makeAddr("superchainProxyAdmin"));
    address l1ProxyAdminOwner = makeAddr("l1ProxyAdminOwner");
    address challenger = makeAddr("challenger");

    function setUp() public virtual {
        // We'll need to store some code on these two addresses so that the deployment script checks pass
        vm.etch(address(superchainConfigProxy), hex"01");
        vm.etch(address(protocolVersionsProxy), hex"01");

        deployImplementations = new DeployImplementations();
    }

    function test_deployImplementation_succeeds() public {
        DeployImplementations.Input memory input = defaultInput();
        DeployImplementations.Output memory output = deployImplementations.run(input);

        assertNotEq(address(output.systemConfigImpl), address(0));
        // V2 contracts should not be deployed with default flag (false)
        assertEq(address(output.faultDisputeGameV2Impl), address(0));
        assertEq(address(output.permissionedDisputeGameV2Impl), address(0));
    }

    function test_reuseImplementation_succeeds() public {
        DeployImplementations.Input memory input = defaultInput();
        DeployImplementations.Output memory output1 = deployImplementations.run(input);
        DeployImplementations.Output memory output2 = deployImplementations.run(input);

        // Assert that the addresses did not change.
        assertEq(address(output1.systemConfigImpl), address(output2.systemConfigImpl), "100");
        assertEq(address(output1.l1CrossDomainMessengerImpl), address(output2.l1CrossDomainMessengerImpl), "200");
        assertEq(address(output1.l1ERC721BridgeImpl), address(output2.l1ERC721BridgeImpl), "300");
        assertEq(address(output1.l1StandardBridgeImpl), address(output2.l1StandardBridgeImpl), "400");
        assertEq(
            address(output1.optimismMintableERC20FactoryImpl), address(output2.optimismMintableERC20FactoryImpl), "500"
        );
        assertEq(address(output1.optimismPortalImpl), address(output2.optimismPortalImpl), "600");
        assertEq(address(output1.delayedWETHImpl), address(output2.delayedWETHImpl), "700");
        assertEq(address(output1.preimageOracleSingleton), address(output2.preimageOracleSingleton), "800");
        assertEq(address(output1.mipsSingleton), address(output2.mipsSingleton), "900");
        assertEq(address(output1.disputeGameFactoryImpl), address(output2.disputeGameFactoryImpl), "1000");
        assertEq(address(output1.anchorStateRegistryImpl), address(output2.anchorStateRegistryImpl), "1100");
        assertEq(address(output1.opcm), address(output2.opcm), "1200");
        assertEq(address(output1.ethLockboxImpl), address(output2.ethLockboxImpl), "1300");
        // V2 contracts should both be address(0) since default flag is false
        assertEq(address(output1.faultDisputeGameV2Impl), address(output2.faultDisputeGameV2Impl), "1400");
        assertEq(address(output1.permissionedDisputeGameV2Impl), address(output2.permissionedDisputeGameV2Impl), "1500");
        assertEq(address(output1.faultDisputeGameV2Impl), address(0), "V2 contracts should be null");
        assertEq(address(output1.permissionedDisputeGameV2Impl), address(0), "V2 contracts should be null");
    }

    function testFuzz_run_memory_succeeds(
        uint256 _withdrawalDelaySeconds,
        uint256 _minProposalSizeBytes,
        uint64 _challengePeriodSeconds,
        uint256 _proofMaturityDelaySeconds,
        uint256 _disputeGameFinalityDelaySeconds,
        address _superchainConfigImpl,
        uint256 _faultGameV2MaxGameDepth,
        uint256 _faultGameV2SplitDepth,
        uint256 _faultGameV2ClockExtension,
        uint256 _faultGameV2MaxClockDuration,
        bytes32 _devFeatureBitmap
    )
        public
    {
        _withdrawalDelaySeconds = bound(_withdrawalDelaySeconds, 1, type(uint256).max);
        _minProposalSizeBytes = bound(_minProposalSizeBytes, 1, 1000000);
        _challengePeriodSeconds = uint64(bound(uint256(_challengePeriodSeconds), 1, type(uint64).max));
        _proofMaturityDelaySeconds = bound(_proofMaturityDelaySeconds, 1, type(uint256).max);
        _disputeGameFinalityDelaySeconds = bound(_disputeGameFinalityDelaySeconds, 1, type(uint256).max);

        // Ensure superchainConfigImpl is not zero address
        vm.assume(_superchainConfigImpl != address(0));
        // Must configure the ProxyAdmin contract.

        superchainProxyAdmin = IProxyAdmin(
            DeployUtils.create1({
                _name: "ProxyAdmin",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IProxyAdmin.__constructor__, (msg.sender)))
            })
        );
        superchainConfigProxy = ISuperchainConfig(
            DeployUtils.create1({
                _name: "Proxy",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(IProxy.__constructor__, (address(superchainProxyAdmin)))
                )
            })
        );

        ISuperchainConfig superchainConfigImpl = ISuperchainConfig(_superchainConfigImpl);
        vm.prank(address(superchainProxyAdmin));
        IProxy(payable(address(superchainConfigProxy))).upgradeTo(address(superchainConfigImpl));

        _faultGameV2MaxGameDepth = bound(_faultGameV2MaxGameDepth, 4, 125);
        _faultGameV2SplitDepth =
            bound(_faultGameV2SplitDepth, 2, _faultGameV2MaxGameDepth > 3 ? _faultGameV2MaxGameDepth - 2 : 2);
        _faultGameV2ClockExtension = bound(_faultGameV2ClockExtension, 1, 7 days);
        _faultGameV2MaxClockDuration = bound(_faultGameV2MaxClockDuration, _faultGameV2ClockExtension * 2, 30 days);

        // When V2 is not enabled, set V2 params to 0 to match script expectations
        // Otherwise ensure they remain within bounds already set
        bool isV2Enabled = DevFeatures.isDevFeatureEnabled(_devFeatureBitmap, DevFeatures.DEPLOY_V2_DISPUTE_GAMES);
        if (!isV2Enabled) {
            _faultGameV2MaxGameDepth = 0;
            _faultGameV2SplitDepth = 0;
            _faultGameV2ClockExtension = 0;
            _faultGameV2MaxClockDuration = 0;
        }

        DeployImplementations.Input memory input = DeployImplementations.Input(
            _withdrawalDelaySeconds,
            _minProposalSizeBytes,
            uint256(_challengePeriodSeconds),
            _proofMaturityDelaySeconds,
            _disputeGameFinalityDelaySeconds,
            StandardConstants.MIPS_VERSION, // mipsVersion
            _devFeatureBitmap, // devFeatureBitmap (fuzzed)
            _faultGameV2MaxGameDepth, // faultGameV2MaxGameDepth (bounded)
            _faultGameV2SplitDepth, // faultGameV2SplitDepth (bounded)
            _faultGameV2ClockExtension, // faultGameV2ClockExtension (bounded)
            _faultGameV2MaxClockDuration, // faultGameV2MaxClockDuration (bounded)
            superchainConfigProxy,
            protocolVersionsProxy,
            superchainProxyAdmin,
            l1ProxyAdminOwner,
            challenger
        );

        DeployImplementations.Output memory output = deployImplementations.run(input);

        // Basic assertions
        assertNotEq(address(output.anchorStateRegistryImpl), address(0), "100");
        assertNotEq(address(output.delayedWETHImpl), address(0), "200");
        assertNotEq(address(output.disputeGameFactoryImpl), address(0), "300");
        assertNotEq(address(output.ethLockboxImpl), address(0), "400");
        assertNotEq(address(output.l1CrossDomainMessengerImpl), address(0), "500");
        assertNotEq(address(output.l1ERC721BridgeImpl), address(0), "500");
        assertNotEq(address(output.l1StandardBridgeImpl), address(0), "600");
        assertNotEq(address(output.mipsSingleton), address(0), "700");
        assertNotEq(address(output.opcm), address(0), "800");
        assertNotEq(address(output.opcmContractsContainer), address(0), "900");
        assertNotEq(address(output.opcmDeployer), address(0), "1000");
        assertNotEq(address(output.opcmGameTypeAdder), address(0), "1100");

        // Check V2 contracts based on feature flag
        bool v2Enabled = DevFeatures.isDevFeatureEnabled(_devFeatureBitmap, DevFeatures.DEPLOY_V2_DISPUTE_GAMES);
        if (v2Enabled) {
            assertNotEq(address(output.faultDisputeGameV2Impl), address(0), "V2 should be deployed when enabled");
            assertNotEq(address(output.permissionedDisputeGameV2Impl), address(0), "V2 should be deployed when enabled");

            // Verify V2 constructor parameters match fuzz inputs
            assertEq(output.faultDisputeGameV2Impl.maxGameDepth(), _faultGameV2MaxGameDepth, "FDGv2 maxGameDepth");
            assertEq(output.faultDisputeGameV2Impl.splitDepth(), _faultGameV2SplitDepth, "FDGv2 splitDepth");
            assertEq(
                output.faultDisputeGameV2Impl.clockExtension().raw(),
                uint64(_faultGameV2ClockExtension),
                "FDGv2 clockExtension"
            );
            assertEq(
                output.faultDisputeGameV2Impl.maxClockDuration().raw(),
                uint64(_faultGameV2MaxClockDuration),
                "FDGv2 maxClockDuration"
            );

            assertEq(
                output.permissionedDisputeGameV2Impl.maxGameDepth(), _faultGameV2MaxGameDepth, "PDGv2 maxGameDepth"
            );
            assertEq(output.permissionedDisputeGameV2Impl.splitDepth(), _faultGameV2SplitDepth, "PDGv2 splitDepth");
            assertEq(
                output.permissionedDisputeGameV2Impl.clockExtension().raw(),
                uint64(_faultGameV2ClockExtension),
                "PDGv2 clockExtension"
            );
            assertEq(
                output.permissionedDisputeGameV2Impl.maxClockDuration().raw(),
                uint64(_faultGameV2MaxClockDuration),
                "PDGv2 maxClockDuration"
            );
        } else {
            assertEq(address(output.faultDisputeGameV2Impl), address(0), "V2 should be null when disabled");
            assertEq(address(output.permissionedDisputeGameV2Impl), address(0), "V2 should be null when disabled");
        }

        // Address contents assertions
        bytes memory empty;

        assertNotEq(address(output.anchorStateRegistryImpl).code, empty, "1200");
        assertNotEq(address(output.delayedWETHImpl).code, empty, "1300");
        assertNotEq(address(output.disputeGameFactoryImpl).code, empty, "1400");
        assertNotEq(address(output.ethLockboxImpl).code, empty, "1500");
        assertNotEq(address(output.l1CrossDomainMessengerImpl).code, empty, "1600");
        assertNotEq(address(output.l1ERC721BridgeImpl).code, empty, "1700");
        assertNotEq(address(output.l1StandardBridgeImpl).code, empty, "1800");
        assertNotEq(address(output.mipsSingleton).code, empty, "1900");
        assertNotEq(address(output.opcm).code, empty, "2000");
        assertNotEq(address(output.opcmContractsContainer).code, empty, "2100");
        assertNotEq(address(output.opcmDeployer).code, empty, "2200");
        assertNotEq(address(output.opcmGameTypeAdder).code, empty, "2300");

        // V2 contracts code existence based on feature flag
        if (v2Enabled) {
            assertNotEq(address(output.faultDisputeGameV2Impl).code, empty, "V2 FDG should have code when enabled");
            assertNotEq(
                address(output.permissionedDisputeGameV2Impl).code, empty, "V2 PDG should have code when enabled"
            );
        } else {
            assertEq(address(output.faultDisputeGameV2Impl).code, empty, "V2 FDG should be empty when disabled");
            assertEq(address(output.permissionedDisputeGameV2Impl).code, empty, "V2 PDG should be empty when disabled");
        }

        // Architecture assertions.
        assertEq(address(output.mipsSingleton.oracle()), address(output.preimageOracleSingleton), "600");
    }

    function test_run_deployMipsV1OnMainnetOrSepolia_reverts() public {
        DeployImplementations.Input memory input = defaultInput();
        input.mipsVersion = 1;

        vm.chainId(Chains.Mainnet);
        vm.expectRevert("DeployImplementations: Only Mips64 should be deployed on Mainnet or Sepolia");
        deployImplementations.run(input);

        vm.chainId(Chains.Sepolia);
        vm.expectRevert("DeployImplementations: Only Mips64 should be deployed on Mainnet or Sepolia");
        deployImplementations.run(input);
    }

    function test_challengePeriodSeconds_valueTooLarge_reverts(uint256 _challengePeriodSeconds) public {
        vm.assume(_challengePeriodSeconds > uint256(type(uint64).max));

        DeployImplementations.Input memory input = defaultInput();
        input.challengePeriodSeconds = _challengePeriodSeconds;

        vm.expectRevert("DeployImplementations: challengePeriodSeconds too large");
        deployImplementations.run(input);
    }

    function test_run_nullInput_reverts() public {
        DeployImplementations.Input memory input;

        input = defaultInput();
        input.withdrawalDelaySeconds = 0;
        vm.expectRevert("DeployImplementations: withdrawalDelaySeconds not set");
        deployImplementations.run(input);

        input = defaultInput();
        input.minProposalSizeBytes = 0;
        vm.expectRevert("DeployImplementations: minProposalSizeBytes not set");
        deployImplementations.run(input);

        input = defaultInput();
        input.challengePeriodSeconds = 0;
        vm.expectRevert("DeployImplementations: challengePeriodSeconds not set");
        deployImplementations.run(input);

        input = defaultInput();
        input.proofMaturityDelaySeconds = 0;
        vm.expectRevert("DeployImplementations: proofMaturityDelaySeconds not set");
        deployImplementations.run(input);

        input = defaultInput();
        input.disputeGameFinalityDelaySeconds = 0;
        vm.expectRevert("DeployImplementations: disputeGameFinalityDelaySeconds not set");
        deployImplementations.run(input);

        input = defaultInput();
        input.mipsVersion = 0;
        vm.expectRevert("DeployImplementations: mipsVersion not set");
        deployImplementations.run(input);

        input = defaultInput();
        input.superchainConfigProxy = ISuperchainConfig(address(0));
        vm.expectRevert("DeployImplementations: superchainConfigProxy not set");
        deployImplementations.run(input);

        input = defaultInput();
        input.protocolVersionsProxy = IProtocolVersions(address(0));
        vm.expectRevert("DeployImplementations: protocolVersionsProxy not set");
        deployImplementations.run(input);

        input = defaultInput();
        input.superchainProxyAdmin = IProxyAdmin(address(0));
        vm.expectRevert("DeployImplementations: superchainProxyAdmin not set");
        deployImplementations.run(input);

        input = defaultInput();
        input.l1ProxyAdminOwner = address(0);
        vm.expectRevert("DeployImplementations: L1ProxyAdminOwner not set");
        deployImplementations.run(input);
    }

    function test_deployImplementation_withV2Enabled_succeeds() public {
        DeployImplementations.Input memory input = defaultInput();
        input.devFeatureBitmap = DevFeatures.DEPLOY_V2_DISPUTE_GAMES;
        DeployImplementations.Output memory output = deployImplementations.run(input);

        assertNotEq(address(output.faultDisputeGameV2Impl), address(0), "FaultDisputeGameV2 should be deployed");
        assertNotEq(
            address(output.permissionedDisputeGameV2Impl), address(0), "PermissionedDisputeGameV2 should be deployed"
        );

        // Validate constructor args for FaultDisputeGameV2
        assertEq(output.faultDisputeGameV2Impl.maxGameDepth(), 73, "FaultDisputeGameV2 maxGameDepth incorrect");
        assertEq(output.faultDisputeGameV2Impl.splitDepth(), 30, "FaultDisputeGameV2 splitDepth incorrect");
        assertEq(
            output.faultDisputeGameV2Impl.clockExtension().raw(), 10800, "FaultDisputeGameV2 clockExtension incorrect"
        );
        assertEq(
            output.faultDisputeGameV2Impl.maxClockDuration().raw(),
            302400,
            "FaultDisputeGameV2 maxClockDuration incorrect"
        );

        // Validate constructor args for PermissionedDisputeGameV2
        assertEq(
            output.permissionedDisputeGameV2Impl.maxGameDepth(), 73, "PermissionedDisputeGameV2 maxGameDepth incorrect"
        );
        assertEq(
            output.permissionedDisputeGameV2Impl.splitDepth(), 30, "PermissionedDisputeGameV2 splitDepth incorrect"
        );
        assertEq(
            output.permissionedDisputeGameV2Impl.clockExtension().raw(),
            10800,
            "PermissionedDisputeGameV2 clockExtension incorrect"
        );
        assertEq(
            output.permissionedDisputeGameV2Impl.maxClockDuration().raw(),
            302400,
            "PermissionedDisputeGameV2 maxClockDuration incorrect"
        );

        assertEq(
            output.opcm.blueprints().permissionedDisputeGame1,
            address(0),
            "PermissionedDisputeGame1 blueprint should not be deployed"
        );
        assertEq(
            output.opcm.blueprints().permissionedDisputeGame2,
            address(0),
            "PermissionedDisputeGame2 blueprint should not be deployed"
        );
        assertEq(
            output.opcm.blueprints().permissionlessDisputeGame1,
            address(0),
            "PermissionlessDisputeGame1 blueprint should not be deployed"
        );
        assertEq(
            output.opcm.blueprints().permissionlessDisputeGame1,
            address(0),
            "PermissionlessDisputeGame2 blueprint should not be deployed"
        );
    }

    function test_v2ParamsValidation_withFlagDisabled_succeeds() public {
        // When V2 flag is disabled, V2 params should be 0 or within safe bounds
        DeployImplementations.Input memory input = defaultInput();
        input.devFeatureBitmap = bytes32(0); // V2 disabled

        // Test that zero values are accepted
        input.faultGameV2MaxGameDepth = 0;
        input.faultGameV2SplitDepth = 0;
        input.faultGameV2ClockExtension = 0;
        input.faultGameV2MaxClockDuration = 0;

        DeployImplementations.Output memory output = deployImplementations.run(input);
        assertEq(address(output.faultDisputeGameV2Impl), address(0), "V2 FDG should be null when disabled");
        assertEq(address(output.permissionedDisputeGameV2Impl), address(0), "V2 PDG should be null when disabled");
    }

    function test_v2ParamsValidation_withHugeValues_reverts() public {
        // When V2 flag is enabled, huge V2 params should be rejected
        DeployImplementations.Input memory input = defaultInput();
        input.devFeatureBitmap = DevFeatures.DEPLOY_V2_DISPUTE_GAMES; // V2 enabled

        // Test that huge clock extension is rejected
        input.faultGameV2ClockExtension = type(uint256).max;
        vm.expectRevert("DeployImplementations: faultGameV2ClockExtension too large for uint64");
        deployImplementations.run(input);

        // Reset and test huge max clock duration
        input.faultGameV2ClockExtension = 100;
        input.faultGameV2MaxClockDuration = type(uint256).max;
        vm.expectRevert("DeployImplementations: faultGameV2MaxClockDuration too large for uint64");
        deployImplementations.run(input);

        // Reset and test huge max game depth
        input.faultGameV2MaxClockDuration = 200;
        input.faultGameV2MaxGameDepth = 300; // > 200
        vm.expectRevert("DeployImplementations: faultGameV2MaxGameDepth out of valid range (1-125)");
        deployImplementations.run(input);

        // Reset and test invalid split depth (too large, >= maxGameDepth)
        input.faultGameV2MaxGameDepth = 50;
        input.faultGameV2SplitDepth = 50; // splitDepth + 1 must be < maxGameDepth
        vm.expectRevert("DeployImplementations: faultGameV2SplitDepth must be >= 2 and splitDepth + 1 < maxGameDepth");
        deployImplementations.run(input);

        // Reset and test invalid split depth (too small, < 2)
        input.faultGameV2MaxGameDepth = 50;
        input.faultGameV2SplitDepth = 1; // < 2
        vm.expectRevert("DeployImplementations: faultGameV2SplitDepth must be >= 2 and splitDepth + 1 < maxGameDepth");
        deployImplementations.run(input);

        // Reset and test clock extension = 0 (must be > 0 when V2 enabled)
        input.faultGameV2SplitDepth = 10;
        input.faultGameV2ClockExtension = 0;
        input.faultGameV2MaxClockDuration = 1000;
        vm.expectRevert("DeployImplementations: faultGameV2ClockExtension must be > 0");
        deployImplementations.run(input);

        // Reset and test maxClockDuration < clockExtension
        input.faultGameV2ClockExtension = 1000;
        input.faultGameV2MaxClockDuration = 500; // < clockExtension
        vm.expectRevert("DeployImplementations: maxClockDuration must be >= clockExtension");
        deployImplementations.run(input);
    }

    function test_deployImplementation_withV2Disabled_succeeds() public {
        DeployImplementations.Input memory input = defaultInput();
        input.devFeatureBitmap = bytes32(0);
        DeployImplementations.Output memory output = deployImplementations.run(input);

        assertEq(address(output.faultDisputeGameV2Impl), address(0), "FaultDisputeGameV2 should not be deployed");
        assertEq(
            address(output.permissionedDisputeGameV2Impl),
            address(0),
            "PermissionedDisputeGameV2 should not be deployed"
        );

        // Ensure other contracts are still deployed
        assertNotEq(address(output.systemConfigImpl), address(0), "SystemConfig should still be deployed");
        assertNotEq(address(output.disputeGameFactoryImpl), address(0), "DisputeGameFactory should still be deployed");
    }

    function test_reuseImplementation_withV2Flags_succeeds() public {
        DeployImplementations.Input memory inputEnabled = defaultInput();
        inputEnabled.devFeatureBitmap = DevFeatures.DEPLOY_V2_DISPUTE_GAMES;
        DeployImplementations.Output memory output1 = deployImplementations.run(inputEnabled);

        DeployImplementations.Input memory inputDisabled = defaultInput();
        inputDisabled.devFeatureBitmap = bytes32(0);
        DeployImplementations.Output memory output2 = deployImplementations.run(inputDisabled);

        // V2 contracts should be different between enabled and disabled
        assertTrue(
            address(output1.faultDisputeGameV2Impl) != address(output2.faultDisputeGameV2Impl),
            "V2 addresses should differ between enabled/disabled"
        );
        assertTrue(
            address(output1.permissionedDisputeGameV2Impl) != address(output2.permissionedDisputeGameV2Impl),
            "V2 addresses should differ between enabled/disabled"
        );

        // Validate constructor args for FaultDisputeGameV2
        assertEq(output1.faultDisputeGameV2Impl.maxGameDepth(), 73, "FaultDisputeGameV2 maxGameDepth incorrect");
        assertEq(output1.faultDisputeGameV2Impl.splitDepth(), 30, "FaultDisputeGameV2 splitDepth incorrect");
        assertEq(
            output1.faultDisputeGameV2Impl.clockExtension().raw(), 10800, "FaultDisputeGameV2 clockExtension incorrect"
        );
        assertEq(
            output1.faultDisputeGameV2Impl.maxClockDuration().raw(),
            302400,
            "FaultDisputeGameV2 maxClockDuration incorrect"
        );

        // Validate constructor args for PermissionedDisputeGameV2
        assertEq(
            output1.permissionedDisputeGameV2Impl.maxGameDepth(), 73, "PermissionedDisputeGameV2 maxGameDepth incorrect"
        );
        assertEq(
            output1.permissionedDisputeGameV2Impl.splitDepth(), 30, "PermissionedDisputeGameV2 splitDepth incorrect"
        );
        assertEq(
            output1.permissionedDisputeGameV2Impl.clockExtension().raw(),
            10800,
            "PermissionedDisputeGameV2 clockExtension incorrect"
        );
        assertEq(
            output1.permissionedDisputeGameV2Impl.maxClockDuration().raw(),
            302400,
            "PermissionedDisputeGameV2 maxClockDuration incorrect"
        );

        // Other contracts should remain the same
        assertEq(
            address(output1.systemConfigImpl),
            address(output2.systemConfigImpl),
            "SystemConfig addresses should be the same"
        );
        assertEq(
            address(output1.disputeGameFactoryImpl),
            address(output2.disputeGameFactoryImpl),
            "DisputeGameFactory addresses should be the same"
        );

        // Running with same flags should produce same results
        DeployImplementations.Output memory output3 = deployImplementations.run(inputEnabled);
        assertEq(
            address(output1.faultDisputeGameV2Impl),
            address(output3.faultDisputeGameV2Impl),
            "V2 enabled addresses should be deterministic"
        );
        assertEq(
            address(output1.permissionedDisputeGameV2Impl),
            address(output3.permissionedDisputeGameV2Impl),
            "V2 enabled addresses should be deterministic"
        );
    }

    function defaultInput() private view returns (DeployImplementations.Input memory input_) {
        input_ = DeployImplementations.Input(
            withdrawalDelaySeconds,
            minProposalSizeBytes,
            challengePeriodSeconds,
            proofMaturityDelaySeconds,
            disputeGameFinalityDelaySeconds,
            StandardConstants.MIPS_VERSION, // mipsVersion
            bytes32(0), // devFeatureBitmap
            73, // faultGameV2MaxGameDepth
            30, // faultGameV2SplitDepth
            10800, // faultGameV2ClockExtension
            302400, // faultGameV2MaxClockDuration
            superchainConfigProxy,
            protocolVersionsProxy,
            superchainProxyAdmin,
            l1ProxyAdminOwner,
            challenger
        );
    }
}
