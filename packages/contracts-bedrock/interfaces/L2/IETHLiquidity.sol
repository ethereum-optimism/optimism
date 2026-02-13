// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IETHLiquidity {
    error Unauthorized();
    error InvalidAmount();

    event LiquidityBurned(address indexed caller, uint256 value);
    event LiquidityMinted(address indexed caller, uint256 value);
    event LiquidityFunded(address indexed funder, uint256 amount);

    function burn() external payable;
    function mint(uint256 _amount) external;
    function fund() external payable;
    function version() external view returns (string memory);

    function __constructor__() external;
}
