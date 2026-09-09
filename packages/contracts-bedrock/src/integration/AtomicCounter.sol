// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

/// @notice Stateful destination used by the atomic-call integration tests.
contract AtomicCounter {
    error AtomicCounter_LimitExceeded(uint256 value, uint256 limit);
    /// @custom:semver 0.1.0

    string public constant version = "0.1.0";
    uint256 public value;
    uint256 public limit;

    /// @notice Sets a demo-only bound; zero disables it.
    function setLimit(uint256 _limit) external {
        limit = _limit;
    }

    function add(uint256 _amount) external returns (uint256) {
        value += _amount;
        if (limit != 0 && value > limit) revert AtomicCounter_LimitExceeded(value, limit);
        return value;
    }
}
