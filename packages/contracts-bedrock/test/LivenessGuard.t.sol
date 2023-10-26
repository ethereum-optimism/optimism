// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { Safe, OwnerManager } from "safe-contracts/Safe.sol";
import { SafeProxyFactory } from "safe-contracts/proxies/SafeProxyFactory.sol";
import { ModuleManager } from "safe-contracts/base/ModuleManager.sol";
import { Enum } from "safe-contracts/common/Enum.sol";
import "test/safe-tools/SafeTestTools.sol";

import { LivenessGuard } from "src/Safe/LivenessGuard.sol";

contract LivenessGuard_TestInit is Test, SafeTestTools {
    using SafeTestLib for SafeInstance;

    event OwnerRecorded(bytes32 indexed txHash, address signer);

    uint256 initTime = 10;
    LivenessGuard livenessGuard;
    SafeInstance safeInstance;

    /// @dev Sets up the test environment
    function setUp() public {
        vm.warp(initTime);
        safeInstance = _setupSafe();
        livenessGuard = new LivenessGuard(safeInstance.safe);
        safeInstance.setGuard(address(livenessGuard));
    }
}

contract LivenessGuard_Constructor_Test is LivenessGuard_TestInit {
    /// @dev Tests that the constructor correctly sets the current time as the lastLive time for each owner
    function test_constructor_works() external {
        address[] memory owners = safeInstance.owners;
        livenessGuard = new LivenessGuard(safeInstance.safe);
        for (uint256 i = 0; i < owners.length; i++) {
            assertEq(livenessGuard.lastLive(owners[i]), initTime);
        }
    }
}

contract LivenessGuard_Getters_Test is LivenessGuard_TestInit {
    /// @dev Tests that the getters return the correct values
    function test_getters_works() external {
        assertEq(address(livenessGuard.safe()), address(safeInstance.safe));
        assertEq(livenessGuard.lastLive(address(0)), 0);
    }
}

contract LivenessGuard_CheckTx_TestFails is LivenessGuard_TestInit {
    /// @dev Tests that the checkTransaction function reverts if the caller is not the Safe
    function test_checkTransaction_callerIsNotSafe_revert() external {
        vm.expectRevert("LivenessGuard: only Safe can call this function");
        livenessGuard.checkTransaction({
            to: address(0),
            value: 0,
            data: hex"00",
            operation: Enum.Operation.Call,
            safeTxGas: 0,
            baseGas: 0,
            gasPrice: 0,
            gasToken: address(0),
            refundReceiver: payable(address(0)),
            signatures: hex"00",
            msgSender: address(0)
        });
    }
}

contract LivenessGuard_CheckTx_Test is LivenessGuard_TestInit {
    using SafeTestLib for SafeInstance;

    /// @dev Tests that the checkTransaction function succeeds
    function test_checkTransaction_succeeds() external {
        // Create an array of the addresses who will sign the transaction. SafeTestTools
        // will generate these signatures up to the threshold by iterating over the owners array.
        address[] memory signers = new address[](safeInstance.threshold);
        signers[0] = safeInstance.owners[0];
        signers[1] = safeInstance.owners[1];

        for (uint256 i; i < signers.length; i++) {
            // Don't check topic1 so that we can avoid the ugly txHash calculation.
            vm.expectEmit(false, true, true, true, address(livenessGuard));
            emit OwnerRecorded(0x0, signers[i]);
        }
        vm.expectCall(address(safeInstance.safe), abi.encodeWithSignature("nonce()"));
        vm.expectCall(address(safeInstance.safe), abi.encodeCall(OwnerManager.getThreshold, ()));
        safeInstance.execTransaction({ to: address(1111), value: 0, data: hex"abba" });

        for (uint256 i; i < safeInstance.threshold; i++) {
            assertEq(livenessGuard.lastLive(safeInstance.owners[i]), block.timestamp);
        }
    }
}

contract LivenessGuard_CheckAfterExecution_TestFails is LivenessGuard_TestInit {
    /// @dev Tests that the checkAfterExecution function reverts if the caller is not the Safe
    function test_checkAfterExecution_callerIsNotSafe_revert() external {
        vm.expectRevert("LivenessGuard: only Safe can call this function");
        livenessGuard.checkAfterExecution(bytes32(0), false);
    }
}

contract LivenessGuard_ShowLiveness_TestFail is LivenessGuard_TestInit {
    /// @dev Tests that the showLiveness function reverts if the caller is not an owner
    function test_showLiveness_callIsNotSafeOwner_reverts() external {
        vm.expectRevert("LivenessGuard: only Safe owners may demonstrate liveness");
        livenessGuard.showLiveness();
    }
}

contract LivenessGuard_ShowLiveness_Test is LivenessGuard_TestInit {
    /// @dev Tests that the showLiveness function succeeds
    function test_showLiveness_succeeds() external {
        // Cache the caller
        address caller = safeInstance.owners[0];

        vm.expectEmit(address(livenessGuard));
        emit OwnerRecorded(0x0, caller);

        vm.prank(caller);
        livenessGuard.showLiveness();

        assertEq(livenessGuard.lastLive(caller), block.timestamp);
    }
}

contract LivenessGuard_OwnerManagement_Test is LivenessGuard_TestInit {
    using SafeTestLib for SafeInstance;

    /// @dev Tests that the guard correctly deletes the owner from the lastLive mapping when it is removed
    function test_removeOwner_succeeds() external {
        address ownerToRemove = safeInstance.owners[0];
        assertGe(livenessGuard.lastLive(ownerToRemove), 0);
        assertTrue(safeInstance.safe.isOwner(ownerToRemove));

        safeInstance.execTransaction({
            to: address(safeInstance.safe),
            value: 0,
            data: abi.encodeWithSelector(OwnerManager.removeOwner.selector, SafeTestLib.SENTINEL_OWNERS, ownerToRemove, 1)
        });

        assertFalse(safeInstance.safe.isOwner(ownerToRemove));
        assertEq(livenessGuard.lastLive(ownerToRemove), 0);
    }

    /// @dev Tests that the guard correctly adds an owner to the lastLive mapping when it is added
    function test_addOwner_succeeds() external {
        address ownerToAdd = makeAddr("new owner");
        assertEq(livenessGuard.lastLive(ownerToAdd), 0);
        assertFalse(safeInstance.safe.isOwner(ownerToAdd));

        safeInstance.execTransaction({
            to: address(safeInstance.safe),
            value: 0,
            data: abi.encodeWithSelector(OwnerManager.addOwnerWithThreshold.selector, ownerToAdd, 1)
        });

        assertTrue(safeInstance.safe.isOwner(ownerToAdd));
        assertEq(livenessGuard.lastLive(ownerToAdd), block.timestamp);
    }
}
