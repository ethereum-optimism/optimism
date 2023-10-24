// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { Safe, OwnerManager } from "safe-contracts/Safe.sol";
import { SafeProxyFactory } from "safe-contracts/proxies/SafeProxyFactory.sol";
import { ModuleManager } from "safe-contracts/base/ModuleManager.sol";
import { Enum } from "safe-contracts/common/Enum.sol";
import "test/safe-tools/SafeTestTools.sol";

import { LivenessGuard } from "src/Safe/LivenessGuard.sol";

// Todo(Maurelian):
// Other tests needed:
//   - EIP1271 signatures
//   - Signatures from contracts
//   - Signatures from non-owners
//   - Signers may call directly to prove liveness (must be an owner).
//   - Unexpected length of signature data

contract LivenessGuard_TestInit is Test, SafeTestTools {
    using SafeTestLib for SafeInstance;

    event SignersRecorded(bytes32 indexed txHash, address[] signers);

    LivenessGuard livenessGuard;
    SafeInstance safeInstance;

    function setUp() public {
        safeInstance = _setupSafe();
        livenessGuard = new LivenessGuard(safeInstance.safe);
        safeInstance.setGuard(address(livenessGuard));
    }
}

contract LivenessGuard_Getters_Test is LivenessGuard_TestInit {
    function test_getters_works() external {
        assertEq(address(livenessGuard.safe()), address(safeInstance.safe));
        assertEq(livenessGuard.lastLive(address(0)), 0);
    }
}

contract LivenessGuard_CheckTx_TestFails is LivenessGuard_TestInit {
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

    function test_checkTransaction_succeeds() external {
        // Create an array of the addresses who will sign the transaction. SafeTestTools
        // will generate these signatures up to the threshold by iterating over the owners array.
        address[] memory signers = new address[](safeInstance.threshold);
        signers[0] = safeInstance.owners[0];
        signers[1] = safeInstance.owners[1];

        // Don't check topic1 so that we can avoid the ugly txHash calculation.
        vm.expectEmit(false, true, true, true, address(livenessGuard));
        emit SignersRecorded(0x0, signers);
        vm.expectCall(address(safeInstance.safe), abi.encodeWithSignature("nonce()"));
        vm.expectCall(address(safeInstance.safe), abi.encodeCall(OwnerManager.getThreshold, ()));
        safeInstance.execTransaction({ to: address(1111), value: 0, data: hex"abba" });

        for (uint256 i; i < safeInstance.threshold; i++) {
            assertEq(livenessGuard.lastLive(safeInstance.owners[i]), block.timestamp);
        }
    }
}

contract LivenessGuard_CheckAfterExecution_TestFails is LivenessGuard_TestInit {
    function test_checkAfterExecution_callerIsNotSafe_revert() external {
        vm.expectRevert("LivenessGuard: only Safe can call this function");
        livenessGuard.checkAfterExecution(bytes32(0), false);
    }
}

contract LivenessGuard_CheckAfterExecution_Test is LivenessGuard_TestInit { }

contract LivenessGuard_ShowLiveness_Test is LivenessGuard_TestInit {
    function test_showLiveness_succeeds() external {
        // Cache the caller
        address caller = safeInstance.owners[0];

        // Construct a signers array with just the caller to identify the expected event.
        address[] memory signers = new address[](1);
        signers[0] = caller;
        vm.expectEmit(address(livenessGuard));
        emit SignersRecorded(0x0, signers);

        vm.prank(caller);
        livenessGuard.showLiveness();

        assertEq(livenessGuard.lastLive(caller), block.timestamp);
    }
}
