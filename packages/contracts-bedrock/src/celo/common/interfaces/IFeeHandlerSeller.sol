// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity ^0.8.15;

import "../FixidityLib.sol";

interface IFeeHandlerSeller {
    function sell(
        address sellTokenAddress,
        address buyTokenAddress,
        uint256 amount,
        uint256 minAmount
    )
        external
        returns (uint256);
    // in case some funds need to be returned or moved to another contract
    function transfer(address token, uint256 amount, address to) external returns (bool);
}
