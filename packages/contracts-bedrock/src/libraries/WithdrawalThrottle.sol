// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/// @title WithdrawalThrottle
/// @notice Storage, arithmetic, and packing helpers for percentage-based withdrawal token buckets.
library WithdrawalThrottle {
    /// @notice Two-slot packed token-bucket state.
    /// @custom:field config Capacity, refill period, maximum basis points, and enabled flag.
    /// @custom:field bucket Available capacity, last-updated timestamp, and fractional remainder.
    struct State {
        uint256 config;
        uint256 bucket;
    }

    /// @notice Logical view of a packed token-bucket state.
    struct Snapshot {
        uint128 capacity;
        uint128 available;
        uint64 refillPeriod;
        uint64 lastUpdated;
        uint64 refillRemainder;
        uint16 maxBps;
        bool enabled;
    }

    /// @notice Thrown when a timestamp cannot be represented by the packed bucket.
    error WithdrawalThrottle_TimestampOverflow();

    /// @notice Denominator used for basis point values.
    uint256 internal constant MAX_BPS = 10_000;

    /// @notice Maximum capacity representable by the packed configuration.
    uint256 internal constant MAX_CAPACITY = type(uint128).max;

    /// @notice Bit offsets in the packed configuration word.
    uint256 internal constant CONFIG_REFILL_PERIOD_OFFSET = 128;
    uint256 internal constant CONFIG_MAX_BPS_OFFSET = 192;
    uint256 internal constant CONFIG_ENABLED_OFFSET = 208;

    /// @notice Bit offsets in the packed mutable bucket word.
    uint256 internal constant BUCKET_LAST_UPDATED_OFFSET = 128;
    uint256 internal constant BUCKET_REFILL_REMAINDER_OFFSET = 192;

    /// @notice Computes bucket capacity as a percentage of an asset stock.
    ///         Values above uint128 are conservatively saturated so an oversized stock cannot
    ///         weaken the throttle or make withdrawals unavailable due to an arithmetic revert.
    /// @param _stock  Current stock of the asset.
    /// @param _maxBps Maximum withdrawable stock in basis points.
    /// @return capacity_ Packed bucket capacity.
    function capacity(uint256 _stock, uint16 _maxBps) internal pure returns (uint128 capacity_) {
        if (_maxBps == 0) return 0;

        uint256 wholeStock = _stock / MAX_BPS;
        if (wholeStock > MAX_CAPACITY / _maxBps) return type(uint128).max;

        uint256 value = wholeStock * _maxBps + ((_stock % MAX_BPS) * _maxBps) / MAX_BPS;
        if (value > MAX_CAPACITY) return type(uint128).max;
        return uint128(value);
    }

    /// @notice Packs the immutable-per-withdrawal configuration fields into one word.
    function config(
        uint128 _capacity,
        uint64 _refillPeriod,
        uint16 _maxBps,
        bool _enabled
    )
        internal
        pure
        returns (uint256 config_)
    {
        config_ = uint256(_capacity) | uint256(_refillPeriod) << CONFIG_REFILL_PERIOD_OFFSET
            | uint256(_maxBps) << CONFIG_MAX_BPS_OFFSET;
        if (_enabled) config_ |= uint256(1) << CONFIG_ENABLED_OFFSET;
    }

    /// @notice Returns the capacity encoded in a packed configuration word.
    function capacity(uint256 _config) internal pure returns (uint128 capacity_) {
        return uint128(_config);
    }

    /// @notice Returns the refill period encoded in a packed configuration word.
    function refillPeriod(uint256 _config) internal pure returns (uint64 refillPeriod_) {
        return uint64(_config >> CONFIG_REFILL_PERIOD_OFFSET);
    }

    /// @notice Returns the maximum basis points encoded in a packed configuration word.
    function maxBps(uint256 _config) internal pure returns (uint16 maxBps_) {
        return uint16(_config >> CONFIG_MAX_BPS_OFFSET);
    }

    /// @notice Returns whether a packed configuration word enables throttling.
    function enabled(uint256 _config) internal pure returns (bool enabled_) {
        return (_config >> CONFIG_ENABLED_OFFSET) & 1 != 0;
    }

    /// @notice Packs mutable token-bucket accounting into one word.
    function bucket(
        uint128 _available,
        uint64 _lastUpdated,
        uint64 _refillRemainder
    )
        internal
        pure
        returns (uint256 bucket_)
    {
        bucket_ = uint256(_available) | uint256(_lastUpdated) << BUCKET_LAST_UPDATED_OFFSET
            | uint256(_refillRemainder) << BUCKET_REFILL_REMAINDER_OFFSET;
    }

    /// @notice Returns stored availability from a packed mutable bucket word.
    function storedAvailable(uint256 _bucket) internal pure returns (uint128 available_) {
        return uint128(_bucket);
    }

    /// @notice Returns the last-updated timestamp from a packed mutable bucket word.
    function lastUpdated(uint256 _bucket) internal pure returns (uint64 lastUpdated_) {
        return uint64(_bucket >> BUCKET_LAST_UPDATED_OFFSET);
    }

    /// @notice Returns the fractional refill numerator from a packed mutable bucket word.
    function refillRemainder(uint256 _bucket) internal pure returns (uint64 refillRemainder_) {
        return uint64(_bucket >> BUCKET_REFILL_REMAINDER_OFFSET);
    }

    /// @notice Checks and narrows a timestamp for packed storage.
    function timestamp(uint256 _timestamp) internal pure returns (uint64 timestamp_) {
        if (_timestamp > type(uint64).max) revert WithdrawalThrottle_TimestampOverflow();
        return uint64(_timestamp);
    }

    /// @notice Computes available capacity after a pending exact fractional refill.
    ///         uint128 capacity multiplied by uint64 elapsed time fits safely in uint192, while
    ///         all intermediate arithmetic is performed in uint256.
    /// @param _capacity     Maximum bucket capacity.
    /// @param _bucket       Packed mutable bucket state.
    /// @param _refillPeriod Time for the bucket to refill from empty to full.
    /// @param _timestamp    Timestamp at which to compute availability.
    /// @return available_ Available capacity at `_timestamp`.
    /// @return remainder_ Fractional refill numerator to preserve when materializing the refill.
    function available(
        uint128 _capacity,
        uint256 _bucket,
        uint64 _refillPeriod,
        uint256 _timestamp
    )
        internal
        pure
        returns (uint128 available_, uint64 remainder_)
    {
        uint128 stored = storedAvailable(_bucket);
        if (stored >= _capacity) return (_capacity, 0);

        uint64 updated = lastUpdated(_bucket);
        if (_timestamp <= updated) return (stored, refillRemainder(_bucket));

        uint256 elapsed = _timestamp - updated;
        if (elapsed >= _refillPeriod) return (_capacity, 0);

        uint256 numerator = uint256(_capacity) * elapsed + refillRemainder(_bucket);
        uint256 refilled = numerator / _refillPeriod;
        uint256 missing = uint256(_capacity) - stored;
        if (refilled >= missing) return (_capacity, 0);

        uint256 value = uint256(stored) + refilled;
        assert(value <= type(uint128).max);
        available_ = uint128(value);

        uint256 remainder = numerator % _refillPeriod;
        assert(remainder <= type(uint64).max);
        remainder_ = uint64(remainder);
    }

    /// @notice Synchronizes a refilled bucket to a new live-stock capacity snapshot.
    ///         Absolute accrued availability and its fractional remainder are preserved when the
    ///         capacity grows. Capacity reductions accrue at the lower live rate, clamp
    ///         availability, and clear the remainder when the reduced bucket is full.
    function sync(
        uint128 _oldCapacity,
        uint256 _bucket,
        uint64 _refillPeriod,
        uint128 _newCapacity,
        uint256 _timestamp
    )
        internal
        pure
        returns (uint128 available_, uint64 remainder_)
    {
        uint128 refillCapacity = _oldCapacity < _newCapacity ? _oldCapacity : _newCapacity;
        (available_, remainder_) = available(refillCapacity, _bucket, _refillPeriod, _timestamp);
        if (available_ >= _newCapacity) return (_newCapacity, 0);
    }

    /// @notice Returns a logical snapshot of the stored state before pending refill is materialized.
    function snapshot(State storage _state) internal view returns (Snapshot memory snapshot_) {
        uint256 config_ = _state.config;
        uint256 bucket_ = _state.bucket;
        snapshot_ = Snapshot({
            capacity: capacity(config_),
            available: storedAvailable(bucket_),
            refillPeriod: refillPeriod(config_),
            lastUpdated: lastUpdated(bucket_),
            refillRemainder: refillRemainder(bucket_),
            maxBps: maxBps(config_),
            enabled: enabled(config_)
        });
    }

    /// @notice Configures a throttle against a live stock value.
    ///         Reconfiguration preserves accrued whole units, resets the fractional remainder,
    ///         and clamps availability to the new capacity.
    function configure(
        State storage _state,
        uint256 _stock,
        uint16 _maxBps,
        uint64 _refillPeriod,
        uint256 _timestamp
    )
        internal
        returns (uint128 capacity_, uint128 available_)
    {
        uint256 oldConfig = _state.config;
        if (enabled(oldConfig)) {
            uint128 liveOldCapacity = capacity(_stock, maxBps(oldConfig));
            (available_,) =
                sync(capacity(oldConfig), _state.bucket, refillPeriod(oldConfig), liveOldCapacity, _timestamp);
        } else {
            available_ = type(uint128).max;
        }

        capacity_ = capacity(_stock, _maxBps);
        if (available_ > capacity_) available_ = capacity_;

        uint64 updated = timestamp(_timestamp);
        _state.config = config(capacity_, _refillPeriod, _maxBps, true);
        _state.bucket = bucket(available_, updated, 0);
    }

    /// @notice Synchronizes a configured throttle against a live stock value and materializes refill.
    /// @dev The caller must pass the state's current config word and ensure the throttle is enabled.
    function refresh(
        State storage _state,
        uint256 _config,
        uint256 _stock,
        uint256 _timestamp
    )
        internal
        returns (uint128 capacity_, uint128 available_)
    {
        uint16 maxBps_ = maxBps(_config);
        uint64 refillPeriod_ = refillPeriod(_config);
        capacity_ = capacity(_stock, maxBps_);
        uint64 remainder;
        (available_, remainder) = sync(capacity(_config), _state.bucket, refillPeriod_, capacity_, _timestamp);

        uint64 updated = timestamp(_timestamp);
        if (capacity_ != capacity(_config)) {
            _state.config = config(capacity_, refillPeriod_, maxBps_, true);
        }
        _state.bucket = bucket(available_, updated, remainder);
    }

    /// @notice Returns currently available capacity against a live stock value.
    /// @dev The caller must pass the state's current config word and ensure the throttle is enabled.
    function availableCapacity(
        State storage _state,
        uint256 _config,
        uint256 _stock,
        uint256 _timestamp
    )
        internal
        view
        returns (uint128 available_)
    {
        uint128 liveCapacity = capacity(_stock, maxBps(_config));
        (available_,) = sync(capacity(_config), _state.bucket, refillPeriod(_config), liveCapacity, _timestamp);
    }

    /// @notice Attempts to consume capacity against a live stock value.
    /// @dev The caller must pass the state's current config word and ensure the throttle is enabled
    ///      and the amount is non-zero. State is unchanged when the requested amount exceeds
    ///      available capacity.
    /// @return capacity_        Capacity synchronized to the supplied live stock.
    /// @return available_       Available capacity before consumption.
    /// @return remaining_       Available capacity after consumption when allowed.
    /// @return capacityChanged_ Whether live-stock synchronization changed stored capacity.
    function consume(
        State storage _state,
        uint256 _config,
        uint256 _stock,
        uint256 _amount,
        uint256 _timestamp
    )
        internal
        returns (uint128 capacity_, uint128 available_, uint128 remaining_, bool capacityChanged_)
    {
        capacity_ = capacity(_stock, maxBps(_config));
        uint64 remainder;
        (available_, remainder) = sync(capacity(_config), _state.bucket, refillPeriod(_config), capacity_, _timestamp);
        if (_amount > available_) return (capacity_, available_, 0, false);

        uint256 remainingValue = uint256(available_) - _amount;
        assert(remainingValue <= type(uint128).max);
        remaining_ = uint128(remainingValue);
        capacityChanged_ = capacity_ != capacity(_config);

        uint64 updated = timestamp(_timestamp);
        if (capacityChanged_) {
            _state.config = config(capacity_, refillPeriod(_config), maxBps(_config), true);
        }
        _state.bucket = bucket(remaining_, updated, remainder);
    }

    /// @notice Disables a throttle and clears both packed slots.
    function disable(State storage _state) internal {
        _state.config = 0;
        _state.bucket = 0;
    }
}
