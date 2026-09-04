// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "interfaces/universal/ISemver.sol";
import { IProxyAdminOwnedBase } from "interfaces/universal/IProxyAdminOwnedBase.sol";

/// @notice Operator commitment covering one contiguous range of public rendering blocks, posted as
///         the leading transaction of the range it describes.
///
///         `privateTerminalParentHash` is here so the public supernode can serve the private
///         chain's complete follow references without any private access. The batcher copies each
///         private block's own L1 origin into its rendering, so origins are equal by construction
///         and the rest of the reference is already derivable from public data; the parent hash was
///         the one remaining piece that was not, so it is published rather than derived.
///
/// @custom:field version                   Claim format version. Must be 1 for this registry.
/// @custom:field firstBlock                First public block covered by the range.
/// @custom:field lastBlock                 Last public block covered by the range.
/// @custom:field privateTerminalBlockHash  The private chain's block hash at `lastBlock`.
/// @custom:field privateTerminalParentHash Parent hash of that private terminal block.
/// @custom:field l1Head                    L1 head the range was derived under.
/// @custom:field rollupConfigHash          Hash of the rollup config the range was derived under.
/// @custom:field depSetHash                Hash of the dependency set the range was derived under.
/// @custom:field privateDataHash           Content hash of the full private derivation input.
/// @custom:field proof                     Proof slot. Must be empty in v1.
struct RangeClaim {
    uint8 version;
    uint64 firstBlock;
    uint64 lastBlock;
    bytes32 privateTerminalBlockHash;
    bytes32 privateTerminalParentHash;
    bytes32 l1Head;
    bytes32 rollupConfigHash;
    bytes32 depSetHash;
    bytes32 privateDataHash;
    bytes proof;
}

/// @title IClaimRegistry
/// @notice Interface for the ClaimRegistry contract.
interface IClaimRegistry is ISemver, IProxyAdminOwnedBase {
    error ClaimRegistry_UnsupportedClaimVersion();
    error ClaimRegistry_ProofNotSupported();
    error ClaimRegistry_InvalidRange();
    error ClaimRegistry_OverlappingRange();

    function CLAIM_VERSION() external view returns (uint8);
    function MAX_PROOF_LENGTH() external view returns (uint256);
    function rangeCount() external view returns (uint64);
    function lastPostedLastBlock() external view returns (uint64);
    function lastClaimHash() external view returns (bytes32);

    function postClaim(RangeClaim calldata _claim) external;

    function __constructor__() external;
}
