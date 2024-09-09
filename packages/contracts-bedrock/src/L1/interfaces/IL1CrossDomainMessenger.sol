// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { OptimismPortal } from "src/L1/OptimismPortal.sol";

/// @title IL1CrossDomainMessenger
/// @notice Interface for the L1CrossDomainMessenger contract.
interface IL1CrossDomainMessenger {
    function PORTAL() external view returns (OptimismPortal);
}
