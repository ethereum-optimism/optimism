// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "interfaces/universal/ISemver.sol";
import { IPreimageOracle } from "interfaces/cannon/IPreimageOracle.sol";

/// @title IMIPS64
/// @notice Interface for the MIPS64 contract.
interface IMIPS64 is ISemver {
    struct State {
        bytes32 memRoot;
        bytes32 preimageKey;
        uint32 preimageOffset;
        uint32 pc;
        uint32 nextPC;
        uint32 lo;
        uint32 hi;
        uint32 heap;
        uint8 exitCode;
        bool exited;
        uint64 step;
        uint32[32] registers;
    }

    error InvalidExitedValue();
    error InvalidPC();
    error InvalidSecondMemoryProof();
    error UnsupportedStateVersion();
    error InvalidMemoryProof();
    error InvalidRMWInstruction();

    function version() external view returns (string memory);
    function oracle() external view returns (IPreimageOracle oracle_);
    function stateVersion() external view returns (uint256 stateVersion_);
    function step(bytes memory _stateData, bytes memory _proof, bytes32 _localContext) external returns (bytes32 postState_);

    function __constructor__(IPreimageOracle _oracle, uint256 _stateVersion) external;
}
