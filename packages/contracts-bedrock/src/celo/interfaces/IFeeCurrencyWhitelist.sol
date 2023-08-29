// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity ^0.8.15;

interface IFeeCurrencyWhitelist {
    function addToken(address) external;
    function getWhitelist() external view returns (address[] memory);
}
