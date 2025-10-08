// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";

import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";

import { UpgradeSuperchainConfig, UpgradeSuperchainConfigInput } from "scripts/deploy/UpgradeSuperchainConfig.s.sol";

contract UpgradeSuperchainConfigInput_Test is Test {
    UpgradeSuperchainConfigInput input;

    function setUp() public {
        input = new UpgradeSuperchainConfigInput();
    }

    function test_getters_whenNotSet_reverts() public {
        vm.expectRevert("UpgradeSuperchainConfigInput: prank not set");
        input.prank();

        vm.expectRevert("UpgradeSuperchainConfigInput: opcm not set");
        input.opcm();

        vm.expectRevert("UpgradeSuperchainConfigInput: superchainConfig not set");
        input.superchainConfig();

        vm.expectRevert("UpgradeSuperchainConfigInput: superchainProxyAdmin not set");
        input.superchainProxyAdmin();
    }

    function test_setAddress_succeeds() public {
        address mockPrank = makeAddr("prank");
        address mockOPCM = makeAddr("opcm");

        // Create mock contract at OPCM address
        vm.etch(mockOPCM, hex"01");

        input.set(input.prank.selector, mockPrank);
        input.set(input.opcm.selector, mockOPCM);

        assertEq(input.prank(), mockPrank);
        assertEq(address(input.opcm()), mockOPCM);
    }

    function test_setAddress_withZeroAddress_reverts() public {
        vm.expectRevert("UpgradeSuperchainConfigInput: cannot set zero address");
        input.set(input.prank.selector, address(0));

        vm.expectRevert("UpgradeSuperchainConfigInput: cannot set zero address");
        input.set(input.opcm.selector, address(0));
    }

    function test_set_withInvalidSelector_reverts() public {
        vm.expectRevert("UpgradeSuperchainConfigInput: unknown selector");
        input.set(bytes4(0xdeadbeef), makeAddr("test"));
    }
}

contract MockOPCM {
    event UpgradeCalled(address indexed superchainConfig, address indexed superchainProxyAdmin);

    function upgradeSuperchainConfig(ISuperchainConfig _superchainConfig, IProxyAdmin _superchainProxyAdmin) public {
        emit UpgradeCalled(address(_superchainConfig), address(_superchainProxyAdmin));
    }
}

contract UpgradeSuperchainConfig_Test is Test {
    MockOPCM mockOPCM;
    UpgradeSuperchainConfigInput uoci;
    UpgradeSuperchainConfig upgradeSuperchainConfig;
    address prank;
    ISuperchainConfig superchainConfig;
    IProxyAdmin superchainProxyAdmin;

    event UpgradeCalled(address indexed superchainConfig, address indexed superchainProxyAdmin);

    function setUp() public virtual {
        mockOPCM = new MockOPCM();
        uoci = new UpgradeSuperchainConfigInput();
        uoci.set(uoci.opcm.selector, address(mockOPCM));

        superchainConfig = ISuperchainConfig(makeAddr("superchainConfig"));
        superchainProxyAdmin = IProxyAdmin(makeAddr("superchainProxyAdmin"));
        prank = makeAddr("prank");

        uoci.set(uoci.superchainConfig.selector, address(superchainConfig));
        uoci.set(uoci.superchainProxyAdmin.selector, address(superchainProxyAdmin));
        uoci.set(uoci.prank.selector, prank);

        upgradeSuperchainConfig = new UpgradeSuperchainConfig();
    }

    function test_upgrade_succeeds() public {
        // UpgradeCalled should be emitted by the prank since it's a delegate call.
        vm.expectEmit(address(prank));
        emit UpgradeCalled(address(superchainConfig), address(superchainProxyAdmin));
        upgradeSuperchainConfig.run(uoci);
    }
}
