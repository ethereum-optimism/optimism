// SPDX-License-Identifier: MIT
pragma solidity >=0.5.13 <0.9.0;

/// Possibly not final version
interface IOracle {
    function getExchangeRate(address token) external view returns (uint256 numerator, uint256 denominator);
}
