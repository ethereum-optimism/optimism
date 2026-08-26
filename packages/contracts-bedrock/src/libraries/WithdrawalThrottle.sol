// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/// @title WithdrawalThrottle
/// @notice Arithmetic helpers for percentage-based withdrawal token buckets.
library WithdrawalThrottle {
    /// @notice Denominator used for basis point values.
    uint256 internal constant MAX_BPS = 10_000;

    /// @notice Computes bucket capacity as a percentage of an asset stock.
    /// @param _stock  Current stock of the asset.
    /// @param _maxBps Maximum withdrawable stock in basis points.
    /// @return Bucket capacity.
    function capacity(uint256 _stock, uint16 _maxBps) internal pure returns (uint256) {
        return (_stock / MAX_BPS) * _maxBps + ((_stock % MAX_BPS) * _maxBps) / MAX_BPS;
    }

    /// @notice Computes available capacity after a pending linear refill.
    /// @param _capacity        Maximum bucket capacity.
    /// @param _available       Stored available capacity.
    /// @param _refillRemainder Stored fractional refill numerator.
    /// @param _refillPeriod    Time for the bucket to refill from empty to full.
    /// @param _lastUpdated     Timestamp of the last materialized refill.
    /// @param _timestamp       Timestamp at which to compute availability.
    /// @return available_ Available capacity at `_timestamp`.
    /// @return remainder_ Fractional refill numerator to preserve when materializing the refill.
    function available(
        uint256 _capacity,
        uint256 _available,
        uint64 _refillRemainder,
        uint64 _refillPeriod,
        uint64 _lastUpdated,
        uint256 _timestamp
    )
        internal
        pure
        returns (uint256 available_, uint64 remainder_)
    {
        if (_available >= _capacity) return (_capacity, 0);

        uint256 elapsed = _timestamp - _lastUpdated;
        if (elapsed >= _refillPeriod) return (_capacity, 0);

        uint256 numerator = (_capacity % _refillPeriod) * elapsed + _refillRemainder;
        uint256 refilled = (_capacity / _refillPeriod) * elapsed + numerator / _refillPeriod;
        uint256 missing = _capacity - _available;
        if (refilled >= missing) return (_capacity, 0);

        return (_available + refilled, uint64(numerator % _refillPeriod));
    }
}
