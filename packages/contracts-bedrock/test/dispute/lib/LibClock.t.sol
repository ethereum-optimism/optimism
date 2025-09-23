// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { Test } from "forge-std/Test.sol";
import { LibClock } from "src/dispute/lib/LibUDT.sol";
import "src/dispute/lib/Types.sol";

/// @title LibClock_Wrap_Test
/// @notice Tests the `wrap` function of the `LibClock` library.
contract LibClock_Wrap_Test is Test {
    /// @notice Tests that the `wrap` function correctly shifts out the `Duration` from a packed
    ///         `Clock` type.
    function testFuzz_wrap_duration_succeeds(Duration _duration, Timestamp _timestamp) public pure {
        Clock clock = LibClock.wrap(_duration, _timestamp);
        assertEq(Duration.unwrap(clock.duration()), Duration.unwrap(_duration));
    }

    /// @notice Tests that the `wrap` function correctly shifts out the `Timestamp` from a packed
    ///         `Clock` type.
    function testFuzz_wrap_timestamp_succeeds(Duration _duration, Timestamp _timestamp) public pure {
        Clock clock = LibClock.wrap(_duration, _timestamp);
        assertEq(Timestamp.unwrap(clock.timestamp()), Timestamp.unwrap(_timestamp));
    }
}
