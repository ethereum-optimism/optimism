// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test, StdUtils } from "forge-std/Test.sol";
import { Safe } from "safe-contracts/Safe.sol";
import { SafeProxyFactory } from "safe-contracts/proxies/SafeProxyFactory.sol";
import { ModuleManager } from "safe-contracts/base/ModuleManager.sol";
import { Enum } from "safe-contracts/common/Enum.sol";
import "test/safe-tools/SafeTestTools.sol";

import { LivenessModule } from "src/Safe/LivenessModule.sol";
import { LivenessGuard } from "src/Safe/LivenessGuard.sol";

contract LivenessModule_TestInit is Test, SafeTestTools {
    using SafeTestLib for SafeInstance;

    event SignersRecorded(bytes32 indexed txHash, address[] signers);

    LivenessModule livenessModule;
    LivenessGuard livenessGuard;
    SafeInstance safeInstance;
    address fallbackOwner;

    function setUp() public {
        // Create a Safe with 10 owners
        (, uint256[] memory keys) = makeAddrsAndKeys(10);
        safeInstance = _setupSafe(keys, 8);
        livenessGuard = new LivenessGuard(safeInstance.safe);
        fallbackOwner = makeAddr("fallbackOwner");
        livenessModule = new LivenessModule({
            _safe: safeInstance.safe,
            _livenessGuard: livenessGuard,
            _livenessInterval: 30 days,
            _minOwners: 6,
            _fallbackOwner: fallbackOwner
        });
        safeInstance.enableModule(address(livenessModule));
        safeInstance.setGuard(address(livenessGuard));
    }
}

contract LivenessModule_Getters_Test is LivenessModule_TestInit {
    function test_getters_works() external {
        assertEq(address(livenessModule.safe()), address(safeInstance.safe));
        assertEq(address(livenessModule.livenessGuard()), address(livenessGuard));
        assertEq(livenessModule.livenessInterval(), 30 days);
        assertEq(livenessModule.minOwners(), 6);
        assertEq(livenessModule.fallbackOwner(), fallbackOwner);
    }
}

contract LivenessModule_Get75PercentThreshold_Test is LivenessModule_TestInit {
    /// @dev check the return values of the get75PercentThreshold function against manually
    ///      calculated values.
    function test_get75PercentThreshold_Works() external {
        assertEq(livenessModule.get75PercentThreshold(20), 15);
        assertEq(livenessModule.get75PercentThreshold(19), 15);
        assertEq(livenessModule.get75PercentThreshold(18), 14);
        assertEq(livenessModule.get75PercentThreshold(17), 13);
        assertEq(livenessModule.get75PercentThreshold(16), 12);
        assertEq(livenessModule.get75PercentThreshold(15), 12);
        assertEq(livenessModule.get75PercentThreshold(14), 11);
        assertEq(livenessModule.get75PercentThreshold(13), 10);
        assertEq(livenessModule.get75PercentThreshold(12), 9);
        assertEq(livenessModule.get75PercentThreshold(11), 9);
        assertEq(livenessModule.get75PercentThreshold(10), 8);
        assertEq(livenessModule.get75PercentThreshold(9), 7);
        assertEq(livenessModule.get75PercentThreshold(8), 6);
        assertEq(livenessModule.get75PercentThreshold(7), 6);
        assertEq(livenessModule.get75PercentThreshold(6), 5);
        assertEq(livenessModule.get75PercentThreshold(5), 4);
        assertEq(livenessModule.get75PercentThreshold(4), 3);
        assertEq(livenessModule.get75PercentThreshold(3), 3);
        assertEq(livenessModule.get75PercentThreshold(2), 2);
        assertEq(livenessModule.get75PercentThreshold(1), 1);
    }
}

contract LivenessModule_RemoveOwner_Test is LivenessModule_TestInit {
    function test_removeOwner_oneOwner_succeeds() external {
        uint256 ownersBefore = safeInstance.owners.length;
        vm.warp(block.timestamp + 30 days);
        livenessModule.removeOwner(safeInstance.owners[0]);
        assertEq(safeInstance.safe.getOwners().length, ownersBefore - 1);
    }

    function test_removeOwner_allOwners_succeeds() external {
        vm.warp(block.timestamp + 30 days);
        // The safe is initialized with 10 owners, so we need to remove 3 to get below the minOwners threshold
        livenessModule.removeOwner(safeInstance.owners[0]);
        livenessModule.removeOwner(safeInstance.owners[1]);
        livenessModule.removeOwner(safeInstance.owners[2]);
    }
}
