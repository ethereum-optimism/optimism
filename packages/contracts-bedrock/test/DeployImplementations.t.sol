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
    DeployImplementationsInput dii;

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
        dii = new DeployImplementationsInput();
    }

    function test_loadInput_succeeds() public {
        dii.loadInput(input);

        assertTrue(dii.inputSet(), "100");

        // Compare the test input struct to the getter methods.
        assertEq(input.withdrawalDelaySeconds, dii.withdrawalDelaySeconds(), "200");
        assertEq(input.minProposalSizeBytes, dii.minProposalSizeBytes(), "300");
        assertEq(input.challengePeriodSeconds, dii.challengePeriodSeconds(), "400");
        assertEq(input.proofMaturityDelaySeconds, dii.proofMaturityDelaySeconds(), "500");
        assertEq(input.disputeGameFinalityDelaySeconds, dii.disputeGameFinalityDelaySeconds(), "600");

        // Compare the test input struct to the `input` getter method.
        assertEq(keccak256(abi.encode(input)), keccak256(abi.encode(dii.input())), "800");
    }

    function test_getters_whenNotSet_revert() public {
        bytes memory expectedErr = "DeployImplementationsInput: input not set";

        vm.expectRevert(expectedErr);
        dii.withdrawalDelaySeconds();

        vm.expectRevert(expectedErr);
        dii.minProposalSizeBytes();

        vm.expectRevert(expectedErr);
        dii.challengePeriodSeconds();

        vm.expectRevert(expectedErr);
        dii.proofMaturityDelaySeconds();

        vm.expectRevert(expectedErr);
        dii.disputeGameFinalityDelaySeconds();
    }
}

contract DeployImplementationsOutput_Test is Test {
    DeployImplementationsOutput dio;

    function setUp() public {
        dio = new DeployImplementationsOutput();
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
        dio.set(dio.opsm.selector, address(output.opsm));
        dio.set(dio.optimismPortalImpl.selector, address(output.optimismPortalImpl));
        dio.set(dio.delayedWETHImpl.selector, address(output.delayedWETHImpl));
        dio.set(dio.preimageOracleSingleton.selector, address(output.preimageOracleSingleton));
        dio.set(dio.mipsSingleton.selector, address(output.mipsSingleton));
        dio.set(dio.systemConfigImpl.selector, address(output.systemConfigImpl));
        dio.set(dio.l1CrossDomainMessengerImpl.selector, address(output.l1CrossDomainMessengerImpl));
        dio.set(dio.l1ERC721BridgeImpl.selector, address(output.l1ERC721BridgeImpl));
        dio.set(dio.l1StandardBridgeImpl.selector, address(output.l1StandardBridgeImpl));
        dio.set(dio.optimismMintableERC20FactoryImpl.selector, address(output.optimismMintableERC20FactoryImpl));
        dio.set(dio.disputeGameFactoryImpl.selector, address(output.disputeGameFactoryImpl));

        assertEq(address(output.opsm), address(dio.opsm()), "50");
        assertEq(address(output.optimismPortalImpl), address(dio.optimismPortalImpl()), "100");
        assertEq(address(output.delayedWETHImpl), address(dio.delayedWETHImpl()), "200");
        assertEq(address(output.preimageOracleSingleton), address(dio.preimageOracleSingleton()), "300");
        assertEq(address(output.mipsSingleton), address(dio.mipsSingleton()), "400");
        assertEq(address(output.systemConfigImpl), address(dio.systemConfigImpl()), "500");
        assertEq(address(output.l1CrossDomainMessengerImpl), address(dio.l1CrossDomainMessengerImpl()), "600");
        assertEq(address(output.l1ERC721BridgeImpl), address(dio.l1ERC721BridgeImpl()), "700");
        assertEq(address(output.l1StandardBridgeImpl), address(dio.l1StandardBridgeImpl()), "800");
        assertEq(
            address(output.optimismMintableERC20FactoryImpl), address(dio.optimismMintableERC20FactoryImpl()), "900"
        );
        assertEq(address(output.disputeGameFactoryImpl), address(dio.disputeGameFactoryImpl()), "950");

        assertEq(keccak256(abi.encode(output)), keccak256(abi.encode(dio.output())), "1000");
    }

    function test_getters_whenNotSet_revert() public {
        bytes memory expectedErr = "DeployUtils: zero address";

        vm.expectRevert(expectedErr);
        dio.optimismPortalImpl();

        vm.expectRevert(expectedErr);
        dio.delayedWETHImpl();

        vm.expectRevert(expectedErr);
        dio.preimageOracleSingleton();

        vm.expectRevert(expectedErr);
        dio.mipsSingleton();

        vm.expectRevert(expectedErr);
        dio.systemConfigImpl();

        vm.expectRevert(expectedErr);
        dio.l1CrossDomainMessengerImpl();

        vm.expectRevert(expectedErr);
        dio.l1ERC721BridgeImpl();

        vm.expectRevert(expectedErr);
        dio.l1StandardBridgeImpl();

        vm.expectRevert(expectedErr);
        dio.optimismMintableERC20FactoryImpl();

        vm.expectRevert(expectedErr);
        dio.disputeGameFactoryImpl();
    }

    function test_getters_whenAddrHasNoCode_reverts() public {
        address emptyAddr = makeAddr("emptyAddr");
        bytes memory expectedErr = bytes(string.concat("DeployUtils: no code at ", vm.toString(emptyAddr)));

        dio.set(dio.optimismPortalImpl.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dio.optimismPortalImpl();

        dio.set(dio.delayedWETHImpl.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dio.delayedWETHImpl();

        dio.set(dio.preimageOracleSingleton.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dio.preimageOracleSingleton();

        dio.set(dio.mipsSingleton.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dio.mipsSingleton();

        dio.set(dio.systemConfigImpl.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dio.systemConfigImpl();

        dio.set(dio.l1CrossDomainMessengerImpl.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dio.l1CrossDomainMessengerImpl();

        dio.set(dio.l1ERC721BridgeImpl.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dio.l1ERC721BridgeImpl();

        dio.set(dio.l1StandardBridgeImpl.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dio.l1StandardBridgeImpl();

        dio.set(dio.optimismMintableERC20FactoryImpl.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dio.optimismMintableERC20FactoryImpl();
    }
}

contract DeployImplementations_Test is Test {
    DeployImplementations deployImplementations;
    DeployImplementationsInput dii;
    DeployImplementationsOutput dio;

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
        (dii, dio) = deployImplementations.getIOContracts();
    }

    // By deploying the `DeployImplementations` contract with this virtual function, we provide a
    // hook that child contracts can override to return a different implementation of the contract.
    // This lets us test e.g. the `DeployImplementationsInterop` contract without duplicating test code.
    function createDeployImplementationsContract() internal virtual returns (DeployImplementations) {
        return new DeployImplementations();
    }

    function testFuzz_run_succeeds(DeployImplementationsInput.Input memory _input) public {
        // This is a requirement in the PreimageOracle contract.
        _input.challengePeriodSeconds = bound(_input.challengePeriodSeconds, 0, type(uint64).max);

        DeployImplementationsOutput.Output memory output = deployImplementations.run(_input);

        // Assert that individual input fields were properly set based on the input struct.
        assertEq(_input.withdrawalDelaySeconds, dii.withdrawalDelaySeconds(), "100");
        assertEq(_input.minProposalSizeBytes, dii.minProposalSizeBytes(), "200");
        assertEq(_input.challengePeriodSeconds, dii.challengePeriodSeconds(), "300");
        assertEq(_input.proofMaturityDelaySeconds, dii.proofMaturityDelaySeconds(), "400");
        assertEq(_input.disputeGameFinalityDelaySeconds, dii.disputeGameFinalityDelaySeconds(), "500");

        // Assert that individual output fields were properly set based on the output struct.
        assertEq(address(output.optimismPortalImpl), address(dio.optimismPortalImpl()), "600");
        assertEq(address(output.delayedWETHImpl), address(dio.delayedWETHImpl()), "700");
        assertEq(address(output.preimageOracleSingleton), address(dio.preimageOracleSingleton()), "800");
        assertEq(address(output.mipsSingleton), address(dio.mipsSingleton()), "900");
        assertEq(address(output.systemConfigImpl), address(dio.systemConfigImpl()), "1000");
        assertEq(address(output.l1CrossDomainMessengerImpl), address(dio.l1CrossDomainMessengerImpl()), "1100");
        assertEq(address(output.l1ERC721BridgeImpl), address(dio.l1ERC721BridgeImpl()), "1200");
        assertEq(address(output.l1StandardBridgeImpl), address(dio.l1StandardBridgeImpl()), "1300");
        assertEq(
            address(output.optimismMintableERC20FactoryImpl), address(dio.optimismMintableERC20FactoryImpl()), "1400"
        );
        assertEq(address(output.disputeGameFactoryImpl), address(dio.disputeGameFactoryImpl()), "1450");

        // Assert that the full input and output structs were properly set.
        assertEq(keccak256(abi.encode(_input)), keccak256(abi.encode(DeployImplementationsInput(dii).input())), "1500");
        assertEq(
            keccak256(abi.encode(output)), keccak256(abi.encode(DeployImplementationsOutput(dio).output())), "1600"
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
        dio.checkOutput();
    }

    function testFuzz_run_largeChallengePeriodSeconds_reverts(uint256 _challengePeriodSeconds) public {
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
