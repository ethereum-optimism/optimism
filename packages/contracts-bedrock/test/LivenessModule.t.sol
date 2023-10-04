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

contract LivnessModule_TestInit is Test, SafeTestTools {
    using SafeTestLib for SafeInstance;

    event SignersRecorded(bytes32 indexed txHash, address[] signers);

    LivenessModule livenessModule;
    LivenessGuard livenessGuard;
    SafeInstance safeInstance;

    function makeKeys(uint256 num) public pure returns (uint256[] memory keys_) {
        keys_ = new uint256[](num);
        for (uint256 i; i < num; i++) {
            keys_[i] = uint256(keccak256(abi.encodePacked(i)));
        }
    }

    function setUp() public {
        // Create a Safe with 10 owners
        uint256[] memory keys = makeKeys(10);
        safeInstance = _setupSafe(keys, 8);
        livenessGuard = new LivenessGuard(safeInstance.safe);
        livenessModule = new LivenessModule({
            _safe: safeInstance.safe,
            _livenessGuard: livenessGuard,
            _livenessInterval: 30 days,
            _minOwners: 6,
            _fallbackOwner: makeAddr("fallbackOwner")
        });
        safeInstance.enableModule(address(livenessModule));
    }
}

contract LivenessModule_RemoveOwner_Test is LivnessModule_TestInit {
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
