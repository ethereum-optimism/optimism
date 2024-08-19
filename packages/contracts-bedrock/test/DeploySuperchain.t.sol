// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";

import { Proxy } from "src/universal/Proxy.sol";
import { ProtocolVersion } from "src/L1/ProtocolVersions.sol";
import { DeploySuperchainInput, DeploySuperchain, DeploySuperchainOutput } from "scripts/DeploySuperchain.s.sol";

/// @notice Deploys the Superchain contracts that can be shared by many chains.
contract DeploySuperchain_Test is Test {
    DeploySuperchain deploySuperchain;

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
    }

    function unwrap(ProtocolVersion _pv) internal pure returns (uint256) {
        return ProtocolVersion.unwrap(_pv);
    }

    function test_run_succeeds(DeploySuperchainInput.Input memory _input) public {
        vm.assume(_input.roles.proxyAdminOwner != address(0));
        vm.assume(_input.roles.protocolVersionsOwner != address(0));
        vm.assume(_input.roles.guardian != address(0));

        DeploySuperchainOutput.Output memory output = deploySuperchain.run(_input);

        // Assert the inputs were properly set.
        assertEq(address(output.superchainProxyAdmin.owner()), _input.roles.proxyAdminOwner, "100");
        assertEq(address(output.protocolVersionsProxy.owner()), _input.roles.protocolVersionsOwner, "200");
        assertEq(address(output.superchainConfigProxy.guardian()), _input.roles.guardian, "300");
        assertEq(output.superchainConfigProxy.paused(), _input.paused, "400");
        assertEq(unwrap(output.protocolVersionsProxy.required()), unwrap(_input.requiredProtocolVersion), "500");
        assertEq(unwrap(output.protocolVersionsProxy.recommended()), unwrap(_input.recommendedProtocolVersion), "600");

        // Architecture assertions.
        // We prank as the zero address due to the Proxy's `proxyCallIfNotAdmin` modifier.
        Proxy superchainConfigProxy = Proxy(payable(address(output.superchainConfigProxy)));
        Proxy protocolVersionsProxy = Proxy(payable(address(output.protocolVersionsProxy)));
        vm.startPrank(address(0));

        assertEq(superchainConfigProxy.implementation(), address(output.superchainConfigImpl), "700");
        assertEq(protocolVersionsProxy.implementation(), address(output.protocolVersionsImpl), "800");
        assertEq(superchainConfigProxy.admin(), protocolVersionsProxy.admin(), "900");
        assertEq(superchainConfigProxy.admin(), address(output.superchainProxyAdmin), "1000");
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
