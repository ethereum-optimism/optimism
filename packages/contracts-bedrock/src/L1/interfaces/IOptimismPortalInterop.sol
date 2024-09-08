// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ConfigType } from "src/L2/L1BlockInterop.sol";

/// @title IOptimismPortalInterop
/// @notice Interface for the OptimismPortalInterop contract.
interface IOptimismPortalInterop {
    /// @notice Thrown when a non-depositor account attempts update static configuration.
    error Unauthorized();

    function setConfig(ConfigType _type, bytes memory _value) external;
}
