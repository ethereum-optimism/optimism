// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";

import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";

import { UpgradeSuperchainConfig } from "scripts/deploy/UpgradeSuperchainConfig.s.sol";
import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";

contract MockOPCM {
    event UpgradeCalled(address indexed superchainConfig, address indexed superchainProxyAdmin);

    function upgradeSuperchainConfig(ISuperchainConfig _superchainConfig, IProxyAdmin _superchainProxyAdmin) public {
        emit UpgradeCalled(address(_superchainConfig), address(_superchainProxyAdmin));
    }
}

contract UpgradeSuperchainConfig_Test is Test {
    MockOPCM mockOPCM;
    UpgradeSuperchainConfig.Input input;
    UpgradeSuperchainConfig upgradeSuperchainConfig;
    address prank;
    ISuperchainConfig superchainConfig;
    IProxyAdmin superchainProxyAdmin;

    event UpgradeCalled(address indexed superchainConfig, address indexed superchainProxyAdmin);

    function setUp() public virtual {
        mockOPCM = new MockOPCM();

        input.opcm = IOPContractsManager(address(mockOPCM));

        superchainConfig = ISuperchainConfig(makeAddr("superchainConfig"));
        superchainProxyAdmin = IProxyAdmin(makeAddr("superchainProxyAdmin"));
        prank = makeAddr("prank");

        input.superchainConfig = superchainConfig;
        input.superchainProxyAdmin = superchainProxyAdmin;
        input.prank = prank;

        upgradeSuperchainConfig = new UpgradeSuperchainConfig();
    }

    function test_upgrade_succeeds() public {
        // UpgradeCalled should be emitted by the prank since it's a delegate call.
        vm.expectEmit(address(prank));
        emit UpgradeCalled(address(superchainConfig), address(superchainProxyAdmin));
        upgradeSuperchainConfig.run(input);
    }
}
