// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ICompliance } from "interfaces/universal/ICompliance.sol";

/// @title IL1Compliance
/// @notice Interface for the L1Compliance contract.
interface IL1Compliance is ICompliance {
    function __constructor__() external;
}
