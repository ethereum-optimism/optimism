// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity ^0.8.15;

interface IUniswapV2FactoryMin {
    function getPair(address tokenA, address tokenB) external view returns (address pair);
}
