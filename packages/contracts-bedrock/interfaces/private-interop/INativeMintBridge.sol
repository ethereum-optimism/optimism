// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title INativeMintBridge
/// @notice Interface for the NativeMintBridge contract.
interface INativeMintBridge is ISemver {
    error NativeMintBridge_Unauthorized();
    error NativeMintBridge_InvalidCrossDomainSender();
    error NativeMintBridge_InvalidCrossDomainSource();
    error NativeMintBridge_ZeroAddress();
    error NativeMintBridge_ZeroAmount();

    event MintRelayed(address indexed to, uint256 amount);
    event BurnAndUnlockSent(address indexed from, address indexed to, uint256 amount, bytes32 msgHash);

    function counterpartyChainId() external view returns (uint256);
    function lockVault() external view returns (address);

    function relayMint(address _to, uint256 _amount) external;
    function burnAndUnlock(address _to) external payable returns (bytes32 msgHash_);

    function __constructor__(uint256 _counterpartyChainId, address _lockVault) external;
}
