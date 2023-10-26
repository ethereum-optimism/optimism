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

contract OwnerSimulator is OwnerManager {
    constructor(address[] memory _owners, uint256 _threshold) {
        setupOwners(_owners, _threshold);
    }

    function removeOwnerWrapped(address prevOwner, address owner, uint256 _threshold) public {
        OwnerManager(address(this)).removeOwner(prevOwner, owner, _threshold);
    }
}

contract LivenessModule_TestInit is Test, SafeTestTools {
    using SafeTestLib for SafeInstance;

    /// @notice The address of the first owner in the linked list of owners
    address internal constant SENTINEL_OWNERS = address(0x1);

    event SignersRecorded(bytes32 indexed txHash, address[] signers);

    LivenessModule livenessModule;
    LivenessGuard livenessGuard;
    SafeInstance safeInstance;
    OwnerSimulator ownerSimulator;
    address fallbackOwner;

    /// @notice Get the previous owner in the linked list of owners
    /// @param _owner The owner whose previous owner we want to find
    /// @param _owners The list of owners
    function _getPrevOwner(address _owner, address[] memory _owners) internal pure returns (address prevOwner_) {
        for (uint256 i = 0; i < _owners.length; i++) {
            if (_owners[i] != _owner) continue;
            if (i == 0) {
                prevOwner_ = SENTINEL_OWNERS;
                break;
            }
            prevOwner_ = _owners[i - 1];
        }
    }

    /// @notice Given an arrary of owners to remove, this function will return an array of the previous owners
    ///         in the order that they must be provided to the LivenessMoules's removeOwners() function.
    ///         Because owners are removed one at a time, and not necessarily in order, we need to simulate
    ///         the owners list after each removal, in order to identify the correct previous owner.
    /// @param _ownersToRemove The owners to remove
    /// @return prevOwners_ The previous owners in the linked list
    function _getPrevOwners(address[] memory _ownersToRemove) internal returns (address[] memory prevOwners_) {
        prevOwners_ = new address[](_ownersToRemove.length);
        address[] memory currentOwners;
        for (uint256 i = 0; i < _ownersToRemove.length; i++) {
            currentOwners = ownerSimulator.getOwners();
            prevOwners_[i] = _getPrevOwner(safeInstance.owners[i], currentOwners);
            if (currentOwners.length == 1) break;
            ownerSimulator.removeOwnerWrapped(prevOwners_[i], _ownersToRemove[i], 1);
        }
    }

    function setUp() public {
        // Create a Safe with 10 owners
        (, uint256[] memory keys) = makeAddrsAndKeys(10);
        safeInstance = _setupSafe(keys, 8);
        ownerSimulator = new OwnerSimulator(safeInstance.owners, 1);

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
        address[] memory prevOwners = new address[](1);
        address[] memory ownersToRemove = new address[](1);
        ownersToRemove[0] = safeInstance.owners[0];
        prevOwners[0] = _getPrevOwner(safeInstance.owners[0], safeInstance.owners);

        vm.warp(block.timestamp + 30 days);

        livenessModule.removeOwners(prevOwners, ownersToRemove);
        assertEq(safeInstance.safe.getOwners().length, ownersBefore - 1);
    }

    function test_removeOwner_allOwners_succeeds() external {
        uint256 numOwners = safeInstance.owners.length;

        address[] memory ownersToRemove = new address[](numOwners);
        for (uint256 i = 0; i < numOwners; i++) {
            ownersToRemove[i] = safeInstance.owners[i];
        }
        address[] memory prevOwners = _getPrevOwners(ownersToRemove);

        vm.warp(block.timestamp + 30 days);
        livenessModule.removeOwners(prevOwners, ownersToRemove);
        assertEq(safeInstance.safe.getOwners().length, 1);
        assertEq(safeInstance.safe.getOwners()[0], fallbackOwner);
        assertEq(safeInstance.safe.getThreshold(), 1);
    }
}
