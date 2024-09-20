// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface ISuperchainWETH {
    error NotCustomGasToken();
    error Unauthorized();

    event Approval(address indexed src, address indexed guy, uint256 wad);
    event Deposit(address indexed dst, uint256 wad);
    event RelayERC20(address indexed from, address indexed to, uint256 amount, uint256 source);
    event SendERC20(address indexed from, address indexed to, uint256 amount, uint256 destination);
    event Transfer(address indexed src, address indexed dst, uint256 wad);
    event Withdrawal(address indexed src, uint256 wad);

    fallback() external payable;

    receive() external payable;

    function allowance(address, address) external view returns (uint256);
    function approve(address guy, uint256 wad) external returns (bool);
    function balanceOf(address) external view returns (uint256);
    function decimals() external view returns (uint8);
    function deposit() external payable;
    function name() external view returns (string memory);
    function relayERC20(address from, address dst, uint256 wad) external;
    function sendERC20(address dst, uint256 wad, uint256 chainId) external;
    function symbol() external view returns (string memory);
    function totalSupply() external view returns (uint256);
    function transfer(address dst, uint256 wad) external returns (bool);
    function transferFrom(address src, address dst, uint256 wad) external returns (bool);
    function version() external view returns (string memory);
    function withdraw(uint256 wad) external;
}
