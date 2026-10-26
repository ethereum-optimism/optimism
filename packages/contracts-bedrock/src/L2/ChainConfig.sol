// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Libraries
import { Storage } from "src/libraries/Storage.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title ChainConfig
/// @notice Predeploy that tells contracts which OP Private role this chain has, through its
///         projection chain ID:
///         - 0 on a public chain;
///         - the chain's own chain ID on a projection chain;
///         - the chain ID of its projection chain on a private chain.
///         Genesis writes the value into a fixed storage slot. There is no setter.
/// @custom:proxied true
/// @custom:predeploy 0x4200000000000000000000000000000000000034
contract ChainConfig is ISemver {
    /// @notice Storage slot of the projection chain ID, written in genesis:
    ///         bytes32(uint256(keccak256("chainconfig.projectionchainid")) - 1).
    bytes32 public constant PROJECTION_CHAIN_ID_SLOT =
        0xefaa9774600f1eb033432032cd3ecb9a3805ec370f04f3c89ef5227f0299d2a4;

    /// @notice The semantic version of the ChainConfig contract.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Returns the projection chain ID of this chain.
    /// @return id_ 0 on a public chain, the chain's own ID on a projection chain, and the ID of its
    ///         projection chain on a private chain.
    function projectionChainId() external view returns (uint256 id_) {
        id_ = Storage.getUint(PROJECTION_CHAIN_ID_SLOT);
    }
}
