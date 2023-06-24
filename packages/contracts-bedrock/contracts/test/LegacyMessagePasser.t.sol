// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "./CommonTest.t.sol";

// Testing contract dependencies
import { Predeploys } from "../libraries/Predeploys.sol";

// Target contract
import { LegacyMessagePasser } from "../legacy/LegacyMessagePasser.sol";

contract LegacyMessagePasser_Test is CommonTest {
    LegacyMessagePasser messagePasser;

    /// @dev Sets up the test suite.
    function setUp() public virtual override {
        super.setUp();
        messagePasser = new LegacyMessagePasser();
    }

    /// @dev Tests that `passMessageToL1` succeeds.
    function test_passMessageToL1_succeeds() external {
        vm.prank(alice);
        messagePasser.passMessageToL1(hex"ff");
        assert(messagePasser.sentMessages(keccak256(abi.encodePacked(hex"ff", alice))));
    }
}
