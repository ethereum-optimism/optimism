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
    address upgradeController = makeAddr("upgradeController");
    address proposer = makeAddr("proposer");
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
        address _superchainConfigImpl
    )
        public
    {
        vm.assume(_withdrawalDelaySeconds != 0);
        vm.assume(_minProposalSizeBytes != 0);
        vm.assume(_challengePeriodSeconds != 0);
        vm.assume(_proofMaturityDelaySeconds != 0);
        vm.assume(_disputeGameFinalityDelaySeconds != 0);
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

        DeployImplementations.Input memory input = DeployImplementations.Input(
            _withdrawalDelaySeconds,
            _minProposalSizeBytes,
            uint256(_challengePeriodSeconds),
            _proofMaturityDelaySeconds,
            _disputeGameFinalityDelaySeconds,
            StandardConstants.MIPS_VERSION, // mipsVersion
            bytes32(0), // devFeatureBitmap
            73, // faultGameV2MaxGameDepth
            30, // faultGameV2SplitDepth
            10800, // faultGameV2ClockExtension
            302400, // faultGameV2MaxClockDuration
            superchainConfigProxy,
            protocolVersionsProxy,
            superchainProxyAdmin,
            upgradeController,
            proposer,
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
        // V2 contracts should be null since default flag is false
        assertEq(address(output.faultDisputeGameV2Impl), address(0), "1200");
        assertEq(address(output.permissionedDisputeGameV2Impl), address(0), "1300");

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
        // V2 contracts should have empty code since they're null addresses
        assertEq(address(output.faultDisputeGameV2Impl).code, empty, "2400");
        assertEq(address(output.permissionedDisputeGameV2Impl).code, empty, "2500");

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
        input.upgradeController = address(0);
        vm.expectRevert("DeployImplementations: upgradeController not set");
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
            upgradeController,
            proposer,
            challenger
        );
    }
}
