// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title IChainConfig
/// @notice Interface for the ChainConfig predeploy.
interface IChainConfig is ISemver {
    /// @notice Storage slot of the settlement chain ID, written in genesis.
    function SETTLEMENT_CHAIN_ID_SLOT() external view returns (bytes32);

    /// @notice Returns the settlement chain ID of this chain.
    /// @return id_ 0 on a public chain, the chain's own ID on a settlement chain, and the ID of its
    ///         settlement chain on a private chain.
    function settlementChainId() external view returns (uint256 id_);

    /// @notice Returns true if this chain is a settlement chain.
    /// @return isSettlementChain_ True if the settlement chain ID is this chain's own chain ID.
    function isSettlementChain() external view returns (bool isSettlementChain_);

    /// @notice Returns true if this chain is a private chain.
    /// @return isPrivateChain_ True if the settlement chain ID is set and is another chain's ID.
    function isPrivateChain() external view returns (bool isPrivateChain_);

    /// @notice Constructor placeholder for the interface.
    function __constructor__() external;
}
