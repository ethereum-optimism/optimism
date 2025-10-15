// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { SafeSend } from "src/universal/SafeSend.sol";
import { CommonTest } from "test/setup/CommonTest.sol";

/// @title SafeSend_Constructor_Test
/// @notice Tests the `constructor` function of the `SafeSend` contract.
contract SafeSend_Constructor_Test is CommonTest {
    /// @notice Tests that sending ETH to an EOA succeeds with various amounts.
    /// @param _amount Amount of ETH to send.
    function testFuzz_constructor_toEOA_succeeds(uint256 _amount) public {
        _amount = bound(_amount, 0, type(uint96).max);

        assertNotEq(alice, address(0));
        assertNotEq(bob, address(0));
        assertEq(bob.code.length, 0);

        vm.deal(alice, _amount);

        uint256 aliceBalanceBefore = alice.balance;
        uint256 bobBalanceBefore = bob.balance;

        vm.prank(alice);
        SafeSend safeSend = new SafeSend{ value: _amount }(payable(bob));

        assertEq(address(safeSend).code.length, 0);
        assertEq(address(safeSend).balance, 0);
        assertEq(alice.balance, aliceBalanceBefore - _amount);
        assertEq(bob.balance, bobBalanceBefore + _amount);
    }

    /// @notice Tests that sending ETH to a contract succeeds without executing code.
    /// @param _amount Amount of ETH to send.
    function testFuzz_constructor_toContract_succeeds(uint256 _amount) public {
        _amount = bound(_amount, 0, type(uint96).max);

        // Etch reverting code into bob
        vm.etch(bob, hex"fe");
        vm.deal(alice, _amount);

        uint256 aliceBalanceBefore = alice.balance;
        uint256 bobBalanceBefore = bob.balance;

        vm.prank(alice);
        SafeSend safeSend = new SafeSend{ value: _amount }(payable(bob));

        assertEq(address(safeSend).code.length, 0);
        assertEq(address(safeSend).balance, 0);
        assertEq(alice.balance, aliceBalanceBefore - _amount);
        assertEq(bob.balance, bobBalanceBefore + _amount);
    }

    /// @notice Tests that sending to address(0) works correctly.
    /// @param _amount Amount of ETH to send.
    function testFuzz_constructor_toZeroAddress_succeeds(uint256 _amount) public {
        _amount = bound(_amount, 0, type(uint96).max);

        vm.deal(alice, _amount);

        uint256 aliceBalanceBefore = alice.balance;
        uint256 zeroAddressBalanceBefore = address(0).balance;

        vm.prank(alice);
        SafeSend safeSend = new SafeSend{ value: _amount }(payable(address(0)));

        assertEq(address(safeSend).code.length, 0);
        assertEq(address(safeSend).balance, 0);
        assertEq(alice.balance, aliceBalanceBefore - _amount);
        assertEq(address(0).balance, zeroAddressBalanceBefore + _amount);
    }

    /// @notice Tests that sending to any recipient address succeeds.
    /// @param _recipient Random recipient address.
    /// @param _amount Amount of ETH to send.
    function testFuzz_constructor_anyRecipient_succeeds(address _recipient, uint256 _amount) public {
        _amount = bound(_amount, 0, type(uint96).max);

        vm.deal(alice, _amount);

        uint256 aliceBalanceBefore = alice.balance;
        uint256 recipientBalanceBefore = _recipient.balance;

        vm.prank(alice);
        SafeSend safeSend = new SafeSend{ value: _amount }(payable(_recipient));

        assertEq(address(safeSend).code.length, 0);
        assertEq(address(safeSend).balance, 0);
        assertEq(alice.balance, aliceBalanceBefore - _amount);
        assertEq(_recipient.balance, recipientBalanceBefore + _amount);
    }
}
