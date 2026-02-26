// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ICompliance } from "interfaces/universal/ICompliance.sol";

/// @title IL2Compliance
/// @notice Interface for the L2Compliance contract.
interface IL2Compliance is ICompliance {
    function __constructor__() external;
}
