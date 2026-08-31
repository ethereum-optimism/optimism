// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title IETHLockVault
/// @notice Interface for the ETHLockVault contract.
interface IETHLockVault is ISemver {
    error ETHLockVault_Unauthorized();
    error ETHLockVault_InvalidCrossDomainSender();
    error ETHLockVault_InvalidCrossDomainSource();
    error ETHLockVault_ZeroAddress();
    error ETHLockVault_ZeroAmount();

    event ETHLocked(address indexed from, address indexed recipient, uint256 amount, bytes32 msgHash);
    event ETHUnlocked(address indexed to, uint256 amount);

    function privateChainId() external view returns (uint256);
    function privateBridge() external view returns (address);

    function lock(address _recipient) external payable returns (bytes32 msgHash_);
    function unlock(address _to, uint256 _amount) external;

    function __constructor__(uint256 _privateChainId, address _privateBridge) external;
}
