// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title IChainConfig
/// @notice Interface for the ChainConfig predeploy.
interface IChainConfig is ISemver {
    /// @notice Storage slot of the projection chain ID, written in genesis.
    function PROJECTION_CHAIN_ID_SLOT() external view returns (bytes32);

    /// @notice Returns the projection chain ID of this chain.
    /// @return id_ 0 on a public chain, the chain's own ID on a projection chain, and the ID of its
    ///         projection chain on a private chain.
    function projectionChainId() external view returns (uint256 id_);

    /// @notice Constructor placeholder for the interface.
    function __constructor__() external;
}
