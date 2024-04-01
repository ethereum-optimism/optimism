// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Safe, Enum, OwnerManager, ModuleManager } from "safe-contracts/Safe.sol";

import { VetoModule } from "src/Safe/VetoerSet/VetoModule.sol";

import "forge-std/Test.sol";

contract TestVetoModule is Test {
    address private _safe;
    address private _delayedVetoable;
    VetoModule private _sut;

    function setUp() public {
        _safe = makeAddr("Safe");
        _delayedVetoable = makeAddr("DelayedVetoable");
        _sut = new VetoModule({ _safe: Safe(payable(_safe)), _delayedVetoable: _delayedVetoable });
    }

    /// @dev `veto` should revert with `SenderIsNotAnOwner` when the sender is not an owner of the Safe Account.
    function testRevert_Veto_SenderIsNotAnOwner(address sender) public {
        // Mock the dependencies.
        {
            // Mock `safe.isOwner()` to return `false` for the given `sender`.
            vm.mockCall(_safe, abi.encodeCall(OwnerManager.isOwner, (sender)), abi.encode(false));
        }

        vm.expectRevert(abi.encodeWithSelector(VetoModule.SenderIsNotAnOwner.selector, (sender)));
        vm.prank(sender);
        _sut.veto();
    }

    /// @dev `veto` should forward the call to the Safe Account by calling its `execTransactionFromModule` method.
    function test_Veto_CallsExecTransactionFromModule(address sender) public {
        // Mock the dependencies.
        bytes memory execTransactionFromModuleCall;
        {
            // Mock `safe.isOwner()` to return `true` for the given `sender`.
            vm.mockCall(_safe, abi.encodeCall(OwnerManager.isOwner, (sender)), abi.encode(true));

            // Mock `safe.execTransactionFromModule()` to return `true` on the expected call.
            execTransactionFromModuleCall = abi.encodeCall(
                ModuleManager.execTransactionFromModule,
                (_delayedVetoable, 0, abi.encodeWithSelector(VetoModule.veto.selector), Enum.Operation.Call)
            );
            vm.mockCall(_safe, execTransactionFromModuleCall, abi.encode(true));
        }

        vm.expectCall(_safe, execTransactionFromModuleCall);
        vm.prank(sender);
        _sut.veto();
    }
}
