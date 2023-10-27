// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test, StdUtils } from "forge-std/Test.sol";
import { Safe } from "safe-contracts/Safe.sol";
import { SafeProxyFactory } from "safe-contracts/proxies/SafeProxyFactory.sol";
import { ModuleManager } from "safe-contracts/base/ModuleManager.sol";
import { OwnerManager } from "safe-contracts/base/OwnerManager.sol";
import { Enum } from "safe-contracts/common/Enum.sol";
import "test/safe-tools/SafeTestTools.sol";

import { LivenessModule } from "src/Safe/LivenessModule.sol";
import { LivenessGuard } from "src/Safe/LivenessGuard.sol";

contract LivenessModule_TestInit is Test, SafeTestTools {
    using SafeTestLib for SafeInstance;

    event SignersRecorded(bytes32 indexed txHash, address[] signers);

    uint256 initTime = 10;
    uint256 livenessInterval = 30 days;
    uint256 minOwners = 6;
    LivenessModule livenessModule;
    LivenessGuard livenessGuard;
    SafeInstance safeInstance;
    address fallbackOwner;

    /// @dev Removes an owner from the safe
    function _removeAnOwner(address _ownerToRemove, address[] memory _owners) internal {
        address[] memory prevOwners = new address[](1);
        address[] memory ownersToRemove = new address[](1);
        ownersToRemove[0] = _ownerToRemove;
        prevOwners[0] = SafeTestLib.getPrevOwnerFromList(_ownerToRemove, _owners);

        livenessModule.removeOwners(prevOwners, ownersToRemove);
    }

    /// @dev Set the current time to after the liveness interval
    function _warpPastLivenessInterval() internal {
        vm.warp(initTime + livenessInterval + 1);
    }

    /// @dev Sets up the test environment
    function setUp() public {
        // Set the block timestamp to the initTime, so that signatures recorded in the first block
        // are non-zero.
        vm.warp(initTime);

        // Create a Safe with 10 owners
        (, uint256[] memory keys) = SafeTestLib.makeAddrsAndKeys("moduleTest", 10);
        safeInstance = _setupSafe(keys, 8);

        livenessGuard = new LivenessGuard(safeInstance.safe);
        fallbackOwner = makeAddr("fallbackOwner");
        livenessModule = new LivenessModule({
            _safe: safeInstance.safe,
            _livenessGuard: livenessGuard,
            _livenessInterval: livenessInterval,
            _minOwners: minOwners,
            _fallbackOwner: fallbackOwner
        });
        safeInstance.setGuard(address(livenessGuard));
        safeInstance.enableModule(address(livenessModule));
    }
}

contract LivenessModule_Constructor_TestFail is LivenessModule_TestInit {
    /// @dev Tests that the constructor fails if the minOwners is greater than the number of owners
    function test_constructor_minOwnersGreaterThanOwners_reverts() external {
        vm.expectRevert("LivenessModule: minOwners must be less than the number of owners");
        new LivenessModule({
            _safe: safeInstance.safe,
            _livenessGuard: livenessGuard,
            _livenessInterval: livenessInterval,
            _minOwners: 11,
            _fallbackOwner: address(0)
        });
    }

    /// @dev Tests that the constructor fails if the minOwners is greater than the number of owners
    function test_constructor_wrongThreshold_reverts() external {
        uint256 wrongThreshold = livenessModule.get75PercentThreshold(safeInstance.owners.length) - 1;
        vm.mockCall(
            address(safeInstance.safe), abi.encodeCall(OwnerManager.getThreshold, ()), abi.encode(wrongThreshold)
        );
        vm.expectRevert("LivenessModule: Safe must have a threshold of at least 75% of the number of owners");
        new LivenessModule({
            _safe: safeInstance.safe,
            _livenessGuard: livenessGuard,
            _livenessInterval: livenessInterval,
            _minOwners: minOwners,
            _fallbackOwner: address(0)
        });
    }
}

contract LivenessModule_Getters_Test is LivenessModule_TestInit {
    /// @dev Tests if the getters work correctly
    function test_getters_works() external {
        assertEq(address(livenessModule.safe()), address(safeInstance.safe));
        assertEq(address(livenessModule.livenessGuard()), address(livenessGuard));
        assertEq(livenessModule.livenessInterval(), 30 days);
        assertEq(livenessModule.minOwners(), 6);
        assertEq(livenessModule.fallbackOwner(), fallbackOwner);
    }
}

contract LivenessModule_CanRemove_TestFail is LivenessModule_TestInit {
    /// @dev Tests if canRemove work correctly
    function test_canRemove_notSafeOwner_reverts() external {
        address nonOwner = makeAddr("nonOwner");
        vm.expectRevert("LivenessModule: the owner to remove must be an owner of the Safe");
        livenessModule.canRemove(nonOwner);
    }
}

contract LivenessModule_CanRemove_Test is LivenessModule_TestInit {
    /// @dev Tests if canRemove work correctly
    function test_canRemove_works() external {
        _warpPastLivenessInterval();
        bool canRemove = livenessModule.canRemove(safeInstance.owners[0]);
        assertTrue(canRemove);
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

contract LivenessModule_RemoveOwners_TestFail is LivenessModule_TestInit {
    using SafeTestLib for SafeInstance;

    /// @dev Tests with different length owner arrays
    function test_removeOwners_differentArrayLengths_reverts() external {
        address[] memory ownersToRemove = new address[](1);
        address[] memory prevOwners = new address[](2);
        vm.expectRevert("LivenessModule: arrays must be the same length");
        livenessModule.removeOwners(prevOwners, ownersToRemove);
    }

    /// @dev Test removing an owner which has recently signed a transaction
    function test_removeOwners_ownerHasSignedRecently_reverts() external {
        /// Will sign a transaction with the first M owners in the owners list
        safeInstance.execTransaction({ to: address(1111), value: 0, data: hex"abba" });

        address[] memory owners = safeInstance.safe.getOwners();

        vm.expectRevert("LivenessModule: the owner to remove has signed recently");
        _removeAnOwner(safeInstance.owners[0], owners);
    }

    /// @dev Test removing an owner which has recently called showLiveness
    function test_removeOwners_ownerHasShownLivenessRecently_reverts() external {
        /// Will sign a transaction with the first M owners in the owners list
        vm.prank(safeInstance.owners[0]);
        livenessGuard.showLiveness();
        address[] memory owners = safeInstance.safe.getOwners();
        vm.expectRevert("LivenessModule: the owner to remove has signed recently");
        _removeAnOwner(safeInstance.owners[0], owners);
    }

    /// @dev Test removing an owner with an incorrect previous owner
    function test_removeOwners_wrongPreviousOwner_reverts() external {
        address[] memory prevOwners = new address[](1);
        address[] memory ownersToRemove = new address[](1);
        ownersToRemove[0] = safeInstance.owners[0];
        prevOwners[0] = ownersToRemove[0]; // incorrect.

        _warpPastLivenessInterval();
        vm.expectRevert("LivenessModule: failed to remove owner");
        livenessModule.removeOwners(prevOwners, ownersToRemove);
    }

    /// @dev Tests if removing all owners works correctly
    function test_removeOwners_swapToFallBackOwner_reverts() external {
        uint256 numOwners = safeInstance.owners.length;

        address[] memory ownersToRemove = new address[](numOwners);
        for (uint256 i = 0; i < numOwners; i++) {
            ownersToRemove[i] = safeInstance.owners[i];
        }
        address[] memory prevOwners = safeInstance.getPrevOwners(ownersToRemove);

        // Incorrectly set the final owner to address(0)
        ownersToRemove[ownersToRemove.length - 1] = address(0);

        _warpPastLivenessInterval();
        vm.expectRevert("LivenessModule: failed to swap to fallback owner");
        livenessModule.removeOwners(prevOwners, ownersToRemove);
    }

    /// @dev Tests if remove owners reverts if it removes too many owners without removing all of them
    function test_removeOwners_belowMinButNotEmptied_reverts() external {
        // Remove all but one owner
        uint256 numOwners = safeInstance.owners.length - 1;

        address[] memory ownersToRemove = new address[](numOwners);
        for (uint256 i = 0; i < numOwners; i++) {
            ownersToRemove[i] = safeInstance.owners[i];
        }
        address[] memory prevOwners = safeInstance.getPrevOwners(ownersToRemove);

        _warpPastLivenessInterval();
        vm.expectRevert("LivenessModule: must transfer ownership to fallback owner");
        livenessModule.removeOwners(prevOwners, ownersToRemove);
    }

    /// @dev Tests if remove owners reverts if the current Safe.guard does note match the expected
    ///      livenessGuard address.
    function test_removeOwners_guardChanged_reverts() external {
        address[] memory ownersToRemove = new address[](1);
        ownersToRemove[0] = safeInstance.owners[0];
        address[] memory prevOwners = safeInstance.getPrevOwners(ownersToRemove);

        // Change the guard
        livenessGuard = new LivenessGuard(safeInstance.safe);
        safeInstance.setGuard(address(livenessGuard));

        _warpPastLivenessInterval();
        vm.expectRevert("LivenessModule: guard has been changed");
        livenessModule.removeOwners(prevOwners, ownersToRemove);
    }

    function test_removeOwners_invalidThreshold_reverts() external {
        address[] memory ownersToRemove = new address[](0);
        address[] memory prevOwners = new address[](0);
        uint256 wrongThreshold = safeInstance.safe.getThreshold() + 1;

        vm.mockCall(
            address(safeInstance.safe), abi.encodeCall(OwnerManager.getThreshold, ()), abi.encode(wrongThreshold)
        );

        _warpPastLivenessInterval();
        vm.expectRevert("LivenessModule: Safe must have a threshold of 75% of the number of owners");
        livenessModule.removeOwners(prevOwners, ownersToRemove);
    }
}

contract LivenessModule_RemoveOwners_Test is LivenessModule_TestInit {
    using SafeTestLib for SafeInstance;

    /// @dev Tests if removing one owner works correctly
    function test_removeOwners_oneOwner_succeeds() external {
        uint256 ownersBefore = safeInstance.owners.length;
        address ownerToRemove = safeInstance.owners[0];

        _warpPastLivenessInterval();
        _removeAnOwner(ownerToRemove, safeInstance.owners);

        assertFalse(safeInstance.safe.isOwner(ownerToRemove));
        assertEq(safeInstance.safe.getOwners().length, ownersBefore - 1);
    }

    /// @dev Tests if removing all owners works correctly
    function test_removeOwners_allOwners_succeeds() external {
        uint256 numOwners = safeInstance.owners.length;

        address[] memory ownersToRemove = new address[](numOwners);
        for (uint256 i = 0; i < numOwners; i++) {
            ownersToRemove[i] = safeInstance.owners[i];
        }
        address[] memory prevOwners = safeInstance.getPrevOwners(ownersToRemove);

        _warpPastLivenessInterval();
        livenessModule.removeOwners(prevOwners, ownersToRemove);
        assertEq(safeInstance.safe.getOwners().length, 1);
        assertEq(safeInstance.safe.getOwners()[0], fallbackOwner);
        assertEq(safeInstance.safe.getThreshold(), 1);
    }
}
