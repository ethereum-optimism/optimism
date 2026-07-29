// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";

/// @title IPauseSource
/// @notice Interface for the contract that resolves the pause state, the guardian and the
///         SuperchainConfig for another contract. The address of the IPauseSource is the pause
///         identifier used against the SuperchainConfig, so an IPauseSource must have the same
///         scope as the contracts that point at it. SystemConfig implements this for contracts
///         that belong to exactly one chain. ETHLockbox implements it for contracts that are
///         shared by every chain authorized on that lockbox.
interface IPauseSource {
    function paused() external view returns (bool);
    function guardian() external view returns (address);
    function superchainConfig() external view returns (ISuperchainConfig);
}
