// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Safe, Enum, OwnerManager, ModuleManager } from "safe-contracts/Safe.sol";

import { OwnerGuard } from "src/Safe/VetoerSet/OwnerGuard.sol";
import { AddOwnerModule } from "src/Safe/VetoerSet/AddOwnerModule.sol";

import "forge-std/Test.sol";

contract TestAddOwnerModule is Test {
    address private _safe;
    address private _ownerGuard;
    address private _opFoundation;
    AddOwnerModule private _sut;

    function setUp() public {
        _safe = makeAddr("Safe");
        _ownerGuard = makeAddr("OwnerGuard");
        _opFoundation = makeAddr("OPFoundation");
        _sut = new AddOwnerModule({
            safe_: Safe(payable(_safe)),
            ownerGuard_: OwnerGuard(_ownerGuard),
            opFoundation_: _opFoundation
        });
    }

    /// @dev `addOwner` should revert with `SenderIsNotOpFoundation` when the sender is not the registered OP
    ///       foundation address.
    function testRevert_AddOwner_SenderIsNotOpFoundation(address newOwner, address sender) public {
        // Ensure the inputs are reasonable values.
        {
            vm.assume(sender != _opFoundation);
            vm.assume(newOwner != address(0x0));
            vm.assume(newOwner != address(0x1));
        }

        vm.expectRevert(abi.encodeWithSelector(AddOwnerModule.SenderIsNotOpFoundation.selector, (sender)));
        vm.prank(sender);
        _sut.addOwner(newOwner);
    }

    /// @dev `addOwner` should buble up the `InvalidOwnerCount` error returned by the `OwnerGuard` when
    ///      checking if adding the new owner does not exceed `maxOwnerCount`.
    function testRevert_AddOwner_InvalidOwnerCount(
        address newOwner,
        uint256 initialOwnerCount,
        uint256 maxOwnerCount
    )
        public
    {
        // Ensure the inputs are reasonable values.
        {
            maxOwnerCount = bound(maxOwnerCount, 7, 255);
            initialOwnerCount = bound(initialOwnerCount, maxOwnerCount, 511);
        }

        // Mock the dependencies.
        uint256 newOwnerCount;
        bytes memory invalidOwnerCountError;
        {
            // Mock `safe.getOwners()` to return a list of addresses of length `initialOwnerCount`.
            vm.mockCall(
                _safe,
                abi.encodeWithSelector(OwnerManager.getOwners.selector),
                abi.encode(new address[](initialOwnerCount))
            );

            // Mock `ownerGuard.checkNewOwnerCount()` to revert with the `InvalidOwnerCount` erro.
            newOwnerCount = initialOwnerCount + 1;
            invalidOwnerCountError =
                abi.encodeWithSelector(OwnerGuard.InvalidOwnerCount.selector, newOwnerCount, maxOwnerCount);
            vm.mockCallRevert(
                _ownerGuard, abi.encodeWithSelector(OwnerGuard.checkNewOwnerCount.selector), invalidOwnerCountError
            );
        }

        vm.expectRevert(invalidOwnerCountError);
        vm.prank(_opFoundation);
        _sut.addOwner(newOwner);
    }

    /// @dev `addOwner` should call `execTransactionFromModule` on the Safe Account with the abi encoded call
    ///      to its own `addOwnerWithThreshold` method.
    function test_AddOwner_CallsExecTransactionFromModule(
        address newOwner,
        uint256 initialOwnerCount,
        uint256 maxOwnerCount
    )
        public
    {
        // Ensure the inputs are reasonable values.
        {
            maxOwnerCount = bound(maxOwnerCount, 7, 255);
            initialOwnerCount = bound(initialOwnerCount, 1, maxOwnerCount - 1);
        }

        // Mock the dependencies.
        bytes memory execTransactionFromModuleCall;
        {
            // Mock `safe.getOwners()` to return a list of addresses of length `initialOwnerCount`.
            vm.mockCall(
                _safe,
                abi.encodeWithSelector(OwnerManager.getOwners.selector),
                abi.encode(new address[](initialOwnerCount))
            );

            // Mock `ownerGuard.checkNewOwnerCount()` to return `newThreshold`.
            uint256 newOwnerCount = initialOwnerCount + 1;
            uint256 newThreshold = (newOwnerCount * 66 + 99) / 100;
            vm.mockCall(
                _ownerGuard, abi.encodeWithSelector(OwnerGuard.checkNewOwnerCount.selector), abi.encode(newThreshold)
            );

            // Mock `safe.execTransactionFromModule()` to return `true` on the expected call.
            execTransactionFromModuleCall = abi.encodeCall(
                ModuleManager.execTransactionFromModule,
                (
                    _safe,
                    0,
                    abi.encodeCall(OwnerManager.addOwnerWithThreshold, (newOwner, newThreshold)),
                    Enum.Operation.Call
                )
            );
            vm.mockCall(_safe, execTransactionFromModuleCall, abi.encode(true));
        }

        vm.expectCall(_safe, execTransactionFromModuleCall);
        vm.prank(_opFoundation);
        _sut.addOwner(newOwner);
    }
}
