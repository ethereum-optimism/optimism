// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { SafeSend } from "src/universal/SafeSend.sol";
import { CommonTest } from "test/setup/CommonTest.sol";

/// @title SafeSend_Constructor_Test
/// @notice Tests the `constructor` function of the `SafeSend` contract.
contract SafeSend_Constructor_Test is CommonTest {
    /// @notice Tests that sending to an EOA with various amounts succeeds.
    function testFuzz_constructor_toEOA_succeeds(address _recipient, uint256 _value) public {
        // Ensure recipient is an EOA (no code)
        vm.assume(_recipient != address(0));
        vm.assume(_recipient.code.length == 0);
        // Exclude precompiles (0x01-0x09) and predeploys (0x4200...)
        vm.assume(uint160(_recipient) > 0x09);
        vm.assume(uint160(_recipient) < uint160(0x4200000000000000000000000000000000000000));
        // Bound value to reasonable range
        _value = bound(_value, 0, type(uint128).max);

        // Reset recipient balance to ensure clean test
        vm.deal(_recipient, 0);
        vm.deal(alice, _value);

        uint256 aliceBalanceBefore = alice.balance;
        uint256 recipientBalanceBefore = _recipient.balance;

        vm.prank(alice);
        SafeSend safeSend = new SafeSend{ value: _value }(payable(_recipient));

        assertEq(address(safeSend).code.length, 0);
        assertEq(address(safeSend).balance, 0);
        assertEq(alice.balance, aliceBalanceBefore - _value);
        assertEq(_recipient.balance, recipientBalanceBefore + _value);
    }

    /// @notice Tests that sending to a contract with various amounts succeeds.
    function testFuzz_constructor_toContract_succeeds(address _recipient, uint256 _value) public {
        vm.assume(_recipient != address(0));
        // Exclude precompiles (0x01-0x09) and predeploys (0x4200...)
        vm.assume(uint160(_recipient) > 0x09);
        vm.assume(uint160(_recipient) < uint160(0x4200000000000000000000000000000000000000));
        // Etch reverting code into recipient to make it a contract
        vm.etch(_recipient, hex"fe");
        // Bound value to reasonable range
        _value = bound(_value, 0, type(uint128).max);

        // Reset recipient balance to ensure clean test
        vm.deal(_recipient, 0);
        vm.deal(alice, _value);

        uint256 aliceBalanceBefore = alice.balance;
        uint256 recipientBalanceBefore = _recipient.balance;

        vm.prank(alice);
        SafeSend safeSend = new SafeSend{ value: _value }(payable(_recipient));

        assertEq(address(safeSend).code.length, 0);
        assertEq(address(safeSend).balance, 0);
        assertEq(alice.balance, aliceBalanceBefore - _value);
        assertEq(_recipient.balance, recipientBalanceBefore + _value);
    }

    /// @notice Tests that sending zero value succeeds.
    function test_constructor_zeroValue_succeeds() public {
        vm.deal(alice, 0);

        uint256 aliceBalanceBefore = alice.balance;
        uint256 bobBalanceBefore = bob.balance;

        vm.prank(alice);
        SafeSend safeSend = new SafeSend{ value: 0 }(payable(bob));

        assertEq(address(safeSend).code.length, 0);
        assertEq(address(safeSend).balance, 0);
        assertEq(alice.balance, aliceBalanceBefore);
        assertEq(bob.balance, bobBalanceBefore);
    }

    /// @notice Tests that sending to zero address succeeds.
    function test_constructor_zeroAddress_succeeds() public {
        vm.deal(alice, 100 ether);

        uint256 aliceBalanceBefore = alice.balance;

        vm.prank(alice);
        SafeSend safeSend = new SafeSend{ value: 100 ether }(payable(address(0)));

        assertEq(address(safeSend).code.length, 0);
        assertEq(address(safeSend).balance, 0);
        assertEq(alice.balance, aliceBalanceBefore - 100 ether);
        // Note: ETH sent to address(0) is effectively burned
    }
}
