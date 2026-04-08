// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

// Scripts
import { ReadSuperchainDeployment } from "scripts/deploy/ReadSuperchainDeployment.s.sol";

// Interfaces
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";

// Test addresses declared as constants for convenience.
address constant TEST_SUPERCHAIN_CONFIG_IMPL = address(0x3001);
address constant TEST_SUPERCHAIN_PROXY_ADMIN = address(0x3002);
address constant TEST_GUARDIAN = address(0x3003);
address constant TEST_SUPERCHAIN_PROXY_ADMIN_OWNER = address(0x3004);

/// @title ReadSuperchainDeploymentTest
/// @notice Tests that ReadSuperchainDeployment.run and ReadSuperchainDeployment.runWithBytes succeed.
contract ReadSuperchainDeploymentTest is Test {
    ReadSuperchainDeployment script;
    ReadSuperchainDeployment.Input input;

    function setUp() public {
        script = new ReadSuperchainDeployment();
        input.superchainConfigProxy = ISuperchainConfig(makeAddr("superchainConfigProxy"));
    }

    /// @notice Tests that ReadSuperchainDeployment.run succeeds.
    function test_run_succeeds() public {
        _setUpSuperchainConfigProxy();
        _mockSuperchainConfigCalls();

        ReadSuperchainDeployment.Output memory output = script.run(input);

        assertEq(address(output.superchainConfigProxy), address(input.superchainConfigProxy));
        assertEq(address(output.superchainConfigImpl), TEST_SUPERCHAIN_CONFIG_IMPL);
        assertEq(address(output.superchainProxyAdmin), TEST_SUPERCHAIN_PROXY_ADMIN);
        assertEq(output.guardian, TEST_GUARDIAN);
        assertEq(output.superchainProxyAdminOwner, TEST_SUPERCHAIN_PROXY_ADMIN_OWNER);
    }

    /// @notice Tests that ReadSuperchainDeployment.runWithBytes succeeds.
    function test_runWithBytes_succeeds() public {
        _setUpSuperchainConfigProxy();
        _mockSuperchainConfigCalls();

        bytes memory inputBytes = abi.encode(input);
        bytes memory outputBytes = script.runWithBytes(inputBytes);
        ReadSuperchainDeployment.Output memory output = abi.decode(outputBytes, (ReadSuperchainDeployment.Output));

        assertEq(address(output.superchainConfigProxy), address(input.superchainConfigProxy));
        assertEq(address(output.superchainConfigImpl), TEST_SUPERCHAIN_CONFIG_IMPL);
        assertEq(address(output.superchainProxyAdmin), TEST_SUPERCHAIN_PROXY_ADMIN);
        assertEq(output.guardian, TEST_GUARDIAN);
        assertEq(output.superchainProxyAdminOwner, TEST_SUPERCHAIN_PROXY_ADMIN_OWNER);
    }

    /// @notice Tests that run reverts when superchainConfigProxy has no code.
    function test_run_superchainConfigNoCode_reverts() public {
        // Do not etch code to superchainConfigProxy

        vm.expectRevert("ReadSuperchainDeployment: superchainConfigProxy has no code");
        script.run(input);
    }

    /// @notice Sets up the superchainConfigProxy for testing.
    function _setUpSuperchainConfigProxy() internal {
        // Etch code to superchainConfigProxy
        vm.etch(address(input.superchainConfigProxy), "0x01");
        // Set EIP-1967 admin slot on superchainConfigProxy so getAdmin returns TEST_SUPERCHAIN_PROXY_ADMIN
        EIP1967Helper.setAdmin(address(input.superchainConfigProxy), TEST_SUPERCHAIN_PROXY_ADMIN);
    }

    /// @notice Mocks SuperchainConfig proxy and ProxyAdmin calls.
    function _mockSuperchainConfigCalls() internal {
        _mockExpect(
            address(input.superchainConfigProxy),
            abi.encodeCall(IProxy.implementation, ()),
            abi.encode(TEST_SUPERCHAIN_CONFIG_IMPL)
        );
        _mockExpect(
            address(input.superchainConfigProxy),
            abi.encodeCall(ISuperchainConfig.guardian, ()),
            abi.encode(TEST_GUARDIAN)
        );
        _mockExpect(
            TEST_SUPERCHAIN_PROXY_ADMIN,
            abi.encodeCall(IProxyAdmin.owner, ()),
            abi.encode(TEST_SUPERCHAIN_PROXY_ADMIN_OWNER)
        );
    }

    /// @notice Internal helper to mock and expect calls.
    function _mockExpect(address _target, bytes memory _callData, bytes memory _returnData) internal {
        vm.mockCall(_target, _callData, _returnData);
        vm.expectCall(_target, _callData);
    }
}
