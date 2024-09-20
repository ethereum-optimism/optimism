// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { AddressManager } from "src/legacy/AddressManager.sol";

/// @title IResolvedDelegateProxy
/// @notice Interface for the ResolvedDelegateProxy contract.
interface IResolvedDelegateProxy {
    fallback() external payable;

    function __constructor__(AddressManager _addressManager, string memory _implementationName) external;
}
