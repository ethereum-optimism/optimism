// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IOptimismPortal2 } from "src/L1/interfaces/IOptimismPortal2.sol";

enum ConfigType {
    SET_GAS_PAYING_TOKEN,
    ADD_DEPENDENCY,
    REMOVE_DEPENDENCY
}

/// @title IOptimismPortalInterop
/// @notice Interface for the OptimismPortalInterop contract.
interface IOptimismPortalInterop is IOptimismPortal2 {
    function setConfig(ConfigType _type, bytes memory _value) external;
}
