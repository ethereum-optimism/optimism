// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

/// @title LegacyMessagePasser_PassMessageToL1_Test
/// @notice Tests the `passMessageToL1` function of the `LegacyMessagePasser` contract.
contract LegacyMessagePasser_PassMessageToL1_Test is CommonTest {
    /// @notice Tests that `passMessageToL1` succeeds.
    function test_passMessageToL1_succeeds() external {
        vm.prank(alice);
        legacyMessagePasser.passMessageToL1(hex"ff");
        assert(legacyMessagePasser.sentMessages(keccak256(abi.encodePacked(hex"ff", alice))));
    }
}
