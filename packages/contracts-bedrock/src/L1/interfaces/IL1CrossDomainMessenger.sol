// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ISemver } from "src/universal/interfaces/ISemver.sol";
import { OptimismPortal } from "src/L1/OptimismPortal.sol";

/// @title IL1CrossDomainMessenger
/// @notice Interface for the L1CrossDomainMessenger contract.
interface IL1CrossDomainMessenger is ISemver {
    function PORTAL() external view returns (OptimismPortal);
}
