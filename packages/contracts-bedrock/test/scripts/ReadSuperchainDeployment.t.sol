// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

// Scripts
import { ReadSuperchainDeployment } from "scripts/deploy/ReadSuperchainDeployment.s.sol";

// Interfaces
import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IProtocolVersions, ProtocolVersion } from "interfaces/L1/IProtocolVersions.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";
import { Constants } from "src/libraries/Constants.sol";

// Test addresses declared as constants for convenience.
address constant TEST_SUPERCHAIN_CONFIG_IMPL = address(0x3001);
address constant TEST_SUPERCHAIN_PROXY_ADMIN = address(0x3002);
address constant TEST_GUARDIAN = address(0x3003);
address constant TEST_SUPERCHAIN_PROXY_ADMIN_OWNER = address(0x3004);
address constant TEST_OPCM_IMPL = address(0x2000);
address constant TEST_PROTOCOL_VERSIONS_PROXY = address(0x3005);
address constant TEST_PROTOCOL_VERSIONS_IMPL = address(0x3006);
address constant TEST_PROTOCOL_VERSIONS_OWNER = address(0x3007);
uint256 constant TEST_RECOMMENDED_VERSION = 1;
uint256 constant TEST_REQUIRED_VERSION = 2;

/// @title ReadSuperchainDeploymentTest
/// @notice Tests that ReadSuperchainDeployment.run and ReadSuperchainDeployment.runWithBytes succeed with OPCM V1
/// and OPCM V2.
contract ReadSuperchainDeploymentTest is Test {
    ReadSuperchainDeployment script;
    ReadSuperchainDeployment.Input input;

    function setUp() public {
        script = new ReadSuperchainDeployment();
        input.superchainConfigProxy = ISuperchainConfig(makeAddr("superchainConfigProxy"));
    }

    /// @notice Tests that ReadSuperchainDeployment.run succeeds with OPCM V2 (opcmAddress is zero).
    function test_run_withOPCMV2ZeroAddress_succeeds() public {
        input.opcmAddress = IOPContractsManager(address(0));

        _setUpSuperchainConfigProxy();
        _mockSuperchainConfigCalls();

        ReadSuperchainDeployment.Output memory output = script.run(input);

        assertEq(address(output.superchainConfigProxy), address(input.superchainConfigProxy));
        assertEq(address(output.superchainConfigImpl), TEST_SUPERCHAIN_CONFIG_IMPL);
        assertEq(address(output.superchainProxyAdmin), TEST_SUPERCHAIN_PROXY_ADMIN);
        assertEq(output.guardian, TEST_GUARDIAN);
        assertEq(output.superchainProxyAdminOwner, TEST_SUPERCHAIN_PROXY_ADMIN_OWNER);
        assertEq(address(output.protocolVersionsImpl), address(0));
        assertEq(address(output.protocolVersionsProxy), address(0));
        assertEq(output.protocolVersionsOwner, address(0));
        assertEq(output.recommendedProtocolVersion, bytes32(0));
        assertEq(output.requiredProtocolVersion, bytes32(0));
    }

    /// @notice Tests that ReadSuperchainDeployment.run succeeds with OPCM V2 (opcm version >= 7.0.0).
    function test_run_withOPCMV2VersionGteMin_succeeds() public {
        input.opcmAddress = IOPContractsManager(TEST_OPCM_IMPL);
        vm.etch(TEST_OPCM_IMPL, "0x01");
        _mockExpect(
            TEST_OPCM_IMPL, abi.encodeCall(IOPContractsManager.version, ()), abi.encode(Constants.OPCM_V2_MIN_VERSION)
        );

        _setUpSuperchainConfigProxy();
        _mockSuperchainConfigCalls();

        ReadSuperchainDeployment.Output memory output = script.run(input);

        assertEq(address(output.superchainConfigProxy), address(input.superchainConfigProxy));
        assertEq(address(output.superchainConfigImpl), TEST_SUPERCHAIN_CONFIG_IMPL);
        assertEq(address(output.superchainProxyAdmin), TEST_SUPERCHAIN_PROXY_ADMIN);
        assertEq(output.guardian, TEST_GUARDIAN);
        assertEq(output.superchainProxyAdminOwner, TEST_SUPERCHAIN_PROXY_ADMIN_OWNER);
        assertEq(address(output.protocolVersionsImpl), address(0));
        assertEq(address(output.protocolVersionsProxy), address(0));
        assertEq(output.protocolVersionsOwner, address(0));
        assertEq(output.recommendedProtocolVersion, bytes32(0));
        assertEq(output.requiredProtocolVersion, bytes32(0));
    }

    /// @notice Tests that ReadSuperchainDeployment.runWithBytes succeeds with OPCM V2.
    function test_runWithBytes_withOPCMV2_succeeds() public {
        input.opcmAddress = IOPContractsManager(address(0));
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
        assertEq(address(output.protocolVersionsImpl), address(0));
        assertEq(address(output.protocolVersionsProxy), address(0));
        assertEq(output.protocolVersionsOwner, address(0));
        assertEq(output.recommendedProtocolVersion, bytes32(0));
        assertEq(output.requiredProtocolVersion, bytes32(0));
    }

    /// @notice Tests that run reverts when OPCM V2 and superchainConfigProxy has no code.
    function test_run_opcmV2SuperchainConfigNoCode_reverts() public {
        input.opcmAddress = IOPContractsManager(address(0));
        // Do not etch code to superchainConfigProxy

        vm.expectRevert("ReadSuperchainDeployment: superchainConfigProxy has no code for OPCM v2");
        script.run(input);
    }

    /// @notice Tests that ReadSuperchainDeployment.run succeeds with OPCM V1.
    function test_run_withOPCMV1_succeeds() public {
        _mockOPCMV1();
        _setUpSuperchainConfigProxy();
        _mockSuperchainConfigCalls();
        _mockProtocolVersionsCalls();

        ReadSuperchainDeployment.Output memory output = script.run(input);

        assertEq(address(output.superchainConfigProxy), address(input.superchainConfigProxy));
        assertEq(address(output.superchainConfigImpl), TEST_SUPERCHAIN_CONFIG_IMPL);
        assertEq(address(output.superchainProxyAdmin), TEST_SUPERCHAIN_PROXY_ADMIN);
        assertEq(output.guardian, TEST_GUARDIAN);
        assertEq(output.superchainProxyAdminOwner, TEST_SUPERCHAIN_PROXY_ADMIN_OWNER);
        assertEq(address(output.protocolVersionsImpl), TEST_PROTOCOL_VERSIONS_IMPL);
        assertEq(address(output.protocolVersionsProxy), TEST_PROTOCOL_VERSIONS_PROXY);
        assertEq(output.protocolVersionsOwner, TEST_PROTOCOL_VERSIONS_OWNER);
        assertEq(output.recommendedProtocolVersion, bytes32(TEST_RECOMMENDED_VERSION));
        assertEq(output.requiredProtocolVersion, bytes32(TEST_REQUIRED_VERSION));
    }

    /// @notice Tests that ReadSuperchainDeployment.runWithBytes succeeds with OPCM V1.
    function test_runWithBytes_withOPCMV1_succeeds() public {
        _mockOPCMV1();
        vm.etch(address(input.superchainConfigProxy), "0x01");
        _setUpSuperchainConfigProxy();
        _mockSuperchainConfigCalls();
        _mockProtocolVersionsCalls();

        bytes memory inputBytes = abi.encode(input);
        bytes memory outputBytes = script.runWithBytes(inputBytes);
        ReadSuperchainDeployment.Output memory output = abi.decode(outputBytes, (ReadSuperchainDeployment.Output));

        assertEq(address(output.superchainConfigProxy), address(input.superchainConfigProxy));
        assertEq(address(output.protocolVersionsImpl), TEST_PROTOCOL_VERSIONS_IMPL);
        assertEq(output.recommendedProtocolVersion, bytes32(TEST_RECOMMENDED_VERSION));
        assertEq(output.requiredProtocolVersion, bytes32(TEST_REQUIRED_VERSION));
    }

    /// @notice Tests that run reverts when OPCM address is non-zero but has no code.
    function test_run_opcmCodeLengthZero_reverts() public {
        input.opcmAddress = IOPContractsManager(makeAddr("opcmNoCode"));
        vm.etch(address(input.superchainConfigProxy), "0x01");

        vm.expectRevert("ReadSuperchainDeployment: OPCM address has no code");
        script.run(input);
    }

    /// @notice Sets up the superchainConfigProxy for testing.
    function _setUpSuperchainConfigProxy() internal {
        // Etch code to superchainConfigProxy
        vm.etch(address(input.superchainConfigProxy), "0x01");
        // Set EIP-1967 admin slot on superchainConfigProxy so getAdmin returns TEST_SUPERCHAIN_PROXY_ADMIN
        EIP1967Helper.setAdmin(address(input.superchainConfigProxy), TEST_SUPERCHAIN_PROXY_ADMIN);
    }

    /// @notice Mocks SuperchainConfig proxy and ProxyAdmin calls used in both OPCM v1 and v2 paths.
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

    /// @notice Mocks OPCM V1: version, protocolVersions(), superchainConfig().
    function _mockOPCMV1() internal {
        input.opcmAddress = IOPContractsManager(TEST_OPCM_IMPL);
        vm.etch(TEST_OPCM_IMPL, "0x01");

        _mockExpect(TEST_OPCM_IMPL, abi.encodeCall(IOPContractsManager.version, ()), abi.encode("6.0.0"));
        _mockExpect(
            TEST_OPCM_IMPL,
            abi.encodeCall(IOPContractsManager.protocolVersions, ()),
            abi.encode(TEST_PROTOCOL_VERSIONS_PROXY)
        );
        _mockExpect(
            TEST_OPCM_IMPL,
            abi.encodeCall(IOPContractsManager.superchainConfig, ()),
            abi.encode(address(input.superchainConfigProxy))
        );
    }

    /// @notice Mocks ProtocolVersions proxy: implementation(), owner(), recommended(), required().
    function _mockProtocolVersionsCalls() internal {
        _mockExpect(
            TEST_PROTOCOL_VERSIONS_PROXY,
            abi.encodeCall(IProxy.implementation, ()),
            abi.encode(TEST_PROTOCOL_VERSIONS_IMPL)
        );
        _mockExpect(
            TEST_PROTOCOL_VERSIONS_PROXY,
            abi.encodeCall(IProtocolVersions.owner, ()),
            abi.encode(TEST_PROTOCOL_VERSIONS_OWNER)
        );
        _mockExpect(
            TEST_PROTOCOL_VERSIONS_PROXY,
            abi.encodeCall(IProtocolVersions.recommended, ()),
            abi.encode(ProtocolVersion.wrap(TEST_RECOMMENDED_VERSION))
        );
        _mockExpect(
            TEST_PROTOCOL_VERSIONS_PROXY,
            abi.encodeCall(IProtocolVersions.required, ()),
            abi.encode(ProtocolVersion.wrap(TEST_REQUIRED_VERSION))
        );
    }

    /// @notice Internal helper to mock and expect calls.
    function _mockExpect(address _target, bytes memory _callData, bytes memory _returnData) internal {
        vm.mockCall(_target, _callData, _returnData);
        vm.expectCall(_target, _callData);
    }
}
