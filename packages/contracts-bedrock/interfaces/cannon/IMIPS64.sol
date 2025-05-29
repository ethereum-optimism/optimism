// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IMIPS } from "interfaces/cannon/IMIPS.sol";

/// @title IMIPS64
/// @notice Interface for the MIPS64 contract.
interface IMIPS64 is IMIPS {
    function stateVersion() external view returns (uint256);
}
