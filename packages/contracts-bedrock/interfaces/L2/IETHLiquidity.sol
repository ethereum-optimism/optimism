// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IProxyAdminOwnedBase } from "interfaces/universal/IProxyAdminOwnedBase.sol";

interface IETHLiquidity is IProxyAdminOwnedBase {
    error Unauthorized();
    error InvalidAmount();
    error ETHLiquidity_Paused();
    error ETHLiquidity_NotL1CrossDomainMessenger();

    event LiquidityBurned(address indexed caller, uint256 value);
    event LiquidityMinted(address indexed caller, uint256 value);
    event LiquidityFunded(address indexed funder, uint256 amount);
    event PausedSet(bool paused, uint256 messageNonce);
    event ProxyAdminOwnerPausedSet(bool paused);

    function burn() external payable;
    function mint(uint256 _amount) external;
    function fund() external payable;
    function setPaused(bool _paused, uint256 _messageNonce) external;
    function setProxyAdminOwnerPaused(bool _paused) external;
    function paused() external view returns (bool);
    function l1Paused() external view returns (bool);
    function pausedMessageNonce() external view returns (uint256);
    function proxyAdminOwnerPaused() external view returns (bool);
    function version() external view returns (string memory);

    function __constructor__() external;
}
