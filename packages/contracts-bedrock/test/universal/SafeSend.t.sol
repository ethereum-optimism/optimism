// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { SafeSend } from "src/universal/SafeSend.sol";
import { CommonTest } from "test/setup/CommonTest.sol";

/// @title SafeSend_Constructor_Test
/// @notice Tests the `constructor` function of the `SafeSend` contract.
contract SafeSend_Constructor_Test is CommonTest {
    /// @notice Tests that sending to an EOA succeeds.
    function test_constructor_toEOA_succeeds() public {
        assertNotEq(alice, address(0));
        assertNotEq(bob, address(0));
        assertEq(bob.code.length, 0);

        vm.deal(alice, 100 ether);

        uint256 aliceBalanceBefore = alice.balance;
        uint256 bobBalanceBefore = bob.balance;

        vm.prank(alice);
        SafeSend safeSend = new SafeSend{ value: 100 ether }(payable(bob));

        assertEq(address(safeSend).code.length, 0);
        assertEq(address(safeSend).balance, 0);
        assertEq(alice.balance, aliceBalanceBefore - 100 ether);
        assertEq(bob.balance, bobBalanceBefore + 100 ether);
    }

    /// @notice Tests that sending to a contract succeeds without executing the contract's code
    function test_constructor_toContract_succeeds() public {
        // Etch reverting code into bob
        vm.etch(bob, hex"fe");
        vm.deal(alice, 100 ether);

        uint256 aliceBalanceBefore = alice.balance;
        uint256 bobBalanceBefore = bob.balance;

        vm.prank(alice);
        SafeSend safeSend = new SafeSend{ value: 100 ether }(payable(bob));

        assertEq(address(safeSend).code.length, 0);
        assertEq(address(safeSend).balance, 0);
        assertEq(alice.balance, aliceBalanceBefore - 100 ether);
        assertEq(bob.balance, bobBalanceBefore + 100 ether);
    }
}
