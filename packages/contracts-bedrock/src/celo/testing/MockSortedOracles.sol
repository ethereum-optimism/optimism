// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { IOracle } from "../interfaces/IOracle.sol";

/**
 * @title A mock SortedOracles for testing.
 */
contract MockSortedOracles is IOracle {
    uint256 public constant DENOMINATOR = 1000000000000000000000000;
    mapping(address => uint256) public numerators;
    mapping(address => uint256) public medianTimestamp;
    mapping(address => uint256) public numRates;
    mapping(address => bool) public expired;

    function setMedianRate(address token, uint256 numerator) external returns (bool) {
        numerators[token] = numerator;
        return true;
    }

    function setMedianTimestamp(address token, uint256 timestamp) external {
        medianTimestamp[token] = timestamp;
    }

    function setMedianTimestampToNow(address token) external {
        // solhint-disable-next-line not-rely-on-time
        medianTimestamp[token] = uint128(block.timestamp);
    }

    function setNumRates(address token, uint256 rate) external {
        numRates[token] = rate; // This change may break something, TODO
    }

    function getExchangeRate(address token) external view returns (uint256 numerator, uint256 denominator) {
        return medianRate(token);
    }

    function medianRate(address token) public view returns (uint256, uint256) {
        if (numerators[token] > 0) {
            return (numerators[token], DENOMINATOR);
        }
        return (0, 0);
    }

    function isOldestReportExpired(address token) public view returns (bool, address) {
        return (expired[token], token);
    }

    function setOldestReportExpired(address token) public {
        expired[token] = true;
    }
}
