// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

/// @title LegacyMessagePasser_PassMessageToL1_Test
/// @notice Tests the `passMessageToL1` function of the `LegacyMessagePasser` contract.
contract LegacyMessagePasser_PassMessageToL1_Test is CommonTest {
    /// @notice Tests that `passMessageToL1` succeeds with arbitrary input.
    /// @param _message Arbitrary message to pass to L1.
    /// @param _sender Address sending the message.
    function testFuzz_passMessageToL1_arbitraryInput_succeeds(bytes memory _message, address _sender) external {
        // Bound message length to prevent out-of-gas in heavy-fuzz CI
        vm.assume(_message.length <= 4096);

        vm.prank(_sender);
        legacyMessagePasser.passMessageToL1(_message);

        // Verify the message was recorded for this sender
        assertTrue(legacyMessagePasser.sentMessages(keccak256(abi.encodePacked(_message, _sender))));

        // Verify per-sender separation: a different sender's key should remain false
        address otherSender = address(uint160(_sender) ^ 1);
        assertFalse(legacyMessagePasser.sentMessages(keccak256(abi.encodePacked(_message, otherSender))));
    }
}

/// @title LegacyMessagePasser_Version_Test
/// @notice Tests the `version` function of the `LegacyMessagePasser` contract.
contract LegacyMessagePasser_Version_Test is CommonTest {
    /// @notice Tests that `version` returns a valid semver format string.
    function test_version_validFormat_succeeds() external view {
        string memory version = legacyMessagePasser.version();
        // Validate non-empty and contains expected semver structure (x.y.z)
        assertGt(bytes(version).length, 0);
        // Count dots to ensure semver format (should have exactly 2 dots)
        uint256 dotCount = 0;
        for (uint256 i = 0; i < bytes(version).length; i++) {
            if (bytes(version)[i] == ".") {
                dotCount++;
            }
        }
        assertEq(dotCount, 2);
    }
}
