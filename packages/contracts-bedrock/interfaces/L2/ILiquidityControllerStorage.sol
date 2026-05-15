// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface ILiquidityControllerStorage {
    function minters(address) external view returns (bool);
    function gasPayingTokenName() external view returns (string memory);
    function gasPayingTokenSymbol() external view returns (string memory);
}
