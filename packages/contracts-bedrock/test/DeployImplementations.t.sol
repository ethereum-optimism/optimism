// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";

import { DelayedWETH } from "src/dispute/weth/DelayedWETH.sol";
import { PreimageOracle } from "src/cannon/PreimageOracle.sol";
import { MIPS } from "src/cannon/MIPS.sol";
import { DisputeGameFactory } from "src/dispute/DisputeGameFactory.sol";

import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { ProtocolVersions } from "src/L1/ProtocolVersions.sol";
import { OPStackManager } from "src/L1/OPStackManager.sol";
import { OptimismPortal2 } from "src/L1/OptimismPortal2.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { L1CrossDomainMessenger } from "src/L1/L1CrossDomainMessenger.sol";
import { L1ERC721Bridge } from "src/L1/L1ERC721Bridge.sol";
import { L1StandardBridge } from "src/L1/L1StandardBridge.sol";
import { OptimismMintableERC20Factory } from "src/universal/OptimismMintableERC20Factory.sol";

import {
    DeployImplementationsInput,
    DeployImplementations,
    DeployImplementationsInterop,
    DeployImplementationsOutput
} from "scripts/DeployImplementations.s.sol";

contract DeployImplementationsInput_Test is Test {
    DeployImplementationsInput dsi;

    DeployImplementationsInput.Input input = DeployImplementationsInput.Input({
        withdrawalDelaySeconds: 100,
        minProposalSizeBytes: 200,
        challengePeriodSeconds: 300,
        proofMaturityDelaySeconds: 400,
        disputeGameFinalityDelaySeconds: 500,
        release: "op-contracts/latest",
        superchainConfigProxy: SuperchainConfig(makeAddr("superchainConfigProxy")),
        protocolVersionsProxy: ProtocolVersions(makeAddr("protocolVersionsProxy"))
    });

    function setUp() public {
        dsi = new DeployImplementationsInput();
    }

    function test_loadInput_succeeds() public {
        dsi.loadInput(input);

        assertTrue(dsi.inputSet(), "100");

        // Compare the test input struct to the getter methods.
        assertEq(input.withdrawalDelaySeconds, dsi.withdrawalDelaySeconds(), "200");
        assertEq(input.minProposalSizeBytes, dsi.minProposalSizeBytes(), "300");
        assertEq(input.challengePeriodSeconds, dsi.challengePeriodSeconds(), "400");
        assertEq(input.proofMaturityDelaySeconds, dsi.proofMaturityDelaySeconds(), "500");
        assertEq(input.disputeGameFinalityDelaySeconds, dsi.disputeGameFinalityDelaySeconds(), "600");

        // Compare the test input struct to the `input` getter method.
        assertEq(keccak256(abi.encode(input)), keccak256(abi.encode(dsi.input())), "800");
    }

    function test_getters_whenNotSet_revert() public {
        bytes memory expectedErr = "DeployImplementationsInput: input not set";

        vm.expectRevert(expectedErr);
        dsi.withdrawalDelaySeconds();

        vm.expectRevert(expectedErr);
        dsi.minProposalSizeBytes();

        vm.expectRevert(expectedErr);
        dsi.challengePeriodSeconds();

        vm.expectRevert(expectedErr);
        dsi.proofMaturityDelaySeconds();

        vm.expectRevert(expectedErr);
        dsi.disputeGameFinalityDelaySeconds();
    }
}

contract DeployImplementationsOutput_Test is Test {
    DeployImplementationsOutput dso;

    function setUp() public {
        dso = new DeployImplementationsOutput();
    }

    function test_set_succeeds() public {
        DeployImplementationsOutput.Output memory output = DeployImplementationsOutput.Output({
            opsm: OPStackManager(makeAddr("opsm")),
            optimismPortalImpl: OptimismPortal2(payable(makeAddr("optimismPortalImpl"))),
            delayedWETHImpl: DelayedWETH(payable(makeAddr("delayedWETHImpl"))),
            preimageOracleSingleton: PreimageOracle(makeAddr("preimageOracleSingleton")),
            mipsSingleton: MIPS(makeAddr("mipsSingleton")),
            systemConfigImpl: SystemConfig(makeAddr("systemConfigImpl")),
            l1CrossDomainMessengerImpl: L1CrossDomainMessenger(makeAddr("l1CrossDomainMessengerImpl")),
            l1ERC721BridgeImpl: L1ERC721Bridge(makeAddr("l1ERC721BridgeImpl")),
            l1StandardBridgeImpl: L1StandardBridge(payable(makeAddr("l1StandardBridgeImpl"))),
            optimismMintableERC20FactoryImpl: OptimismMintableERC20Factory(makeAddr("optimismMintableERC20FactoryImpl")),
            disputeGameFactoryImpl: DisputeGameFactory(makeAddr("disputeGameFactoryImpl"))
        });

        vm.etch(address(output.opsm), hex"01");
        vm.etch(address(output.optimismPortalImpl), hex"01");
        vm.etch(address(output.delayedWETHImpl), hex"01");
        vm.etch(address(output.preimageOracleSingleton), hex"01");
        vm.etch(address(output.mipsSingleton), hex"01");
        vm.etch(address(output.systemConfigImpl), hex"01");
        vm.etch(address(output.l1CrossDomainMessengerImpl), hex"01");
        vm.etch(address(output.l1ERC721BridgeImpl), hex"01");
        vm.etch(address(output.l1StandardBridgeImpl), hex"01");
        vm.etch(address(output.optimismMintableERC20FactoryImpl), hex"01");
        vm.etch(address(output.disputeGameFactoryImpl), hex"01");
        dso.set(dso.opsm.selector, address(output.opsm));
        dso.set(dso.optimismPortalImpl.selector, address(output.optimismPortalImpl));
        dso.set(dso.delayedWETHImpl.selector, address(output.delayedWETHImpl));
        dso.set(dso.preimageOracleSingleton.selector, address(output.preimageOracleSingleton));
        dso.set(dso.mipsSingleton.selector, address(output.mipsSingleton));
        dso.set(dso.systemConfigImpl.selector, address(output.systemConfigImpl));
        dso.set(dso.l1CrossDomainMessengerImpl.selector, address(output.l1CrossDomainMessengerImpl));
        dso.set(dso.l1ERC721BridgeImpl.selector, address(output.l1ERC721BridgeImpl));
        dso.set(dso.l1StandardBridgeImpl.selector, address(output.l1StandardBridgeImpl));
        dso.set(dso.optimismMintableERC20FactoryImpl.selector, address(output.optimismMintableERC20FactoryImpl));
        dso.set(dso.disputeGameFactoryImpl.selector, address(output.disputeGameFactoryImpl));

        assertEq(address(output.opsm), address(dso.opsm()), "50");
        assertEq(address(output.optimismPortalImpl), address(dso.optimismPortalImpl()), "100");
        assertEq(address(output.delayedWETHImpl), address(dso.delayedWETHImpl()), "200");
        assertEq(address(output.preimageOracleSingleton), address(dso.preimageOracleSingleton()), "300");
        assertEq(address(output.mipsSingleton), address(dso.mipsSingleton()), "400");
        assertEq(address(output.systemConfigImpl), address(dso.systemConfigImpl()), "500");
        assertEq(address(output.l1CrossDomainMessengerImpl), address(dso.l1CrossDomainMessengerImpl()), "600");
        assertEq(address(output.l1ERC721BridgeImpl), address(dso.l1ERC721BridgeImpl()), "700");
        assertEq(address(output.l1StandardBridgeImpl), address(dso.l1StandardBridgeImpl()), "800");
        assertEq(
            address(output.optimismMintableERC20FactoryImpl), address(dso.optimismMintableERC20FactoryImpl()), "900"
        );
        assertEq(address(output.disputeGameFactoryImpl), address(dso.disputeGameFactoryImpl()), "950");

        assertEq(keccak256(abi.encode(output)), keccak256(abi.encode(dso.output())), "1000");
    }

    function test_getters_whenNotSet_revert() public {
        bytes memory expectedErr = "DeployUtils: zero address";

        vm.expectRevert(expectedErr);
        dso.optimismPortalImpl();

        vm.expectRevert(expectedErr);
        dso.delayedWETHImpl();

        vm.expectRevert(expectedErr);
        dso.preimageOracleSingleton();

        vm.expectRevert(expectedErr);
        dso.mipsSingleton();

        vm.expectRevert(expectedErr);
        dso.systemConfigImpl();

        vm.expectRevert(expectedErr);
        dso.l1CrossDomainMessengerImpl();

        vm.expectRevert(expectedErr);
        dso.l1ERC721BridgeImpl();

        vm.expectRevert(expectedErr);
        dso.l1StandardBridgeImpl();

        vm.expectRevert(expectedErr);
        dso.optimismMintableERC20FactoryImpl();

        vm.expectRevert(expectedErr);
        dso.disputeGameFactoryImpl();
    }

    function test_getters_whenAddrHasNoCode_reverts() public {
        address emptyAddr = makeAddr("emptyAddr");
        bytes memory expectedErr = bytes(string.concat("DeployUtils: no code at ", vm.toString(emptyAddr)));

        dso.set(dso.optimismPortalImpl.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dso.optimismPortalImpl();

        dso.set(dso.delayedWETHImpl.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dso.delayedWETHImpl();

        dso.set(dso.preimageOracleSingleton.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dso.preimageOracleSingleton();

        dso.set(dso.mipsSingleton.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dso.mipsSingleton();

        dso.set(dso.systemConfigImpl.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dso.systemConfigImpl();

        dso.set(dso.l1CrossDomainMessengerImpl.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dso.l1CrossDomainMessengerImpl();

        dso.set(dso.l1ERC721BridgeImpl.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dso.l1ERC721BridgeImpl();

        dso.set(dso.l1StandardBridgeImpl.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dso.l1StandardBridgeImpl();

        dso.set(dso.optimismMintableERC20FactoryImpl.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dso.optimismMintableERC20FactoryImpl();
    }
}

contract DeployImplementations_Test is Test {
    DeployImplementations deployImplementations;
    DeployImplementationsInput dsi;
    DeployImplementationsOutput dso;

    // Define a default input struct for testing.
    DeployImplementationsInput.Input input = DeployImplementationsInput.Input({
        withdrawalDelaySeconds: 100,
        minProposalSizeBytes: 200,
        challengePeriodSeconds: 300,
        proofMaturityDelaySeconds: 400,
        disputeGameFinalityDelaySeconds: 500,
        release: "op-contracts/latest",
        superchainConfigProxy: SuperchainConfig(makeAddr("superchainConfigProxy")),
        protocolVersionsProxy: ProtocolVersions(makeAddr("protocolVersionsProxy"))
    });

    function setUp() public virtual {
        deployImplementations = new DeployImplementations();
        (dsi, dso) = deployImplementations.getIOContracts();
    }

    // By deploying the `DeployImplementations` contract with this virtual function, we provide a
    // hook that child contracts can override to return a different implementation of the contract.
    // This lets us test e.g. the `DeployImplementationsInterop` contract without duplicating test code.
    function createDeployImplementationsContract() internal virtual returns (DeployImplementations) {
        return new DeployImplementations();
    }

    function test_run_succeeds(DeployImplementationsInput.Input memory _input) public {
        // This is a requirement in the PreimageOracle contract.
        _input.challengePeriodSeconds = bound(_input.challengePeriodSeconds, 0, type(uint64).max);

        DeployImplementationsOutput.Output memory output = deployImplementations.run(_input);

        // Assert that individual input fields were properly set based on the input struct.
        assertEq(_input.withdrawalDelaySeconds, dsi.withdrawalDelaySeconds(), "100");
        assertEq(_input.minProposalSizeBytes, dsi.minProposalSizeBytes(), "200");
        assertEq(_input.challengePeriodSeconds, dsi.challengePeriodSeconds(), "300");
        assertEq(_input.proofMaturityDelaySeconds, dsi.proofMaturityDelaySeconds(), "400");
        assertEq(_input.disputeGameFinalityDelaySeconds, dsi.disputeGameFinalityDelaySeconds(), "500");

        // Assert that individual output fields were properly set based on the output struct.
        assertEq(address(output.optimismPortalImpl), address(dso.optimismPortalImpl()), "600");
        assertEq(address(output.delayedWETHImpl), address(dso.delayedWETHImpl()), "700");
        assertEq(address(output.preimageOracleSingleton), address(dso.preimageOracleSingleton()), "800");
        assertEq(address(output.mipsSingleton), address(dso.mipsSingleton()), "900");
        assertEq(address(output.systemConfigImpl), address(dso.systemConfigImpl()), "1000");
        assertEq(address(output.l1CrossDomainMessengerImpl), address(dso.l1CrossDomainMessengerImpl()), "1100");
        assertEq(address(output.l1ERC721BridgeImpl), address(dso.l1ERC721BridgeImpl()), "1200");
        assertEq(address(output.l1StandardBridgeImpl), address(dso.l1StandardBridgeImpl()), "1300");
        assertEq(
            address(output.optimismMintableERC20FactoryImpl), address(dso.optimismMintableERC20FactoryImpl()), "1400"
        );
        assertEq(address(output.disputeGameFactoryImpl), address(dso.disputeGameFactoryImpl()), "1450");

        // Assert that the full input and output structs were properly set.
        assertEq(keccak256(abi.encode(_input)), keccak256(abi.encode(DeployImplementationsInput(dsi).input())), "1500");
        assertEq(
            keccak256(abi.encode(output)), keccak256(abi.encode(DeployImplementationsOutput(dso).output())), "1600"
        );

        // Assert inputs were properly passed through to the contract initializers.
        assertEq(output.delayedWETHImpl.delay(), _input.withdrawalDelaySeconds, "1700");
        assertEq(output.preimageOracleSingleton.challengePeriod(), _input.challengePeriodSeconds, "1800");
        assertEq(output.preimageOracleSingleton.minProposalSize(), _input.minProposalSizeBytes, "1900");
        assertEq(output.optimismPortalImpl.proofMaturityDelaySeconds(), _input.proofMaturityDelaySeconds, "2000");
        assertEq(
            output.optimismPortalImpl.disputeGameFinalityDelaySeconds(), _input.disputeGameFinalityDelaySeconds, "2100"
        );

        // Architecture assertions.
        assertEq(address(output.mipsSingleton.oracle()), address(output.preimageOracleSingleton), "2200");

        // Ensure that `checkOutput` passes. This is called by the `run` function during execution,
        // so this just acts as a sanity check. It reverts on failure.
        dso.checkOutput();
    }

    function test_run_largeChallengePeriodSeconds_reverts(uint256 _challengePeriodSeconds) public {
        input.challengePeriodSeconds = bound(_challengePeriodSeconds, uint256(type(uint64).max) + 1, type(uint256).max);
        vm.expectRevert("DeployImplementationsInput: challenge period too large");
        deployImplementations.run(input);
    }
}

contract DeployImplementationsInterop_Test is DeployImplementations_Test {
    function createDeployImplementationsContract() internal override returns (DeployImplementations) {
        return new DeployImplementationsInterop();
    }
}
