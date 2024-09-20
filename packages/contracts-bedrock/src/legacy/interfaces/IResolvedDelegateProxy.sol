// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title IResolvedDelegateProxy
/// @notice Interface for the ResolvedDelegateProxy contract.
interface IResolvedDelegateProxy {
    fallback() external payable;
}
