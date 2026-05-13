//! Public types of the pure derivation deriver.
//!
//! Phase 3 of the pure-derivation migration: these types form the IO boundary
//! between the caller (kona-node, kona-client) and `Deriver`. The boundary
//! follows two non-negotiable rules:
//!
//! 1. **Filter-aggressive, parse-lazy.** The caller filters L1 data using static rollup-config
//!    fields and decodes whole-byte blobs. The deriver parses every fallible structure (deposit
//!    logs, config update logs, frames, channels, batches, span batches) and emits a [`TraceEntry`]
//!    for each per-item failure. No fallible parsing in `extract_l1_input`.
//!
//! 2. **Raw bytes throughout.** Transactions move as `Vec<Bytes>`, never as decoded envelopes. The
//!    span-batch overlap content check compares bytes directly, with no encode/decode roundtrip.
//!
//! These two rules let the trace be single-sourced: every dropped item shows
//! up in exactly one place that tests can assert against, and the trace's
//! exhaustive-match property holds compiler-wide.

use crate::core as core_mod;
use ::core::ops::RangeInclusive;
use alloc::vec::Vec;
use alloy_consensus::Header;
use alloy_primitives::{Address, Bytes, Log};
use kona_genesis::{SystemConfigUpdateError, SystemConfigUpdateKind};
use kona_protocol::{
    BatchDropReason, BlockInfo, ChannelError, DepositError, FrameParseError, L2BlockInfo,
    OpAttributesWithParent, SpanBatchError,
};

// ---------------------------------------------------------------------------
// Inputs the caller hands to the deriver.
// ---------------------------------------------------------------------------

/// One L1 block of pre-filtered derivation inputs.
///
/// The caller produces this with [`crate::extract_l1_input`]. The deriver
/// owns every fallible decode that can produce a [`TraceEntry`] (frame
/// parsing, deposit log decoding, system config log decoding) — `L1Input`
/// carries raw bytes and unparsed [`Log`]s by design.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct L1Input {
    /// The L1 block header for this input.
    pub header: Header,
    /// Calldata or decoded blob bytes for every transaction whose `to` field
    /// equals the batch inbox address, paired with the transaction's `from`
    /// (sender) so the deriver can apply the rolling `batcher_addr` filter.
    pub batch_inbox_data: Vec<(Address, Bytes)>,
    /// Deposit event logs, pre-filtered by `address == deposit_contract` and
    /// `topic[0] == DEPOSIT_EVENT_ABI_HASH`. The deriver decodes each one
    /// internally and emits [`TraceEntry::DepositLogDropped`] on failure.
    pub deposit_logs: Vec<Log>,
    /// System config update logs, pre-filtered by
    /// `address == l1_system_config_address` and
    /// `topic[0] == CONFIG_UPDATE_TOPIC`. The deriver decodes each one
    /// internally and emits [`TraceEntry::SystemConfigUpdateDropped`] on
    /// failure.
    pub config_logs: Vec<Log>,
}

/// Caller's response to [`Derivation::NeedSpanBatchOverlap`].
///
/// Carries the L2 blocks that span batch's prefix is claimed to overlap, so
/// the deriver can run the full byte-wise overlap content check. The
/// `parent` is `L2BlockInfo` (not just a hash) because the prefix check
/// reads `parent.l1_origin.number` for two early-exit sanity checks. See
/// `kona_protocol::SpanBatch::check_batch_prefix` lines 705 and 742.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SpanBatchOverlap {
    /// Parent of the span batch's first block, as identified by the deriver
    /// in the [`Derivation::NeedSpanBatchOverlap`] request.
    pub parent: L2BlockInfo,
    /// The L2 blocks the span batch overlaps — `parent.number + 1
    /// ..= safe_head.number`. Order matches the `content` range.
    pub blocks: Vec<SpanBatchOverlapBlock>,
}

/// One overlap-target L2 block in [`SpanBatchOverlap`].
///
/// Carries raw transaction RLP bytes so the byte-wise overlap content check
/// does not need an encode/decode roundtrip. The L1 origin number is decoded
/// on demand from the first deposit transaction (the synthetic L1 info
/// deposit per the `L1InfoTx` scheme).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SpanBatchOverlapBlock {
    /// The L2 block number.
    pub number: u64,
    /// All transactions in the block, in execution order, as raw RLP bytes.
    /// The first transaction must be the L1 info deposit.
    pub txs: Vec<Bytes>,
}

// ---------------------------------------------------------------------------
// What the deriver returns from `derive`.
// ---------------------------------------------------------------------------

/// What the deriver wants next, after a `derive(safe_head)` call.
///
/// Exhaustively matched everywhere inside `pure/`. Adding a variant forces
/// every internal match site to be updated.
//
// The `Attributes` variant is large (`OpAttributesWithParent` holds an
// `OpPayloadAttributes` which holds a `Vec<Bytes>`). Boxing it would add
// an allocation on every emit, which is the common path; keeping the
// enum tag-large is the right trade for derivation throughput.
#[derive(Debug, Clone, PartialEq, Eq)]
#[allow(clippy::large_enum_variant)]
pub enum Derivation {
    /// The deriver produced payload attributes for the next L2 block.
    Attributes {
        /// The attributes, with parent/derived-from set.
        attrs: OpAttributesWithParent,
    },
    /// The deriver is blocked waiting for the next contiguous L1 block.
    /// Caller fetches it, runs [`crate::extract_l1_input`], and feeds the
    /// result back via [`crate::Deriver::add_l1_input`].
    NeedL1Input,
    /// The deriver encountered an overlapping span batch and needs the L2
    /// overlap content to run the full byte-wise check. Caller fetches
    /// `content`'s blocks and feeds them back via
    /// [`crate::Deriver::add_span_batch_overlap`].
    NeedSpanBatchOverlap {
        /// L2 parent block of the span batch's first overlap block.
        parent: L2BlockInfo,
        /// Inclusive range of L2 block numbers to fetch. Empty range encoded
        /// as `start = end + 1` is invalid; the deriver only emits non-empty
        /// ranges.
        content: RangeInclusive<u64>,
    },
    /// The deriver is idle — no batches to validate against the safe head,
    /// and no L1 input request. Caller should advance the safe head (e.g.
    /// after the engine commits an attribute) and call `derive` again.
    Idle,
}

// ---------------------------------------------------------------------------
// DeriveTrace: the structured event stream.
// ---------------------------------------------------------------------------

/// Lifecycle event from a single [`crate::Deriver::derive`] call.
///
/// Strict typed reason enums everywhere — no `String`, no `&'static str`.
/// Adding a variant forces every internal match in `pure/` to update.
///
/// Not `Clone` because `DepositError` (a kona-protocol type) is not `Clone`.
/// Bumping that would require an `alloy` dep change; not in scope for
/// phase 3.
#[derive(Debug, PartialEq, Eq)]
pub enum TraceEntry {
    // ---- Frame layer ----
    /// A frame was successfully parsed out of an L1 batch-inbox tx.
    FrameParsed {
        /// L1 origin the frame was inclusion in.
        origin: BlockInfo,
        /// Channel id this frame belongs to.
        channel_id: [u8; 16],
        /// Frame's number within its channel.
        frame_number: u16,
        /// Whether this is the last frame in its channel.
        is_last: bool,
    },
    /// A frame was dropped.
    FrameDropped {
        /// L1 origin the frame was included in.
        origin: BlockInfo,
        /// Reason for the drop.
        reason: FrameDropReason,
    },
    /// A batch-inbox tx failed to parse into any frames.
    FramesParseFailed {
        /// L1 origin the tx was included in.
        origin: BlockInfo,
        /// The batch-inbox tx's sender.
        from: Address,
        /// Parsing error.
        reason: FrameParseError,
    },
    /// A batch-inbox tx was discarded because its sender did not match the
    /// rolling batcher address. The dynamic batcher filter lives in the
    /// deriver because the rolling sysconfig lives in the deriver — see
    /// design rule 8 in the brainstorm's Key Decisions table.
    BatchInboxTxIgnoredFromMismatch {
        /// L1 origin the tx was included in.
        origin: BlockInfo,
        /// The tx's sender that did not match.
        from: Address,
    },

    // ---- Channel layer ----
    /// A new channel was opened.
    ChannelOpened {
        /// L1 origin in which the opening frame appeared.
        origin: BlockInfo,
        /// Channel id.
        channel_id: [u8; 16],
    },
    /// A channel completed its frame sequence and is ready for batch decode.
    ChannelReady {
        /// L1 origin in which the closing frame appeared.
        origin: BlockInfo,
        /// Channel id.
        channel_id: [u8; 16],
    },
    /// A channel timed out.
    ChannelTimedOut {
        /// L1 origin at which the timeout was detected.
        origin: BlockInfo,
        /// Channel id.
        channel_id: [u8; 16],
        /// L1 block number at which the channel was opened.
        open_block: u64,
    },
    /// A channel's compressed bytes could not be decompressed.
    ChannelDecompressionFailed {
        /// L1 origin in which the channel closed.
        origin: BlockInfo,
        /// Channel id.
        channel_id: [u8; 16],
    },
    /// A batch decoded out of a channel's decompressed bytes was malformed.
    ChannelBatchDecodeFailed {
        /// L1 origin in which the channel closed.
        origin: BlockInfo,
        /// Channel id.
        channel_id: [u8; 16],
    },

    // ---- Batch layer ----
    /// A batch was decoded out of a channel.
    BatchDecoded {
        /// L1 origin the batch was decoded from.
        origin: BlockInfo,
        /// The batch's "kind" — single vs span.
        kind: BatchKind,
    },
    /// A batch's verdict was determined.
    BatchVerdict {
        /// L1 origin the batch was decoded from.
        origin: BlockInfo,
        /// Verdict result.
        verdict: BatchVerdict,
    },
    /// An empty batch was synthesized because the sequencing window expired.
    EmptyBatchGenerated {
        /// L2 epoch number for which the empty batch was generated.
        epoch_num: u64,
        /// Reason an empty batch was synthesized.
        reason: EmptyBatchReason,
    },
    /// A span batch's per-block extraction (`get_singular_batches`) failed.
    /// This drops the span batch and flushes the channel.
    SpanBatchExtractionFailed {
        /// L1 origin the span batch was decoded from.
        origin: BlockInfo,
        /// Underlying error.
        reason: SpanBatchError,
    },

    // ---- Attributes layer ----
    /// Payload attributes were successfully built.
    AttributesBuilt {
        /// L2 block number of the built attributes.
        l2_number: u64,
        /// L1 origin the batch was derived from.
        l1_origin: BlockInfo,
        /// Number of user transactions in the batch (excluding `L1Info` + deposits + upgrades).
        user_tx_count: usize,
    },
    /// Payload attributes building hit the time invariant: next L2 time is
    /// earlier than the L1 origin's timestamp. Caller must reset.
    AttributesBrokenTimeInvariant {
        /// L1 origin the attributes were attempted against.
        l1_origin: BlockInfo,
    },
    /// Building the L1 info transaction envelope failed.
    AttributesL1InfoTxBuildFailed {
        /// L1 origin the attributes were attempted against.
        l1_origin: BlockInfo,
    },

    // ---- System config / deposits ----
    /// A system config update log was successfully applied.
    SystemConfigUpdated {
        /// L1 origin the update was applied from.
        origin: BlockInfo,
        /// What field of the system config was updated.
        kind: SystemConfigUpdateKind,
    },
    /// A system config update log was malformed and could not be applied.
    SystemConfigUpdateDropped {
        /// L1 origin the malformed log was included in.
        origin: BlockInfo,
        /// Reason for the drop.
        reason: SystemConfigUpdateError,
    },
    /// A deposit log was malformed and could not be decoded into a deposit
    /// transaction. The L1 block is otherwise processed.
    DepositLogDropped {
        /// L1 origin the malformed log was included in.
        origin: BlockInfo,
        /// Reason for the drop.
        reason: DepositError,
    },

    // ---- Deriver lifecycle ----
    /// The deriver was reset to a new safe head.
    Reset {
        /// L2 safe head the deriver is now anchored at.
        safe_head: L2BlockInfo,
    },
    /// A second empty batch was about to be synthesized in a single
    /// `derive` call. The protocol allows at most one (the Accept path
    /// returns from `derive_inner` immediately); a second means the
    /// synthesized batch is being rejected every iteration (e.g. L1/L2
    /// timeline inconsistency) and the outer loop would spin until OOM.
    /// `derive_inner` returns `Idle` so the caller can investigate.
    EmptyBatchDuplicate,
}

/// Reasons a frame can be dropped during channel assembly.
///
/// Strict-typed — never a `String`.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum FrameDropReason {
    /// `add_frame` rejected the frame (out-of-order, mismatched id, etc.).
    AddFrameFailed(ChannelError),
    /// The cumulative channel size exceeded the max RLP bytes for the active
    /// fork.
    ChannelSizeExceeded,
    /// A non-zero frame arrived with no active channel — Holocene's
    /// single-channel rule.
    NoActiveChannel,
}

impl From<core_mod::channel::FrameDropReason> for FrameDropReason {
    fn from(reason: core_mod::channel::FrameDropReason) -> Self {
        match reason {
            core_mod::channel::FrameDropReason::AddFrameFailed(err) => Self::AddFrameFailed(err),
            core_mod::channel::FrameDropReason::ChannelSizeExceeded => Self::ChannelSizeExceeded,
            core_mod::channel::FrameDropReason::NoActiveChannel => Self::NoActiveChannel,
        }
    }
}

/// Whether a [`TraceEntry::BatchDecoded`] entry came from a single or span
/// batch. Exhaustive.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum BatchKind {
    /// `SingleBatch` decoded out of the channel.
    Single,
    /// `SpanBatch` decoded out of the channel.
    Span,
}

/// What the deriver concluded about a batch.
///
/// Mirrors `kona_protocol::BatchValidity` but uses a structured drop reason
/// so test assertions stay on the typed enum, not the protocol's `Display`
/// string. The `Drop` reason carries the underlying [`BatchDropReason`] from
/// `kona-protocol` for full parity with the existing async pipeline.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum BatchVerdict {
    /// Batch accepted.
    Accept,
    /// Batch dropped; channel is flushed.
    Drop(BatchDropReason),
    /// Batch is older than the safe head; ignored without flushing.
    Past,
    /// Batch is in the future relative to the safe head; surfacing it now
    /// would be premature.
    Future,
    /// Cannot decide yet — needs more L1 origins or L2 overlap data.
    Undecided,
}

/// Why an empty batch was synthesized.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum EmptyBatchReason {
    /// The sequencing window expired and the validator chose to synthesize
    /// an empty batch (the standard mechanism for keeping L2 time advancing
    /// when the sequencer is offline).
    SequencingWindowExpired,
}

// ---------------------------------------------------------------------------
// DeriveTrace wrapper + helpers.
// ---------------------------------------------------------------------------

/// Structured trace of one `derive` call.
///
/// Wrapper struct over `Vec<TraceEntry>` so test helpers (`.drops()`,
/// `.batch_verdicts()`, etc.) can grow without breaking the public type.
#[derive(Debug, Default, PartialEq, Eq)]
pub struct DeriveTrace {
    /// All entries from this `derive` call, in occurrence order.
    pub entries: Vec<TraceEntry>,
}

impl DeriveTrace {
    /// Creates an empty trace.
    pub const fn new() -> Self {
        Self { entries: Vec::new() }
    }

    /// Returns the number of entries.
    pub const fn len(&self) -> usize {
        self.entries.len()
    }

    /// Returns whether the trace is empty.
    pub const fn is_empty(&self) -> bool {
        self.entries.is_empty()
    }

    /// Pushes a new entry.
    pub(crate) fn push(&mut self, entry: TraceEntry) {
        self.entries.push(entry);
    }

    /// Returns an iterator over all `BatchVerdict`s recorded in the trace.
    pub fn batch_verdicts(&self) -> impl Iterator<Item = BatchVerdict> + '_ {
        self.entries.iter().filter_map(|e| {
            if let TraceEntry::BatchVerdict { verdict, .. } = e { Some(*verdict) } else { None }
        })
    }

    /// Returns an iterator over all [`TraceEntry::AttributesBuilt`] L2 numbers.
    pub fn attributes_built(&self) -> impl Iterator<Item = u64> + '_ {
        self.entries.iter().filter_map(|e| {
            if let TraceEntry::AttributesBuilt { l2_number, .. } = e {
                Some(*l2_number)
            } else {
                None
            }
        })
    }
}

// ---------------------------------------------------------------------------
// Critical errors — only `add_*` methods return these.
// ---------------------------------------------------------------------------

/// Caller-contract violation. The deriver itself never returns a critical
/// error from `derive`; only `add_l1_input` and `add_span_batch_overlap`
/// surface these.
#[derive(Debug, Clone, PartialEq, Eq, thiserror::Error)]
pub enum CriticalError {
    /// The supplied L1 input is not contiguous with the deriver's current
    /// position. Caller must reset.
    #[error("non-contiguous L1 input: expected #{expected}, got #{got}")]
    NonContiguousL1Input {
        /// Block number the deriver expected.
        expected: u64,
        /// Block number actually supplied.
        got: u64,
    },
    /// The supplied L1 input's `parent_hash` does not match the deriver's
    /// known last L1 hash. Caller must reset.
    #[error("L1 input parent hash mismatch at #{number}")]
    L1InputParentMismatch {
        /// L1 block number where the mismatch occurred.
        number: u64,
    },
    /// `add_span_batch_overlap` was called when no overlap was requested.
    #[error("unsolicited span batch overlap response")]
    UnsolicitedOverlap,
    /// The supplied overlap response does not match the requested parent.
    #[error("overlap parent mismatch")]
    OverlapParentMismatch,
    /// The supplied overlap response covers the wrong block range.
    #[error(
        "overlap range mismatch: expected {expected_start}..={expected_end}, got {got_start}..={got_end}"
    )]
    OverlapRangeMismatch {
        /// Lower bound the deriver expected.
        expected_start: u64,
        /// Upper bound the deriver expected.
        expected_end: u64,
        /// Lower bound actually supplied.
        got_start: u64,
        /// Upper bound actually supplied.
        got_end: u64,
    },
}
