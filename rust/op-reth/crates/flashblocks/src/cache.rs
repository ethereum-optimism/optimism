//! Sequence cache management for flashblocks.
//!
//! The `SequenceManager` maintains a ring buffer of recently completed flashblock sequences
//! and intelligently selects which sequence to build based on the local chain tip.

use crate::{
    FlashBlock, FlashBlockCompleteSequence, PendingFlashBlock,
    sequence::{FlashBlockPendingSequence, SequenceExecutionOutcome},
    worker::BuildArgs,
};
use alloy_eips::eip2718::WithEncoded;
use alloy_primitives::B256;
use reth_primitives_traits::{NodePrimitives, Recovered, SignedTransaction};
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
            self.completed_cache.enqueue((completed, txs));

            // ensure cache is wiped on new flashblock
            let _ = self.pending.take_cached_reads();
        }

        self.pending_transactions
            .extend(flashblock.recover_transactions().collect::<Result<Vec<_>, _>>()?);
        self.pending.insert(flashblock);
        Ok(())
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
    ///
    /// Returns None if nothing is buildable right now.
    pub(crate) fn next_buildable_args(
        &mut self,
        local_tip_hash: B256,
        local_tip_timestamp: u64,
    ) -> Option<BuildArgs<Vec<WithEncoded<Recovered<T>>>>> {
        // Try to find a buildable sequence: (base, last_fb, transactions, cached_state,
        // source_name)
        let (base, last_flashblock, transactions, cached_state, source_name) =
            // Priority 1: Try current pending sequence
            if let Some(base) = self.pending.payload_base().filter(|b| b.parent_hash == local_tip_hash) {
                let cached_state = self.pending.take_cached_reads().map(|r| (base.parent_hash, r));
                let last_fb = self.pending.last_flashblock()?;
                let transactions = self.pending_transactions.clone();
                (base, last_fb, transactions, cached_state, "pending")
            }
            // Priority 2: Try cached sequence with exact parent match
            else if let Some((cached, txs)) = self.completed_cache.iter().find(|(c, _)| c.payload_base().parent_hash == local_tip_hash) {
                let base = cached.payload_base().clone();
                let last_fb = cached.last();
                let transactions = txs.clone();
                let cached_state = None;
                (base, last_fb, transactions, cached_state, "cached")
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
            "Building from flashblock sequence"
        );

        Some(BuildArgs {
            base,
            transactions,
            cached_state,
            last_flashblock_index: last_flashblock.index,
            last_flashblock_hash: last_flashblock.diff.block_hash,
            compute_state_root,
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
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_utils::TestFlashBlockFactory;
    use alloy_primitives::B256;
    use op_alloy_consensus::OpTxEnvelope;

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

        let args = manager.next_buildable_args(local_tip_hash, local_tip_timestamp);
        assert!(args.is_none());
    }

    #[test]
    fn test_next_buildable_args_matches_pending_parent() {
        let mut manager: SequenceManager<OpTxEnvelope> = SequenceManager::new(true);
        let factory = TestFlashBlockFactory::new();

        let fb0 = factory.flashblock_at(0).build();
        let parent_hash = fb0.base.as_ref().unwrap().parent_hash;
        manager.insert_flashblock(fb0).unwrap();

        let args = manager.next_buildable_args(parent_hash, 1000000);
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
        let args = manager.next_buildable_args(wrong_parent, 1000000);
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
        let args = manager.next_buildable_args(parent_hash, 1000000);
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
        let args = manager.next_buildable_args(parent_hash, 1000000);
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
        let args = manager.next_buildable_args(parent_hash, base_timestamp - block_time);
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

        let args = manager.next_buildable_args(parent_hash, base_timestamp - block_time);
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
        let args = manager.next_buildable_args(parent_hash, base_timestamp - block_time);
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
        let args = manager.next_buildable_args(first_parent, 1000000);
        // Should not find it (evicted from ring buffer)
        assert!(args.is_none());
    }
}
