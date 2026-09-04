// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Libraries
import { WithdrawalThrottle } from "src/libraries/WithdrawalThrottle.sol";

/// @title WithdrawalThrottle_Harness
/// @notice Exposes internal `WithdrawalThrottle` functions for testing.
contract WithdrawalThrottle_Harness {
    function capacity(uint256 _stock, uint16 _maxBps) external pure returns (uint128) {
        return WithdrawalThrottle.capacity(_stock, _maxBps);
    }

    function config(
        uint128 _capacity,
        uint64 _refillPeriod,
        uint16 _maxBps,
        bool _enabled
    )
        external
        pure
        returns (uint256)
    {
        return WithdrawalThrottle.config(_capacity, _refillPeriod, _maxBps, _enabled);
    }

    function capacity(uint256 _config) external pure returns (uint128) {
        return WithdrawalThrottle.capacity(_config);
    }

    function refillPeriod(uint256 _config) external pure returns (uint64) {
        return WithdrawalThrottle.refillPeriod(_config);
    }

    function maxBps(uint256 _config) external pure returns (uint16) {
        return WithdrawalThrottle.maxBps(_config);
    }

    function enabled(uint256 _config) external pure returns (bool) {
        return WithdrawalThrottle.enabled(_config);
    }

    function bucket(uint128 _available, uint64 _lastUpdated, uint64 _refillRemainder) external pure returns (uint256) {
        return WithdrawalThrottle.bucket(_available, _lastUpdated, _refillRemainder);
    }

    function storedAvailable(uint256 _bucket) external pure returns (uint128) {
        return WithdrawalThrottle.storedAvailable(_bucket);
    }

    function lastUpdated(uint256 _bucket) external pure returns (uint64) {
        return WithdrawalThrottle.lastUpdated(_bucket);
    }

    function refillRemainder(uint256 _bucket) external pure returns (uint64) {
        return WithdrawalThrottle.refillRemainder(_bucket);
    }

    function timestamp(uint256 _timestamp) external pure returns (uint64) {
        return WithdrawalThrottle.timestamp(_timestamp);
    }

    function available(
        uint128 _capacity,
        uint256 _bucket,
        uint64 _refillPeriod,
        uint256 _timestamp
    )
        external
        pure
        returns (uint128, uint64)
    {
        return WithdrawalThrottle.available(_capacity, _bucket, _refillPeriod, _timestamp);
    }

    function sync(
        uint128 _oldCapacity,
        uint256 _bucket,
        uint64 _refillPeriod,
        uint128 _newCapacity,
        uint256 _timestamp
    )
        external
        pure
        returns (uint128, uint64)
    {
        return WithdrawalThrottle.sync(_oldCapacity, _bucket, _refillPeriod, _newCapacity, _timestamp);
    }
}

/// @title WithdrawalThrottle_TestInit
/// @notice Reusable initialization for `WithdrawalThrottle` tests.
abstract contract WithdrawalThrottle_TestInit is Test {
    WithdrawalThrottle_Harness internal harness;

    function setUp() public {
        harness = new WithdrawalThrottle_Harness();
    }
}

/// @title WithdrawalThrottle_Config_Test
/// @notice Tests packed withdrawal throttle configuration.
contract WithdrawalThrottle_Config_Test is WithdrawalThrottle_TestInit {
    function test_config_packsAndDecodesFields_succeeds() external view {
        uint128 capacity = type(uint128).max;
        uint64 refillPeriod = type(uint64).max;
        uint16 maxBps = 10_000;
        uint256 config = harness.config(capacity, refillPeriod, maxBps, true);
        uint256 expected = uint256(capacity) | uint256(refillPeriod) << 128 | uint256(maxBps) << 192 | uint256(1) << 208;

        assertEq(config, expected);
        assertEq(harness.capacity(config), capacity);
        assertEq(harness.refillPeriod(config), refillPeriod);
        assertEq(harness.maxBps(config), maxBps);
        assertTrue(harness.enabled(config));
        assertFalse(harness.enabled(harness.config(capacity, refillPeriod, maxBps, false)));
    }
}

/// @title WithdrawalThrottle_Bucket_Test
/// @notice Tests packed mutable bucket accounting.
contract WithdrawalThrottle_Bucket_Test is WithdrawalThrottle_TestInit {
    function test_bucket_packsAndDecodesFields_succeeds() external view {
        uint128 available = type(uint128).max;
        uint64 lastUpdated = type(uint64).max;
        uint64 refillRemainder = type(uint64).max;
        uint256 bucket = harness.bucket(available, lastUpdated, refillRemainder);

        assertEq(bucket, type(uint256).max);
        assertEq(harness.storedAvailable(bucket), available);
        assertEq(harness.lastUpdated(bucket), lastUpdated);
        assertEq(harness.refillRemainder(bucket), refillRemainder);
    }
}

/// @title WithdrawalThrottle_Capacity_Test
/// @notice Tests percentage capacity calculations.
contract WithdrawalThrottle_Capacity_Test is WithdrawalThrottle_TestInit {
    function test_capacity_preservesIntegerPrecision_succeeds() external view {
        assertEq(harness.capacity(12_345, 1234), 1523);
    }

    function test_capacity_uint128BoundaryIsExact_succeeds() external view {
        assertEq(harness.capacity(type(uint128).max, 10_000), type(uint128).max);
    }

    function test_capacity_aboveUint128Saturates_succeeds() external view {
        assertEq(harness.capacity(uint256(type(uint128).max) + 1, 10_000), type(uint128).max);
    }

    function test_capacity_hugeStockSaturatesWithoutOverflow_succeeds() external view {
        assertEq(harness.capacity(type(uint256).max, 9999), type(uint128).max);
    }
}

/// @title WithdrawalThrottle_Available_Test
/// @notice Tests exact fractional refills.
contract WithdrawalThrottle_Available_Test is WithdrawalThrottle_TestInit {
    function test_available_preservesExactFractionalRefill_succeeds() external view {
        uint64 refillPeriod = 3;
        uint256 timestamp = 1000;
        uint256 bucket = harness.bucket(0, uint64(timestamp), 0);

        (uint128 available, uint64 remainder) = harness.available(100, bucket, refillPeriod, timestamp + 1);
        assertEq(available, 33);
        assertEq(remainder, 1);

        bucket = harness.bucket(available, uint64(timestamp + 1), remainder);
        (available, remainder) = harness.available(100, bucket, refillPeriod, timestamp + 2);
        assertEq(available, 66);
        assertEq(remainder, 2);

        bucket = harness.bucket(available, uint64(timestamp + 2), remainder);
        (available, remainder) = harness.available(100, bucket, refillPeriod, timestamp + 3);
        assertEq(available, 100);
        assertEq(remainder, 0);
    }
}

/// @title WithdrawalThrottle_Sync_Test
/// @notice Tests capacity snapshot synchronization.
contract WithdrawalThrottle_Sync_Test is WithdrawalThrottle_TestInit {
    function test_sync_capacityGrowthPreservesAccruedAvailability_succeeds() external view {
        uint256 bucket = harness.bucket(0, 1000, 0);

        (uint128 available, uint64 remainder) = harness.sync(100, bucket, 3, 200, 1001);
        assertEq(available, 33);
        assertEq(remainder, 1);
    }

    function test_sync_capacityReductionUsesLowerRefillRate_succeeds() external view {
        uint256 bucket = harness.bucket(0, 1000, 0);

        (uint128 available, uint64 remainder) = harness.sync(100, bucket, 3, 50, 1002);
        assertEq(available, 33);
        assertEq(remainder, 1);
    }

    function test_sync_saturatedCapacityReductionClamps_succeeds() external view {
        uint128 saturatedCapacity = harness.capacity(type(uint256).max, 10_000);
        uint256 bucket = harness.bucket(saturatedCapacity, 1000, type(uint64).max);

        (uint128 available, uint64 remainder) = harness.sync(saturatedCapacity, bucket, 1, 123, 1000);
        assertEq(available, 123);
        assertEq(remainder, 0);
    }
}

/// @title WithdrawalThrottle_Timestamp_Test
/// @notice Tests checked timestamp narrowing.
contract WithdrawalThrottle_Timestamp_Test is WithdrawalThrottle_TestInit {
    function test_timestamp_uint64BoundaryIsExact_succeeds() external view {
        assertEq(harness.timestamp(type(uint64).max), type(uint64).max);
    }

    function test_timestamp_aboveUint64_reverts() external {
        vm.expectRevert(WithdrawalThrottle.WithdrawalThrottle_TimestampOverflow.selector);
        harness.timestamp(uint256(type(uint64).max) + 1);
    }
}
