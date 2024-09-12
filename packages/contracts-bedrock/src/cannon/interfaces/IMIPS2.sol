// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "src/universal/interfaces/ISemver.sol";

/// @title IMIPS2
/// @notice Interface for the MIPS2 contract.
interface IMIPS2 is ISemver {
    error InvalidExitedValue();
    error InvalidMemoryProof();
    error InvalidSecondMemoryProof();

    function step(bytes memory _stateData, bytes memory _proof, bytes32 _localContext) external returns (bytes32);
}
