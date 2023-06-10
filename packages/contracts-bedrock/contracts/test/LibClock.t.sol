// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { Test } from "forge-std/Test.sol";
import { LibClock } from "../dispute/lib/LibClock.sol";
import "../libraries/DisputeTypes.sol";

/**
 * @notice Tests for `LibClock`
 */
contract LibClock_Test is Test {
    /**
     * @notice Tests that the `duration` function correctly shifts out the `Duration` from a packed `Clock` type.
     */
    function testFuzz_duration_succeeds(uint64 _duration, uint64 _timestamp) public {
        Clock clock = LibClock.wrap(Duration.wrap(_duration), Timestamp.wrap(_timestamp));
        assertEq(Duration.unwrap(LibClock.duration(clock)), _duration);
    }

    /**
     * @notice Tests that the `timestamp` function correctly shifts out the `Timestamp` from a packed `Clock` type.
     */
    function testFuzz_timestamp_succeeds(uint64 _duration, uint64 _timestamp) public {
        Clock clock = LibClock.wrap(Duration.wrap(_duration), Timestamp.wrap(_timestamp));
        assertEq(Timestamp.unwrap(LibClock.timestamp(clock)), _timestamp);
    }
}
