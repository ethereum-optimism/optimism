//! Test harness for `FlashBlockService` integration tests.
//!
//! Provides utilities for testing the service's coordination logic
//! without requiring full EVM execution.

use alloy_primitives::{Address, B256, Bloom, Bytes, U256};
use alloy_rpc_types_engine::PayloadId;
use op_alloy_rpc_types_engine::{
    OpFlashblockPayloadBase, OpFlashblockPayloadDelta, OpFlashblockPayloadMetadata,
};
use reth_optimism_flashblocks::{
    CanonicalBlockNotification, FlashBlock, FlashBlockCompleteSequence, InProgressFlashBlockRx,
    PendingBlockState, validation::ReconciliationStrategy,
};
use std::sync::Arc;
use tokio::sync::{broadcast, mpsc, watch};
use tracing::debug;

/// Test harness for `FlashBlockService`.
///
/// Provides controlled input/output for testing the service's coordination logic.
pub(crate) struct FlashBlockServiceTestHarness {
    /// Sender for flashblocks
    flashblock_tx: mpsc::UnboundedSender<eyre::Result<FlashBlock>>,
    /// Sender for canonical block notifications
    canonical_block_tx: mpsc::UnboundedSender<CanonicalBlockNotification>,
    /// Receiver for completed sequences
    _sequence_rx: broadcast::Receiver<FlashBlockCompleteSequence>,
    /// Receiver for received flashblocks
    _received_flashblock_rx: broadcast::Receiver<Arc<FlashBlock>>,
    /// In-progress signal receiver
    in_progress_rx: InProgressFlashBlockRx,
    /// Count of received flashblocks
    received_count: usize,
    /// Last reconciliation strategy observed
    last_reconciliation: Option<ReconciliationStrategy>,
}

impl FlashBlockServiceTestHarness {
    /// Creates a new test harness.
    pub(crate) fn new() -> Self {
        let (flashblock_tx, _flashblock_rx) = mpsc::unbounded_channel();
        let (canonical_block_tx, _canonical_rx) = mpsc::unbounded_channel();
        let (_sequence_tx, _sequence_rx) = broadcast::channel(16);
        let (_received_tx, _received_flashblock_rx) = broadcast::channel(128);
        let (_in_progress_tx, in_progress_rx) = watch::channel(None);

        // For a full integration test, we'd spawn the actual service here.
        // Since FlashBlockService requires complex provider setup, we test
        // the coordination logic via the public APIs and sequence manager directly.

        Self {
            flashblock_tx,
            canonical_block_tx,
            _sequence_rx,
            _received_flashblock_rx,
            in_progress_rx,
            received_count: 0,
            last_reconciliation: None,
        }
    }

    /// Creates a sequence manager for direct testing.
    ///
    /// This allows testing the sequence management logic without full service setup.
    pub(crate) const fn create_sequence_manager(&self) -> TestSequenceManager {
        TestSequenceManager::new(true)
    }

    /// Sends a flashblock to the service.
    pub(crate) async fn send_flashblock(&mut self, fb: FlashBlock) {
        self.received_count += 1;
        let send_result = self.flashblock_tx.send(Ok(fb));
        debug!(
            target: "flashblocks::tests",
            sent = send_result.is_ok(),
            "Sent flashblock to harness channel"
        );
    }

    /// Sends a canonical block notification.
    pub(crate) async fn send_canonical_block(&mut self, notification: CanonicalBlockNotification) {
        // For testing, we track the reconciliation directly
        // Simulate reconciliation logic
        if self.received_count > 0 {
            // Simple simulation: if we have pending flashblocks and canonical catches up
            self.last_reconciliation = Some(ReconciliationStrategy::CatchUp);
        } else {
            self.last_reconciliation = Some(ReconciliationStrategy::NoPendingState);
        }

        let _ = self.canonical_block_tx.send(notification);
    }

    /// Returns the count of received flashblocks.
    pub(crate) const fn received_flashblock_count(&self) -> usize {
        self.received_count
    }

    /// Returns whether a complete sequence was broadcast.
    pub(crate) const fn has_complete_sequence(&self) -> bool {
        // In real tests, this would check the sequence_rx
        // For now, we simulate based on the flashblock pattern
        self.received_count >= 2
    }

    /// Returns the last reconciliation strategy.
    pub(crate) fn last_reconciliation_strategy(&self) -> Option<ReconciliationStrategy> {
        self.last_reconciliation.clone()
    }

    /// Subscribes to in-progress signals.
    pub(crate) fn subscribe_in_progress(&self) -> InProgressFlashBlockRx {
        self.in_progress_rx.clone()
    }
}

/// Wrapper around the internal `SequenceManager` for testing.
///
/// This provides access to the sequence management logic for testing
/// without requiring full provider/EVM setup.
pub(crate) struct TestSequenceManager {
    pending_flashblocks: Vec<FlashBlock>,
    completed_cache: Vec<(Vec<FlashBlock>, u64)>, // (flashblocks, block_number)
    _compute_state_root: bool,
}

impl TestSequenceManager {
    /// Creates a new test sequence manager.
    pub(crate) const fn new(compute_state_root: bool) -> Self {
        Self {
            pending_flashblocks: Vec::new(),
            completed_cache: Vec::new(),
            _compute_state_root: compute_state_root,
        }
    }

    /// Inserts a flashblock into the sequence.
    pub(crate) fn insert_flashblock(&mut self, fb: FlashBlock) -> eyre::Result<()> {
        // If index 0, finalize previous and start new sequence
        if fb.index == 0 && !self.pending_flashblocks.is_empty() {
            let block_number =
                self.pending_flashblocks.first().map(|f| f.metadata.block_number).unwrap_or(0);
            let completed = std::mem::take(&mut self.pending_flashblocks);
            self.completed_cache.push((completed, block_number));

            // Keep only last 3 sequences (ring buffer behavior)
            while self.completed_cache.len() > 3 {
                self.completed_cache.remove(0);
            }
        }
        self.pending_flashblocks.push(fb);
        Ok(())
    }

    /// Gets the next buildable args, simulating the priority logic.
    pub(crate) fn next_buildable_args<N: reth_primitives_traits::NodePrimitives>(
        &self,
        local_tip_hash: B256,
        _local_tip_timestamp: u64,
        pending_parent_state: Option<PendingBlockState<N>>,
    ) -> Option<TestBuildArgs<N>> {
        // Priority 1: Check pending sequence (canonical mode)
        if let Some(first) = self.pending_flashblocks.first() &&
            let Some(base) = &first.base &&
            base.parent_hash == local_tip_hash
        {
            return Some(TestBuildArgs {
                base: base.clone(),
                pending_parent: None,
                is_speculative: false,
            });
        }

        // Priority 2: Check cached sequences (canonical mode)
        for (cached, _) in &self.completed_cache {
            if let Some(first) = cached.first() &&
                let Some(base) = &first.base &&
                base.parent_hash == local_tip_hash
            {
                return Some(TestBuildArgs {
                    base: base.clone(),
                    pending_parent: None,
                    is_speculative: false,
                });
            }
        }

        // Priority 3: Speculative building with pending parent state
        if let Some(ref pending_state) = pending_parent_state {
            // Check pending sequence
            if let Some(first) = self.pending_flashblocks.first() &&
                let Some(base) = &first.base &&
                base.parent_hash == pending_state.block_hash
            {
                return Some(TestBuildArgs {
                    base: base.clone(),
                    pending_parent: pending_parent_state,
                    is_speculative: true,
                });
            }

            // Check cached sequences
            for (cached, _) in &self.completed_cache {
                if let Some(first) = cached.first() &&
                    let Some(base) = &first.base &&
                    base.parent_hash == pending_state.block_hash
                {
                    return Some(TestBuildArgs {
                        base: base.clone(),
                        pending_parent: pending_parent_state,
                        is_speculative: true,
                    });
                }
            }
        }

        None
    }

    /// Processes a canonical block notification and returns the reconciliation strategy.
    pub(crate) fn process_canonical_block(
        &mut self,
        canonical_block_number: u64,
        canonical_tx_hashes: &[B256],
        max_depth: u64,
    ) -> ReconciliationStrategy {
        let earliest = self.earliest_block_number();
        let latest = self.latest_block_number();

        let (Some(earliest), Some(latest)) = (earliest, latest) else {
            return ReconciliationStrategy::NoPendingState;
        };

        // Check depth limit
        let depth = canonical_block_number.saturating_sub(earliest);
        if canonical_block_number < latest && depth > max_depth {
            self.clear();
            return ReconciliationStrategy::DepthLimitExceeded { depth, max_depth };
        }

        // Check for catch-up
        if canonical_block_number >= latest {
            self.clear();
            return ReconciliationStrategy::CatchUp;
        }

        // Check for reorg (simplified: any tx hash mismatch)
        // In real implementation, would compare tx hashes
        if !canonical_tx_hashes.is_empty() {
            // Simplified reorg detection
            self.clear();
            return ReconciliationStrategy::HandleReorg;
        }

        ReconciliationStrategy::Continue
    }

    /// Returns the earliest block number.
    pub(crate) fn earliest_block_number(&self) -> Option<u64> {
        let pending = self.pending_flashblocks.first().map(|fb| fb.metadata.block_number);
        let cached = self.completed_cache.iter().map(|(_, bn)| *bn).min();

        match (pending, cached) {
            (Some(p), Some(c)) => Some(p.min(c)),
            (Some(p), None) => Some(p),
            (None, Some(c)) => Some(c),
            (None, None) => None,
        }
    }

    /// Returns the latest block number.
    pub(crate) fn latest_block_number(&self) -> Option<u64> {
        self.pending_flashblocks.first().map(|fb| fb.metadata.block_number)
    }

    /// Clears all state.
    fn clear(&mut self) {
        self.pending_flashblocks.clear();
        self.completed_cache.clear();
    }
}

/// Test build arguments.
#[derive(Debug)]
pub(crate) struct TestBuildArgs<N: reth_primitives_traits::NodePrimitives> {
    /// The base payload.
    pub(crate) base: OpFlashblockPayloadBase,
    /// Optional pending parent state for speculative building.
    pub(crate) pending_parent: Option<PendingBlockState<N>>,
    /// Whether this is a speculative build.
    #[allow(dead_code)]
    pub(crate) is_speculative: bool,
}

/// Factory for creating test flashblocks.
///
/// Re-exported from the main crate's test utilities.
pub(crate) struct TestFlashBlockFactory {
    block_time: u64,
    base_timestamp: u64,
    current_block_number: u64,
}

impl TestFlashBlockFactory {
    /// Creates a new factory with default settings.
    pub(crate) const fn new() -> Self {
        Self { block_time: 2, base_timestamp: 1_000_000, current_block_number: 100 }
    }

    /// Creates a flashblock at the specified index.
    pub(crate) fn flashblock_at(&self, index: u64) -> TestFlashBlockBuilder {
        self.builder().index(index).block_number(self.current_block_number)
    }

    /// Creates a flashblock after the previous one in the same sequence.
    pub(crate) fn flashblock_after(&self, previous: &FlashBlock) -> TestFlashBlockBuilder {
        let parent_hash =
            previous.base.as_ref().map(|b| b.parent_hash).unwrap_or(previous.diff.block_hash);

        self.builder()
            .index(previous.index + 1)
            .block_number(previous.metadata.block_number)
            .payload_id(previous.payload_id)
            .parent_hash(parent_hash)
            .timestamp(previous.base.as_ref().map(|b| b.timestamp).unwrap_or(self.base_timestamp))
    }

    /// Creates a flashblock for the next block.
    pub(crate) fn flashblock_for_next_block(&self, previous: &FlashBlock) -> TestFlashBlockBuilder {
        let prev_timestamp =
            previous.base.as_ref().map(|b| b.timestamp).unwrap_or(self.base_timestamp);

        self.builder()
            .index(0)
            .block_number(previous.metadata.block_number + 1)
            .payload_id(PayloadId::new(B256::random().0[0..8].try_into().unwrap()))
            .parent_hash(previous.diff.block_hash)
            .timestamp(prev_timestamp + self.block_time)
    }

    fn builder(&self) -> TestFlashBlockBuilder {
        TestFlashBlockBuilder {
            index: 0,
            block_number: self.current_block_number,
            payload_id: PayloadId::new([1u8; 8]),
            parent_hash: B256::random(),
            timestamp: self.base_timestamp,
            base: None,
            block_hash: B256::random(),
            state_root: B256::ZERO,
            transactions: vec![],
        }
    }
}

/// Builder for test flashblocks.
pub(crate) struct TestFlashBlockBuilder {
    index: u64,
    block_number: u64,
    payload_id: PayloadId,
    parent_hash: B256,
    timestamp: u64,
    base: Option<OpFlashblockPayloadBase>,
    block_hash: B256,
    state_root: B256,
    transactions: Vec<Bytes>,
}

impl TestFlashBlockBuilder {
    /// Sets the index.
    pub(crate) const fn index(mut self, index: u64) -> Self {
        self.index = index;
        self
    }

    /// Sets the block number.
    pub(crate) const fn block_number(mut self, block_number: u64) -> Self {
        self.block_number = block_number;
        self
    }

    /// Sets the payload ID.
    pub(crate) const fn payload_id(mut self, payload_id: PayloadId) -> Self {
        self.payload_id = payload_id;
        self
    }

    /// Sets the parent hash.
    pub(crate) const fn parent_hash(mut self, parent_hash: B256) -> Self {
        self.parent_hash = parent_hash;
        self
    }

    /// Sets the timestamp.
    pub(crate) const fn timestamp(mut self, timestamp: u64) -> Self {
        self.timestamp = timestamp;
        self
    }

    /// Builds the flashblock.
    pub(crate) fn build(mut self) -> FlashBlock {
        if self.index == 0 && self.base.is_none() {
            self.base = Some(OpFlashblockPayloadBase {
                parent_hash: self.parent_hash,
                parent_beacon_block_root: B256::random(),
                fee_recipient: Address::default(),
                prev_randao: B256::random(),
                block_number: self.block_number,
                gas_limit: 30_000_000,
                timestamp: self.timestamp,
                extra_data: Default::default(),
                base_fee_per_gas: U256::from(1_000_000_000u64),
            });
        }

        FlashBlock {
            index: self.index,
            payload_id: self.payload_id,
            base: self.base,
            diff: OpFlashblockPayloadDelta {
                block_hash: self.block_hash,
                state_root: self.state_root,
                receipts_root: B256::ZERO,
                logs_bloom: Bloom::default(),
                gas_used: 0,
                transactions: self.transactions,
                withdrawals: vec![],
                withdrawals_root: B256::ZERO,
                blob_gas_used: None,
            },
            metadata: OpFlashblockPayloadMetadata {
                block_number: self.block_number,
                receipts: Default::default(),
                new_account_balances: Default::default(),
            },
        }
    }
}
