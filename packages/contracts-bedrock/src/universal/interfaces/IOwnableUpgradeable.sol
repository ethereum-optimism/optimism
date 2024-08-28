// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IInitializable } from "src/universal/interfaces/IInitializable.sol";

/// @title IOwnableUpgradeable
/// @notice Interface for the OwnableUpgradeable contract.
interface IOwnableUpgradeable is IInitializable {
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

    function owner() external view returns (address);
    function renounceOwnership() external;
    function transferOwnership(address newOwner) external; // nosemgrep: sol-style-input-arg-fmt
}
