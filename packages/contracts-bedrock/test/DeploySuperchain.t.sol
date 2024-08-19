// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";

import { Proxy } from "src/universal/Proxy.sol";
import { ProtocolVersion } from "src/L1/ProtocolVersions.sol";
import { DeploySuperchainInput, DeploySuperchain, DeploySuperchainOutput } from "scripts/DeploySuperchain.s.sol";

/// @notice Deploys the Superchain contracts that can be shared by many chains.
contract DeploySuperchain_Test is Test {
    DeploySuperchain deploySuperchain;
    DeploySuperchainInput dsi;
    DeploySuperchainOutput dso;

    // Define a default input struct for testing.
    DeploySuperchainInput.Input input = DeploySuperchainInput.Input({
        roles: DeploySuperchainInput.Roles({
            proxyAdminOwner: makeAddr("defaultProxyAdminOwner"),
            protocolVersionsOwner: makeAddr("defaultProtocolVersionsOwner"),
            guardian: makeAddr("defaultGuardian")
        }),
        paused: false,
        requiredProtocolVersion: ProtocolVersion.wrap(1),
        recommendedProtocolVersion: ProtocolVersion.wrap(2)
    });

    function setUp() public {
        deploySuperchain = new DeploySuperchain();
        (dsi, dso) = deploySuperchain.getIOContracts();
    }

    function unwrap(ProtocolVersion _pv) internal pure returns (uint256) {
        return ProtocolVersion.unwrap(_pv);
    }

    function test_run_succeeds(DeploySuperchainInput.Input memory _input) public {
        vm.assume(_input.roles.proxyAdminOwner != address(0));
        vm.assume(_input.roles.protocolVersionsOwner != address(0));
        vm.assume(_input.roles.guardian != address(0));

        DeploySuperchainOutput.Output memory output = deploySuperchain.run(_input);

        // Assert that individual input fields were properly set based on the input struct.
        assertEq(_input.roles.proxyAdminOwner, dsi.proxyAdminOwner(), "100");
        assertEq(_input.roles.protocolVersionsOwner, dsi.protocolVersionsOwner(), "200");
        assertEq(_input.roles.guardian, dsi.guardian(), "300");
        assertEq(_input.paused, dsi.paused(), "400");
        assertEq(unwrap(_input.requiredProtocolVersion), unwrap(dsi.requiredProtocolVersion()), "500");
        assertEq(unwrap(_input.recommendedProtocolVersion), unwrap(dsi.recommendedProtocolVersion()), "600");

        // Assert that individual output fields were properly set based on the output struct.
        assertEq(address(output.superchainProxyAdmin), address(dso.superchainProxyAdmin()), "700");
        assertEq(address(output.superchainConfigImpl), address(dso.superchainConfigImpl()), "800");
        assertEq(address(output.superchainConfigProxy), address(dso.superchainConfigProxy()), "900");
        assertEq(address(output.protocolVersionsImpl), address(dso.protocolVersionsImpl()), "1000");
        assertEq(address(output.protocolVersionsProxy), address(dso.protocolVersionsProxy()), "1100");

        // Assert that the full input and output structs were properly set.
        assertEq(keccak256(abi.encode(_input)), keccak256(abi.encode(DeploySuperchainInput(dsi).input())), "1200");
        assertEq(keccak256(abi.encode(output)), keccak256(abi.encode(DeploySuperchainOutput(dso).output())), "1300");

        // Assert inputs were properly passed through to the contract initializers.
        assertEq(address(output.superchainProxyAdmin.owner()), _input.roles.proxyAdminOwner, "1400");
        assertEq(address(output.protocolVersionsProxy.owner()), _input.roles.protocolVersionsOwner, "1500");
        assertEq(address(output.superchainConfigProxy.guardian()), _input.roles.guardian, "1600");
        assertEq(output.superchainConfigProxy.paused(), _input.paused, "1700");
        assertEq(unwrap(output.protocolVersionsProxy.required()), unwrap(_input.requiredProtocolVersion), "1800");
        assertEq(unwrap(output.protocolVersionsProxy.recommended()), unwrap(_input.recommendedProtocolVersion), "1900");

        // Architecture assertions.
        // We prank as the zero address due to the Proxy's `proxyCallIfNotAdmin` modifier.
        Proxy superchainConfigProxy = Proxy(payable(address(output.superchainConfigProxy)));
        Proxy protocolVersionsProxy = Proxy(payable(address(output.protocolVersionsProxy)));

        vm.startPrank(address(0));
        assertEq(superchainConfigProxy.implementation(), address(output.superchainConfigImpl), "900");
        assertEq(protocolVersionsProxy.implementation(), address(output.protocolVersionsImpl), "1000");
        assertEq(superchainConfigProxy.admin(), protocolVersionsProxy.admin(), "1100");
        assertEq(superchainConfigProxy.admin(), address(output.superchainProxyAdmin), "1200");
        vm.stopPrank();

        // Ensure that `checkOutput` passes. This is called by the `run` function during execution,
        // so this just acts as a sanity check. It reverts on failure.
        dso.checkOutput();
    }

    function test_run_ZeroAddressRoles_reverts() public {
        // Snapshot the state so we can revert to the default `input` struct between assertions.
        uint256 snapshotId = vm.snapshot();

        // Assert over each role being set to the zero address.
        input.roles.proxyAdminOwner = address(0);
        vm.expectRevert("DeploySuperchainInput: Null proxyAdminOwner");
        deploySuperchain.run(input);

        vm.revertTo(snapshotId);
        input.roles.protocolVersionsOwner = address(0);
        vm.expectRevert("DeploySuperchainInput: Null protocolVersionsOwner");
        deploySuperchain.run(input);

        vm.revertTo(snapshotId);
        input.roles.guardian = address(0);
        vm.expectRevert("DeploySuperchainInput: Null guardian");
        deploySuperchain.run(input);
    }
}
