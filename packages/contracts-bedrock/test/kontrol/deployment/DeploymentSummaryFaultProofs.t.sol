// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Libraries
import { Constants } from "src/libraries/Constants.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

// Target contract dependencies
import { DisputeGameFactory } from "src/dispute/DisputeGameFactory.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { OptimismPortal2 } from "src/L1/OptimismPortal2.sol";
import { L1CrossDomainMessenger } from "src/L1/L1CrossDomainMessenger.sol";
import { DeploymentSummaryFaultProofs } from "../proofs/utils/DeploymentSummaryFaultProofs.sol";
import { L1ERC721Bridge } from "src/L1/L1ERC721Bridge.sol";
import { L1StandardBridge } from "src/L1/L1StandardBridge.sol";
import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import { LegacyMintableERC20 } from "src/legacy/LegacyMintableERC20.sol";

// Tests
import { L1CrossDomainMessenger_Test } from "test/L1/L1CrossDomainMessenger.t.sol";
import { OptimismPortal2_Test } from "test/L1/OptimismPortal2.t.sol";
import { L1ERC721Bridge_Test, TestERC721 } from "test/L1/L1ERC721Bridge.t.sol";
import {
    L1StandardBridge_Getter_Test,
    L1StandardBridge_Initialize_Test,
    L1StandardBridge_Pause_Test
} from "test/L1/L1StandardBridge.t.sol";

/// @dev Contract testing the deployment summary correctness
contract DeploymentSummaryFaultProofs_TestOptimismPortal is DeploymentSummaryFaultProofs, OptimismPortal2_Test {
    /// @notice super.setUp is not called on purpose
    function setUp() public override {
        // Recreate Deployment Summary state changes
        DeploymentSummaryFaultProofs deploymentSummary = new DeploymentSummaryFaultProofs();
        deploymentSummary.recreateDeployment();

        // Set summary addresses
        optimismPortal2 = OptimismPortal2(payable(optimismPortalProxyAddress));
        superchainConfig = SuperchainConfig(superchainConfigProxyAddress);
        disputeGameFactory = DisputeGameFactory(disputeGameFactoryProxyAddress);
        systemConfig = SystemConfig(systemConfigProxyAddress);

        // Set up utilized addresses
        depositor = makeAddr("depositor");
        alice = makeAddr("alice");
        bob = makeAddr("bob");
        vm.deal(alice, 10000 ether);
        vm.deal(bob, 10000 ether);
    }

    /// @dev Skips the first line of `super.test_constructor_succeeds` because
    ///      we're not exercising the `Deploy` logic in these tests. However,
    ///      the remaining assertions of the test are important to check
    function test_constructor_succeeds() external view override {
        // OptimismPortal opImpl = OptimismPortal(payable(deploy.mustGetAddress("OptimismPortal")));
        OptimismPortal2 opImpl = OptimismPortal2(payable(optimismPortal2Address));
        assertEq(address(opImpl.disputeGameFactory()), address(0));
        assertEq(address(opImpl.systemConfig()), address(0));
        assertEq(address(opImpl.superchainConfig()), address(0));
        assertEq(opImpl.l2Sender(), Constants.DEFAULT_L2_SENDER);
    }

    /// @dev Skips the first line of `super.test_initialize_succeeds` because
    ///      we're not exercising the `Deploy` logic in these tests. However,
    ///      the remaining assertions of the test are important to check
    function test_initialize_succeeds() external view override {
        // address guardian = deploy.cfg().superchainConfigGuardian();
        address guardian = superchainConfig.guardian();
        assertEq(address(optimismPortal2.disputeGameFactory()), address(disputeGameFactory));
        assertEq(address(optimismPortal2.systemConfig()), address(systemConfig));
        assertEq(optimismPortal2.guardian(), guardian);
        assertEq(address(optimismPortal2.superchainConfig()), address(superchainConfig));
        assertEq(optimismPortal2.l2Sender(), Constants.DEFAULT_L2_SENDER);
        assertEq(optimismPortal2.paused(), false);
    }
}
