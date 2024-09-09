// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @notice Enum representing different types of configurations that can be set on L1BlockInterop.
/// @custom:value SET_GAS_PAYING_TOKEN  Represents the config type for setting the gas paying token.
/// @custom:value ADD_DEPENDENCY        Represents the config type for adding a chain to the interop dependency set.
/// @custom:value REMOVE_DEPENDENCY     Represents the config type for removing a chain from the interop dependency set.
enum ConfigType {
    SET_GAS_PAYING_TOKEN,
    ADD_DEPENDENCY,
    REMOVE_DEPENDENCY
}

/// @title IL1BlockInterop
/// @notice Interface for the L1BlockInterop contract.
interface IL1BlockInterop {
    function dependencySetSize() external view returns (uint8);
    function setConfig(ConfigType _type, bytes memory _value) external;
}
