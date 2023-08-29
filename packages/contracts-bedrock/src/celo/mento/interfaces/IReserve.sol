// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity ^0.8.15;

interface IReserve {
    function setTobinTaxStalenessThreshold(uint256) external;

    function addToken(address) external returns (bool);

    function removeToken(address, uint256) external returns (bool);

    function transferGold(address payable, uint256) external returns (bool);

    function transferExchangeGold(address payable, uint256) external returns (bool);

    function getReserveGoldBalance() external view returns (uint256);

    function getUnfrozenReserveGoldBalance() external view returns (uint256);

    function getOrComputeTobinTax() external returns (uint256, uint256);

    function getTokens() external view returns (address[] memory);

    function getReserveRatio() external view returns (uint256);

    function addExchangeSpender(address) external;

    function removeExchangeSpender(address, uint256) external;

    function addSpender(address) external;

    function removeSpender(address) external;
}
