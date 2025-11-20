// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

/// @title LegacyMessagePasser_PassMessageToL1_Test
/// @notice Tests the `passMessageToL1` function of the `LegacyMessagePasser` contract.
contract LegacyMessagePasser_PassMessageToL1_Test is CommonTest {
    /// @notice Tests that `passMessageToL1` succeeds with any message and sender.
    /// @param _message Arbitrary message to pass to L1.
    /// @param _sender Address sending the message.
    function testFuzz_passMessageToL1_succeeds(bytes memory _message, address _sender) external {
        vm.prank(_sender);
        legacyMessagePasser.passMessageToL1(_message);
        assertTrue(legacyMessagePasser.sentMessages(keccak256(abi.encodePacked(_message, _sender))));
    }
}

/// @title LegacyMessagePasser_Version_Test
/// @notice Tests the `version` function of the `LegacyMessagePasser` contract.
contract LegacyMessagePasser_Version_Test is CommonTest {
    /// @notice Tests that `version` returns a valid semver string.
    function test_version_succeeds() external view {
        assertEq(legacyMessagePasser.version(), "1.1.2");
    }
}
