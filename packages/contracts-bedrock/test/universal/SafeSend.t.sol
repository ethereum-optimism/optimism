// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { SafeSend } from "src/universal/SafeSend.sol";
import { CommonTest } from "test/setup/CommonTest.sol";

/// @title SafeSend_Constructor_Test
/// @notice Tests the `constructor` function of the `SafeSend` contract.
contract SafeSend_Constructor_Test is CommonTest {
    /// @notice Tests sending ETH to an EOA with various amounts.
    function testFuzz_constructor_eoaRecipient_succeeds(uint256 _amount) public {
        _amount = bound(_amount, 0, 10000 ether);

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

    /// @notice Tests sending ETH to a contract without triggering code.
    function testFuzz_constructor_contractRecipient_succeeds(uint256 _amount) public {
        _amount = bound(_amount, 0, 10000 ether);

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

    /// @notice Tests sending to address(0) succeeds.
    function testFuzz_constructor_zeroAddress_succeeds(uint256 _amount) public {
        _amount = bound(_amount, 0, 10000 ether);

        vm.deal(alice, _amount);

        uint256 aliceBalanceBefore = alice.balance;
        uint256 zeroBalanceBefore = address(0).balance;

        vm.prank(alice);
        SafeSend safeSend = new SafeSend{ value: _amount }(payable(address(0)));

        assertEq(address(safeSend).code.length, 0);
        assertEq(address(safeSend).balance, 0);
        assertEq(alice.balance, aliceBalanceBefore - _amount);
        assertEq(address(0).balance, zeroBalanceBefore + _amount);
    }

    /// @notice Tests sending to various recipient addresses.
    function testFuzz_constructor_arbitraryRecipient_succeeds(address _recipient, uint256 _amount) public {
        _amount = bound(_amount, 0, 10000 ether);

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
