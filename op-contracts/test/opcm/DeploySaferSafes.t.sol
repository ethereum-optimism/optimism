// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Scripts
import { DeploySaferSafes } from "scripts/deploy/DeploySaferSafes.s.sol";

// Interfaces
import { ISaferSafes } from "interfaces/safe/ISaferSafes.sol";

/// @title DeploySaferSafes_Test
/// @notice Tests for the DeploySaferSafes script.
contract DeploySaferSafes_Test is Test {
    DeploySaferSafes deploySaferSafes;

    /// @notice Sets up the test suite.
    function setUp() public {
        deploySaferSafes = new DeploySaferSafes();
    }

    /// @notice Tests that the DeploySaferSafes script succeeds.
    function test_run_succeeds() public {
        DeploySaferSafes.Output memory output = deploySaferSafes.run();

        // Verify the SaferSafes singleton is deployed.
        assertNotEq(address(output.saferSafesSingleton), address(0), "SaferSafes address is zero");

        // Verify the contract has code.
        assertGt(address(output.saferSafesSingleton).code.length, 0, "SaferSafes has no code");

        // Verify the version is correct.
        assertEq(output.saferSafesSingleton.version(), "1.10.1", "SaferSafes version mismatch");
    }

    /// @notice Tests that the deployment is deterministic and reuses addresses.
    function test_reuseAddresses_succeeds() public {
        DeploySaferSafes.Output memory output1 = deploySaferSafes.run();
        DeploySaferSafes.Output memory output2 = deploySaferSafes.run();

        // Verify that the same address is reused.
        assertEq(
            address(output1.saferSafesSingleton),
            address(output2.saferSafesSingleton),
            "SaferSafes address should be reused"
        );
    }

    /// @notice Tests that assertValidOutput succeeds with valid output.
    function test_assertValidOutput_succeeds() public {
        DeploySaferSafes.Output memory output = deploySaferSafes.run();

        // This should not revert.
        deploySaferSafes.assertValidOutput(output);
    }

    /// @notice Tests that assertValidOutput reverts when the address is zero.
    function test_assertValidOutput_zeroAddress_reverts() public {
        DeploySaferSafes.Output memory output;
        output.saferSafesSingleton = ISaferSafes(address(0));

        vm.expectRevert("DeployUtils: zero address");
        deploySaferSafes.assertValidOutput(output);
    }

    /// @notice Tests that assertValidOutput reverts when the contract has no code.
    function test_assertValidOutput_noCode_reverts() public {
        DeploySaferSafes.Output memory output;
        address noCodeAddr = makeAddr("noCode");
        output.saferSafesSingleton = ISaferSafes(noCodeAddr);

        vm.expectRevert(bytes(string.concat("DeployUtils: no code at ", vm.toString(noCodeAddr))));
        deploySaferSafes.assertValidOutput(output);
    }

    /// @notice Tests that assertValidOutput reverts when the version is incorrect.
    function test_assertValidOutput_wrongVersion_reverts() public {
        // Deploy a mock contract with a different version.
        MockSaferSafes mockSaferSafes = new MockSaferSafes();

        DeploySaferSafes.Output memory output;
        output.saferSafesSingleton = ISaferSafes(address(mockSaferSafes));

        vm.expectRevert("DeploySaferSafes: unexpected version");
        deploySaferSafes.assertValidOutput(output);
    }

    /// @notice Tests that the deployment uses CREATE2 for deterministic addresses.
    function test_deterministicDeployment_succeeds() public {
        // First deployment.
        DeploySaferSafes.Output memory output1 = deploySaferSafes.run();

        // The contract should be deployed at a deterministic address.
        // Running again should return the same address without redeploying.
        DeploySaferSafes.Output memory output2 = deploySaferSafes.run();

        // Verify that the same address is used.
        assertEq(
            address(output1.saferSafesSingleton),
            address(output2.saferSafesSingleton),
            "SaferSafes address should be deterministic"
        );

        // Verify that the contract has code (it wasn't redeployed).
        assertGt(address(output2.saferSafesSingleton).code.length, 0, "Contract should have code");
    }

    /// @notice Tests that multiple runs do not redeploy the contract.
    function test_multipleRuns_succeeds() public {
        DeploySaferSafes.Output memory output1 = deploySaferSafes.run();
        DeploySaferSafes.Output memory output2 = deploySaferSafes.run();
        DeploySaferSafes.Output memory output3 = deploySaferSafes.run();

        // All deployments should use the same address.
        assertEq(address(output1.saferSafesSingleton), address(output2.saferSafesSingleton), "Second run mismatch");
        assertEq(address(output2.saferSafesSingleton), address(output3.saferSafesSingleton), "Third run mismatch");
    }

    /// @notice Tests that the deployed contract has the expected version string format.
    function test_versionFormat_succeeds() public {
        DeploySaferSafes.Output memory output = deploySaferSafes.run();

        string memory version = output.saferSafesSingleton.version();

        // Verify the version is not empty.
        assertTrue(bytes(version).length > 0, "Version should not be empty");

        // Verify the version matches the expected format.
        assertEq(version, "1.10.1", "Version should be 1.10.1");
    }
}

/// @title MockSaferSafes
/// @notice A mock SaferSafes contract with a different version for testing.
contract MockSaferSafes {
    function version() external pure returns (string memory) {
        return "0.0.0";
    }
}
