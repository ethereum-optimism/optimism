// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Libraries
import { Constants } from "src/libraries/Constants.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

// Target contract dependencies
import { L2OutputOracle } from "src/L1/L2OutputOracle.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { OptimismPortal } from "src/L1/OptimismPortal.sol";
import { L1CrossDomainMessenger } from "src/L1/L1CrossDomainMessenger.sol";
import { DeploymentSummary } from "../proofs/utils/DeploymentSummary.sol";
import { OptimismPortal_Test } from "test/L1/OptimismPortal.t.sol";
import { L1CrossDomainMessenger_Test } from "test/L1/L1CrossDomainMessenger.t.sol";

/// @dev Contract testing the deployment summary correctness
contract DeploymentSummary_TestOptimismPortal is DeploymentSummary, OptimismPortal_Test {
    /// @notice super.setUp is not called on purpose
    function setUp() public override {
        // Recreate Deployment Summary state changes
        DeploymentSummary deploymentSummary = new DeploymentSummary();
        deploymentSummary.recreateDeployment();

        // Set summary addresses
        optimismPortal = OptimismPortal(payable(optimismPortalProxyAddress));
        superchainConfig = SuperchainConfig(superchainConfigProxyAddress);
        l2OutputOracle = L2OutputOracle(l2OutputOracleProxyAddress);
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
    function test_constructor_succeeds() external override {
        // OptimismPortal opImpl = OptimismPortal(payable(deploy.mustGetAddress("OptimismPortal")));
        OptimismPortal opImpl = OptimismPortal(payable(optimismPortalAddress));
        assertEq(address(opImpl.L2_ORACLE()), address(0));
        assertEq(address(opImpl.l2Oracle()), address(0));
        assertEq(address(opImpl.SYSTEM_CONFIG()), address(0));
        assertEq(address(opImpl.systemConfig()), address(0));
        assertEq(address(opImpl.superchainConfig()), address(0));
        assertEq(opImpl.l2Sender(), Constants.DEFAULT_L2_SENDER);
    }

    /// @dev Skips the first line of `super.test_initialize_succeeds` because
    ///      we're not exercising the `Deploy` logic in these tests. However,
    ///      the remaining assertions of the test are important to check
    function test_initialize_succeeds() external override {
        // address guardian = deploy.cfg().superchainConfigGuardian();
        address guardian = superchainConfig.guardian();
        assertEq(address(optimismPortal.L2_ORACLE()), address(l2OutputOracle));
        assertEq(address(optimismPortal.l2Oracle()), address(l2OutputOracle));
        assertEq(address(optimismPortal.SYSTEM_CONFIG()), address(systemConfig));
        assertEq(address(optimismPortal.systemConfig()), address(systemConfig));
        assertEq(optimismPortal.GUARDIAN(), guardian);
        assertEq(optimismPortal.guardian(), guardian);
        assertEq(address(optimismPortal.superchainConfig()), address(superchainConfig));
        assertEq(optimismPortal.l2Sender(), Constants.DEFAULT_L2_SENDER);
        assertEq(optimismPortal.paused(), false);
    }

    /// @notice This test is overridden because `KontrolDeployment` doesn't initialize
    ///         the L2OutputOracle, which is needed in this test
    function test_simple_isOutputFinalized_succeeds() external override { }

    /// @notice This test is overridden because `KontrolDeployment` doesn't initialize
    ///         the L2OutputOracle, which is needed in this test
    function test_isOutputFinalized_succeeds() external override { }
}

contract DeploymentSummary_TestL1CrossDomainMessenger is DeploymentSummary, L1CrossDomainMessenger_Test {
    /// @notice super.setUp is not called on purpose
    function setUp() public override {
        // Recreate Deployment Summary state changes
        DeploymentSummary deploymentSummary = new DeploymentSummary();
        deploymentSummary.recreateDeployment();

        // Set summary addresses
        optimismPortal = OptimismPortal(payable(optimismPortalProxyAddress));
        superchainConfig = SuperchainConfig(superchainConfigProxyAddress);
        l2OutputOracle = L2OutputOracle(l2OutputOracleProxyAddress);
        systemConfig = SystemConfig(systemConfigProxyAddress);
        l1CrossDomainMessenger = L1CrossDomainMessenger(l1CrossDomainMessengerProxyAddress);

        // Set up utilized addresses
        alice = makeAddr("alice");
        bob = makeAddr("bob");
        vm.deal(alice, 10000 ether);
        vm.deal(bob, 10000 ether);
    }

    /// @dev Skips the first line of `super.test_constructor_succeeds` because
    ///      we're not exercising the `Deploy` logic in these tests. However,
    ///      the remaining assertions of the test are important to check
    function test_constructor_succeeds() external override {
        // L1CrossDomainMessenger impl = L1CrossDomainMessenger(deploy.mustGetAddress("L1CrossDomainMessenger"));
        L1CrossDomainMessenger impl = L1CrossDomainMessenger(l1CrossDomainMessengerAddress);
        assertEq(address(impl.superchainConfig()), address(0));
        assertEq(address(impl.PORTAL()), address(0));
        assertEq(address(impl.portal()), address(0));
        assertEq(address(impl.OTHER_MESSENGER()), Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        assertEq(address(impl.otherMessenger()), Predeploys.L2_CROSS_DOMAIN_MESSENGER);
    }

    /// @notice This test is overridden because `KontrolDeployment` doesn't deploy
    ///         L2CrossDomainMessenger, which is needed in this test
    function test_relayMessage_v2_reverts() external override { }
}
