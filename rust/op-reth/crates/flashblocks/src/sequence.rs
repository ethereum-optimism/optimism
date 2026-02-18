use crate::{FlashBlock, FlashBlockCompleteSequenceRx};
use alloy_primitives::{B256, Bytes};
use alloy_rpc_types_engine::PayloadId;
use core::mem;
use eyre::{OptionExt, bail};
use op_alloy_rpc_types_engine::OpFlashblockPayloadBase;
use reth_revm::cached::CachedReads;
use std::{collections::BTreeMap, ops::Deref};
use tokio::sync::broadcast;
use tracing::*;

/// The size of the broadcast channel for completed flashblock sequences.
const FLASHBLOCK_SEQUENCE_CHANNEL_SIZE: usize = 128;

/// Outcome from executing a flashblock sequence.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[allow(unnameable_types)]
pub struct SequenceExecutionOutcome {
    /// The block hash of the executed pending block
    pub block_hash: B256,
    /// Properly computed state root
    pub state_root: B256,
}

/// An ordered B-tree keeping the track of a sequence of [`FlashBlock`]s by their indices.
#[derive(Debug)]
pub struct FlashBlockPendingSequence {
    /// tracks the individual flashblocks in order
    inner: BTreeMap<u64, FlashBlock>,
    /// Broadcasts flashblocks to subscribers.
    block_broadcaster: broadcast::Sender<FlashBlockCompleteSequence>,
    /// Optional execution outcome from building the current sequence.
    execution_outcome: Option<SequenceExecutionOutcome>,
    /// Cached state reads for the current block.
    /// Current `PendingFlashBlock` is built out of a sequence of `FlashBlocks`, and executed again
    /// when fb received on top of the same block. Avoid redundant I/O across multiple
    /// executions within the same block.
    cached_reads: Option<CachedReads>,
}

impl FlashBlockPendingSequence {
    /// Create a new pending sequence.
    pub fn new() -> Self {
        // Note: if the channel is full, send will not block but rather overwrite the oldest
        // messages. Order is preserved.
        let (tx, _) = broadcast::channel(FLASHBLOCK_SEQUENCE_CHANNEL_SIZE);
        Self {
            inner: BTreeMap::new(),
            block_broadcaster: tx,
            execution_outcome: None,
            cached_reads: None,
        }
    }

    /// Returns the sender half of the [`FlashBlockCompleteSequence`] channel.
    pub const fn block_sequence_broadcaster(
        &self,
    ) -> &broadcast::Sender<FlashBlockCompleteSequence> {
        &self.block_broadcaster
    }

    /// Gets a subscriber to the flashblock sequences produced.
    pub fn subscribe_block_sequence(&self) -> FlashBlockCompleteSequenceRx {
        self.block_broadcaster.subscribe()
    }

    /// Inserts a new block into the sequence.
    ///
    /// A [`FlashBlock`] with index 0 resets the set.
    pub fn insert(&mut self, flashblock: FlashBlock) {
        if flashblock.index == 0 {
            trace!(target: "flashblocks", number=%flashblock.block_number(), "Tracking new flashblock sequence");
            self.inner.insert(flashblock.index, flashblock);
            return;
        }

        // only insert if we previously received the same block and payload, assume we received
        // index 0
        let same_block = self.block_number() == Some(flashblock.block_number());
        let same_payload = self.payload_id() == Some(flashblock.payload_id);

        if same_block && same_payload {
            trace!(target: "flashblocks", number=%flashblock.block_number(), index = %flashblock.index, block_count = self.inner.len()  ,"Received followup flashblock");
            self.inner.insert(flashblock.index, flashblock);
        } else {
            trace!(target: "flashblocks", number=%flashblock.block_number(), index = %flashblock.index, current=?self.block_number()  ,"Ignoring untracked flashblock following");
        }
    }

    /// Set execution outcome from building the flashblock sequence
    pub const fn set_execution_outcome(
        &mut self,
        execution_outcome: Option<SequenceExecutionOutcome>,
    ) {
        self.execution_outcome = execution_outcome;
    }

    /// Set cached reads for this sequence
    pub fn set_cached_reads(&mut self, cached_reads: CachedReads) {
        self.cached_reads = Some(cached_reads);
    }

    /// Removes the cached reads for this sequence
    pub const fn take_cached_reads(&mut self) -> Option<CachedReads> {
        self.cached_reads.take()
    }

    /// Returns the first block number
    pub fn block_number(&self) -> Option<u64> {
        Some(self.inner.values().next()?.block_number())
    }

    /// Returns the payload base of the first tracked flashblock.
    pub fn payload_base(&self) -> Option<OpFlashblockPayloadBase> {
        self.inner.values().next()?.base.clone()
    }

    /// Returns the number of tracked flashblocks.
    pub fn count(&self) -> usize {
        self.inner.len()
    }

    /// Returns the reference to the last flashblock.
    pub fn last_flashblock(&self) -> Option<&FlashBlock> {
        self.inner.last_key_value().map(|(_, b)| b)
    }

    /// Returns the current/latest flashblock index in the sequence
    pub fn index(&self) -> Option<u64> {
        Some(self.inner.values().last()?.index)
    }
    /// Returns the payload id of the first tracked flashblock in the current sequence.
    pub fn payload_id(&self) -> Option<PayloadId> {
        Some(self.inner.values().next()?.payload_id)
    }

    /// Finalizes the current pending sequence and returns it as a complete sequence.
    ///
    /// Clears the internal state and returns an error if the sequence is empty or validation fails.
    pub fn finalize(&mut self) -> eyre::Result<FlashBlockCompleteSequence> {
        if self.inner.is_empty() {
            bail!("Cannot finalize empty flashblock sequence");
        }

        let flashblocks = mem::take(&mut self.inner);
        let execution_outcome = mem::take(&mut self.execution_outcome);
        self.cached_reads = None;

        FlashBlockCompleteSequence::new(flashblocks.into_values().collect(), execution_outcome)
    }

    /// Returns an iterator over all flashblocks in the sequence.
    pub fn flashblocks(&self) -> impl Iterator<Item = &FlashBlock> {
        self.inner.values()
    }
}

impl Default for FlashBlockPendingSequence {
    fn default() -> Self {
        Self::new()
    }
}

/// A complete sequence of flashblocks, often corresponding to a full block.
///
/// Ensures invariants of a complete flashblocks sequence.
/// If this entire sequence of flashblocks was executed on top of latest block, this also includes
/// the execution outcome with block hash and state root.
#[derive(Debug, Clone)]
pub struct FlashBlockCompleteSequence {
    inner: Vec<FlashBlock>,
    /// Optional execution outcome from building the flashblock sequence
    execution_outcome: Option<SequenceExecutionOutcome>,
}

impl FlashBlockCompleteSequence {
    /// Create a complete sequence from a vector of flashblocks.
    /// Ensure that:
    /// * vector is not empty
    /// * first flashblock have the base payload
    /// * sequence of flashblocks is sound (successive index from 0, same payload id, ...)
    pub fn new(
        blocks: Vec<FlashBlock>,
        execution_outcome: Option<SequenceExecutionOutcome>,
    ) -> eyre::Result<Self> {
        let first_block = blocks.first().ok_or_eyre("No flashblocks in sequence")?;

        // Ensure that first flashblock have base
        first_block.base.as_ref().ok_or_eyre("Flashblock at index 0 has no base")?;

        // Ensure that index are successive from 0, have same block number and payload id
        if !blocks.iter().enumerate().all(|(idx, block)| {
            idx == block.index as usize &&
                block.payload_id == first_block.payload_id &&
                block.block_number() == first_block.block_number()
        }) {
            bail!("Flashblock inconsistencies detected in sequence");
        }

        Ok(Self { inner: blocks, execution_outcome })
    }

    /// Returns the block number
    pub fn block_number(&self) -> u64 {
        self.inner.first().unwrap().block_number()
    }

    /// Returns the payload base of the first flashblock.
    pub fn payload_base(&self) -> &OpFlashblockPayloadBase {
        self.inner.first().unwrap().base.as_ref().unwrap()
    }

    /// Returns the number of flashblocks in the sequence.
    pub const fn count(&self) -> usize {
        self.inner.len()
    }

    /// Returns the last flashblock in the sequence.
    pub fn last(&self) -> &FlashBlock {
        self.inner.last().unwrap()
    }

    /// Returns the execution outcome of the sequence.
    pub const fn execution_outcome(&self) -> Option<SequenceExecutionOutcome> {
        self.execution_outcome
    }

    /// Updates execution outcome of the sequence.
    pub const fn set_execution_outcome(
        &mut self,
        execution_outcome: Option<SequenceExecutionOutcome>,
    ) {
        self.execution_outcome = execution_outcome;
    }

    /// Returns all transactions from all flashblocks in the sequence
    pub fn all_transactions(&self) -> Vec<Bytes> {
        self.inner.iter().flat_map(|fb| fb.diff.transactions.iter().cloned()).collect()
    }
}

impl Deref for FlashBlockCompleteSequence {
    type Target = Vec<FlashBlock>;

    fn deref(&self) -> &Self::Target {
        &self.inner
    }
}

impl TryFrom<FlashBlockPendingSequence> for FlashBlockCompleteSequence {
    type Error = eyre::Error;
    fn try_from(sequence: FlashBlockPendingSequence) -> Result<Self, Self::Error> {
        Self::new(sequence.inner.into_values().collect(), sequence.execution_outcome)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_utils::TestFlashBlockFactory;

    mod pending_sequence_insert {
        use super::*;

        #[test]
        fn test_insert_index_zero_creates_new_sequence() {
            let mut sequence = FlashBlockPendingSequence::new();
            let factory = TestFlashBlockFactory::new();
            let fb0 = factory.flashblock_at(0).build();
            let payload_id = fb0.payload_id;

            sequence.insert(fb0);

            assert_eq!(sequence.count(), 1);
            assert_eq!(sequence.block_number(), Some(100));
            assert_eq!(sequence.payload_id(), Some(payload_id));
        }

        #[test]
        fn test_insert_followup_same_block_and_payload() {
            let mut sequence = FlashBlockPendingSequence::new();
            let factory = TestFlashBlockFactory::new();

            let fb0 = factory.flashblock_at(0).build();
            sequence.insert(fb0.clone());

            let fb1 = factory.flashblock_after(&fb0).build();
            sequence.insert(fb1.clone());

            let fb2 = factory.flashblock_after(&fb1).build();
            sequence.insert(fb2);

            assert_eq!(sequence.count(), 3);
            assert_eq!(sequence.index(), Some(2));
        }

        #[test]
        fn test_insert_ignores_different_block_number() {
            let mut sequence = FlashBlockPendingSequence::new();
            let factory = TestFlashBlockFactory::new();

            let fb0 = factory.flashblock_at(0).build();
            sequence.insert(fb0.clone());

            // Try to insert followup with different block number
            let fb1 = factory.flashblock_after(&fb0).block_number(101).build();
            sequence.insert(fb1);

            assert_eq!(sequence.count(), 1);
            assert_eq!(sequence.block_number(), Some(100));
        }

        #[test]
        fn test_insert_ignores_different_payload_id() {
            let mut sequence = FlashBlockPendingSequence::new();
            let factory = TestFlashBlockFactory::new();

            let fb0 = factory.flashblock_at(0).build();
            let payload_id1 = fb0.payload_id;
            sequence.insert(fb0.clone());

            // Try to insert followup with different payload_id
            let payload_id2 = alloy_rpc_types_engine::PayloadId::new([2u8; 8]);
            let fb1 = factory.flashblock_after(&fb0).payload_id(payload_id2).build();
            sequence.insert(fb1);

            assert_eq!(sequence.count(), 1);
            assert_eq!(sequence.payload_id(), Some(payload_id1));
        }

        #[test]
        fn test_insert_maintains_btree_order() {
            let mut sequence = FlashBlockPendingSequence::new();
            let factory = TestFlashBlockFactory::new();

            let fb0 = factory.flashblock_at(0).build();
            sequence.insert(fb0.clone());

            let fb2 = factory.flashblock_after(&fb0).index(2).build();
            sequence.insert(fb2);

            let fb1 = factory.flashblock_after(&fb0).build();
            sequence.insert(fb1);

            let indices: Vec<u64> = sequence.flashblocks().map(|fb| fb.index).collect();
            assert_eq!(indices, vec![0, 1, 2]);
        }
    }

    mod pending_sequence_finalize {
        use super::*;

        #[test]
        fn test_finalize_empty_sequence_fails() {
            let mut sequence = FlashBlockPendingSequence::new();
            let result = sequence.finalize();

            assert!(result.is_err());
            assert_eq!(
                result.unwrap_err().to_string(),
                "Cannot finalize empty flashblock sequence"
            );
        }

        #[test]
        fn test_finalize_clears_pending_state() {
            let mut sequence = FlashBlockPendingSequence::new();
            let factory = TestFlashBlockFactory::new();

            let fb0 = factory.flashblock_at(0).build();
            sequence.insert(fb0);

            assert_eq!(sequence.count(), 1);

            let _complete = sequence.finalize().unwrap();

            // After finalize, sequence should be empty
            assert_eq!(sequence.count(), 0);
            assert_eq!(sequence.block_number(), None);
        }

        #[test]
        fn test_finalize_preserves_execution_outcome() {
            let mut sequence = FlashBlockPendingSequence::new();
            let factory = TestFlashBlockFactory::new();

            let fb0 = factory.flashblock_at(0).build();
            sequence.insert(fb0);

            let outcome =
                SequenceExecutionOutcome { block_hash: B256::random(), state_root: B256::random() };
            sequence.set_execution_outcome(Some(outcome));

            let complete = sequence.finalize().unwrap();

            assert_eq!(complete.execution_outcome(), Some(outcome));
        }

        #[test]
        fn test_finalize_clears_cached_reads() {
            let mut sequence = FlashBlockPendingSequence::new();
            let factory = TestFlashBlockFactory::new();

            let fb0 = factory.flashblock_at(0).build();
            sequence.insert(fb0);

            let cached_reads = CachedReads::default();
            sequence.set_cached_reads(cached_reads);
            assert!(sequence.take_cached_reads().is_some());

            let _complete = sequence.finalize().unwrap();

            // Cached reads should be cleared
            assert!(sequence.take_cached_reads().is_none());
        }

        #[test]
        fn test_finalize_multiple_times_after_refill() {
            let mut sequence = FlashBlockPendingSequence::new();
            let factory = TestFlashBlockFactory::new();

            // First sequence
            let fb0 = factory.flashblock_at(0).build();
            sequence.insert(fb0);

            let complete1 = sequence.finalize().unwrap();
            assert_eq!(complete1.count(), 1);

            // Add new sequence for next block
            let fb1 = factory.flashblock_for_next_block(&complete1.last().clone()).build();
            sequence.insert(fb1);

            let complete2 = sequence.finalize().unwrap();
            assert_eq!(complete2.count(), 1);
            assert_eq!(complete2.block_number(), 101);
        }
    }

    mod complete_sequence_invariants {
        use super::*;

        #[test]
        fn test_new_empty_sequence_fails() {
            let result = FlashBlockCompleteSequence::new(vec![], None);
            assert!(result.is_err());
            assert_eq!(result.unwrap_err().to_string(), "No flashblocks in sequence");
        }

        #[test]
        fn test_new_requires_base_at_index_zero() {
            let factory = TestFlashBlockFactory::new();
            // Use builder() with index 1 first to create a flashblock, then change its index to 0
            // to bypass the auto-base creation logic
            let mut fb0_no_base = factory.flashblock_at(1).build();
            fb0_no_base.index = 0;
            fb0_no_base.base = None;

            let result = FlashBlockCompleteSequence::new(vec![fb0_no_base], None);
            assert!(result.is_err());
            assert_eq!(result.unwrap_err().to_string(), "Flashblock at index 0 has no base");
        }

        #[test]
        fn test_new_validates_successive_indices() {
            let factory = TestFlashBlockFactory::new();

            let fb0 = factory.flashblock_at(0).build();
            // Skip index 1, go straight to 2
            let fb2 = factory.flashblock_after(&fb0).index(2).build();

            let result = FlashBlockCompleteSequence::new(vec![fb0, fb2], None);
            assert!(result.is_err());
            assert_eq!(
                result.unwrap_err().to_string(),
                "Flashblock inconsistencies detected in sequence"
            );
        }

        #[test]
        fn test_new_validates_same_block_number() {
            let factory = TestFlashBlockFactory::new();

            let fb0 = factory.flashblock_at(0).build();
            let fb1 = factory.flashblock_after(&fb0).block_number(101).build();

            let result = FlashBlockCompleteSequence::new(vec![fb0, fb1], None);
            assert!(result.is_err());
            assert_eq!(
                result.unwrap_err().to_string(),
                "Flashblock inconsistencies detected in sequence"
            );
        }

        #[test]
        fn test_new_validates_same_payload_id() {
            let factory = TestFlashBlockFactory::new();

            let fb0 = factory.flashblock_at(0).build();
            let payload_id2 = alloy_rpc_types_engine::PayloadId::new([2u8; 8]);
            let fb1 = factory.flashblock_after(&fb0).payload_id(payload_id2).build();

            let result = FlashBlockCompleteSequence::new(vec![fb0, fb1], None);
            assert!(result.is_err());
            assert_eq!(
                result.unwrap_err().to_string(),
                "Flashblock inconsistencies detected in sequence"
            );
        }

        #[test]
        fn test_new_valid_single_flashblock() {
            let factory = TestFlashBlockFactory::new();
            let fb0 = factory.flashblock_at(0).build();

            let result = FlashBlockCompleteSequence::new(vec![fb0], None);
            assert!(result.is_ok());

            let complete = result.unwrap();
            assert_eq!(complete.count(), 1);
            assert_eq!(complete.block_number(), 100);
        }

        #[test]
        fn test_new_valid_multiple_flashblocks() {
            let factory = TestFlashBlockFactory::new();

            let fb0 = factory.flashblock_at(0).build();
            let fb1 = factory.flashblock_after(&fb0).build();
            let fb2 = factory.flashblock_after(&fb1).build();

            let result = FlashBlockCompleteSequence::new(vec![fb0, fb1, fb2], None);
            assert!(result.is_ok());

            let complete = result.unwrap();
            assert_eq!(complete.count(), 3);
            assert_eq!(complete.last().index, 2);
        }

        #[test]
        fn test_all_transactions_aggregates_correctly() {
            let factory = TestFlashBlockFactory::new();

            let fb0 = factory
                .flashblock_at(0)
                .transactions(vec![Bytes::from_static(&[1, 2, 3]), Bytes::from_static(&[4, 5, 6])])
                .build();

            let fb1 = factory
                .flashblock_after(&fb0)
                .transactions(vec![Bytes::from_static(&[7, 8, 9])])
                .build();

            let complete = FlashBlockCompleteSequence::new(vec![fb0, fb1], None).unwrap();
            let all_txs = complete.all_transactions();

            assert_eq!(all_txs.len(), 3);
            assert_eq!(all_txs[0], Bytes::from_static(&[1, 2, 3]));
            assert_eq!(all_txs[1], Bytes::from_static(&[4, 5, 6]));
            assert_eq!(all_txs[2], Bytes::from_static(&[7, 8, 9]));
        }

        #[test]
        fn test_payload_base_returns_first_block_base() {
            let factory = TestFlashBlockFactory::new();

            let fb0 = factory.flashblock_at(0).build();
            let fb1 = factory.flashblock_after(&fb0).build();

            let complete = FlashBlockCompleteSequence::new(vec![fb0.clone(), fb1], None).unwrap();

            assert_eq!(complete.payload_base().block_number, fb0.base.unwrap().block_number);
        }

        #[test]
        fn test_execution_outcome_mutation() {
            let factory = TestFlashBlockFactory::new();
            let fb0 = factory.flashblock_at(0).build();

            let mut complete = FlashBlockCompleteSequence::new(vec![fb0], None).unwrap();
            assert!(complete.execution_outcome().is_none());

            let outcome =
                SequenceExecutionOutcome { block_hash: B256::random(), state_root: B256::random() };
            complete.set_execution_outcome(Some(outcome));

            assert_eq!(complete.execution_outcome(), Some(outcome));
        }

        #[test]
        fn test_deref_provides_vec_access() {
            let factory = TestFlashBlockFactory::new();

            let fb0 = factory.flashblock_at(0).build();
            let fb1 = factory.flashblock_after(&fb0).build();

            let complete = FlashBlockCompleteSequence::new(vec![fb0, fb1], None).unwrap();

            // Use deref to access Vec methods
            assert_eq!(complete.len(), 2);
            assert!(!complete.is_empty());
        }
    }

    mod sequence_conversion {
        use super::*;

        #[test]
        fn test_try_from_pending_to_complete_valid() {
            let mut pending = FlashBlockPendingSequence::new();
            let factory = TestFlashBlockFactory::new();

            let fb0 = factory.flashblock_at(0).build();
            pending.insert(fb0);

            let complete: Result<FlashBlockCompleteSequence, _> = pending.try_into();
            assert!(complete.is_ok());
            assert_eq!(complete.unwrap().count(), 1);
        }

        #[test]
        fn test_try_from_pending_to_complete_empty_fails() {
            let pending = FlashBlockPendingSequence::new();

            let complete: Result<FlashBlockCompleteSequence, _> = pending.try_into();
            assert!(complete.is_err());
        }

        #[test]
        fn test_try_from_preserves_execution_outcome() {
            let mut pending = FlashBlockPendingSequence::new();
            let factory = TestFlashBlockFactory::new();

            let fb0 = factory.flashblock_at(0).build();
            pending.insert(fb0);

            let outcome =
                SequenceExecutionOutcome { block_hash: B256::random(), state_root: B256::random() };
            pending.set_execution_outcome(Some(outcome));

            let complete: FlashBlockCompleteSequence = pending.try_into().unwrap();
            assert_eq!(complete.execution_outcome(), Some(outcome));
        }
    }

    mod pending_sequence_helpers {
        use super::*;

        #[test]
        fn test_last_flashblock_returns_highest_index() {
            let mut sequence = FlashBlockPendingSequence::new();
            let factory = TestFlashBlockFactory::new();

            let fb0 = factory.flashblock_at(0).build();
            sequence.insert(fb0.clone());

            let fb1 = factory.flashblock_after(&fb0).build();
            sequence.insert(fb1);

            let last = sequence.last_flashblock().unwrap();
            assert_eq!(last.index, 1);
        }

        #[test]
        fn test_subscribe_block_sequence_channel() {
            let sequence = FlashBlockPendingSequence::new();
            let mut rx = sequence.subscribe_block_sequence();

            // Spawn a task that sends a complete sequence
            let tx = sequence.block_sequence_broadcaster().clone();
            std::thread::spawn(move || {
                let factory = TestFlashBlockFactory::new();
                let fb0 = factory.flashblock_at(0).build();
                let complete = FlashBlockCompleteSequence::new(vec![fb0], None).unwrap();
                let _ = tx.send(complete);
            });

            // Should receive the broadcast
            let received = rx.blocking_recv();
            assert!(received.is_ok());
            assert_eq!(received.unwrap().count(), 1);
        }
    }
}
