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
import { OwnerGuard } from "src/Safe/VetoerSet/OwnerGuard.sol";

contract OwnerGuard_TestInit is Test, SafeTestTools {
    using SafeTestLib for SafeInstance;

    uint256 initTime = 10;
    OwnerGuard ownerGuard;
    SafeInstance safeInstance;

    /// @dev Sets up the test environment
    function setUp() public {
        vm.warp(initTime);
        safeInstance = _setupSafe();
        ownerGuard = new OwnerGuard(safeInstance.safe);
        safeInstance.setGuard(address(ownerGuard));
    }
}

contract OwnerGuard_UpdateMaxCount_test is OwnerGuard_TestInit {
    function test_updateMaxCount() public {
        vm.prank(address(safeInstance.safe));
        ownerGuard.updateMaxCount(10);
        assertEq(ownerGuard.maxCount(), 10);
    }
}

contract OwnerGuard_UnauthedUpdateMaxCount_test is OwnerGuard_TestInit {
    function test_unauthedupdateMaxCount() public {
        vm.expectRevert("OwnerGuard: only Safe can call this function");
        ownerGuard.updateMaxCount(10);
    }
}

contract OwnerGuard_CheckAfterExecution_test is OwnerGuard_TestInit {
    using SafeTestLib for SafeInstance;

    function test_checkAfterExecution() public {
        vm.prank(address(safeInstance.safe));
        safeInstance.safe.addOwnerWithThreshold(vm.addr(1), 1);
        vm.prank(address(safeInstance.safe));
        vm.expectRevert("OwnerGuard: Safe must have a threshold of at least 66% of the number of owners");
        ownerGuard.checkAfterExecution(0, false);
    }
}
