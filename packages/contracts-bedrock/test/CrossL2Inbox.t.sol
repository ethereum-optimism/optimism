// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { Test } from "forge-std/Test.sol";
import { CrossL2Inbox, InboxEntry } from "src/interop/CrossL2Inbox.sol";

/// @notice Tests for `CrossL2Inbox`
contract CrossL2Inbox_Test is Test {
    CrossL2Inbox inbox;
    address alice = address(128);
    address bob = address(256);

    function setUp() public {
        vm.deal(alice, type(uint64).max);
        vm.deal(bob, type(uint64).max);

        vm.label(alice, "alice");
        vm.label(bob, "bob");

        // alice will be the postie
        inbox = new CrossL2Inbox(alice);
    }

    /// @dev Tests that `deliverMail` can have 0 mail.
    function test_deliverMail_empty_succeeds() external {
        assertEq(inbox.superchainPostie(), alice);
        vm.prank(alice);

        InboxEntry[] memory mail = new InboxEntry[](0);
        inbox.deliverMail(mail);
    }

    /// @dev Tests that `deliverMail` successfully updates the inbox.
    function test_deliverMail_single_succeeds() external {
        assertEq(inbox.superchainPostie(), alice);
        vm.prank(alice);

        InboxEntry[] memory mail = new InboxEntry[](1);
        mail[0] = InboxEntry({ chain: bytes32(uint256(10)), output: bytes32(uint256(0xaa)) });
        inbox.deliverMail(mail);

        assertEq(inbox.roots(mail[0].chain, mail[0].output), true);
    }

    /// @dev Tests that `deliverMail` can deliver multiple entries.
    function test_deliverMail_two_succeeds() external {
        assertEq(inbox.superchainPostie(), alice);
        vm.prank(alice);

        InboxEntry[] memory mail = new InboxEntry[](2);
        mail[0] = InboxEntry({ chain: bytes32(uint256(10)), output: bytes32(uint256(0xaa)) });
        mail[1] = InboxEntry({ chain: bytes32(uint256(42)), output: bytes32(uint256(0xbb)) });
        inbox.deliverMail(mail);

        assertEq(inbox.roots(mail[0].chain, mail[0].output), true);
        assertEq(inbox.roots(mail[1].chain, mail[1].output), true);
    }

    /// @dev Tests that `deliverMail` reverts when called by a non-SUPERCHAIN_POSTIE.
    function test_deliverMail_reverts() external {
        assertEq(inbox.superchainPostie(), alice);
        assertTrue(inbox.superchainPostie() != bob);
        vm.expectRevert("CrossL2Inbox: only postie can deliver mail");
        vm.prank(bob);

        InboxEntry[] memory mail = new InboxEntry[](1);
        mail[0] = InboxEntry({ chain: bytes32(uint256(10)), output: bytes32(uint256(0xaa)) });
        inbox.deliverMail(mail);

        assertEq(inbox.roots(mail[0].chain, mail[0].output), false);
    }
}
