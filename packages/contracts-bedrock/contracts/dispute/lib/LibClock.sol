// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "../../libraries/DisputeTypes.sol";

/**
 * @title LibClock
 * @author clabby <https://github.com/clabby>
 * @notice This library contains helper functions for working with the `Clock` type.
 */
library LibClock {
    /**
     * @notice Packs a `Duration` and `Timestamp` into a `Clock` type.
     * @param _duration The `Duration` to pack into the `Clock` type.
     * @param _timestamp The `Timestamp` to pack into the `Clock` type.
     * @return _clock The `Clock` containing the `_duration` and `_timestamp`.
     */
    function wrap(Duration _duration, Timestamp _timestamp) internal pure returns (Clock _clock) {
        assembly {
            _clock := or(shl(0x80, _duration), _timestamp)
        }
    }

    /**
     * @notice Pull the `Duration` out of a `Clock` type.
     * @param _clock The `Clock` type to pull the `Duration` out of.
     * @return _duration The `Duration` pulled out of `_clock`.
     */
    function duration(Clock _clock) internal pure returns (Duration _duration) {
        // Shift the high-order 128 bits into the low-order 128 bits, leaving only the `duration`.
        assembly {
            _duration := shr(0x80, _clock)
        }
    }

    /**
     * @notice Pull the `Timestamp` out of a `Clock` type.
     * @param _clock The `Clock` type to pull the `Timestamp` out of.
     * @return _timestamp The `Timestamp` pulled out of `_clock`.
     */
    function timestamp(Clock _clock) internal pure returns (Timestamp _timestamp) {
        // Clean the high-order 128 bits by shifting the clock left and then right again, leaving
        // only the `timestamp`.
        assembly {
            _timestamp := shr(0x80, shl(0x80, _clock))
        }
    }
}
