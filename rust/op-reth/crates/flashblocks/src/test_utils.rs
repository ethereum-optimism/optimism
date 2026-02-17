//! Test utilities for flashblocks.
//!
//! Provides a factory for creating test flashblocks with automatic timestamp management.
//!
//! # Examples
//!
//! ## Simple: Create a flashblock sequence for the same block
//!
//! ```ignore
//! let factory = TestFlashBlockFactory::new(); // Default 2 second block time
//! let fb0 = factory.flashblock_at(0).build();
//! let fb1 = factory.flashblock_after(&fb0).build();
//! let fb2 = factory.flashblock_after(&fb1).build();
//! ```
//!
//! ## Create flashblocks with transactions
//!
//! ```ignore
//! let factory = TestFlashBlockFactory::new();
//! let fb0 = factory.flashblock_at(0).build();
//! let txs = vec![Bytes::from_static(&[1, 2, 3])];
//! let fb1 = factory.flashblock_after(&fb0).transactions(txs).build();
//! ```
//!
//! ## Test across multiple blocks (timestamps auto-increment)
//!
//! ```ignore
//! let factory = TestFlashBlockFactory::new(); // Default 2 second blocks
//!
//! // Block 100 at timestamp 1000000
//! let fb0 = factory.flashblock_at(0).build();
//! let fb1 = factory.flashblock_after(&fb0).build();
//!
//! // Block 101 at timestamp 1000002 (auto-incremented by block_time)
//! let fb2 = factory.flashblock_for_next_block(&fb1).build();
//! let fb3 = factory.flashblock_after(&fb2).build();
//! ```
//!
//! ## Full control with builder
//!
//! ```ignore
//! let factory = TestFlashBlockFactory::new();
//! let fb = factory.builder()
//!     .block_number(100)
//!     .parent_hash(specific_hash)
//!     .state_root(computed_root)
//!     .transactions(txs)
//!     .build();
//! ```

use crate::FlashBlock;
use alloy_primitives::{Address, B256, Bloom, Bytes, U256};
use alloy_rpc_types_engine::PayloadId;
use op_alloy_rpc_types_engine::{
    OpFlashblockPayloadBase, OpFlashblockPayloadDelta, OpFlashblockPayloadMetadata,
};

/// Factory for creating test flashblocks with automatic timestamp management.
///
/// Tracks `block_time` to automatically increment timestamps when creating new blocks.
/// Returns builders that can be further customized before calling `build()`.
///
/// # Examples
///
/// ```ignore
/// let factory = TestFlashBlockFactory::new(); // Default 2 second block time
/// let fb0 = factory.flashblock_at(0).build();
/// let fb1 = factory.flashblock_after(&fb0).build();
/// let fb2 = factory.flashblock_for_next_block(&fb1).build(); // timestamp auto-increments
/// ```
#[derive(Debug)]
pub(crate) struct TestFlashBlockFactory {
    /// Block time in seconds (used to auto-increment timestamps)
    block_time: u64,
    /// Starting timestamp for the first block
    base_timestamp: u64,
    /// Current block number being tracked
    current_block_number: u64,
}

impl TestFlashBlockFactory {
    /// Creates a new factory with a default block time of 2 seconds.
    ///
    /// Use [`with_block_time`](Self::with_block_time) to customize the block time.
    pub(crate) fn new() -> Self {
        Self { block_time: 2, base_timestamp: 1_000_000, current_block_number: 100 }
    }

    pub(crate) fn with_block_time(mut self, block_time: u64) -> Self {
        self.block_time = block_time;
        self
    }

    /// Creates a builder for a flashblock at the specified index (within the current block).
    ///
    /// Returns a builder with index set, allowing further customization before building.
    ///
    /// # Examples
    ///
    /// ```ignore
    /// let factory = TestFlashBlockFactory::new();
    /// let fb0 = factory.flashblock_at(0).build(); // Simple usage
    /// let fb1 = factory.flashblock_at(1).state_root(specific_root).build(); // Customize
    /// ```
    pub(crate) fn flashblock_at(&self, index: u64) -> TestFlashBlockBuilder {
        self.builder().index(index).block_number(self.current_block_number)
    }

    /// Creates a builder for a flashblock following the previous one in the same sequence.
    ///
    /// Automatically increments the index and maintains `block_number` and `payload_id`.
    /// Returns a builder allowing further customization.
    ///
    /// # Examples
    ///
    /// ```ignore
    /// let factory = TestFlashBlockFactory::new();
    /// let fb0 = factory.flashblock_at(0).build();
    /// let fb1 = factory.flashblock_after(&fb0).build(); // Simple
    /// let fb2 = factory.flashblock_after(&fb1).transactions(txs).build(); // With txs
    /// ```
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

    /// Creates a builder for a flashblock for the next block, starting a new sequence at index 0.
    ///
    /// Increments block number, uses previous `block_hash` as `parent_hash`, generates new
    /// `payload_id`, and automatically increments the timestamp by `block_time`.
    /// Returns a builder allowing further customization.
    ///
    /// # Examples
    ///
    /// ```ignore
    /// let factory = TestFlashBlockFactory::new(); // 2 second blocks
    /// let fb0 = factory.flashblock_at(0).build(); // Block 100, timestamp 1000000
    /// let fb1 = factory.flashblock_for_next_block(&fb0).build(); // Block 101, timestamp 1000002
    /// let fb2 = factory.flashblock_for_next_block(&fb1).transactions(txs).build(); // Customize
    /// ```
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

    /// Returns a custom builder for full control over flashblock creation.
    ///
    /// Use this when the convenience methods don't provide enough control.
    ///
    /// # Examples
    ///
    /// ```ignore
    /// let factory = TestFlashBlockFactory::new();
    /// let fb = factory.builder()
    ///     .index(5)
    ///     .block_number(200)
    ///     .parent_hash(specific_hash)
    ///     .state_root(computed_root)
    ///     .build();
    /// ```
    pub(crate) fn builder(&self) -> TestFlashBlockBuilder {
        TestFlashBlockBuilder {
            index: 0,
            block_number: self.current_block_number,
            payload_id: PayloadId::new([1u8; 8]),
            parent_hash: B256::random(),
            timestamp: self.base_timestamp,
            base: None,
            block_hash: B256::random(),
            state_root: B256::ZERO,
            receipts_root: B256::ZERO,
            logs_bloom: Bloom::default(),
            gas_used: 0,
            transactions: vec![],
            withdrawals: vec![],
            withdrawals_root: B256::ZERO,
            blob_gas_used: None,
        }
    }
}

/// Custom builder for creating test flashblocks with full control.
///
/// Created via [`TestFlashBlockFactory::builder()`].
#[derive(Debug)]
pub(crate) struct TestFlashBlockBuilder {
    index: u64,
    block_number: u64,
    payload_id: PayloadId,
    parent_hash: B256,
    timestamp: u64,
    base: Option<OpFlashblockPayloadBase>,
    block_hash: B256,
    state_root: B256,
    receipts_root: B256,
    logs_bloom: Bloom,
    gas_used: u64,
    transactions: Vec<Bytes>,
    withdrawals: Vec<alloy_eips::eip4895::Withdrawal>,
    withdrawals_root: B256,
    blob_gas_used: Option<u64>,
}

impl TestFlashBlockBuilder {
    /// Sets the flashblock index.
    pub(crate) fn index(mut self, index: u64) -> Self {
        self.index = index;
        self
    }

    /// Sets the block number.
    pub(crate) fn block_number(mut self, block_number: u64) -> Self {
        self.block_number = block_number;
        self
    }

    /// Sets the payload ID.
    pub(crate) fn payload_id(mut self, payload_id: PayloadId) -> Self {
        self.payload_id = payload_id;
        self
    }

    /// Sets the parent hash.
    pub(crate) fn parent_hash(mut self, parent_hash: B256) -> Self {
        self.parent_hash = parent_hash;
        self
    }

    /// Sets the timestamp.
    pub(crate) fn timestamp(mut self, timestamp: u64) -> Self {
        self.timestamp = timestamp;
        self
    }

    /// Sets the base payload. Automatically created for index 0 if not set.
    #[allow(dead_code)]
    pub(crate) fn base(mut self, base: OpFlashblockPayloadBase) -> Self {
        self.base = Some(base);
        self
    }

    /// Sets the block hash in the diff.
    #[allow(dead_code)]
    pub(crate) fn block_hash(mut self, block_hash: B256) -> Self {
        self.block_hash = block_hash;
        self
    }

    /// Sets the state root in the diff.
    #[allow(dead_code)]
    pub(crate) fn state_root(mut self, state_root: B256) -> Self {
        self.state_root = state_root;
        self
    }

    /// Sets the receipts root in the diff.
    #[allow(dead_code)]
    pub(crate) fn receipts_root(mut self, receipts_root: B256) -> Self {
        self.receipts_root = receipts_root;
        self
    }

    /// Sets the transactions in the diff.
    pub(crate) fn transactions(mut self, transactions: Vec<Bytes>) -> Self {
        self.transactions = transactions;
        self
    }

    /// Sets the gas used in the diff.
    #[allow(dead_code)]
    pub(crate) fn gas_used(mut self, gas_used: u64) -> Self {
        self.gas_used = gas_used;
        self
    }

    /// Builds the flashblock.
    ///
    /// If index is 0 and no base was explicitly set, creates a default base.
    pub(crate) fn build(mut self) -> FlashBlock {
        // Auto-create base for index 0 if not set
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
                receipts_root: self.receipts_root,
                logs_bloom: self.logs_bloom,
                gas_used: self.gas_used,
                transactions: self.transactions,
                withdrawals: self.withdrawals,
                withdrawals_root: self.withdrawals_root,
                blob_gas_used: self.blob_gas_used,
            },
            metadata: OpFlashblockPayloadMetadata {
                block_number: self.block_number,
                receipts: Default::default(),
                new_account_balances: Default::default(),
            },
        }
    }
}
