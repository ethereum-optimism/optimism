// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { ProxyAdminOwnedBase } from "src/universal/ProxyAdminOwnedBase.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { IL1Block } from "interfaces/L2/IL1Block.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { RangeClaim } from "interfaces/private-interop/IClaimRegistry.sol";

/// @custom:proxied true
/// @title ClaimRegistry
/// @notice On-chain home for the private chain's range claims, on the public rendering. A claim is
///         the operator's commitment for one contiguous range of rendered blocks: the range it
///         covers, the private chain's own block hash and parent hash at the range's last block,
///         the L1 head and config hashes the range was derived under, the content hash of the full
///         private derivation input, continuation commitments, and a proof slot.
///
///         The two terminal hashes are what let the public supernode serve the private chain's
///         complete follow references with no private access at all. Everything else in such a
///         reference is already derivable from public data, because the batcher copies each private
///         block's own L1 origin into its rendering and origins are therefore equal by
///         construction. The parent hash was the single remaining piece that was not derivable, so
///         it is published here rather than reconstructed.
///
///         The claim is the LEADING transaction of the range it describes: the first transaction
///         of the range's first block, committing to the range it is about to open rather than to
///         the one just closed. That is possible because the private range completes before the
///         rendering of it does, so every field — both private terminal hashes included — is known
///         at build time. The genesis range opens with its own claim like any other; the only
///         special case is the first post, which has no predecessor to sit after.
///
///         The registry checks exactly what it can check cheaply and locally: the current batcher
///         is the caller, the claim version,
///         that each range starts strictly after the last posted range ended,
///         and that the proof slot is bounded. It does NOT check that the range's contents match the
///         private chain: the current dummy verifier trusts the operator for that correspondence. Claim N's
///         stored hash folds into claim N+1's, so the posted sequence is a hash chain an auditor
///         can walk from `lastClaimHash`.
///
///         RANGES MAY LEAVE FORWARD GAPS. The rule is `firstBlock > lastPostedLastBlock`, not
///         `== lastPostedLastBlock + 1`: no overlap and no regression, but a jump forward is fine.
///         A range whose opening block is invalidated and replaced — stock interop invalidation
///         today, the proof gate in proven mode later — never executes its claim transaction at
///         all, so the registry cannot advance for a range that was voided. Under a strict
///         contiguity rule that voided range would permanently wedge the next honest claim, since
///         nothing could ever satisfy `+ 1` again. A gap in the record is therefore the
///         self-documenting mark of a voided range, not an error.
///
///         THIS REGISTRY EMITS NO LOGS, deliberately, against the usual rule that a state-changing
///         function emits an event. The claim is the first transaction of a range-opening block, so
///         a log here would sit ahead of every message the block renders and shift each one's log
///         index — and on this chain a message's log position IS its identity, so a rendering-only
///         log would silently break the canonical-position rule the whole design rests on. The
///         durable record is the claim transaction's own calldata: readers scan transactions sent
///         to this address and decode the argument. The hash chain and the range cursor are
///         readable from the getters. In the derivation-gated projection profile, admitted claim
///         calldata remains authoritative even when the registry call reverts. Storage accounting
///         is not a second consensus admission gate. Per-block output records are calldata-only.
///
///         Proof policy is enforced by projection derivation before the span reaches execution.
///         The initial verifier accepts dummy bytes and provides NO private execution proof.
///         This contract bounds the slot and records it without interpreting the proof.
contract ClaimRegistry is ProxyAdminOwnedBase, ISemver {
    /// @notice Thrown when someone other than the current batcher tries to post a claim.
    error ClaimRegistry_NotBatcher();

    /// @notice Thrown when the claim version is not the version this registry accepts.
    error ClaimRegistry_UnsupportedClaimVersion();

    /// @notice Thrown when proof bytes exceed the protocol bound.
    error ClaimRegistry_ProofTooLarge();

    /// @notice Thrown when the claim's range is empty or inverted.
    error ClaimRegistry_InvalidRange();

    /// @notice Thrown when the claim's range does not begin strictly after the last posted range
    ///         ended. Overlaps, regressions and duplicate posts all land here; a forward gap does
    ///         not, and is accepted.
    error ClaimRegistry_OverlappingRange();

    /// @notice Claim version this registry accepts.
    uint8 public constant CLAIM_VERSION = 2;

    /// @notice Upper bound on the proof slot, independent of the derivation verifier mode.
    uint256 public constant MAX_PROOF_LENGTH = 65_536;

    /// @notice Semantic version.
    /// @custom:semver 3.0.0
    string public constant version = "3.0.0";

    /// @notice Number of claims posted so far. Zero means no range has been posted, which is the
    ///         only state in which an arbitrary `firstBlock` is accepted.
    uint64 public rangeCount;

    /// @notice Last block of the most recently posted range.
    uint64 public lastPostedLastBlock;

    /// @notice Running hash of the posted claim sequence. Zero before the first post.
    bytes32 public lastClaimHash;

    /// @notice Logless per-block private output record. The canonical calldata is
    ///         consensus data authenticated by projection derivation. This method
    ///         deliberately stores nothing and performs no execution verification.
    ///         Deposit calls cannot create checkpoints; readers exclude deposits.
    function recordOutput(bytes32) external pure { }

    /// @notice Posts the claim for the range this transaction opens. Reverts unless the range
    ///         begins strictly after the last posted range ended, so posted ranges never overlap,
    ///         never run backwards and can never be posted twice — but may skip forward, which is
    ///         how a voided range leaves its mark instead of wedging the registry. See the
    ///         contract-level notice. The first-ever post has nothing to sit after and sets the
    ///         starting point.
    ///
    ///         Emits nothing: see the contract-level notice. The transaction's calldata is the
    ///         record, and the resulting chain state is readable from `lastClaimHash`,
    ///         `lastPostedLastBlock` and `rangeCount`.
    ///
    ///         Neither `privateTerminalBlockHash` nor `privateTerminalParentHash` is checked here.
    ///         They name blocks on the private chain, which this chain cannot see at all, and even
    ///         the rendering's own terminal block does not exist yet when the claim that opens the
    ///         range is posted. Binding a claim to the blocks it claims is verifier policy,
    ///         performed off-chain by anyone reading the rendering, and is the check a future proof
    ///         would subsume.
    ///
    /// @param _claim Claim describing the range this transaction opens.
    function postClaim(RangeClaim calldata _claim) external {
        // Deposits must not advance the range cursor on behalf of an unrelated account: doing so
        // could block every later honest claim. The standard L1 attributes track batcher rotation.
        if (bytes32(uint256(uint160(msg.sender))) != IL1Block(Predeploys.L1_BLOCK_ATTRIBUTES).batcherHash()) {
            revert ClaimRegistry_NotBatcher();
        }
        if (_claim.version != CLAIM_VERSION) revert ClaimRegistry_UnsupportedClaimVersion();

        // Derivation owns proof policy. This registry enforces only the size bound.
        if (_claim.proof.length > MAX_PROOF_LENGTH) revert ClaimRegistry_ProofTooLarge();

        if (_claim.lastBlock < _claim.firstBlock) revert ClaimRegistry_InvalidRange();

        uint64 index = rangeCount;
        if (index != 0 && _claim.firstBlock <= lastPostedLastBlock) {
            revert ClaimRegistry_OverlappingRange();
        }

        rangeCount = index + 1;
        lastPostedLastBlock = _claim.lastBlock;
        lastClaimHash = keccak256(abi.encode(lastClaimHash, abi.encode(_claim)));
    }
}
