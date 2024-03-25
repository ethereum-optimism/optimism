// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test, StdUtils } from "forge-std/Test.sol";
import { CommonTest } from "test/setup/CommonTest.sol";
import { Safe } from "safe-contracts/Safe.sol";
import { Enum } from "safe-contracts/common/Enum.sol";
import "test/safe-tools/SafeTestTools.sol";

import { DeputyGuardianModule } from "src/Safe/DeputyGuardianModule.sol";
import { LivenessGuard } from "src/Safe/LivenessGuard.sol";

contract DeputyGuardianModule_TestInit is CommonTest, SafeTestTools {
    using SafeTestLib for SafeInstance;

    // uint256 initTime = 10;
    DeputyGuardianModule deputyGuardianModule;
    SafeInstance safeInstance;
    address deputyGuardian;

    /// @dev Sets up the test environment
    function setUp() public virtual override {
        super.setUp();
        // Set the block timestamp to the initTime, so that signatures recorded in the first block
        // are non-zero.
        // vm.warp(initTime);

        // Create a Safe with 10 owners
        (, uint256[] memory keys) = SafeTestLib.makeAddrsAndKeys("moduleTest", 10);
        safeInstance = _setupSafe(keys, 10);

        // Set the Safe as the Guardian of the SuperchainConfig
        vm.store(
            address(superchainConfig),
            superchainConfig.GUARDIAN_SLOT(),
            bytes32(uint256(uint160(address(safeInstance.safe))))
        );

        deputyGuardian = makeAddr("deputyGuardian");

        deputyGuardianModule = new DeputyGuardianModule({
            _safe: safeInstance.safe,
            _superchainConfig: superchainConfig,
            _deputyGuardian: deputyGuardian
        });
        safeInstance.enableModule(address(deputyGuardianModule));
    }
}

contract DeputyGuardianModule_Getters_Test is DeputyGuardianModule_TestInit {
    /// @dev Tests that the constructor sets the correct values
    function test_getters_works() external {
        assertEq(address(deputyGuardianModule.safe()), address(safeInstance.safe));
        assertEq(address(deputyGuardianModule.deputyGuardian()), address(deputyGuardian));
        assertEq(address(deputyGuardianModule.superchainConfig()), address(superchainConfig));
    }
}

contract DeputyGuardianModule_Pause_Test is DeputyGuardianModule_TestInit {
    /// @dev Tests that `pause` successfully pauses when called by the deputy guardian.
    function test_pause_succeeds() external {
        // Pause the SuperchainConfig contract
        vm.prank(address(deputyGuardian));
        deputyGuardianModule.pause();
        assertEq(superchainConfig.paused(), true);
    }
}

contract DeputyGuardianModule_Pause_TestFail is DeputyGuardianModule_TestInit {
    /// @dev Tests that `pause` reverts when called by a non deputy guardian.
    function test_pause_notDeputyGuardian_reverts() external {
        vm.expectRevert("DeputyGuardianModule: Only the deputy guardian can pause the contract");
        deputyGuardianModule.pause();
    }
}

contract DeputyGuardianModule_Unpause_Test is DeputyGuardianModule_TestInit {
    /// @dev Sets up the test environment with the SuperchainConfig paused
    function setUp() public override {
        super.setUp();
        vm.prank(address(deputyGuardian));
        deputyGuardianModule.pause();
        assertEq(superchainConfig.paused(), true);
    }

    /// @dev Tests that `unpause` successfully unpauses when called by the deputy guardian.
    function test_unpause_succeeds() external {
        assertEq(superchainConfig.paused(), true);

        vm.prank(address(deputyGuardian));
        deputyGuardianModule.unpause();
        assertEq(superchainConfig.paused(), false);
    }
}

contract DeputyGuardianModule_Unpause_TestFail is DeputyGuardianModule_Unpause_Test {
    /// @dev Tests that `unpause` reverts when called by a non deputy guardian.
    function test_pause_notDeputyGuardian_reverts() external {
        assertEq(superchainConfig.paused(), true);

        vm.expectRevert();
        deputyGuardianModule.pause();
    }
}

contract DeputyGuardianModule_BlacklistDisputeGame_Test is DeputyGuardianModule_TestInit { }

contract DeputyGuardianModule_SetRespectedGameType_Test is DeputyGuardianModule_TestInit { }
