// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IZkBridgeNativeTokenVault {
    function deposit() external payable;

    function withdraw(uint256 wad) external;
}
