//! Sequence cache management for flashblocks.
//!
//! The `SequenceManager` maintains a ring buffer of recently completed flashblock sequences
//! and intelligently selects which sequence to build based on the local chain tip.

use crate::{
    FlashBlock, FlashBlockCompleteSequence, PendingFlashBlock,
    pending_state::PendingBlockState,
    sequence::{FlashBlockPendingSequence, SequenceExecutionOutcome},
    validation::{
        CanonicalBlockFingerprint, CanonicalBlockReconciler, ReconciliationStrategy, ReorgDetector,
        TrackedBlockFingerprint,
    },
    worker::BuildArgs,
};
use alloy_eips::eip2718::WithEncoded;
use alloy_primitives::B256;
use alloy_rpc_types_engine::PayloadId;
use reth_primitives_traits::{
    NodePrimitives, Recovered, SignedTransaction, transaction::TxHashRef,
};
use reth_revm::cached::CachedReads;
use ringbuffer::{AllocRingBuffer, RingBuffer};
use std::collections::{BTreeMap, HashSet};
use tokio::sync::broadcast;
use tracing::*;

/// Maximum number of cached sequences in the ring buffer.
const CACHE_SIZE: usize = 3;
/// 200 ms flashblock time.
pub(crate) const FLASHBLOCK_BLOCK_TIME: u64 = 200;

/// Stable identity for a tracked flashblock sequence.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub(crate) struct SequenceId {
    pub(crate) block_number: u64,
    pub(crate) payload_id: PayloadId,
    pub(crate) parent_hash: B256,
}

impl SequenceId {
    fn from_pending(sequence: &FlashBlockPendingSequence) -> Option<Self> {
        let base = sequence.payload_base()?;
        let payload_id = sequence.payload_id()?;
        Some(Self { block_number: base.block_number, payload_id, parent_hash: base.parent_hash })
    }

    fn from_complete(sequence: &FlashBlockCompleteSequence) -> Self {
        Self {
            block_number: sequence.block_number(),
            payload_id: sequence.payload_id(),
            parent_hash: sequence.payload_base().parent_hash,
        }
    }
}

/// Snapshot selector for build-completion matching.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
enum SequenceSnapshot {
    Pending { revision: u64 },
    Cached,
}

/// Opaque ticket that identifies the exact sequence snapshot selected for a build.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub(crate) struct BuildTicket {
    sequence_id: SequenceId,
    snapshot: SequenceSnapshot,
}

impl BuildTicket {
    const fn pending(sequence_id: SequenceId, revision: u64) -> Self {
        Self { sequence_id, snapshot: SequenceSnapshot::Pending { revision } }
    }

    const fn cached(sequence_id: SequenceId) -> Self {
        Self { sequence_id, snapshot: SequenceSnapshot::Cached }
    }
}

/// Result of attempting to apply a build completion to tracked sequence state.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) enum BuildApplyOutcome {
    SkippedNoBuildResult,
    AppliedPending,
    AppliedCached {
        rebroadcasted: bool,
    },
    RejectedPendingSequenceMismatch {
        ticket_sequence_id: SequenceId,
        current_sequence_id: Option<SequenceId>,
    },
    RejectedPendingRevisionStale {
        sequence_id: SequenceId,
        ticket_revision: u64,
        current_revision: u64,
    },
    RejectedCachedSequenceMissing {
        sequence_id: SequenceId,
    },
}

impl BuildApplyOutcome {
    pub(crate) const fn is_applied(self) -> bool {
        matches!(self, Self::AppliedPending | Self::AppliedCached { .. })
    }
}

/// A buildable sequence plus the stable identity that selected it.
pub(crate) struct BuildCandidate<I, N: NodePrimitives> {
    pub(crate) ticket: BuildTicket,
    pub(crate) args: BuildArgs<I, N>,
}

impl<I, N: NodePrimitives> std::ops::Deref for BuildCandidate<I, N> {
    type Target = BuildArgs<I, N>;

    fn deref(&self) -> &Self::Target {
        &self.args
    }
}

/// In-progress pending sequence state.
///
/// Keeps accepted flashblocks and recovered transactions in lockstep by index.
#[derive(Debug)]
struct PendingSequence<T: SignedTransaction> {
    sequence: FlashBlockPendingSequence,
    recovered_transactions_by_index: BTreeMap<u64, Vec<WithEncoded<Recovered<T>>>>,
    revision: u64,
    applied_revision: Option<u64>,
}

impl<T: SignedTransaction> PendingSequence<T> {
    fn new() -> Self {
        Self {
            sequence: FlashBlockPendingSequence::new(),
            recovered_transactions_by_index: BTreeMap::new(),
            revision: 0,
            applied_revision: None,
        }
    }

    const fn sequence(&self) -> &FlashBlockPendingSequence {
        &self.sequence
    }

    fn count(&self) -> usize {
        self.sequence.count()
    }

    const fn revision(&self) -> u64 {
        self.revision
    }

    fn clear(&mut self) {
        self.sequence = FlashBlockPendingSequence::new();
        self.recovered_transactions_by_index.clear();
        self.applied_revision = None;
    }

    const fn bump_revision(&mut self) {
        self.revision = self.revision.wrapping_add(1);
    }

    fn is_revision_applied(&self, revision: u64) -> bool {
        self.applied_revision == Some(revision)
    }

    const fn mark_revision_applied(&mut self, revision: u64) {
        self.applied_revision = Some(revision);
    }

    fn insert_flashblock(&mut self, flashblock: FlashBlock) -> eyre::Result<()> {
        if !self.sequence.can_accept(&flashblock) {
            self.sequence.insert(flashblock);
            return Ok(());
        }

        // Only recover transactions once we've validated that this flashblock is accepted.
        let recovered_txs = flashblock.recover_transactions().collect::<Result<Vec<_>, _>>()?;
        let flashblock_index = flashblock.index;

        // Index 0 starts a fresh pending block, so clear any stale in-progress data.
        if flashblock_index == 0 {
            self.clear();
        }

        self.sequence.insert(flashblock);
        self.recovered_transactions_by_index.insert(flashblock_index, recovered_txs);
        self.bump_revision();
        Ok(())
    }

    fn finalize(
        &mut self,
    ) -> eyre::Result<(FlashBlockCompleteSequence, Vec<WithEncoded<Recovered<T>>>)> {
        let finalized = self.sequence.finalize();
        let recovered_by_index = std::mem::take(&mut self.recovered_transactions_by_index);

        match finalized {
            Ok(completed) => Ok((completed, recovered_by_index.into_values().flatten().collect())),
            Err(err) => Err(err),
        }
    }

    fn transactions(&self) -> Vec<WithEncoded<Recovered<T>>> {
        self.recovered_transactions_by_index.values().flatten().cloned().collect()
    }

    fn tx_hashes(&self) -> Vec<B256> {
        self.recovered_transactions_by_index.values().flatten().map(|tx| *tx.tx_hash()).collect()
    }

    #[cfg(test)]
    fn transaction_count(&self) -> usize {
        self.recovered_transactions_by_index.values().map(Vec::len).sum()
    }
}

/// Manages flashblock sequences with caching support.
///
/// This struct handles:
/// - Tracking the current pending sequence
/// - Caching completed sequences in a fixed-size ring buffer
/// - Finding the best sequence to build based on local chain tip
/// - Broadcasting completed sequences to subscribers
#[derive(Debug)]
pub(crate) struct SequenceManager<T: SignedTransaction> {
    /// Current pending sequence being built up from incoming flashblocks
    pending: PendingSequence<T>,
    /// Ring buffer of recently completed sequences bundled with their decoded transactions (FIFO,
    /// size 3)
    completed_cache: AllocRingBuffer<(FlashBlockCompleteSequence, Vec<WithEncoded<Recovered<T>>>)>,
    /// Cached sequence identities that already had a build completion applied.
    applied_cached_sequences: HashSet<SequenceId>,
    /// Cached minimum block number currently present in `completed_cache`.
    cached_min_block_number: Option<u64>,
    /// Broadcast channel for completed sequences
    block_broadcaster: broadcast::Sender<FlashBlockCompleteSequence>,
    /// Whether to compute state roots when building blocks
    compute_state_root: bool,
}

impl<T: SignedTransaction> SequenceManager<T> {
    /// Creates a new sequence manager.
    pub(crate) fn new(compute_state_root: bool) -> Self {
        let (block_broadcaster, _) = broadcast::channel(128);
        Self {
            pending: PendingSequence::new(),
            completed_cache: AllocRingBuffer::new(CACHE_SIZE),
            applied_cached_sequences: HashSet::new(),
            cached_min_block_number: None,
            block_broadcaster,
            compute_state_root,
        }
    }

    /// Returns the sender half of the flashblock sequence broadcast channel.
    pub(crate) const fn block_sequence_broadcaster(
        &self,
    ) -> &broadcast::Sender<FlashBlockCompleteSequence> {
        &self.block_broadcaster
    }

    /// Gets a subscriber to the flashblock sequences produced.
    pub(crate) fn subscribe_block_sequence(&self) -> crate::FlashBlockCompleteSequenceRx {
        self.block_broadcaster.subscribe()
    }

    /// Inserts a new flashblock into the pending sequence.
    ///
    /// When a flashblock with index 0 arrives (indicating a new block), the current
    /// pending sequence is finalized, cached, and broadcast immediately. If the sequence
    /// is later built on top of local tip, `on_build_complete()` will broadcast again
    /// with computed `state_root`.
    ///
    /// Transactions are recovered once and cached for reuse during block building.
    pub(crate) fn insert_flashblock(&mut self, flashblock: FlashBlock) -> eyre::Result<()> {
        // If this starts a new block, finalize and cache the previous sequence BEFORE inserting
        if flashblock.index == 0 && self.pending.count() > 0 {
            let (completed, txs) = self.pending.finalize()?;
            let block_number = completed.block_number();
            let parent_hash = completed.payload_base().parent_hash;

            trace!(
                target: "flashblocks",
                block_number,
                %parent_hash,
                cache_size = self.completed_cache.len(),
                "Caching completed flashblock sequence"
            );

            // Broadcast immediately to consensus client (even without state_root)
            // This ensures sequences are forwarded during catch-up even if not buildable on tip.
            // ConsensusClient checks execution_outcome and skips newPayload if state_root is zero.
            if self.block_broadcaster.receiver_count() > 0 {
                let _ = self.block_broadcaster.send(completed.clone());
            }

            // Bundle completed sequence with its decoded transactions and push to cache
            // Ring buffer automatically evicts oldest entry when full
            self.push_completed_sequence(completed, txs);
        }

        self.pending.insert_flashblock(flashblock)?;
        Ok(())
    }

    /// Pushes a completed sequence into the cache and maintains cached min block-number metadata.
    fn push_completed_sequence(
        &mut self,
        completed: FlashBlockCompleteSequence,
        txs: Vec<WithEncoded<Recovered<T>>>,
    ) {
        let block_number = completed.block_number();
        let completed_sequence_id = SequenceId::from_complete(&completed);
        let evicted_block_number = if self.completed_cache.is_full() {
            self.completed_cache.front().map(|(seq, _)| seq.block_number())
        } else {
            None
        };
        let evicted_sequence_id = if self.completed_cache.is_full() {
            self.completed_cache.front().map(|(seq, _)| SequenceId::from_complete(seq))
        } else {
            None
        };

        if let Some(sequence_id) = evicted_sequence_id {
            self.applied_cached_sequences.remove(&sequence_id);
        }
        // Re-tracking a sequence identity should always start as unapplied.
        self.applied_cached_sequences.remove(&completed_sequence_id);

        self.completed_cache.enqueue((completed, txs));

        self.cached_min_block_number = match self.cached_min_block_number {
            None => Some(block_number),
            Some(current_min) if block_number < current_min => Some(block_number),
            Some(current_min) if Some(current_min) == evicted_block_number => {
                self.recompute_cache_min_block_number()
            }
            Some(current_min) => Some(current_min),
        };
    }

    /// Recomputes the minimum block number in `completed_cache`.
    fn recompute_cache_min_block_number(&self) -> Option<u64> {
        self.completed_cache.iter().map(|(seq, _)| seq.block_number()).min()
    }

    /// Returns the newest cached sequence that matches `parent_hash` and still needs execution.
    ///
    /// Cached sequences that already had build completion applied are skipped to avoid redundant
    /// rebuild loops.
    fn newest_unexecuted_cached_for_parent(
        &self,
        parent_hash: B256,
    ) -> Option<&(FlashBlockCompleteSequence, Vec<WithEncoded<Recovered<T>>>)> {
        self.completed_cache.iter().rev().find(|(seq, _)| {
            let sequence_id = SequenceId::from_complete(seq);
            seq.payload_base().parent_hash == parent_hash &&
                !self.applied_cached_sequences.contains(&sequence_id)
        })
    }

    /// Returns a mutable cached sequence entry by exact sequence identity.
    fn cached_entry_mut_by_id(
        &mut self,
        sequence_id: SequenceId,
    ) -> Option<&mut (FlashBlockCompleteSequence, Vec<WithEncoded<Recovered<T>>>)> {
        self.completed_cache
            .iter_mut()
            .find(|(seq, _)| SequenceId::from_complete(seq) == sequence_id)
    }

    /// Returns the current pending sequence for inspection.
    pub(crate) const fn pending(&self) -> &FlashBlockPendingSequence {
        self.pending.sequence()
    }

    /// Finds the next sequence to build and returns the selected sequence identity
    /// with ready-to-use `BuildArgs`.
    ///
    /// Priority order:
    /// 1. Current pending sequence (if parent matches local tip)
    /// 2. Cached sequence with exact parent match
    /// 3. Speculative: pending sequence with pending parent state (if provided)
    ///
    /// Returns None if nothing is buildable right now.
    pub(crate) fn next_buildable_args<N: NodePrimitives<SignedTx = T>>(
        &mut self,
        local_tip_hash: B256,
        local_tip_timestamp: u64,
        pending_parent_state: Option<PendingBlockState<N>>,
    ) -> Option<BuildCandidate<Vec<WithEncoded<Recovered<T>>>, N>> {
        // Try to find a buildable sequence: (ticket, base, last_fb, transactions,
        // cached_state, source_name, pending_parent)
        let (ticket, base, last_flashblock, transactions, cached_state, source_name, pending_parent) =
            // Priority 1: Try current pending sequence (canonical mode)
            if let Some(base) = self.pending.sequence.payload_base().filter(|b| b.parent_hash == local_tip_hash) {
                let revision = self.pending.revision();
                if self.pending.is_revision_applied(revision) {
                    trace!(
                        target: "flashblocks",
                        block_number = base.block_number,
                        revision,
                        parent_hash = ?base.parent_hash,
                        "Skipping rebuild for already-applied pending revision"
                    );
                    return None;
                }
                let sequence_id = SequenceId::from_pending(self.pending.sequence())?;
                let ticket = BuildTicket::pending(sequence_id, revision);
                let cached_state = self.pending.sequence.take_cached_reads().map(|r| (base.parent_hash, r));
                let last_fb = self.pending.sequence.last_flashblock()?;
                let transactions = self.pending.transactions();
                (ticket, base, last_fb, transactions, cached_state, "pending", None)
            }
            // Priority 2: Try cached sequence with exact parent match (canonical mode)
            else if let Some((cached, txs)) = self.newest_unexecuted_cached_for_parent(local_tip_hash) {
                let sequence_id = SequenceId::from_complete(cached);
                let ticket = BuildTicket::cached(sequence_id);
                let base = cached.payload_base().clone();
                let last_fb = cached.last();
                let transactions = txs.clone();
                let cached_state = None;
                (ticket, base, last_fb, transactions, cached_state, "cached", None)
            }
            // Priority 3: Try speculative building with pending parent state
            else if let Some(ref pending_state) = pending_parent_state {
                // Check if pending sequence's parent matches the pending state's block
                if let Some(base) = self.pending.sequence.payload_base().filter(|b| b.parent_hash == pending_state.block_hash) {
                    let revision = self.pending.revision();
                    if self.pending.is_revision_applied(revision) {
                        trace!(
                            target: "flashblocks",
                            block_number = base.block_number,
                            revision,
                            speculative_parent = ?pending_state.block_hash,
                            "Skipping speculative rebuild for already-applied pending revision"
                        );
                        return None;
                    }
                    let sequence_id = SequenceId::from_pending(self.pending.sequence())?;
                    let ticket = BuildTicket::pending(sequence_id, revision);
                    let cached_state = self.pending.sequence.take_cached_reads().map(|r| (base.parent_hash, r));
                    let last_fb = self.pending.sequence.last_flashblock()?;
                    let transactions = self.pending.transactions();
                    (
                        ticket,
                        base,
                        last_fb,
                        transactions,
                        cached_state,
                        "speculative-pending",
                        pending_parent_state,
                    )
                }
                // Check cached sequences
                else if let Some((cached, txs)) = self.newest_unexecuted_cached_for_parent(pending_state.block_hash) {
                    let sequence_id = SequenceId::from_complete(cached);
                    let ticket = BuildTicket::cached(sequence_id);
                    let base = cached.payload_base().clone();
                    let last_fb = cached.last();
                    let transactions = txs.clone();
                    let cached_state = None;
                    (
                        ticket,
                        base,
                        last_fb,
                        transactions,
                        cached_state,
                        "speculative-cached",
                        pending_parent_state,
                    )
                } else {
                    return None;
                }
            } else {
                return None;
            };

        // Auto-detect when to compute state root: only if the builder didn't provide it (sent
        // B256::ZERO) and we're near the expected final flashblock index.
        //
        // Background: Each block period receives multiple flashblocks at regular intervals.
        // The sequencer sends an initial "base" flashblock at index 0 when a new block starts,
        // then subsequent flashblocks are produced every FLASHBLOCK_BLOCK_TIME intervals (200ms).
        //
        // Examples with different block times:
        // - Base (2s blocks):    expect 2000ms / 200ms = 10 intervals → Flashblocks: index 0 (base)
        //   + indices 1-10 = potentially 11 total
        //
        // - Unichain (1s blocks): expect 1000ms / 200ms = 5 intervals → Flashblocks: index 0 (base)
        //   + indices 1-5 = potentially 6 total
        //
        // Why compute at N-1 instead of N:
        // 1. Timing variance in flashblock producing time may mean only N flashblocks were produced
        //    instead of N+1 (missing the final one). Computing at N-1 ensures we get the state root
        //    for most common cases.
        //
        // 2. The +1 case (index 0 base + N intervals): If all N+1 flashblocks do arrive, we'll
        //    still calculate state root for flashblock N, which sacrifices a little performance but
        //    still ensures correctness for common cases.
        //
        // Note: Pathological cases may result in fewer flashblocks than expected (e.g., builder
        // downtime, flashblock execution exceeding timing budget). When this occurs, we won't
        // compute the state root, causing FlashblockConsensusClient to lack precomputed state for
        // engine_newPayload. This is safe: we still have op-node as backstop to maintain
        // chain progression.
        let block_time_ms = base.timestamp.saturating_sub(local_tip_timestamp) * 1000;
        let expected_final_flashblock = block_time_ms / FLASHBLOCK_BLOCK_TIME;
        let compute_state_root = self.compute_state_root &&
            last_flashblock.diff.state_root.is_zero() &&
            last_flashblock.index >= expected_final_flashblock.saturating_sub(1);

        trace!(
            target: "flashblocks",
            block_number = base.block_number,
            source = source_name,
            ticket = ?ticket,
            flashblock_index = last_flashblock.index,
            expected_final_flashblock,
            compute_state_root_enabled = self.compute_state_root,
            state_root_is_zero = last_flashblock.diff.state_root.is_zero(),
            will_compute_state_root = compute_state_root,
            is_speculative = pending_parent.is_some(),
            "Building from flashblock sequence"
        );

        Some(BuildCandidate {
            ticket,
            args: BuildArgs {
                base,
                transactions,
                cached_state,
                last_flashblock_index: last_flashblock.index,
                last_flashblock_hash: last_flashblock.diff.block_hash,
                compute_state_root,
                pending_parent,
            },
        })
    }

    /// Records the result of building a sequence and re-broadcasts with execution outcome.
    ///
    /// Updates execution outcome and cached reads. For cached sequences (already broadcast
    /// once during finalize), this broadcasts again with the computed `state_root`, allowing
    /// the consensus client to submit via `engine_newPayload`.
    pub(crate) fn on_build_complete<N: NodePrimitives>(
        &mut self,
        ticket: BuildTicket,
        result: Option<(PendingFlashBlock<N>, CachedReads)>,
    ) -> BuildApplyOutcome {
        let Some((computed_block, cached_reads)) = result else {
            return BuildApplyOutcome::SkippedNoBuildResult;
        };

        // Extract execution outcome
        let execution_outcome = computed_block.computed_state_root().map(|state_root| {
            SequenceExecutionOutcome { block_hash: computed_block.block().hash(), state_root }
        });

        let outcome = self.apply_build_outcome(ticket, execution_outcome, cached_reads);
        match outcome {
            BuildApplyOutcome::SkippedNoBuildResult | BuildApplyOutcome::AppliedPending => {}
            BuildApplyOutcome::AppliedCached { rebroadcasted } => {
                trace!(
                    target: "flashblocks",
                    ticket = ?ticket,
                    rebroadcasted,
                    "Applied cached build completion"
                );
            }
            BuildApplyOutcome::RejectedPendingSequenceMismatch {
                ticket_sequence_id,
                current_sequence_id,
            } => {
                trace!(
                    target: "flashblocks",
                    ticket = ?ticket,
                    ?ticket_sequence_id,
                    ?current_sequence_id,
                    "Rejected build completion: pending sequence mismatch"
                );
            }
            BuildApplyOutcome::RejectedPendingRevisionStale {
                sequence_id,
                ticket_revision,
                current_revision,
            } => {
                trace!(
                    target: "flashblocks",
                    ticket = ?ticket,
                    ?sequence_id,
                    ticket_revision,
                    current_revision,
                    "Rejected build completion: pending revision stale"
                );
            }
            BuildApplyOutcome::RejectedCachedSequenceMissing { sequence_id } => {
                trace!(
                    target: "flashblocks",
                    ticket = ?ticket,
                    ?sequence_id,
                    "Rejected build completion: cached sequence missing"
                );
            }
        }
        outcome
    }

    /// Applies build output to the exact sequence targeted by the build job.
    ///
    /// Returns the apply outcome with explicit rejection reasons for observability.
    fn apply_build_outcome(
        &mut self,
        ticket: BuildTicket,
        execution_outcome: Option<SequenceExecutionOutcome>,
        cached_reads: CachedReads,
    ) -> BuildApplyOutcome {
        match ticket.snapshot {
            SequenceSnapshot::Pending { revision } => {
                let current_sequence_id = SequenceId::from_pending(self.pending.sequence());
                if current_sequence_id != Some(ticket.sequence_id) {
                    return BuildApplyOutcome::RejectedPendingSequenceMismatch {
                        ticket_sequence_id: ticket.sequence_id,
                        current_sequence_id,
                    };
                }

                let current_revision = self.pending.revision();
                if current_revision != revision {
                    return BuildApplyOutcome::RejectedPendingRevisionStale {
                        sequence_id: ticket.sequence_id,
                        ticket_revision: revision,
                        current_revision,
                    };
                }

                {
                    self.pending.sequence.set_execution_outcome(execution_outcome);
                    self.pending.sequence.set_cached_reads(cached_reads);
                    self.pending.mark_revision_applied(current_revision);
                    trace!(
                        target: "flashblocks",
                        block_number = self.pending.sequence.block_number(),
                        ticket = ?ticket,
                        has_computed_state_root = execution_outcome.is_some(),
                        "Updated pending sequence with build results"
                    );
                }
                BuildApplyOutcome::AppliedPending
            }
            SequenceSnapshot::Cached => {
                if let Some((cached, _)) = self.cached_entry_mut_by_id(ticket.sequence_id) {
                    let (needs_rebroadcast, rebroadcast_sequence) = {
                        // Only re-broadcast if we computed new information (state_root was
                        // missing). If sequencer already provided
                        // state_root, we already broadcast in
                        // insert_flashblock, so skip re-broadcast to avoid duplicate FCU calls.
                        let needs_rebroadcast =
                            execution_outcome.is_some() && cached.execution_outcome().is_none();

                        cached.set_execution_outcome(execution_outcome);

                        let rebroadcast_sequence = needs_rebroadcast.then_some(cached.clone());
                        (needs_rebroadcast, rebroadcast_sequence)
                    };
                    self.applied_cached_sequences.insert(ticket.sequence_id);

                    if let Some(sequence) = rebroadcast_sequence &&
                        self.block_broadcaster.receiver_count() > 0
                    {
                        trace!(
                            target: "flashblocks",
                            block_number = sequence.block_number(),
                            ticket = ?ticket,
                            "Re-broadcasting sequence with computed state_root"
                        );
                        let _ = self.block_broadcaster.send(sequence);
                    }
                    BuildApplyOutcome::AppliedCached { rebroadcasted: needs_rebroadcast }
                } else {
                    BuildApplyOutcome::RejectedCachedSequenceMissing {
                        sequence_id: ticket.sequence_id,
                    }
                }
            }
        }
    }

    /// Returns the earliest block number in the pending or cached sequences.
    pub(crate) fn earliest_block_number(&self) -> Option<u64> {
        match (self.pending.sequence.block_number(), self.cached_min_block_number) {
            (Some(pending_block), Some(cache_min)) => Some(cache_min.min(pending_block)),
            (Some(pending_block), None) => Some(pending_block),
            (None, Some(cache_min)) => Some(cache_min),
            (None, None) => None,
        }
    }

    /// Returns the latest block number in the pending or cached sequences.
    pub(crate) fn latest_block_number(&self) -> Option<u64> {
        // Pending is always the latest if it exists
        if let Some(pending_block) = self.pending.sequence.block_number() {
            return Some(pending_block);
        }

        // Fall back to cache
        self.completed_cache.iter().map(|(seq, _)| seq.block_number()).max()
    }

    /// Returns the tracked block fingerprint for the given block number from pending or cached
    /// sequences, if available.
    fn tracked_fingerprint_for_block(&self, block_number: u64) -> Option<TrackedBlockFingerprint> {
        // Check pending sequence
        if self.pending.sequence.block_number() == Some(block_number) {
            let base = self.pending.sequence.payload_base()?;
            let last_flashblock = self.pending.sequence.last_flashblock()?;
            let tx_hashes = self.pending.tx_hashes();
            return Some(TrackedBlockFingerprint {
                block_number,
                block_hash: last_flashblock.diff.block_hash,
                parent_hash: base.parent_hash,
                tx_hashes,
            });
        }

        // Check cached sequences (newest first). Multiple payload variants for the same block
        // number can coexist in cache; reorg checks must use the newest tracked variant.
        for (seq, txs) in self.completed_cache.iter().rev() {
            if seq.block_number() == block_number {
                let tx_hashes = txs.iter().map(|tx| *tx.tx_hash()).collect();
                return Some(TrackedBlockFingerprint {
                    block_number,
                    block_hash: seq.last().diff.block_hash,
                    parent_hash: seq.payload_base().parent_hash,
                    tx_hashes,
                });
            }
        }

        None
    }

    /// Processes a canonical block and reconciles pending state.
    ///
    /// This method determines how to handle the pending flashblock state when a new
    /// canonical block arrives. It uses the [`CanonicalBlockReconciler`] to decide
    /// the appropriate strategy based on:
    /// - Whether canonical has caught up to pending
    /// - Whether a reorg was detected (transaction mismatch)
    /// - Whether pending is too far ahead of canonical
    ///
    /// Returns the reconciliation strategy that was applied.
    pub(crate) fn process_canonical_block(
        &mut self,
        canonical: CanonicalBlockFingerprint,
        max_depth: u64,
    ) -> ReconciliationStrategy {
        let canonical_block_number = canonical.block_number;
        let earliest = self.earliest_block_number();
        let latest = self.latest_block_number();

        // Only run reorg detection if we actually track the canonical block number.
        let reorg_detected = self
            .tracked_fingerprint_for_block(canonical_block_number)
            .map(|tracked| ReorgDetector::detect(&tracked, &canonical).is_reorg())
            .unwrap_or(false);

        // Determine reconciliation strategy
        let strategy = CanonicalBlockReconciler::reconcile(
            earliest,
            latest,
            canonical_block_number,
            max_depth,
            reorg_detected,
        );

        match &strategy {
            ReconciliationStrategy::CatchUp => {
                trace!(
                    target: "flashblocks",
                    ?latest,
                    canonical_block_number,
                    "Canonical caught up - clearing pending state"
                );
                self.clear_all();
            }
            ReconciliationStrategy::HandleReorg => {
                warn!(
                    target: "flashblocks",
                    canonical_block_number,
                    canonical_tx_count = canonical.tx_hashes.len(),
                    canonical_parent_hash = ?canonical.parent_hash,
                    canonical_block_hash = ?canonical.block_hash,
                    "Reorg detected - clearing pending state"
                );
                self.clear_all();
            }
            ReconciliationStrategy::DepthLimitExceeded { depth, max_depth } => {
                trace!(
                    target: "flashblocks",
                    depth,
                    max_depth,
                    "Depth limit exceeded - clearing pending state"
                );
                self.clear_all();
            }
            ReconciliationStrategy::Continue => {
                trace!(
                    target: "flashblocks",
                    ?earliest,
                    ?latest,
                    canonical_block_number,
                    "Canonical behind pending - continuing"
                );
            }
            ReconciliationStrategy::NoPendingState => {
                trace!(
                    target: "flashblocks",
                    canonical_block_number,
                    "No pending state to reconcile"
                );
            }
        }

        strategy
    }

    /// Clears all pending and cached state.
    fn clear_all(&mut self) {
        self.pending.clear();
        self.completed_cache.clear();
        self.applied_cached_sequences.clear();
        self.cached_min_block_number = None;
    }

    #[cfg(test)]
    fn pending_transaction_count(&self) -> usize {
        self.pending.transaction_count()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        test_utils::TestFlashBlockFactory,
        validation::{CanonicalBlockFingerprint, ReconciliationStrategy},
    };
    use alloy_primitives::B256;
    use alloy_rpc_types_engine::PayloadId;
    use op_alloy_consensus::OpTxEnvelope;
    use reth_optimism_primitives::OpPrimitives;

    fn canonical_for(
        manager: &SequenceManager<OpTxEnvelope>,
        block_number: u64,
        tx_hashes: Vec<B256>,
    ) -> CanonicalBlockFingerprint {
        if let Some(tracked) = manager.tracked_fingerprint_for_block(block_number) {
            CanonicalBlockFingerprint {
                block_number,
                block_hash: tracked.block_hash,
                parent_hash: tracked.parent_hash,
                tx_hashes,
            }
        } else {
            CanonicalBlockFingerprint {
                block_number,
                block_hash: B256::repeat_byte(0xFE),
                parent_hash: B256::repeat_byte(0xFD),
                tx_hashes,
            }
        }
    }

    #[test]
    fn test_sequence_manager_new() {
        let manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        assert_eq!(manager.pending().count(), 0);
    }

    #[test]
    fn test_insert_flashblock_creates_pending_sequence() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        let fb0 = factory.flashblock_at(0).build();
        manager.insert_flashblock(fb0).unwrap();

        assert_eq!(manager.pending().count(), 1);
        assert_eq!(manager.pending().block_number(), Some(100));
    }

    #[test]
    fn test_insert_flashblock_caches_completed_sequence() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        // Build first sequence
        let fb0 = factory.flashblock_at(0).build();
        manager.insert_flashblock(fb0.clone()).unwrap();

        let fb1 = factory.flashblock_after(&fb0).build();
        manager.insert_flashblock(fb1).unwrap();

        // Insert new base (index 0) which should finalize and cache previous sequence
        let fb2 = factory.flashblock_for_next_block(&fb0).build();
        manager.insert_flashblock(fb2).unwrap();

        // New sequence should be pending
        assert_eq!(manager.pending().count(), 1);
        assert_eq!(manager.pending().block_number(), Some(101));
        assert_eq!(manager.completed_cache.len(), 1);
        let (cached_sequence, _txs) = manager.completed_cache.get(0).unwrap();
        assert_eq!(cached_sequence.block_number(), 100);
    }

    #[test]
    fn test_next_buildable_args_returns_none_when_empty() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let local_tip_hash = B256::random();
        let local_tip_timestamp = 1000;

        let args =
            manager.next_buildable_args::<OpPrimitives>(local_tip_hash, local_tip_timestamp, None);
        assert!(args.is_none());
    }

    #[test]
    fn test_next_buildable_args_matches_pending_parent() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        let fb0 = factory.flashblock_at(0).build();
        let parent_hash = fb0.base.as_ref().unwrap().parent_hash;
        manager.insert_flashblock(fb0).unwrap();

        let args = manager.next_buildable_args::<OpPrimitives>(parent_hash, 1000000, None);
        assert!(args.is_some());

        let build_args = args.unwrap();
        assert_eq!(build_args.last_flashblock_index, 0);
    }

    #[test]
    fn test_next_buildable_args_returns_none_when_parent_mismatch() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        let fb0 = factory.flashblock_at(0).build();
        manager.insert_flashblock(fb0).unwrap();

        // Use different parent hash
        let wrong_parent = B256::random();
        let args = manager.next_buildable_args::<OpPrimitives>(wrong_parent, 1000000, None);
        assert!(args.is_none());
    }

    #[test]
    fn test_next_buildable_args_prefers_pending_over_cached() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        // Create and finalize first sequence
        let fb0 = factory.flashblock_at(0).build();
        manager.insert_flashblock(fb0.clone()).unwrap();

        // Create new sequence (finalizes previous)
        let fb1 = factory.flashblock_for_next_block(&fb0).build();
        let parent_hash = fb1.base.as_ref().unwrap().parent_hash;
        manager.insert_flashblock(fb1).unwrap();

        // Request with first sequence's parent (should find cached)
        let args = manager.next_buildable_args::<OpPrimitives>(parent_hash, 1000000, None);
        assert!(args.is_some());
    }

    #[test]
    fn test_next_buildable_args_finds_cached_sequence() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        // Build and cache first sequence
        let fb0 = factory.flashblock_at(0).build();
        let parent_hash = fb0.base.as_ref().unwrap().parent_hash;
        manager.insert_flashblock(fb0.clone()).unwrap();

        // Start new sequence to finalize first
        let fb1 = factory.flashblock_for_next_block(&fb0).build();
        manager.insert_flashblock(fb1.clone()).unwrap();

        // Clear pending by starting another sequence
        let fb2 = factory.flashblock_for_next_block(&fb1).build();
        manager.insert_flashblock(fb2).unwrap();

        // Request first sequence's parent - should find in cache
        let args = manager.next_buildable_args::<OpPrimitives>(parent_hash, 1000000, None);
        assert!(args.is_some());
    }

    #[test]
    fn test_next_buildable_args_uses_newest_cached_when_parent_hash_shared() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        let shared_parent = B256::repeat_byte(0x44);
        let payload_a = PayloadId::new([0xAA; 8]);
        let payload_b = PayloadId::new([0xBB; 8]);

        // Sequence A for block 100 (will become cached first).
        let fb_a0 = factory
            .flashblock_at(0)
            .block_number(100)
            .parent_hash(shared_parent)
            .payload_id(payload_a)
            .build();
        manager.insert_flashblock(fb_a0).unwrap();

        // Sequence B for the same parent hash and block number (different payload id).
        // Inserting index 0 finalizes/caches sequence A.
        let fb_b0 = factory
            .flashblock_at(0)
            .block_number(100)
            .parent_hash(shared_parent)
            .payload_id(payload_b)
            .build();
        manager.insert_flashblock(fb_b0.clone()).unwrap();

        // Finalize/cache sequence B.
        let fb_next = factory.flashblock_for_next_block(&fb_b0).build();
        manager.insert_flashblock(fb_next).unwrap();

        let candidate = manager
            .next_buildable_args::<OpPrimitives>(shared_parent, 1_000_000, None)
            .expect("shared parent should resolve to a cached sequence");

        // Newest sequence (B) should be selected deterministically.
        assert_eq!(candidate.ticket.sequence_id.payload_id, payload_b);
        assert_eq!(candidate.last_flashblock_hash, fb_b0.diff.block_hash);
    }

    #[test]
    fn test_next_buildable_args_skips_executed_cached_and_advances_speculative() {
        use crate::pending_state::PendingBlockState;
        use reth_execution_types::BlockExecutionOutput;
        use reth_revm::cached::CachedReads;
        use std::sync::Arc;

        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        // Block 100 with three flashblocks.
        let fb100_0 = factory.flashblock_at(0).build();
        let local_tip_hash = fb100_0.base.as_ref().unwrap().parent_hash;
        manager.insert_flashblock(fb100_0.clone()).unwrap();
        let fb100_1 = factory.flashblock_after(&fb100_0).build();
        manager.insert_flashblock(fb100_1.clone()).unwrap();
        let fb100_2 = factory.flashblock_after(&fb100_1).build();
        manager.insert_flashblock(fb100_2.clone()).unwrap();

        // First flashblock of block 101 finalizes block 100 into cache.
        let fb101_0 = factory.flashblock_for_next_block(&fb100_2).build();
        manager.insert_flashblock(fb101_0.clone()).unwrap();

        // First build picks canonical-attached cached block 100.
        let first = manager
            .next_buildable_args::<OpPrimitives>(local_tip_hash, 1_000_000, None)
            .expect("cached block should be buildable first");
        assert!(matches!(first.ticket.snapshot, SequenceSnapshot::Cached));
        assert_eq!(first.base.block_number, fb100_0.block_number());

        // Mark cached block 100 as executed.
        let applied = manager.apply_build_outcome(
            first.ticket,
            Some(SequenceExecutionOutcome {
                block_hash: B256::repeat_byte(0x33),
                state_root: B256::repeat_byte(0x44),
            }),
            CachedReads::default(),
        );
        assert!(matches!(
            applied,
            BuildApplyOutcome::AppliedCached { rebroadcasted: true | false }
        ));

        // Speculative state for block 100 should unlock block 101/index0.
        let pending_state = PendingBlockState::<OpPrimitives> {
            block_hash: fb101_0.base.as_ref().unwrap().parent_hash,
            block_number: fb100_0.block_number(),
            parent_hash: local_tip_hash,
            canonical_anchor_hash: local_tip_hash,
            execution_outcome: Arc::new(BlockExecutionOutput::default()),
            cached_reads: CachedReads::default(),
            sealed_header: None,
        };

        let second = manager
            .next_buildable_args(local_tip_hash, 1_000_000, Some(pending_state))
            .expect("speculative pending block should be buildable next");
        assert!(matches!(second.ticket.snapshot, SequenceSnapshot::Pending { .. }));
        assert_eq!(second.base.block_number, fb101_0.block_number());
        assert!(second.pending_parent.is_some());
    }

    #[test]
    fn test_cached_sequence_with_provided_state_root_not_reselected_after_apply() {
        use reth_revm::cached::CachedReads;

        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();
        let provided_root = B256::repeat_byte(0xA5);

        // Block 100 sequence has non-zero state root from sequencer.
        let fb100_0 = factory.flashblock_at(0).state_root(provided_root).build();
        let local_tip_hash = fb100_0.base.as_ref().unwrap().parent_hash;
        manager.insert_flashblock(fb100_0.clone()).unwrap();

        let fb100_1 = factory.flashblock_after(&fb100_0).state_root(provided_root).build();
        manager.insert_flashblock(fb100_1.clone()).unwrap();

        let fb100_2 = factory.flashblock_after(&fb100_1).state_root(provided_root).build();
        manager.insert_flashblock(fb100_2.clone()).unwrap();

        // First flashblock of block 101 finalizes block 100 into cache.
        let fb101_0 = factory.flashblock_for_next_block(&fb100_2).build();
        manager.insert_flashblock(fb101_0).unwrap();

        let candidate = manager
            .next_buildable_args::<OpPrimitives>(local_tip_hash, 1_000_000, None)
            .expect("cached sequence should be buildable once");
        assert!(matches!(candidate.ticket.snapshot, SequenceSnapshot::Cached));
        assert!(
            !candidate.compute_state_root,
            "non-zero sequencer root should skip local root compute"
        );

        let applied = manager.apply_build_outcome(candidate.ticket, None, CachedReads::default());
        assert!(matches!(applied, BuildApplyOutcome::AppliedCached { rebroadcasted: false }));

        let repeated = manager.next_buildable_args::<OpPrimitives>(local_tip_hash, 1_000_000, None);
        assert!(
            repeated.is_none(),
            "cached sequence with provided state root must not be reselected after apply"
        );
    }

    #[test]
    fn test_delayed_canonical_allows_speculative_next_block_index_zero() {
        use crate::pending_state::PendingBlockState;
        use reth_execution_types::BlockExecutionOutput;
        use reth_revm::cached::CachedReads;
        use std::sync::Arc;

        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        // Canonical tip is block 9. Flashblocks for block 10 all build on block 9.
        let canonical_9_hash = B256::repeat_byte(0x09);
        let fb10_0 = factory
            .flashblock_at(0)
            .block_number(10)
            .parent_hash(canonical_9_hash)
            .block_hash(B256::repeat_byte(0x10))
            .build();
        manager.insert_flashblock(fb10_0.clone()).unwrap();

        let fb10_1 = factory.flashblock_after(&fb10_0).block_hash(B256::repeat_byte(0x11)).build();
        manager.insert_flashblock(fb10_1.clone()).unwrap();

        let fb10_2 = factory.flashblock_after(&fb10_1).block_hash(B256::repeat_byte(0x12)).build();
        manager.insert_flashblock(fb10_2.clone()).unwrap();

        // First flashblock for block 11 arrives before canonical block 10.
        let fb11_0 =
            factory.flashblock_for_next_block(&fb10_2).block_hash(B256::repeat_byte(0x20)).build();
        manager.insert_flashblock(fb11_0.clone()).unwrap();

        // Build block 10 first from canonical tip (cached canonical-attached sequence).
        let block10_candidate = manager
            .next_buildable_args::<OpPrimitives>(canonical_9_hash, 1_000_000, None)
            .expect("block 10 should be buildable from canonical tip");
        assert_eq!(block10_candidate.base.block_number, 10);
        assert!(matches!(block10_candidate.ticket.snapshot, SequenceSnapshot::Cached));

        let applied = manager.apply_build_outcome(
            block10_candidate.ticket,
            Some(SequenceExecutionOutcome {
                block_hash: fb11_0.base.as_ref().unwrap().parent_hash,
                state_root: B256::repeat_byte(0xAA),
            }),
            CachedReads::default(),
        );
        assert!(matches!(
            applied,
            BuildApplyOutcome::AppliedCached { rebroadcasted: true | false }
        ));

        // Speculative state produced by block 10 should unlock block 11/index 0
        // even though canonical block 10 has not arrived yet.
        let pending_state_10 = PendingBlockState::<OpPrimitives> {
            block_hash: fb11_0.base.as_ref().unwrap().parent_hash,
            block_number: 10,
            parent_hash: canonical_9_hash,
            canonical_anchor_hash: canonical_9_hash,
            execution_outcome: Arc::new(BlockExecutionOutput::default()),
            cached_reads: CachedReads::default(),
            sealed_header: None,
        };

        let before_canonical_10 = manager
            .next_buildable_args(canonical_9_hash, 1_000_000, Some(pending_state_10.clone()))
            .expect("block 11/index0 should be buildable speculatively before canonical block 10");
        assert_eq!(before_canonical_10.base.block_number, 11);
        assert!(before_canonical_10.pending_parent.is_some());
        assert_eq!(
            before_canonical_10.pending_parent.as_ref().unwrap().canonical_anchor_hash,
            canonical_9_hash
        );

        // Canonical block 10 arrives later: strategy must be Continue (do not clear pending state).
        let strategy = manager.process_canonical_block(canonical_for(&manager, 10, vec![]), 64);
        assert_eq!(strategy, ReconciliationStrategy::Continue);

        // Block 11/index0 must remain buildable after delayed canonical block 10.
        let after_canonical_10 = manager
            .next_buildable_args(canonical_9_hash, 1_000_000, Some(pending_state_10))
            .expect("block 11/index0 should remain buildable after delayed canonical block 10");
        assert_eq!(after_canonical_10.base.block_number, 11);
        assert!(after_canonical_10.pending_parent.is_some());
    }

    #[test]
    fn test_cached_entry_lookup_is_exact_by_sequence_id() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        let shared_parent = B256::repeat_byte(0x55);
        let payload_a = PayloadId::new([0x0A; 8]);
        let payload_b = PayloadId::new([0x0B; 8]);

        let fb_a0 = factory
            .flashblock_at(0)
            .block_number(100)
            .parent_hash(shared_parent)
            .payload_id(payload_a)
            .build();
        manager.insert_flashblock(fb_a0).unwrap();

        let fb_b0 = factory
            .flashblock_at(0)
            .block_number(100)
            .parent_hash(shared_parent)
            .payload_id(payload_b)
            .build();
        manager.insert_flashblock(fb_b0.clone()).unwrap();

        // Finalize/cache sequence B.
        let fb_next = factory.flashblock_for_next_block(&fb_b0).build();
        manager.insert_flashblock(fb_next).unwrap();

        let seq_a_id =
            SequenceId { block_number: 100, payload_id: payload_a, parent_hash: shared_parent };
        let seq_b_id =
            SequenceId { block_number: 100, payload_id: payload_b, parent_hash: shared_parent };

        let (seq_a, _) = manager
            .cached_entry_mut_by_id(seq_a_id)
            .expect("sequence A should be found by exact id");
        assert_eq!(seq_a.payload_id(), payload_a);

        let (seq_b, _) = manager
            .cached_entry_mut_by_id(seq_b_id)
            .expect("sequence B should be found by exact id");
        assert_eq!(seq_b.payload_id(), payload_b);
    }

    #[test]
    fn test_reorg_detection_uses_newest_cached_variant_for_block_number() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        let shared_parent = B256::repeat_byte(0x66);
        let payload_a = PayloadId::new([0x1A; 8]);
        let payload_b = PayloadId::new([0x1B; 8]);

        // Sequence A for block 100 (cached first).
        let fb_a0 = factory
            .flashblock_at(0)
            .block_number(100)
            .parent_hash(shared_parent)
            .payload_id(payload_a)
            .block_hash(B256::repeat_byte(0xA1))
            .build();
        manager.insert_flashblock(fb_a0).unwrap();

        // Sequence B for the same block number/parent (cached second = newest).
        let fb_b0 = factory
            .flashblock_at(0)
            .block_number(100)
            .parent_hash(shared_parent)
            .payload_id(payload_b)
            .block_hash(B256::repeat_byte(0xB1))
            .build();
        manager.insert_flashblock(fb_b0.clone()).unwrap();

        // Finalize/cache B and start pending block 101.
        let fb_next = factory.flashblock_for_next_block(&fb_b0).build();
        manager.insert_flashblock(fb_next).unwrap();

        let tracked = manager
            .tracked_fingerprint_for_block(100)
            .expect("tracked fingerprint for block 100 should exist");
        assert_eq!(
            tracked.block_hash, fb_b0.diff.block_hash,
            "reorg detection must use newest cached variant for a shared block number"
        );

        // Canonical matches newest variant B; this must not be treated as reorg.
        let canonical = CanonicalBlockFingerprint {
            block_number: 100,
            block_hash: fb_b0.diff.block_hash,
            parent_hash: shared_parent,
            tx_hashes: tracked.tx_hashes,
        };

        let strategy = manager.process_canonical_block(canonical, 64);
        assert_eq!(strategy, ReconciliationStrategy::Continue);
        assert_eq!(manager.pending().block_number(), Some(101));
        assert!(!manager.completed_cache.is_empty());
    }

    #[test]
    fn test_on_build_complete_ignores_unknown_sequence_id() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        // Build one cached sequence and one pending sequence.
        let fb0 = factory.flashblock_at(0).build();
        manager.insert_flashblock(fb0.clone()).unwrap();
        let fb1 = factory.flashblock_for_next_block(&fb0).build();
        manager.insert_flashblock(fb1.clone()).unwrap();

        assert_eq!(manager.completed_cache.len(), 1);
        assert!(manager.completed_cache.get(0).unwrap().0.execution_outcome().is_none());

        let pending_parent = manager.pending().payload_base().unwrap().parent_hash;
        let before = manager
            .next_buildable_args::<OpPrimitives>(pending_parent, 1_000_000, None)
            .expect("pending sequence should be buildable");
        assert!(before.cached_state.is_none(), "pending sequence must start without cached reads");

        let cached = &manager.completed_cache.get(0).unwrap().0;
        let stale_payload = if cached.payload_id() == PayloadId::new([0xEE; 8]) {
            PayloadId::new([0xEF; 8])
        } else {
            PayloadId::new([0xEE; 8])
        };
        let stale_id = SequenceId {
            block_number: cached.block_number(),
            payload_id: stale_payload,
            parent_hash: cached.payload_base().parent_hash,
        };
        let stale_ticket = BuildTicket::cached(stale_id);

        let applied = manager.apply_build_outcome(
            stale_ticket,
            Some(SequenceExecutionOutcome {
                block_hash: B256::repeat_byte(0x11),
                state_root: B256::repeat_byte(0x22),
            }),
            reth_revm::cached::CachedReads::default(),
        );
        assert!(matches!(applied, BuildApplyOutcome::RejectedCachedSequenceMissing { .. }));

        // Unknown sequence IDs must never mutate tracked pending/cached state.
        let after = manager
            .next_buildable_args::<OpPrimitives>(pending_parent, 1_000_000, None)
            .expect("pending sequence should remain buildable");
        assert!(after.cached_state.is_none(), "stale completion must not attach cached reads");

        // Finalize current pending sequence and ensure no synthetic execution outcome was injected.
        let pending_block_number = manager.pending().block_number().unwrap();
        let fb2 = factory.flashblock_for_next_block(&fb1).build();
        manager.insert_flashblock(fb2).unwrap();
        let finalized_pending = manager
            .completed_cache
            .iter()
            .find(|(seq, _)| seq.block_number() == pending_block_number)
            .expect("pending sequence should be finalized into cache")
            .0
            .clone();
        assert!(finalized_pending.execution_outcome().is_none());

        assert!(manager.completed_cache.get(0).unwrap().0.execution_outcome().is_none());
    }

    #[test]
    fn test_pending_build_ticket_rejects_stale_revision() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        let fb0 = factory.flashblock_at(0).build();
        let parent_hash = fb0.base.as_ref().unwrap().parent_hash;
        manager.insert_flashblock(fb0.clone()).unwrap();

        let first_candidate = manager
            .next_buildable_args::<OpPrimitives>(parent_hash, 1_000_000, None)
            .expect("initial pending sequence should be buildable");
        let stale_ticket = first_candidate.ticket;

        // Pending sequence advances while the old build would be in-flight.
        let fb1 = factory.flashblock_after(&fb0).build();
        manager.insert_flashblock(fb1.clone()).unwrap();

        let stale_applied = manager.apply_build_outcome(
            stale_ticket,
            Some(SequenceExecutionOutcome {
                block_hash: B256::repeat_byte(0x31),
                state_root: B256::repeat_byte(0x32),
            }),
            reth_revm::cached::CachedReads::default(),
        );
        assert!(
            matches!(stale_applied, BuildApplyOutcome::RejectedPendingRevisionStale { .. }),
            "stale pending ticket must be rejected"
        );

        // Fresh ticket for the current revision should still apply.
        let fresh_candidate = manager
            .next_buildable_args::<OpPrimitives>(parent_hash, 1_000_000, None)
            .expect("advanced pending sequence should remain buildable");
        assert_eq!(fresh_candidate.last_flashblock_hash, fb1.diff.block_hash);
        assert!(fresh_candidate.cached_state.is_none());

        let fresh_applied = manager.apply_build_outcome(
            fresh_candidate.ticket,
            Some(SequenceExecutionOutcome {
                block_hash: B256::repeat_byte(0x41),
                state_root: B256::repeat_byte(0x42),
            }),
            reth_revm::cached::CachedReads::default(),
        );
        assert!(matches!(fresh_applied, BuildApplyOutcome::AppliedPending));

        let with_same_revision =
            manager.next_buildable_args::<OpPrimitives>(parent_hash, 1_000_000, None);
        assert!(
            with_same_revision.is_none(),
            "applied pending revision must not be rebuilt until sequence revision advances"
        );

        // Once pending data advances, the next revision should be buildable and use cached reads.
        let fb2 = factory.flashblock_after(&fb1).build();
        manager.insert_flashblock(fb2.clone()).unwrap();

        let with_cached_state = manager
            .next_buildable_args::<OpPrimitives>(parent_hash, 1_000_000, None)
            .expect("pending sequence should be buildable after revision advances");
        assert_eq!(with_cached_state.last_flashblock_hash, fb2.diff.block_hash);
        assert!(
            with_cached_state.cached_state.is_some(),
            "fresh completion should attach cached reads once pending revision advances"
        );
    }

    #[test]
    fn test_compute_state_root_logic_near_expected_final() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let block_time = 2u64;
        let factory = TestFlashBlockFactory::new().with_block_time(block_time);

        // Create sequence with zero state root (needs computation)
        let fb0 = factory.flashblock_at(0).state_root(B256::ZERO).build();
        let parent_hash = fb0.base.as_ref().unwrap().parent_hash;
        let base_timestamp = fb0.base.as_ref().unwrap().timestamp;
        manager.insert_flashblock(fb0.clone()).unwrap();

        // Add flashblocks up to expected final index (2000ms / 200ms = 10)
        for i in 1..=9 {
            let fb = factory.flashblock_after(&fb0).index(i).state_root(B256::ZERO).build();
            manager.insert_flashblock(fb).unwrap();
        }

        // Request with proper timing - should compute state root for index 9
        let args = manager.next_buildable_args::<OpPrimitives>(
            parent_hash,
            base_timestamp - block_time,
            None,
        );
        assert!(args.is_some());
        assert!(args.unwrap().compute_state_root);
    }

    #[test]
    fn test_no_compute_state_root_when_provided_by_sequencer() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let block_time = 2u64;
        let factory = TestFlashBlockFactory::new().with_block_time(block_time);

        // Create sequence with non-zero state root (provided by sequencer)
        let fb0 = factory.flashblock_at(0).state_root(B256::random()).build();
        let parent_hash = fb0.base.as_ref().unwrap().parent_hash;
        let base_timestamp = fb0.base.as_ref().unwrap().timestamp;
        manager.insert_flashblock(fb0).unwrap();

        let args = manager.next_buildable_args::<OpPrimitives>(
            parent_hash,
            base_timestamp - block_time,
            None,
        );
        assert!(args.is_some());
        assert!(!args.unwrap().compute_state_root);
    }

    #[test]
    fn test_no_compute_state_root_when_disabled() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(false);
        let block_time = 2u64;
        let factory = TestFlashBlockFactory::new().with_block_time(block_time);

        // Create sequence with zero state root (needs computation)
        let fb0 = factory.flashblock_at(0).state_root(B256::ZERO).build();
        let parent_hash = fb0.base.as_ref().unwrap().parent_hash;
        let base_timestamp = fb0.base.as_ref().unwrap().timestamp;
        manager.insert_flashblock(fb0.clone()).unwrap();

        // Add flashblocks up to expected final index (2000ms / 200ms = 10)
        for i in 1..=9 {
            let fb = factory.flashblock_after(&fb0).index(i).state_root(B256::ZERO).build();
            manager.insert_flashblock(fb).unwrap();
        }

        // Request with proper timing - should compute state root for index 9
        let args = manager.next_buildable_args::<OpPrimitives>(
            parent_hash,
            base_timestamp - block_time,
            None,
        );
        assert!(args.is_some());
        assert!(!args.unwrap().compute_state_root);
    }

    #[test]
    fn test_compute_state_root_with_timestamp_skew_does_not_underflow() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        let fb0 = factory.flashblock_at(0).state_root(B256::ZERO).build();
        let parent_hash = fb0.base.as_ref().unwrap().parent_hash;
        let base_timestamp = fb0.base.as_ref().unwrap().timestamp;
        manager.insert_flashblock(fb0).unwrap();

        // Local tip timestamp can be ahead briefly in skewed/out-of-order conditions.
        // This should not panic due to arithmetic underflow.
        let args =
            manager.next_buildable_args::<OpPrimitives>(parent_hash, base_timestamp + 1, None);
        assert!(args.is_some());
    }

    #[test]
    fn test_cache_ring_buffer_evicts_oldest() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        // Fill cache with 4 sequences (cache size is 3, so oldest should be evicted)
        let mut last_fb = factory.flashblock_at(0).build();
        manager.insert_flashblock(last_fb.clone()).unwrap();

        for _ in 0..3 {
            last_fb = factory.flashblock_for_next_block(&last_fb).build();
            manager.insert_flashblock(last_fb.clone()).unwrap();
        }

        // The first sequence should have been evicted, so we can't build it
        let first_parent = factory.flashblock_at(0).build().base.unwrap().parent_hash;
        let args = manager.next_buildable_args::<OpPrimitives>(first_parent, 1000000, None);
        // Should not find it (evicted from ring buffer)
        assert!(args.is_none());
    }

    // ==================== Canonical Block Reconciliation Tests ====================

    #[test]
    fn test_process_canonical_block_no_pending_state() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);

        // No pending state, should return NoPendingState
        let canonical = canonical_for(&manager, 100, vec![]);
        let strategy = manager.process_canonical_block(canonical, 10);
        assert_eq!(strategy, ReconciliationStrategy::NoPendingState);
    }

    #[test]
    fn test_process_canonical_block_catchup() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        // Insert a flashblock sequence for block 100
        let fb0 = factory.flashblock_at(0).build();
        manager.insert_flashblock(fb0).unwrap();

        assert_eq!(manager.pending().block_number(), Some(100));

        // Canonical catches up to block 100
        let canonical = canonical_for(&manager, 100, vec![]);
        let strategy = manager.process_canonical_block(canonical, 10);
        assert_eq!(strategy, ReconciliationStrategy::CatchUp);

        // Pending state should be cleared
        assert!(manager.pending().block_number().is_none());
    }

    #[test]
    fn test_process_canonical_block_continue() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        // Insert flashblocks for block 100-102
        let fb0 = factory.flashblock_at(0).build();
        manager.insert_flashblock(fb0.clone()).unwrap();

        let fb1 = factory.flashblock_for_next_block(&fb0).build();
        manager.insert_flashblock(fb1.clone()).unwrap();

        let fb2 = factory.flashblock_for_next_block(&fb1).build();
        manager.insert_flashblock(fb2).unwrap();

        // Canonical at 99 (behind pending)
        let canonical = canonical_for(&manager, 99, vec![]);
        let strategy = manager.process_canonical_block(canonical, 10);
        assert_eq!(strategy, ReconciliationStrategy::Continue);

        // Pending state should still exist
        assert!(manager.pending().block_number().is_some());
    }

    #[test]
    fn test_process_canonical_block_depth_limit_exceeded() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        // Insert flashblocks for block 100-102
        let fb0 = factory.flashblock_at(0).build();
        manager.insert_flashblock(fb0.clone()).unwrap();

        let fb1 = factory.flashblock_for_next_block(&fb0).build();
        manager.insert_flashblock(fb1.clone()).unwrap();

        let fb2 = factory.flashblock_for_next_block(&fb1).build();
        manager.insert_flashblock(fb2).unwrap();

        // At this point: earliest=100, latest=102
        // Canonical at 105 with max_depth of 2 (depth = 105 - 100 = 5, which exceeds 2)
        // But wait - if canonical >= latest, it's CatchUp. So canonical must be < latest (102).
        // Let's use canonical=101, which is < 102 but depth = 101 - 100 = 1 > 0
        let canonical = canonical_for(&manager, 101, vec![]);
        let strategy = manager.process_canonical_block(canonical, 0);
        assert!(matches!(strategy, ReconciliationStrategy::DepthLimitExceeded { .. }));

        // Pending state should be cleared
        assert!(manager.pending().block_number().is_none());
    }

    #[test]
    fn test_earliest_and_latest_block_numbers() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        // Initially no blocks
        assert!(manager.earliest_block_number().is_none());
        assert!(manager.latest_block_number().is_none());

        // Insert first flashblock (block 100)
        let fb0 = factory.flashblock_at(0).build();
        manager.insert_flashblock(fb0.clone()).unwrap();

        assert_eq!(manager.earliest_block_number(), Some(100));
        assert_eq!(manager.latest_block_number(), Some(100));

        // Insert next block (block 101) - this caches block 100
        let fb1 = factory.flashblock_for_next_block(&fb0).build();
        manager.insert_flashblock(fb1.clone()).unwrap();

        assert_eq!(manager.earliest_block_number(), Some(100));
        assert_eq!(manager.latest_block_number(), Some(101));

        // Insert another block (block 102)
        let fb2 = factory.flashblock_for_next_block(&fb1).build();
        manager.insert_flashblock(fb2).unwrap();

        assert_eq!(manager.earliest_block_number(), Some(100));
        assert_eq!(manager.latest_block_number(), Some(102));
    }

    #[test]
    fn test_earliest_block_number_tracks_cache_rollover() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        let fb0 = factory.flashblock_at(0).build();
        manager.insert_flashblock(fb0.clone()).unwrap();

        let fb1 = factory.flashblock_for_next_block(&fb0).build();
        manager.insert_flashblock(fb1.clone()).unwrap();

        let fb2 = factory.flashblock_for_next_block(&fb1).build();
        manager.insert_flashblock(fb2.clone()).unwrap();

        let fb3 = factory.flashblock_for_next_block(&fb2).build();
        manager.insert_flashblock(fb3.clone()).unwrap();

        let fb4 = factory.flashblock_for_next_block(&fb3).build();
        manager.insert_flashblock(fb4).unwrap();

        // Cache size is 3, so block 100 should have been evicted.
        assert_eq!(manager.earliest_block_number(), Some(101));
        assert_eq!(manager.latest_block_number(), Some(104));
    }

    // ==================== Speculative Building Tests ====================

    #[test]
    fn test_speculative_build_with_pending_parent_state() {
        use crate::pending_state::PendingBlockState;
        use reth_execution_types::BlockExecutionOutput;
        use reth_revm::cached::CachedReads;
        use std::sync::Arc;

        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        // Create a flashblock for block 101
        let fb0 = factory.flashblock_at(0).block_number(101).build();
        // The parent_hash of block 101 should be the hash of block 100
        let block_100_hash = fb0.base.as_ref().unwrap().parent_hash;
        manager.insert_flashblock(fb0).unwrap();

        // Local tip is block 99 (not matching block 100's hash)
        let local_tip_hash = B256::random();

        // Without pending parent state, no args should be returned
        let args = manager.next_buildable_args::<OpPrimitives>(local_tip_hash, 1000000, None);
        assert!(args.is_none());

        // Create pending parent state for block 100 (its block_hash matches fb0's parent_hash)
        let parent_hash = B256::random();
        let pending_state: PendingBlockState<OpPrimitives> = PendingBlockState {
            block_hash: block_100_hash,
            block_number: 100,
            parent_hash,
            canonical_anchor_hash: parent_hash,
            execution_outcome: Arc::new(BlockExecutionOutput::default()),
            cached_reads: CachedReads::default(),
            sealed_header: None,
        };

        // With pending parent state, should return args for speculative building
        let args = manager.next_buildable_args(local_tip_hash, 1000000, Some(pending_state));
        assert!(args.is_some());
        let build_args = args.unwrap();
        assert!(build_args.pending_parent.is_some());
        assert_eq!(build_args.pending_parent.as_ref().unwrap().block_number, 100);
    }

    #[test]
    fn test_speculative_build_uses_cached_sequence() {
        use crate::pending_state::PendingBlockState;
        use reth_execution_types::BlockExecutionOutput;
        use reth_revm::cached::CachedReads;
        use std::sync::Arc;

        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        // Create and cache first sequence for block 100
        let fb0 = factory.flashblock_at(0).build();
        let block_99_hash = fb0.base.as_ref().unwrap().parent_hash;
        manager.insert_flashblock(fb0.clone()).unwrap();

        // Create second sequence for block 101 (this caches block 100)
        let fb1 = factory.flashblock_for_next_block(&fb0).build();
        manager.insert_flashblock(fb1.clone()).unwrap();

        // Create third sequence for block 102 (this caches block 101)
        let fb2 = factory.flashblock_for_next_block(&fb1).build();
        manager.insert_flashblock(fb2).unwrap();

        // Local tip is some random hash (not matching any sequence parent)
        let local_tip_hash = B256::random();

        // Create pending parent state that matches the cached block 100 sequence's parent
        let parent_hash = B256::random();
        let pending_state: PendingBlockState<OpPrimitives> = PendingBlockState {
            block_hash: block_99_hash,
            block_number: 99,
            parent_hash,
            canonical_anchor_hash: parent_hash,
            execution_outcome: Arc::new(BlockExecutionOutput::default()),
            cached_reads: CachedReads::default(),
            sealed_header: None,
        };

        // Should find cached sequence for block 100 (whose parent is block_99_hash)
        let args = manager.next_buildable_args(local_tip_hash, 1000000, Some(pending_state));
        assert!(args.is_some());
        let build_args = args.unwrap();
        assert!(build_args.pending_parent.is_some());
        assert_eq!(build_args.base.block_number, 100);
    }

    #[test]
    fn test_canonical_build_takes_priority_over_speculative() {
        use crate::pending_state::PendingBlockState;
        use reth_execution_types::BlockExecutionOutput;
        use reth_revm::cached::CachedReads;
        use std::sync::Arc;

        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        // Create a flashblock for block 100
        let fb0 = factory.flashblock_at(0).build();
        let parent_hash = fb0.base.as_ref().unwrap().parent_hash;
        manager.insert_flashblock(fb0).unwrap();

        // Create pending parent state with a different block hash
        let pending_parent_hash = B256::random();
        let pending_state: PendingBlockState<OpPrimitives> = PendingBlockState {
            block_hash: B256::repeat_byte(0xAA),
            block_number: 99,
            parent_hash: pending_parent_hash,
            canonical_anchor_hash: pending_parent_hash,
            execution_outcome: Arc::new(BlockExecutionOutput::default()),
            cached_reads: CachedReads::default(),
            sealed_header: None,
        };

        // Local tip matches the sequence parent (canonical mode should take priority)
        let args = manager.next_buildable_args(parent_hash, 1000000, Some(pending_state));
        assert!(args.is_some());
        let build_args = args.unwrap();
        // Should be canonical build (no pending_parent)
        assert!(build_args.pending_parent.is_none());
    }

    // ==================== Reconciliation Cache Clearing Tests ====================

    #[test]
    fn test_catchup_clears_all_cached_sequences() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        // Build up cached sequences for blocks 100, 101, 102
        let fb0 = factory.flashblock_at(0).build();
        manager.insert_flashblock(fb0.clone()).unwrap();

        let fb1 = factory.flashblock_for_next_block(&fb0).build();
        manager.insert_flashblock(fb1.clone()).unwrap();

        let fb2 = factory.flashblock_for_next_block(&fb1).build();
        manager.insert_flashblock(fb2).unwrap();

        // Verify we have cached sequences
        assert_eq!(manager.completed_cache.len(), 2);
        assert!(manager.pending().block_number().is_some());

        // Canonical catches up to 102 - should clear everything
        let canonical = canonical_for(&manager, 102, vec![]);
        let strategy = manager.process_canonical_block(canonical, 10);
        assert_eq!(strategy, ReconciliationStrategy::CatchUp);

        // Verify all state is cleared
        assert!(manager.pending().block_number().is_none());
        assert_eq!(manager.completed_cache.len(), 0);
    }

    #[test]
    fn test_reorg_clears_all_cached_sequences() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        // Build pending sequence for block 100
        let fb0 = factory.flashblock_at(0).build();
        manager.insert_flashblock(fb0.clone()).unwrap();

        // Add another sequence
        let fb1 = factory.flashblock_for_next_block(&fb0).build();
        manager.insert_flashblock(fb1).unwrap();

        // Verify we have state
        assert!(manager.pending().block_number().is_some());
        assert!(!manager.completed_cache.is_empty());

        // Simulate reorg at block 100: canonical has different tx than our cached
        // We need to insert a tx in the sequence to make reorg detection work
        // The reorg detection compares our pending transactions vs canonical
        // Since we have no pending transactions (TestFlashBlockFactory creates empty tx lists),
        // we need to use a different approach - process with tx hashes that don't match empty

        // Actually, let's verify the state clearing on HandleReorg by checking
        // that any non-empty canonical_tx_hashes when we have state triggers reorg
        let canonical_tx_hashes = vec![B256::repeat_byte(0xAA)];
        let canonical = canonical_for(&manager, 100, canonical_tx_hashes);
        let strategy = manager.process_canonical_block(canonical, 10);

        // Should detect reorg (canonical has txs, we have none for that block)
        assert_eq!(strategy, ReconciliationStrategy::HandleReorg);

        // Verify all state is cleared
        assert!(manager.pending().block_number().is_none());
        assert_eq!(manager.completed_cache.len(), 0);
    }

    #[test]
    fn test_depth_limit_exceeded_clears_all_state() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        // Build sequences for blocks 100-102
        let fb0 = factory.flashblock_at(0).build();
        manager.insert_flashblock(fb0.clone()).unwrap();

        let fb1 = factory.flashblock_for_next_block(&fb0).build();
        manager.insert_flashblock(fb1.clone()).unwrap();

        let fb2 = factory.flashblock_for_next_block(&fb1).build();
        manager.insert_flashblock(fb2).unwrap();

        // Verify state exists
        assert_eq!(manager.earliest_block_number(), Some(100));
        assert_eq!(manager.latest_block_number(), Some(102));

        // Canonical at 101 with max_depth of 0 (depth = 101 - 100 = 1 > 0)
        // Since canonical < latest (102), this should trigger depth limit exceeded
        let canonical = canonical_for(&manager, 101, vec![]);
        let strategy = manager.process_canonical_block(canonical, 0);
        assert!(matches!(strategy, ReconciliationStrategy::DepthLimitExceeded { .. }));

        // Verify all state is cleared
        assert!(manager.pending().block_number().is_none());
        assert_eq!(manager.completed_cache.len(), 0);
    }

    #[test]
    fn test_continue_preserves_all_state() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        // Build sequences for blocks 100-102
        let fb0 = factory.flashblock_at(0).build();
        manager.insert_flashblock(fb0.clone()).unwrap();

        let fb1 = factory.flashblock_for_next_block(&fb0).build();
        manager.insert_flashblock(fb1.clone()).unwrap();

        let fb2 = factory.flashblock_for_next_block(&fb1).build();
        manager.insert_flashblock(fb2).unwrap();

        let cached_count = manager.completed_cache.len();

        // Canonical at 99 (behind pending) with reasonable depth limit
        let canonical = canonical_for(&manager, 99, vec![]);
        let strategy = manager.process_canonical_block(canonical, 10);
        assert_eq!(strategy, ReconciliationStrategy::Continue);

        // Verify state is preserved
        assert_eq!(manager.pending().block_number(), Some(102));
        assert_eq!(manager.completed_cache.len(), cached_count);
    }

    #[test]
    fn test_clear_all_removes_pending_and_cache() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        // Build up state
        let fb0 = factory.flashblock_at(0).build();
        manager.insert_flashblock(fb0.clone()).unwrap();

        let fb1 = factory.flashblock_for_next_block(&fb0).build();
        manager.insert_flashblock(fb1).unwrap();

        // Verify state exists
        assert!(manager.pending().block_number().is_some());
        assert!(!manager.completed_cache.is_empty());
        assert!(manager.pending_transaction_count() > 0 || manager.pending().count() > 0);

        // Clear via catchup
        let canonical = canonical_for(&manager, 101, vec![]);
        manager.process_canonical_block(canonical, 10);

        // Verify complete clearing
        assert!(manager.pending().block_number().is_none());
        assert_eq!(manager.pending().count(), 0);
        assert!(manager.completed_cache.is_empty());
        assert_eq!(manager.pending_transaction_count(), 0);
    }

    // ==================== Tracked Fingerprint Tests ====================

    #[test]
    fn test_tracked_fingerprint_returns_none_for_unknown_block() {
        let manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);

        // No flashblocks inserted, should return none
        let fingerprint = manager.tracked_fingerprint_for_block(100);
        assert!(fingerprint.is_none());
    }

    #[test]
    fn test_no_false_reorg_for_untracked_block() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        // Build pending sequence for block 100
        let fb0 = factory.flashblock_at(0).build();
        manager.insert_flashblock(fb0.clone()).unwrap();

        // Add another sequence for block 101
        let fb1 = factory.flashblock_for_next_block(&fb0).build();
        manager.insert_flashblock(fb1).unwrap();

        // Verify we have state for blocks 100 (cached) and 101 (pending)
        assert_eq!(manager.earliest_block_number(), Some(100));
        assert_eq!(manager.latest_block_number(), Some(101));

        // Process canonical block 99 (not tracked) with transactions
        // This should NOT trigger reorg detection because we don't track block 99
        let canonical_tx_hashes = vec![B256::repeat_byte(0xAA)];
        let canonical = canonical_for(&manager, 99, canonical_tx_hashes);
        let strategy = manager.process_canonical_block(canonical, 10);

        // Should continue (not reorg) because block 99 is outside our tracked window
        assert_eq!(strategy, ReconciliationStrategy::Continue);

        // State should be preserved
        assert_eq!(manager.pending().block_number(), Some(101));
        assert!(!manager.completed_cache.is_empty());
    }

    #[test]
    fn test_reorg_detected_for_tracked_block_with_different_txs() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        // Build pending sequence for block 100
        let fb0 = factory.flashblock_at(0).build();
        manager.insert_flashblock(fb0.clone()).unwrap();

        // Add another sequence for block 101
        let fb1 = factory.flashblock_for_next_block(&fb0).build();
        manager.insert_flashblock(fb1).unwrap();

        // Process canonical block 100 (which IS tracked) with different transactions
        // Our tracked block 100 has empty tx list, canonical has non-empty
        let canonical_tx_hashes = vec![B256::repeat_byte(0xAA)];
        let canonical = canonical_for(&manager, 100, canonical_tx_hashes);
        let strategy = manager.process_canonical_block(canonical, 10);

        // Should detect reorg because we track block 100 and txs don't match
        assert_eq!(strategy, ReconciliationStrategy::HandleReorg);

        // State should be cleared
        assert!(manager.pending().block_number().is_none());
        assert!(manager.completed_cache.is_empty());
    }

    #[test]
    fn test_reorg_detected_for_tracked_block_with_parent_hash_mismatch() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        // Build pending sequence for block 100 and cache it by starting block 101.
        let fb0 = factory.flashblock_at(0).build();
        manager.insert_flashblock(fb0.clone()).unwrap();
        let fb1 = factory.flashblock_for_next_block(&fb0).build();
        manager.insert_flashblock(fb1).unwrap();

        let tracked = manager
            .tracked_fingerprint_for_block(100)
            .expect("tracked fingerprint for block 100 should exist");
        let canonical = CanonicalBlockFingerprint {
            block_number: 100,
            block_hash: tracked.block_hash,
            parent_hash: B256::repeat_byte(0xAA), // Different parent hash, identical txs.
            tx_hashes: tracked.tx_hashes,
        };

        let strategy = manager.process_canonical_block(canonical, 10);
        assert_eq!(strategy, ReconciliationStrategy::HandleReorg);
        assert!(manager.pending().block_number().is_none());
        assert!(manager.completed_cache.is_empty());
    }

    #[test]
    fn test_reorg_detected_for_tracked_block_with_block_hash_mismatch() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        // Build pending sequence for block 100 and cache it by starting block 101.
        let fb0 = factory.flashblock_at(0).build();
        manager.insert_flashblock(fb0.clone()).unwrap();
        let fb1 = factory.flashblock_for_next_block(&fb0).build();
        manager.insert_flashblock(fb1).unwrap();

        let tracked = manager
            .tracked_fingerprint_for_block(100)
            .expect("tracked fingerprint for block 100 should exist");
        let canonical = CanonicalBlockFingerprint {
            block_number: 100,
            block_hash: B256::repeat_byte(0xBB), // Different block hash, identical parent+txs.
            parent_hash: tracked.parent_hash,
            tx_hashes: tracked.tx_hashes,
        };

        let strategy = manager.process_canonical_block(canonical, 10);
        assert_eq!(strategy, ReconciliationStrategy::HandleReorg);
        assert!(manager.pending().block_number().is_none());
        assert!(manager.completed_cache.is_empty());
    }

    #[test]
    fn test_tracked_fingerprint_for_pending_block() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        // Create flashblock without transactions (empty tx list is valid)
        let fb0 = factory.flashblock_at(0).build();
        manager.insert_flashblock(fb0).unwrap();

        // Should find tracked fingerprint for block 100
        let fingerprint = manager.tracked_fingerprint_for_block(100);
        assert!(fingerprint.is_some());
        assert!(fingerprint.unwrap().tx_hashes.is_empty()); // No transactions in this flashblock
    }

    #[test]
    fn test_tracked_fingerprint_for_cached_block() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        // Create first flashblock for block 100
        let fb0 = factory.flashblock_at(0).build();
        manager.insert_flashblock(fb0.clone()).unwrap();

        // Create second flashblock for block 101 (caches block 100)
        let fb1 = factory.flashblock_for_next_block(&fb0).build();
        manager.insert_flashblock(fb1).unwrap();

        // Should find tracked fingerprint for cached block 100
        let fingerprint = manager.tracked_fingerprint_for_block(100);
        assert!(fingerprint.is_some());
        assert!(fingerprint.as_ref().unwrap().tx_hashes.is_empty());

        // Should find tracked fingerprint for pending block 101
        let fingerprint = manager.tracked_fingerprint_for_block(101);
        assert!(fingerprint.is_some());
        assert!(fingerprint.as_ref().unwrap().tx_hashes.is_empty());
    }
}
