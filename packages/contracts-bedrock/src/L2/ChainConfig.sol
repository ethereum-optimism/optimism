// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Libraries
import { Storage } from "src/libraries/Storage.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title ChainConfig
/// @notice Predeploy that tells contracts which OP Private role this chain has, through its
///         settlement chain ID:
///         - 0 on a public chain;
///         - the chain's own chain ID on a settlement chain;
///         - the chain ID of its settlement chain on a private chain.
///         Genesis writes the value into a fixed storage slot. There is no setter.
/// @custom:proxied true
/// @custom:predeploy 0x4200000000000000000000000000000000000034
contract ChainConfig is ISemver {
    /// @notice Storage slot of the settlement chain ID, written in genesis:
    ///         bytes32(uint256(keccak256("chainconfig.settlementchainid")) - 1).
    bytes32 public constant SETTLEMENT_CHAIN_ID_SLOT =
        0x519bd363f00ade2be7cab3da84f88ba43e7e22819543246b4a737af4e3553f4f;

    /// @notice The semantic version of the ChainConfig contract.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Returns the settlement chain ID of this chain.
    /// @return id_ 0 on a public chain, the chain's own ID on a settlement chain, and the ID of its
    ///         settlement chain on a private chain.
    function settlementChainId() public view returns (uint256 id_) {
        id_ = Storage.getUint(SETTLEMENT_CHAIN_ID_SLOT);
    }

    /// @notice Returns true if this chain is a settlement chain.
    /// @return isSettlementChain_ True if the settlement chain ID is this chain's own chain ID.
    function isSettlementChain() external view returns (bool isSettlementChain_) {
        isSettlementChain_ = settlementChainId() == block.chainid;
    }

    /// @notice Returns true if this chain is a private chain.
    /// @return isPrivateChain_ True if the settlement chain ID is set and is another chain's ID.
    function isPrivateChain() external view returns (bool isPrivateChain_) {
        uint256 id = settlementChainId();
        isPrivateChain_ = id != 0 && id != block.chainid;
    }
}
