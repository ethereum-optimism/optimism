// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";

import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";

import { UpgradeSuperchainConfig } from "scripts/deploy/UpgradeSuperchainConfig.s.sol";
import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";

/// @title MockOPCM
/// @notice This contract is used to mock the OPCM contract and emit an event which we check for in the test.
contract MockOPCM {
    event UpgradeCalled(address indexed superchainConfig);

    function upgradeSuperchainConfig(ISuperchainConfig _superchainConfig) public {
        emit UpgradeCalled(address(_superchainConfig));
    }
}

/// @title UpgradeSuperchainConfig_Test
/// @notice This test is used to test the UpgradeSuperchainConfig script.
contract UpgradeSuperchainConfig_Run_Test is Test {
    MockOPCM mockOPCM;
    UpgradeSuperchainConfig.Input input;
    UpgradeSuperchainConfig upgradeSuperchainConfig;
    address prank;
    ISuperchainConfig superchainConfig;

    event UpgradeCalled(address indexed superchainConfig);

    /// @notice Sets up the test suite.
    function setUp() public virtual {
        mockOPCM = new MockOPCM();

        input.opcm = IOPContractsManager(address(mockOPCM));

        superchainConfig = ISuperchainConfig(makeAddr("superchainConfig"));
        prank = makeAddr("prank");

        input.superchainConfig = superchainConfig;
        input.prank = prank;

        upgradeSuperchainConfig = new UpgradeSuperchainConfig();
    }

    /// @notice Tests that the UpgradeSuperchainConfig script succeeds when called with non-zero input values.
    function test_upgrade_succeeds() public {
        // UpgradeCalled should be emitted by the prank since it's a delegate call.
        vm.expectEmit(address(prank));
        emit UpgradeCalled(address(superchainConfig));
        upgradeSuperchainConfig.run(input);
    }

    /// @notice Tests that the UpgradeSuperchainConfig script reverts when called with zero input values.
    function test_run_nullInput_reverts() public {
        input.prank = address(0);
        vm.expectRevert("UpgradeSuperchainConfig: prank not set");
        upgradeSuperchainConfig.run(input);
        input.prank = prank;

        input.opcm = IOPContractsManager(address(0));
        vm.expectRevert("UpgradeSuperchainConfig: opcm not set");
        upgradeSuperchainConfig.run(input);
        input.opcm = IOPContractsManager(address(mockOPCM));

        input.superchainConfig = ISuperchainConfig(address(0));
        vm.expectRevert("UpgradeSuperchainConfig: superchainConfig not set");
        upgradeSuperchainConfig.run(input);
        input.superchainConfig = ISuperchainConfig(address(superchainConfig));
    }
}
