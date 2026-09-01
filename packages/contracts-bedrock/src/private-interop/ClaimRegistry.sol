// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { ProxyAdminOwnedBase } from "src/universal/ProxyAdminOwnedBase.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { RangeClaim } from "interfaces/private-interop/IClaimRegistry.sol";

/// @custom:proxied true
/// @title ClaimRegistry
/// @notice On-chain home for the private chain's range claims, on the public rendering. A claim is
///         the operator's commitment for one contiguous range of rendered blocks: the range it
///         covers, the private chain's own block hash and parent hash at the range's last block,
///         the L1 head and config hashes the range was derived under, the content hash of the full
///         private derivation input held in the operator's object store, and a proof slot.
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
///         The registry checks exactly what it can check cheaply and locally: the claim version,
///         that each range starts strictly after the last posted range ended,
///         and that the proof slot is empty. It does NOT check that the range's contents match the
///         private chain — in v1 that is the operator's attestation, unproven by design. Claim N's
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
///         readable from the getters.
///
///         v1 REFUSES A NON-EMPTY PROOF SLOT. A proof-accepting registry is a future upgrade with
///         a verifier behind it; until then, accepting proof bytes here would let an operator
///         publish something that merely looks proven. The empty-slot rule is what makes "this
///         range is attested, not proven" unambiguous on-chain.
contract ClaimRegistry is ProxyAdminOwnedBase, ISemver {
    /// @notice Thrown when the claim version is not the version this registry accepts.
    error ClaimRegistry_UnsupportedClaimVersion();

    /// @notice Thrown when the claim carries proof bytes. v1 is attested, never proven.
    error ClaimRegistry_ProofNotSupported();

    /// @notice Thrown when the claim's range is empty or inverted.
    error ClaimRegistry_InvalidRange();

    /// @notice Thrown when the claim's range does not begin strictly after the last posted range
    ///         ended. Overlaps, regressions and duplicate posts all land here; a forward gap does
    ///         not, and is accepted.
    error ClaimRegistry_OverlappingRange();

    /// @notice Claim version this registry accepts.
    uint8 public constant CLAIM_VERSION = 1;

    /// @notice Upper bound a future proof-accepting registry will place on the proof slot. Named
    ///         here so the upgrade inherits a bound that was decided before it was needed; v1
    ///         itself only ever accepts a zero-length proof, so nothing enforces it yet.
    uint256 public constant MAX_PROOF_LENGTH = 65_536;

    /// @notice Semantic version.
    /// @custom:semver 2.0.0
    string public constant version = "2.0.0";

    /// @notice Number of claims posted so far. Zero means no range has been posted, which is the
    ///         only state in which an arbitrary `firstBlock` is accepted.
    uint64 public rangeCount;

    /// @notice Last block of the most recently posted range.
    uint64 public lastPostedLastBlock;

    /// @notice Running hash of the posted claim sequence. Zero before the first post.
    bytes32 public lastClaimHash;

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
        if (_claim.version != CLAIM_VERSION) revert ClaimRegistry_UnsupportedClaimVersion();

        // v1 is attested, never proven: the slot must be empty. A proof-accepting registry is a
        // future upgrade, and it is the one that will enforce `MAX_PROOF_LENGTH`.
        if (_claim.proof.length != 0) revert ClaimRegistry_ProofNotSupported();

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
