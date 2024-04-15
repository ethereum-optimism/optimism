// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Safe, Enum, OwnerManager, ModuleManager } from "safe-contracts/Safe.sol";

import { OwnerGuard } from "src/Safe/VetoerSet/OwnerGuard.sol";

import "forge-std/Test.sol";

contract TestOwnerGuard is Test {
    address private _safe;
    OwnerGuard private _sut;

    function setUp() public {
        _safe = makeAddr("Safe");

        // Mock the dependencies.
        {
            // Mock `safe.getOwners()` to return a list of addresses of length `3`.
            vm.mockCall(_safe, abi.encodeWithSelector(OwnerManager.getOwners.selector), abi.encode(new address[](3)));
        }

        _sut = new OwnerGuard({ _safe: Safe(payable(_safe)) });
    }

    /// @dev `constructor` should initialize `maxOwnerCount` to the max between `INITIAL_MAX_OWNER_COUNT` and the
    ///      current number of owners of the Safe Account.
    function test_Constructor_SetMaxOwnerCount(uint8 safeOwnerCount) public {
        // Mock the dependencies.
        {
            // Mock `safe.getOwners()` to return a list of addresses of length `safeOwnerCount`.
            vm.mockCall(
                _safe,
                abi.encodeWithSelector(OwnerManager.getOwners.selector),
                abi.encode(new address[](safeOwnerCount))
            );
        }

        OwnerGuard sut = new OwnerGuard({ _safe: Safe(payable(_safe)) });
        uint256 initialMaxOwnerCount = sut.INITIAL_MAX_OWNER_COUNT();

        uint256 maxOwnerCount = sut.maxOwnerCount();
        uint256 expectedMaxOwnerCount = safeOwnerCount > initialMaxOwnerCount ? safeOwnerCount : initialMaxOwnerCount;
        assertEq(maxOwnerCount, expectedMaxOwnerCount);
    }

    /// @dev `checkAfterExecution` should revert with `OwnerCountTooHigh` when `maxOwnerCount` is exceeded.
    function testRevert_CheckAfterExecution_OwnerCountTooHigh(
        bytes32 txHash,
        bool success,
        uint256 newOwnerCount
    )
        public
    {
        // Ensure the inputs are reasonable values.
        {
            newOwnerCount = bound(newOwnerCount, _sut.maxOwnerCount() + 1, 255);
        }

        // Mock the dependencies.
        {
            // Mock `safe.getOwners()` to return a list of addresses of length `newOwnerCount`.
            vm.mockCall(
                _safe, abi.encodeWithSelector(OwnerManager.getOwners.selector), abi.encode(new address[](newOwnerCount))
            );
        }

        vm.expectRevert(
            abi.encodeWithSelector(OwnerGuard.OwnerCountTooHigh.selector, newOwnerCount, _sut.INITIAL_MAX_OWNER_COUNT())
        );
        _sut.checkAfterExecution(txHash, success);
    }

    /// @dev `checkAfterExecution` should revert with `InvalidSafeAccountThreshold` when the new threshold does not
    /// match
    ///      with the registered Safe Account threshold.
    function testRevert_CheckAfterExecution_InvalidSafeAccountThreshold(
        bytes32 txHash,
        bool success,
        uint256 newOwnerCount,
        uint256 safeThreshold
    )
        public
    {
        // Ensure the inputs are reasonable values.
        uint256 newThreshold;
        {
            newOwnerCount = bound(newOwnerCount, 0, _sut.maxOwnerCount());
            safeThreshold = bound(safeThreshold, 0, newOwnerCount);

            newThreshold = (newOwnerCount * 66 + 99) / 100;
            vm.assume(safeThreshold != newThreshold);
        }

        // Mock the dependencies.
        {
            // Mock `safe.getOwners()` to return a list of addresses of length `newOwnerCount`.
            vm.mockCall(
                _safe, abi.encodeWithSelector(OwnerManager.getOwners.selector), abi.encode(new address[](newOwnerCount))
            );

            // Mock `safe.getThreshold()` to return `safeThreshold`.
            vm.mockCall(_safe, abi.encodeWithSelector(OwnerManager.getThreshold.selector), abi.encode(safeThreshold));
        }

        vm.expectRevert(
            abi.encodeWithSelector(OwnerGuard.InvalidSafeAccountThreshold.selector, safeThreshold, newThreshold)
        );
        _sut.checkAfterExecution(txHash, success);
    }

    /// @dev `updateMaxOwnerCount` should revert with `SenderIsNotSafeAccount` when the sender is not the Safe Account.
    function testRevert_UpdateMaxOwnerCount_SenderIsNotSafeAccount(uint8 newMaxOwnerCount, address sender) public {
        // Ensure the inputs are reasonable values.
        {
            vm.assume(sender != _safe);
        }

        vm.expectRevert(abi.encodeWithSelector(OwnerGuard.SenderIsNotSafeAccount.selector, sender));
        vm.prank(sender);
        _sut.updateMaxOwnerCount(newMaxOwnerCount);
    }

    /// @dev `updateMaxOwnerCount` should revert with `MaxOwnerCountTooLow` when the `newMaxOwnerCount` is below the
    /// current
    ///      number of owners of the Safe Account.
    function testRevert_UpdateMaxOwnerCount_MaxOwnerCountTooLow(
        uint8 newMaxOwnerCount,
        uint256 safeOwnerCount
    )
        public
    {
        // Ensure the inputs are reasonable values.
        {
            safeOwnerCount = bound(safeOwnerCount, uint256(newMaxOwnerCount) + 1, 511);
        }

        // Mock the dependencies.
        {
            // Mock `safe.getOwners()` to return a list of addresses of length `safeOwnerCount`.
            vm.mockCall(
                _safe,
                abi.encodeWithSelector(OwnerManager.getOwners.selector),
                abi.encode(new address[](safeOwnerCount))
            );
        }

        vm.expectRevert(
            abi.encodeWithSelector(OwnerGuard.MaxOwnerCountTooLow.selector, newMaxOwnerCount, safeOwnerCount)
        );
        vm.prank(_safe);
        _sut.updateMaxOwnerCount(newMaxOwnerCount);
    }

    /// @dev `updateMaxOwnerCount` should update `maxOwnerCount`.
    function test_UpdateMaxOwnerCount_UpdateMaxOwnerCount(uint8 newMaxOwnerCount, uint256 safeOwnerCount) public {
        // Ensure the inputs are reasonable values.
        {
            safeOwnerCount = bound(safeOwnerCount, 0, newMaxOwnerCount);
        }

        // Mock the dependencies.
        {
            // Mock `safe.getOwners()` to return a list of addresses of length `safeOwnerCount`.
            vm.mockCall(
                _safe,
                abi.encodeWithSelector(OwnerManager.getOwners.selector),
                abi.encode(new address[](safeOwnerCount))
            );
        }

        vm.prank(_safe);
        _sut.updateMaxOwnerCount(newMaxOwnerCount);
        assertEq(_sut.maxOwnerCount(), newMaxOwnerCount);
    }

    /// @dev `checkNewOwnerCount` should revert with `OwnerCountTooHigh` when `maxOwnerCount` is exceeded.
    function testRevert_CheckNewOwnerCount_OwnerCountTooHigh(uint256 newOwnerCount) public {
        // Ensure the inputs are reasonable values.
        {
            newOwnerCount = bound(newOwnerCount, _sut.maxOwnerCount() + 1, 255);
        }

        vm.expectRevert(
            abi.encodeWithSelector(OwnerGuard.OwnerCountTooHigh.selector, newOwnerCount, _sut.INITIAL_MAX_OWNER_COUNT())
        );
        _sut.checkNewOwnerCount(newOwnerCount);
    }

    /// @dev `checkNewOwnerCount` should return the 66% threshold for `newOwnerCount` owners.
    function test_CheckNewOwnerCount_Returns66PercentThreshold(uint256 newOwnerCount) public {
        // Ensure the inputs are reasonable values.
        {
            newOwnerCount = bound(newOwnerCount, 0, _sut.maxOwnerCount());
        }

        uint256 newThresold = _sut.checkNewOwnerCount(newOwnerCount);
        uint256 expectedNewThreshold = (newOwnerCount * 66 + 99) / 100;
        assertEq(newThresold, expectedNewThreshold);
    }
}
