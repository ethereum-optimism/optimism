// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";

import {
    DeployImplementationsInput,
    DeployImplementations,
    DeployImplementationsOutput
} from "scripts/DeployImplementations.s.sol";

/// @notice Deploys the Superchain contracts that can be shared by many chains.
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
        disputeGameFinalityDelaySeconds: 500
    });

    function setUp() public {
        deployImplementations = new DeployImplementations();
        (dsi, dso) = deployImplementations.getIOContracts();
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
        assertEq(address(output.optimismPortal2Impl), address(dso.optimismPortal2Impl()), "600");
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

        // Assert that the full input and output structs were properly set.
        assertEq(keccak256(abi.encode(_input)), keccak256(abi.encode(DeployImplementationsInput(dsi).input())), "1500");
        assertEq(
            keccak256(abi.encode(output)), keccak256(abi.encode(DeployImplementationsOutput(dso).output())), "1600"
        );

        // Assert inputs were properly passed through to the contract initializers.
        assertEq(output.delayedWETHImpl.delay(), _input.withdrawalDelaySeconds, "1700");
        assertEq(output.preimageOracleSingleton.challengePeriod(), _input.challengePeriodSeconds, "1800");
        assertEq(output.preimageOracleSingleton.minProposalSize(), _input.minProposalSizeBytes, "1900");
        assertEq(output.optimismPortal2Impl.proofMaturityDelaySeconds(), _input.proofMaturityDelaySeconds, "2000");
        assertEq(
            output.optimismPortal2Impl.disputeGameFinalityDelaySeconds(), _input.disputeGameFinalityDelaySeconds, "2100"
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
