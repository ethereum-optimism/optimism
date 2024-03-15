// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { StdUtils } from "forge-std/StdUtils.sol";
import { StdCheats } from "forge-std/StdCheats.sol";
import { Safe, OwnerManager } from "safe-contracts/Safe.sol";
import { SafeProxyFactory } from "safe-contracts/proxies/SafeProxyFactory.sol";
import { ModuleManager } from "safe-contracts/base/ModuleManager.sol";
import { Enum } from "safe-contracts/common/Enum.sol";
import "test/safe-tools/SafeTestTools.sol";
import { EnumerableSet } from "@openzeppelin/contracts/utils/structs/EnumerableSet.sol";

import { LivenessGuard } from "src/Safe/LivenessGuard.sol";
import { VetoerGuard } from "src/Safe/VetoerSet/VetoerGuard.sol";

contract VetoerGuard_TestInit is Test, SafeTestTools {
    using SafeTestLib for SafeInstance;

    uint256 initTime = 10;
    VetoerGuard vetoerGuard;
    SafeInstance safeInstance;

    /// @dev Sets up the test environment
    function setUp() public {
        vm.warp(initTime);
        safeInstance = _setupSafe();
        vetoerGuard = new VetoerGuard(safeInstance.safe);
        safeInstance.setGuard(address(vetoerGuard));
    }
}

contract VetoerGuard_UpdateMaxCount_test is VetoerGuard_TestInit {
    function test_updateMaxCount() public {
        vm.prank(address(safeInstance.safe));
        vetoerGuard.updateMaxCount(10);
        assertEq(vetoerGuard.maxCount(), 10);
    }
}

contract VetoerGuard_UnauthedUpdateMaxCount_test is VetoerGuard_TestInit {
    function test_unauthedupdateMaxCount() public {
        vm.expectRevert("VetoerGuard: only Safe can call this function");
        vetoerGuard.updateMaxCount(10);
    }
}

contract VetoerGuard_CheckAfterExecution_test is VetoerGuard_TestInit {
    using SafeTestLib for SafeInstance;

    function test_checkAfterExecution() public {
        vm.prank(address(safeInstance.safe));
        safeInstance.safe.addOwnerWithThreshold(vm.addr(1), 1);
        vm.prank(address(safeInstance.safe));
        vm.expectRevert("VetoerGuard: Safe must have a threshold of at least 66% of the number of owners");
        vetoerGuard.checkAfterExecution(0, false);
    }
}
