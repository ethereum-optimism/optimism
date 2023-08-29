// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "../stability/interfaces/ISortedOracles.sol";

/**
 * @title A mock SortedOracles for testing.
 */
contract MockSortedOracles is ISortedOracles {
    uint256 public constant DENOMINATOR = 1e24;

    mapping(address => uint256) public numerators;

    function addOracle(address, address) external { }

    function removeOracle(address, address, uint256) external { }

    function report(address, uint256, address, address) external { }

    function removeExpiredReports(address, uint256) external { }

    function isOldestReportExpired(address) external pure returns (bool, address) {
        return (false, address(0x000000000000000000000000000000000000ce10));
    }

    function numRates(address) external pure returns (uint256) {
        return 1;
    }

    function medianRate(address token) external pure returns (uint256, uint256) {
        if (token == address(0x000000000000000000000000000000000000cE16)) {
            return (2 * DENOMINATOR, DENOMINATOR);
        }
        return (0, 0);
    }

    function numTimestamps(address) external pure returns (uint256) {
        return 0;
    }

    function medianTimestamp(address) external pure returns (uint256) {
        return 0;
    }
}
