//! Mock L1 chain for testing the derivation pipeline.
//!
//! Provides a deterministic in-memory L1 chain that implements [`ChainProvider`]
//! and [`BlobProvider`] from `kona-derive`.

mod block;
mod blob_store;

pub use block::*;
pub use blob_store::*;

use alloy_consensus::{Header, Receipt, TxEnvelope};
use alloy_eips::eip4844::Blob;
use alloy_primitives::{B256, map::HashMap};
use async_trait::async_trait;
use kona_derive::{BlobProvider, ChainProvider, PipelineError, PipelineErrorKind};
use kona_protocol::BlockInfo;
use std::sync::{Arc, RwLock};

/// Error type for the mock L1 chain.
#[derive(Debug, thiserror::Error)]
pub enum MockL1Error {
    /// Block not found.
    #[error("Block not found: {0}")]
    BlockNotFound(u64),
    /// Header not found.
    #[error("Header not found: {0}")]
    HeaderNotFound(B256),
    /// Receipts not found.
    #[error("Receipts not found: {0}")]
    ReceiptsNotFound(B256),
    /// Blob not found.
    #[error("Blob not found: {0}")]
    BlobNotFound(B256),
}

impl From<MockL1Error> for PipelineErrorKind {
    fn from(e: MockL1Error) -> Self {
        PipelineError::Provider(e.to_string()).temp()
    }
}

/// Inner state of the mock L1 chain, behind a RwLock for shared access.
#[derive(Debug, Default)]
struct MockL1Inner {
    /// Ordered chain of L1 blocks.
    blocks: Vec<L1Block>,
    /// Blob store: versioned_hash -> blob.
    blob_store: HashMap<B256, Box<Blob>>,
    /// Safe head number.
    safe_number: u64,
    /// Finalized head number.
    finalized_number: u64,
    /// Pending block state for building.
    pending_txs: Vec<TxEnvelope>,
    pending_receipts: Vec<Receipt>,
}

/// A deterministic in-memory L1 chain.
///
/// Implements [`ChainProvider`] and [`BlobProvider`] for use in the derivation pipeline.
/// All access is synchronized via interior mutability (RwLock).
#[derive(Debug, Clone, Default)]
pub struct MockL1Chain {
    inner: Arc<RwLock<MockL1Inner>>,
}

impl MockL1Chain {
    /// Creates a new [`MockL1Chain`] with a genesis block.
    pub fn new(genesis: BlockInfo) -> Self {
        let genesis_block = L1Block::genesis(genesis);
        let chain = Self::default();
        {
            let mut inner = chain.inner.write().unwrap();
            inner.blocks.push(genesis_block);
        }
        chain
    }

    /// Starts building a new block with the given timestamp delta from the parent.
    pub fn act_start_block(&self, _timestamp_delta: u64) {
        let mut inner = self.inner.write().unwrap();
        inner.pending_txs.clear();
        inner.pending_receipts.clear();
    }

    /// Includes a transaction in the pending block.
    pub fn act_include_tx(&self, tx: TxEnvelope) {
        let mut inner = self.inner.write().unwrap();
        // Create a simple receipt for this tx
        let receipt = Receipt {
            status: alloy_consensus::Eip658Value::Eip658(true),
            cumulative_gas_used: 21_000 * (inner.pending_receipts.len() as u64 + 1),
            logs: Vec::new(),
        };
        inner.pending_txs.push(tx);
        inner.pending_receipts.push(receipt);
    }

    /// Finalizes the pending block and appends it to the chain.
    /// Returns the [`BlockInfo`] for the new block.
    pub fn act_end_block(&self, timestamp_delta: u64) -> BlockInfo {
        let mut inner = self.inner.write().unwrap();
        let parent = inner.blocks.last().expect("chain must have genesis");
        let parent_info = parent.info;
        let number = parent_info.number + 1;
        let timestamp = parent_info.timestamp + timestamp_delta;

        let txs = std::mem::take(&mut inner.pending_txs);
        let receipts = std::mem::take(&mut inner.pending_receipts);

        let block = L1Block::new(number, timestamp, parent_info.hash, txs, receipts);
        let info = block.info;
        inner.blocks.push(block);
        info
    }

    /// Advances the safe head by one block.
    pub fn act_safe_next(&self) {
        let mut inner = self.inner.write().unwrap();
        let max = inner.blocks.last().map(|b| b.info.number).unwrap_or(0);
        if inner.safe_number < max {
            inner.safe_number += 1;
        }
    }

    /// Advances the finalized head by one block.
    pub fn act_finalize_next(&self) {
        let mut inner = self.inner.write().unwrap();
        if inner.finalized_number < inner.safe_number {
            inner.finalized_number += 1;
        }
    }

    /// Stores a blob in the blob store.
    pub fn store_blob(&self, versioned_hash: B256, blob: Box<Blob>) {
        let mut inner = self.inner.write().unwrap();
        inner.blob_store.insert(versioned_hash, blob);
    }

    /// Returns the latest block info.
    pub fn latest(&self) -> BlockInfo {
        let inner = self.inner.read().unwrap();
        inner.blocks.last().map(|b| b.info).unwrap_or_default()
    }

    /// Returns the safe head block info.
    pub fn safe_head(&self) -> BlockInfo {
        let inner = self.inner.read().unwrap();
        inner
            .blocks
            .get(inner.safe_number as usize)
            .map(|b| b.info)
            .unwrap_or_default()
    }

    /// Returns the finalized head block info.
    pub fn finalized_head(&self) -> BlockInfo {
        let inner = self.inner.read().unwrap();
        inner
            .blocks
            .get(inner.finalized_number as usize)
            .map(|b| b.info)
            .unwrap_or_default()
    }

    /// Returns the number of blocks in the chain (including genesis).
    pub fn len(&self) -> usize {
        let inner = self.inner.read().unwrap();
        inner.blocks.len()
    }

    /// Returns true if the chain only has genesis.
    pub fn is_empty(&self) -> bool {
        self.len() <= 1
    }

    /// Returns block info by number.
    pub fn block_info_at(&self, number: u64) -> Option<BlockInfo> {
        let inner = self.inner.read().unwrap();
        inner.blocks.get(number as usize).map(|b| b.info)
    }
}

#[async_trait]
impl ChainProvider for MockL1Chain {
    type Error = MockL1Error;

    async fn header_by_hash(&mut self, hash: B256) -> Result<Header, Self::Error> {
        let inner = self.inner.read().unwrap();
        inner
            .blocks
            .iter()
            .find(|b| b.info.hash == hash)
            .map(|b| b.header.clone())
            .ok_or(MockL1Error::HeaderNotFound(hash))
    }

    async fn block_info_by_number(&mut self, number: u64) -> Result<BlockInfo, Self::Error> {
        let inner = self.inner.read().unwrap();
        inner
            .blocks
            .get(number as usize)
            .map(|b| b.info)
            .ok_or(MockL1Error::BlockNotFound(number))
    }

    async fn receipts_by_hash(&mut self, hash: B256) -> Result<Vec<Receipt>, Self::Error> {
        let inner = self.inner.read().unwrap();
        inner
            .blocks
            .iter()
            .find(|b| b.info.hash == hash)
            .map(|b| b.receipts.clone())
            .ok_or(MockL1Error::ReceiptsNotFound(hash))
    }

    async fn block_info_and_transactions_by_hash(
        &mut self,
        hash: B256,
    ) -> Result<(BlockInfo, Vec<TxEnvelope>), Self::Error> {
        let inner = self.inner.read().unwrap();
        inner
            .blocks
            .iter()
            .find(|b| b.info.hash == hash)
            .map(|b| (b.info, b.transactions.clone()))
            .ok_or(MockL1Error::BlockNotFound(0))
    }
}

#[async_trait]
impl BlobProvider for MockL1Chain {
    type Error = MockL1Error;

    async fn get_and_validate_blobs(
        &mut self,
        _block_ref: &BlockInfo,
        blob_hashes: &[B256],
    ) -> Result<Vec<Box<Blob>>, Self::Error> {
        let inner = self.inner.read().unwrap();
        let mut blobs = Vec::with_capacity(blob_hashes.len());
        for hash in blob_hashes {
            let blob = inner
                .blob_store
                .get(hash)
                .cloned()
                .ok_or(MockL1Error::BlobNotFound(*hash))?;
            blobs.push(blob);
        }
        Ok(blobs)
    }
}
