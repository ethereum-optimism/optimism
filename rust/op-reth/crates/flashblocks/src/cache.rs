//! Sequence cache management for flashblocks.
//!
//! The `SequenceManager` maintains a ring buffer of recently completed flashblock sequences
//! and intelligently selects which sequence to build based on the local chain tip.

use crate::{
    FlashBlock, FlashBlockCompleteSequence, PendingFlashBlock,
    pending_state::PendingBlockState,
    sequence::{FlashBlockPendingSequence, SequenceExecutionOutcome},
    validation::{CanonicalBlockReconciler, ReconciliationStrategy, ReorgDetector},
    worker::BuildArgs,
};
use alloy_eips::eip2718::WithEncoded;
use alloy_primitives::B256;
use reth_primitives_traits::{
    NodePrimitives, Recovered, SignedTransaction, transaction::TxHashRef,
};
use reth_revm::cached::CachedReads;
use ringbuffer::{AllocRingBuffer, RingBuffer};
use tokio::sync::broadcast;
use tracing::*;

/// Maximum number of cached sequences in the ring buffer.
const CACHE_SIZE: usize = 3;
/// 200 ms flashblock time.
pub(crate) const FLASHBLOCK_BLOCK_TIME: u64 = 200;

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
    pending: FlashBlockPendingSequence,
    /// Cached recovered transactions for the pending sequence
    pending_transactions: Vec<WithEncoded<Recovered<T>>>,
    /// Ring buffer of recently completed sequences bundled with their decoded transactions (FIFO,
    /// size 3)
    completed_cache: AllocRingBuffer<(FlashBlockCompleteSequence, Vec<WithEncoded<Recovered<T>>>)>,
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
            pending: FlashBlockPendingSequence::new(),
            pending_transactions: Vec::new(),
            completed_cache: AllocRingBuffer::new(CACHE_SIZE),
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
            let completed = self.pending.finalize()?;
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
            let txs = std::mem::take(&mut self.pending_transactions);
            self.push_completed_sequence(completed, txs);

            // ensure cache is wiped on new flashblock
            let _ = self.pending.take_cached_reads();
        }

        self.pending_transactions
            .extend(flashblock.recover_transactions().collect::<Result<Vec<_>, _>>()?);
        self.pending.insert(flashblock);
        Ok(())
    }

    /// Pushes a completed sequence into the cache and maintains cached min block-number metadata.
    fn push_completed_sequence(
        &mut self,
        completed: FlashBlockCompleteSequence,
        txs: Vec<WithEncoded<Recovered<T>>>,
    ) {
        let block_number = completed.block_number();
        let evicted_block_number = if self.completed_cache.is_full() {
            self.completed_cache.front().map(|(seq, _)| seq.block_number())
        } else {
            None
        };

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

    /// Returns the current pending sequence for inspection.
    pub(crate) const fn pending(&self) -> &FlashBlockPendingSequence {
        &self.pending
    }

    /// Finds the next sequence to build and returns ready-to-use `BuildArgs`.
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
    ) -> Option<BuildArgs<Vec<WithEncoded<Recovered<T>>>, N>> {
        // Try to find a buildable sequence: (base, last_fb, transactions, cached_state,
        // source_name, pending_parent)
        let (base, last_flashblock, transactions, cached_state, source_name, pending_parent) =
            // Priority 1: Try current pending sequence (canonical mode)
            if let Some(base) = self.pending.payload_base().filter(|b| b.parent_hash == local_tip_hash) {
                let cached_state = self.pending.take_cached_reads().map(|r| (base.parent_hash, r));
                let last_fb = self.pending.last_flashblock()?;
                let transactions = self.pending_transactions.clone();
                (base, last_fb, transactions, cached_state, "pending", None)
            }
            // Priority 2: Try cached sequence with exact parent match (canonical mode)
            else if let Some((cached, txs)) = self.completed_cache.iter().find(|(c, _)| c.payload_base().parent_hash == local_tip_hash) {
                let base = cached.payload_base().clone();
                let last_fb = cached.last();
                let transactions = txs.clone();
                let cached_state = None;
                (base, last_fb, transactions, cached_state, "cached", None)
            }
            // Priority 3: Try speculative building with pending parent state
            else if let Some(ref pending_state) = pending_parent_state {
                // Check if pending sequence's parent matches the pending state's block
                if let Some(base) = self.pending.payload_base().filter(|b| b.parent_hash == pending_state.block_hash) {
                    let cached_state = self.pending.take_cached_reads().map(|r| (base.parent_hash, r));
                    let last_fb = self.pending.last_flashblock()?;
                    let transactions = self.pending_transactions.clone();
                    (base, last_fb, transactions, cached_state, "speculative-pending", pending_parent_state)
                }
                // Check cached sequences
                else if let Some((cached, txs)) = self.completed_cache.iter().find(|(c, _)| c.payload_base().parent_hash == pending_state.block_hash) {
                    let base = cached.payload_base().clone();
                    let last_fb = cached.last();
                    let transactions = txs.clone();
                    let cached_state = None;
                    (base, last_fb, transactions, cached_state, "speculative-cached", pending_parent_state)
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
        let block_time_ms = (base.timestamp - local_tip_timestamp) * 1000;
        let expected_final_flashblock = block_time_ms / FLASHBLOCK_BLOCK_TIME;
        let compute_state_root = self.compute_state_root &&
            last_flashblock.diff.state_root.is_zero() &&
            last_flashblock.index >= expected_final_flashblock.saturating_sub(1);

        trace!(
            target: "flashblocks",
            block_number = base.block_number,
            source = source_name,
            flashblock_index = last_flashblock.index,
            expected_final_flashblock,
            compute_state_root_enabled = self.compute_state_root,
            state_root_is_zero = last_flashblock.diff.state_root.is_zero(),
            will_compute_state_root = compute_state_root,
            is_speculative = pending_parent.is_some(),
            "Building from flashblock sequence"
        );

        Some(BuildArgs {
            base,
            transactions,
            cached_state,
            last_flashblock_index: last_flashblock.index,
            last_flashblock_hash: last_flashblock.diff.block_hash,
            compute_state_root,
            pending_parent,
        })
    }

    /// Records the result of building a sequence and re-broadcasts with execution outcome.
    ///
    /// Updates execution outcome and cached reads. For cached sequences (already broadcast
    /// once during finalize), this broadcasts again with the computed `state_root`, allowing
    /// the consensus client to submit via `engine_newPayload`.
    pub(crate) fn on_build_complete<N: NodePrimitives>(
        &mut self,
        parent_hash: B256,
        result: Option<(PendingFlashBlock<N>, CachedReads)>,
    ) {
        let Some((computed_block, cached_reads)) = result else {
            return;
        };

        // Extract execution outcome
        let execution_outcome = computed_block.computed_state_root().map(|state_root| {
            SequenceExecutionOutcome { block_hash: computed_block.block().hash(), state_root }
        });

        // Update pending sequence with execution results
        if self.pending.payload_base().is_some_and(|base| base.parent_hash == parent_hash) {
            self.pending.set_execution_outcome(execution_outcome);
            self.pending.set_cached_reads(cached_reads);
            trace!(
                target: "flashblocks",
                block_number = self.pending.block_number(),
                has_computed_state_root = execution_outcome.is_some(),
                "Updated pending sequence with build results"
            );
        }
        // Check if this completed sequence in cache and broadcast with execution outcome
        else if let Some((cached, _)) = self
            .completed_cache
            .iter_mut()
            .find(|(c, _)| c.payload_base().parent_hash == parent_hash)
        {
            // Only re-broadcast if we computed new information (state_root was missing).
            // If sequencer already provided state_root, we already broadcast in insert_flashblock,
            // so skip re-broadcast to avoid duplicate FCU calls.
            let needs_rebroadcast =
                execution_outcome.is_some() && cached.execution_outcome().is_none();

            cached.set_execution_outcome(execution_outcome);

            if needs_rebroadcast && self.block_broadcaster.receiver_count() > 0 {
                trace!(
                    target: "flashblocks",
                    block_number = cached.block_number(),
                    "Re-broadcasting sequence with computed state_root"
                );
                let _ = self.block_broadcaster.send(cached.clone());
            }
        }
    }

    /// Returns the earliest block number in the pending or cached sequences.
    pub(crate) fn earliest_block_number(&self) -> Option<u64> {
        match (self.pending.block_number(), self.cached_min_block_number) {
            (Some(pending_block), Some(cache_min)) => Some(cache_min.min(pending_block)),
            (Some(pending_block), None) => Some(pending_block),
            (None, Some(cache_min)) => Some(cache_min),
            (None, None) => None,
        }
    }

    /// Returns the latest block number in the pending or cached sequences.
    pub(crate) fn latest_block_number(&self) -> Option<u64> {
        // Pending is always the latest if it exists
        if let Some(pending_block) = self.pending.block_number() {
            return Some(pending_block);
        }

        // Fall back to cache
        self.completed_cache.iter().map(|(seq, _)| seq.block_number()).max()
    }

    /// Returns transaction hashes for a specific block number from pending or cached sequences.
    pub(crate) fn get_transaction_hashes_for_block(&self, block_number: u64) -> Vec<B256> {
        // Check pending sequence
        if self.pending.block_number() == Some(block_number) {
            return self.pending_transactions.iter().map(|tx| *tx.tx_hash()).collect();
        }

        // Check cached sequences
        for (seq, txs) in self.completed_cache.iter() {
            if seq.block_number() == block_number {
                return txs.iter().map(|tx| *tx.tx_hash()).collect();
            }
        }

        Vec::new()
    }

    /// Returns true if the given block number is tracked in pending or cached sequences.
    fn tracks_block_number(&self, block_number: u64) -> bool {
        // Check pending sequence
        if self.pending.block_number() == Some(block_number) {
            return true;
        }

        // Check cached sequences
        self.completed_cache.iter().any(|(seq, _)| seq.block_number() == block_number)
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
        canonical_block_number: u64,
        canonical_tx_hashes: &[B256],
        max_depth: u64,
    ) -> ReconciliationStrategy {
        let earliest = self.earliest_block_number();
        let latest = self.latest_block_number();

        // Only run reorg detection if we actually track the canonical block number.
        // If we don't track it (block number outside our pending/cached window),
        // comparing empty tracked hashes to non-empty canonical hashes would falsely
        // trigger reorg detection.
        let reorg_detected = if self.tracks_block_number(canonical_block_number) {
            let tracked_tx_hashes = self.get_transaction_hashes_for_block(canonical_block_number);
            let reorg_result = ReorgDetector::detect(&tracked_tx_hashes, canonical_tx_hashes);
            reorg_result.is_reorg()
        } else {
            false
        };

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
                    canonical_tx_count = canonical_tx_hashes.len(),
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
        self.pending = FlashBlockPendingSequence::new();
        self.pending_transactions.clear();
        self.completed_cache.clear();
        self.cached_min_block_number = None;
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{test_utils::TestFlashBlockFactory, validation::ReconciliationStrategy};
    use alloy_primitives::B256;
    use op_alloy_consensus::OpTxEnvelope;
    use reth_optimism_primitives::OpPrimitives;

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
        let strategy = manager.process_canonical_block(100, &[], 10);
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
        let strategy = manager.process_canonical_block(100, &[], 10);
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
        let strategy = manager.process_canonical_block(99, &[], 10);
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
        let strategy = manager.process_canonical_block(101, &[], 0);
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
        let strategy = manager.process_canonical_block(102, &[], 10);
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
        let strategy = manager.process_canonical_block(100, &canonical_tx_hashes, 10);

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
        let strategy = manager.process_canonical_block(101, &[], 0);
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
        let strategy = manager.process_canonical_block(99, &[], 10);
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
        assert!(!manager.pending_transactions.is_empty() || manager.pending().count() > 0);

        // Clear via catchup
        manager.process_canonical_block(101, &[], 10);

        // Verify complete clearing
        assert!(manager.pending().block_number().is_none());
        assert_eq!(manager.pending().count(), 0);
        assert!(manager.completed_cache.is_empty());
        assert!(manager.pending_transactions.is_empty());
    }

    // ==================== Transaction Hash Tracking Tests ====================

    #[test]
    fn test_get_transaction_hashes_returns_empty_for_unknown_block() {
        let manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);

        // No flashblocks inserted, should return empty
        let hashes = manager.get_transaction_hashes_for_block(100);
        assert!(hashes.is_empty());
    }

    #[test]
    fn test_get_transaction_hashes_for_pending_block() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        // Create flashblock without transactions (empty tx list is valid)
        let fb0 = factory.flashblock_at(0).build();
        manager.insert_flashblock(fb0).unwrap();

        // Should find (empty) transaction hashes for block 100
        let hashes = manager.get_transaction_hashes_for_block(100);
        assert!(hashes.is_empty()); // No transactions in this flashblock
    }

    #[test]
    fn test_get_transaction_hashes_for_cached_block() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        // Create first flashblock for block 100
        let fb0 = factory.flashblock_at(0).build();
        manager.insert_flashblock(fb0.clone()).unwrap();

        // Create second flashblock for block 101 (caches block 100)
        let fb1 = factory.flashblock_for_next_block(&fb0).build();
        manager.insert_flashblock(fb1).unwrap();

        // Should find transaction hashes for cached block 100
        let hashes = manager.get_transaction_hashes_for_block(100);
        assert!(hashes.is_empty()); // No transactions in these flashblocks

        // Should find transaction hashes for pending block 101
        let hashes = manager.get_transaction_hashes_for_block(101);
        assert!(hashes.is_empty()); // No transactions in these flashblocks
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
        let strategy = manager.process_canonical_block(99, &canonical_tx_hashes, 10);

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
        let strategy = manager.process_canonical_block(100, &canonical_tx_hashes, 10);

        // Should detect reorg because we track block 100 and txs don't match
        assert_eq!(strategy, ReconciliationStrategy::HandleReorg);

        // State should be cleared
        assert!(manager.pending().block_number().is_none());
        assert!(manager.completed_cache.is_empty());
    }
}
