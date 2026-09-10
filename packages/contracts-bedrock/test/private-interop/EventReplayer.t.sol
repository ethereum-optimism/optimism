// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";
import { Vm } from "forge-std/Vm.sol";

// Contracts
import { EventReplayer } from "src/private-interop/EventReplayer.sol";

// Interfaces
import { IEventReplayer } from "interfaces/private-interop/IEventReplayer.sol";

/// @title EventReplayer_TestInit
/// @notice Reusable test initialization for `EventReplayer` tests.
abstract contract EventReplayer_TestInit is Test {
    /// @notice Event replayer under test.
    EventReplayer internal eventReplayer;

    /// @notice Test setup.
    function setUp() public virtual {
        eventReplayer = new EventReplayer();
    }

    /// @notice Builds a topics array of the requested length with distinguishable values.
    function _topics(uint256 _count) internal pure returns (bytes32[] memory topics_) {
        topics_ = new bytes32[](_count);
        for (uint256 i = 0; i < _count; i++) {
            topics_[i] = keccak256(abi.encode("topic", i));
        }
    }
}

/// @title EventReplayer_Version_Test
contract EventReplayer_Version_Test is EventReplayer_TestInit {
    /// @notice Tests that the version is set.
    function test_version_succeeds() external view {
        assertTrue(bytes(eventReplayer.version()).length > 0);
    }
}

/// @title EventReplayer_ReplayEvent_Test
/// @notice Tests the `replayEvent` function of the `EventReplayer` contract.
contract EventReplayer_ReplayEvent_Test is EventReplayer_TestInit {
    /// @notice Tests that a log with any legal number of topics is emitted verbatim from this
    ///         contract's own address.
    function test_replayEvent_succeeds() external {
        bytes memory data = hex"00112233445566778899aabbccddeeff";

        for (uint256 count = 0; count <= 4; count++) {
            bytes32[] memory topics = _topics(count);

            vm.recordLogs();
            eventReplayer.replayEvent(topics, data);

            Vm.Log[] memory logs = vm.getRecordedLogs();
            assertEq(logs.length, 1);
            assertEq(logs[0].emitter, address(eventReplayer));
            assertEq(logs[0].topics.length, count);
            for (uint256 i = 0; i < count; i++) {
                assertEq(logs[0].topics[i], topics[i]);
            }
            assertEq(logs[0].data, data);
        }
    }

    /// @notice Tests that an empty data section is emitted as an empty data section.
    function test_replayEvent_emptyData_succeeds() external {
        bytes32[] memory topics = _topics(2);

        vm.recordLogs();
        eventReplayer.replayEvent(topics, hex"");

        Vm.Log[] memory logs = vm.getRecordedLogs();
        assertEq(logs.length, 1);
        assertEq(logs[0].topics.length, 2);
        assertEq(logs[0].data.length, 0);
    }

    /// @notice Tests that more than four topics is refused, since the EVM has no `log5`.
    function test_replayEvent_tooManyTopics_reverts() external {
        vm.expectRevert(IEventReplayer.EventReplayer_TooManyTopics.selector);
        eventReplayer.replayEvent(_topics(5), hex"1234");
    }
}
