// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Target contract dependencies
import { L2OutputOracle } from "src/L1/L2OutputOracle.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { OptimismPortal } from "src/L1/OptimismPortal.sol";
import { DeploymentSummary } from "../proofs/utils/DeploymentSummary.sol";
import { OptimismPortal_Test } from "test/L1/OptimismPortal.t.sol";

/// @dev Contract testing the deployment summary correctness
contract DeploymentSummary_Test is DeploymentSummary, OptimismPortal_Test {
    /// @notice super.setUp is not called on purpose
    function setUp() public override {
        // Recreate Deployment Summary state changes
        DeploymentSummary deploymentSummary = new DeploymentSummary();
        deploymentSummary.recreateDeployment();

        // Set summary addresses
        optimismPortal = OptimismPortal(payable(OptimismPortalProxyAddress));
        superchainConfig = SuperchainConfig(SuperchainConfigProxyAddress);
        l2OutputOracle = L2OutputOracle(L2OutputOracleProxyAddress);
        systemConfig = SystemConfig(SystemConfigProxyAddress);

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
        // address guardian = deploy.cfg().superchainConfigGuardian();
        address guardian = superchainConfig.guardian();
        assertEq(address(optimismPortal.L2_ORACLE()), address(l2OutputOracle));
        assertEq(address(optimismPortal.l2Oracle()), address(l2OutputOracle));
        assertEq(optimismPortal.GUARDIAN(), guardian);
        assertEq(optimismPortal.guardian(), guardian);
        assertEq(optimismPortal.l2Sender(), 0x000000000000000000000000000000000000dEaD);
        assertEq(optimismPortal.paused(), false);
    }

    /// @notice This test is skipped because `KontrolDeployment` doesn't initialize
    ///         the L2OutputOracle
    function test_simple_isOutputFinalized_succeeds() external override {
        vm.skip(true);
    }

    /// @notice This test is skipped because `KontrolDeployment` doesn't initialize
    ///         the L2OutputOracle
    function test_isOutputFinalized_succeeds() external override {
        vm.skip(true);
    }
}
