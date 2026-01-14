//! Test Utilities for chain provider traits

use crate::{
    errors::{PipelineError, PipelineErrorKind},
    traits::{ChainProvider, L2ChainProvider},
};
use alloc::{boxed::Box, string::ToString, sync::Arc, vec::Vec};
use alloy_consensus::{Header, Receipt, TxEnvelope};
use alloy_primitives::{B256, map::HashMap};
use async_trait::async_trait;
use kona_genesis::{RollupConfig, SystemConfig};
use kona_protocol::{BatchValidationProvider, BlockInfo, L2BlockInfo};
use op_alloy_consensus::OpBlock;
use thiserror::Error;

/// A mock chain provider for testing.
#[derive(Debug, Clone, Default)]
pub struct TestChainProvider {
    /// Maps block numbers to block information using a tuple list.
    pub blocks: Vec<(u64, BlockInfo)>,
    /// Maps block hashes to header information using a tuple list.
    pub headers: Vec<(B256, Header)>,
    /// Maps block hashes to receipts using a tuple list.
    pub receipts: Vec<(B256, Vec<Receipt>)>,
    /// Maps block hashes to transactions using a tuple list.
    pub transactions: Vec<(B256, Vec<TxEnvelope>)>,
}

impl TestChainProvider {
    /// Insert a block into the mock chain provider.
    pub fn insert_block(&mut self, number: u64, block: BlockInfo) {
        self.blocks.push((number, block));
    }

    /// Insert a block with transactions into the mock chain provider.
    pub fn insert_block_with_transactions(
        &mut self,
        number: u64,
        block: BlockInfo,
        txs: Vec<TxEnvelope>,
    ) {
        self.blocks.push((number, block));
        self.transactions.push((block.hash, txs));
    }

    /// Insert receipts into the mock chain provider.
    pub fn insert_receipts(&mut self, hash: B256, receipts: Vec<Receipt>) {
        self.receipts.push((hash, receipts));
    }

    /// Insert a header into the mock chain provider.
    pub fn insert_header(&mut self, hash: B256, header: Header) {
        self.headers.push((hash, header));
    }

    /// Clears headers from the mock chain provider.
    pub fn clear_headers(&mut self) {
        self.headers.clear();
    }

    /// Clears blocks from the mock chain provider.
    pub fn clear_blocks(&mut self) {
        self.blocks.clear();
    }

    /// Clears receipts from the mock chain provider.
    pub fn clear_receipts(&mut self) {
        self.receipts.clear();
    }

    /// Clears all blocks and receipts from the mock chain provider.
    pub fn clear(&mut self) {
        self.clear_blocks();
        self.clear_receipts();
        self.clear_headers();
    }
}

/// An error for the [`TestChainProvider`] and [`TestL2ChainProvider`].
#[derive(Error, Debug)]
pub enum TestProviderError {
    /// The block was not found.
    #[error("Block not found")]
    BlockNotFound,
    /// The header was not found.
    #[error("Header not found")]
    HeaderNotFound,
    /// The receipts were not found.
    #[error("Receipts not found")]
    ReceiptsNotFound,
    /// The L2 block was not found.
    #[error("L2 Block not found")]
    L2BlockNotFound,
    /// The system config was not found.
    #[error("System config not found")]
    SystemConfigNotFound(u64),
}

impl From<TestProviderError> for PipelineErrorKind {
    fn from(val: TestProviderError) -> Self {
        PipelineError::Provider(val.to_string()).temp()
    }
}

#[async_trait]
impl ChainProvider for TestChainProvider {
    type Error = TestProviderError;

    async fn header_by_hash(&mut self, hash: B256) -> Result<Header, Self::Error> {
        if let Some((_, header)) = self.headers.iter().find(|(_, b)| b.hash_slow() == hash) {
            Ok(header.clone())
        } else {
            Err(TestProviderError::HeaderNotFound)
        }
    }

    async fn block_info_by_number(&mut self, _number: u64) -> Result<BlockInfo, Self::Error> {
        if let Some((_, block)) = self.blocks.iter().find(|(n, _)| *n == _number) {
            Ok(*block)
        } else {
            Err(TestProviderError::BlockNotFound)
        }
    }

    async fn receipts_by_hash(&mut self, _hash: B256) -> Result<Vec<Receipt>, Self::Error> {
        if let Some((_, receipts)) = self.receipts.iter().find(|(h, _)| *h == _hash) {
            Ok(receipts.clone())
        } else {
            Err(TestProviderError::ReceiptsNotFound)
        }
    }

    async fn block_info_and_transactions_by_hash(
        &mut self,
        hash: B256,
    ) -> Result<(BlockInfo, Vec<TxEnvelope>), Self::Error> {
        let block = self
            .blocks
            .iter()
            .find(|(_, b)| b.hash == hash)
            .map(|(_, b)| *b)
            .ok_or_else(|| TestProviderError::BlockNotFound)?;
        let txs = self
            .transactions
            .iter()
            .find(|(h, _)| *h == hash)
            .map(|(_, txs)| txs.clone())
            .unwrap_or_default();
        Ok((block, txs))
    }
}

/// An [`L2ChainProvider`] implementation for testing.
#[derive(Debug, Default, Clone)]
pub struct TestL2ChainProvider {
    /// Blocks
    pub blocks: Vec<L2BlockInfo>,
    /// Short circuit the block return to be the first block.
    pub short_circuit: bool,
    /// Blocks
    pub op_blocks: Vec<OpBlock>,
    /// System configs
    pub system_configs: HashMap<u64, SystemConfig>,
}

impl TestL2ChainProvider {
    /// Creates a new [`TestL2ChainProvider`] with the given origin and batches.
    pub const fn new(
        blocks: Vec<L2BlockInfo>,
        op_blocks: Vec<OpBlock>,
        system_configs: HashMap<u64, SystemConfig>,
    ) -> Self {
        Self { blocks, short_circuit: false, op_blocks, system_configs }
    }
}

#[async_trait]
impl BatchValidationProvider for TestL2ChainProvider {
    type Error = TestProviderError;

    async fn l2_block_info_by_number(&mut self, number: u64) -> Result<L2BlockInfo, Self::Error> {
        if self.short_circuit {
            return self.blocks.first().copied().ok_or_else(|| TestProviderError::BlockNotFound);
        }
        self.blocks
            .iter()
            .find(|b| b.block_info.number == number)
            .cloned()
            .ok_or_else(|| TestProviderError::BlockNotFound)
    }

    async fn block_by_number(&mut self, number: u64) -> Result<OpBlock, Self::Error> {
        self.op_blocks
            .iter()
            .find(|p| p.header.number == number)
            .cloned()
            .ok_or_else(|| TestProviderError::L2BlockNotFound)
    }
}

#[async_trait]
impl L2ChainProvider for TestL2ChainProvider {
    type Error = TestProviderError;

    async fn system_config_by_number(
        &mut self,
        number: u64,
        _: Arc<RollupConfig>,
    ) -> Result<SystemConfig, <Self as L2ChainProvider>::Error> {
        self.system_configs
            .get(&number)
            .ok_or_else(|| TestProviderError::SystemConfigNotFound(number))
            .cloned()
    }
}
