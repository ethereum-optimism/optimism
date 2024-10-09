// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "src/universal/interfaces/ISemver.sol";
import { IPreimageOracle } from "src/cannon/interfaces/IPreimageOracle.sol";

/// @title IMIPS
/// @notice Interface for the MIPS contract.
interface IMIPS is ISemver {
    error InvalidMemoryProof();
    error InvalidRMWInstruction();

    function oracle() external view returns (IPreimageOracle oracle_);
    function step(bytes memory _stateData, bytes memory _proof, bytes32 _localContext) external returns (bytes32);

    function __constructor__(IPreimageOracle _oracle) external;
}
