//! Test L2 chain provider backed by in-memory state.

use alloy_primitives::map::HashMap;
use async_trait::async_trait;
use kona_derive::{L2ChainProvider, PipelineError, PipelineErrorKind};
use kona_genesis::{RollupConfig, SystemConfig};
use kona_protocol::{BatchValidationProvider, L2BlockInfo};
use op_alloy_consensus::OpBlock;
use std::sync::{Arc, RwLock};

/// Error from the test L2 provider.
#[derive(Debug, thiserror::Error)]
pub enum TestL2ProviderError {
    /// Block not found.
    #[error("L2 block not found: {0}")]
    BlockNotFound(u64),
    /// System config not found.
    #[error("System config not found for block {0}")]
    SystemConfigNotFound(u64),
}

impl From<TestL2ProviderError> for PipelineErrorKind {
    fn from(e: TestL2ProviderError) -> Self {
        PipelineError::Provider(e.to_string()).temp()
    }
}

/// Inner state for the test L2 provider.
#[derive(Debug, Default)]
struct TestL2ProviderInner {
    /// L2 block infos by number.
    blocks: Vec<L2BlockInfo>,
    /// OpBlocks by number (for batch validation).
    op_blocks: Vec<OpBlock>,
    /// System configs by block number.
    system_configs: HashMap<u64, SystemConfig>,
}

/// A test L2 chain provider that stores L2 state in memory.
#[derive(Debug, Clone, Default)]
pub struct TestL2Provider {
    inner: Arc<RwLock<TestL2ProviderInner>>,
}

impl TestL2Provider {
    /// Creates a new [`TestL2Provider`].
    pub fn new() -> Self {
        Self::default()
    }

    /// Inserts an L2 block info.
    pub fn insert_block(&self, block: L2BlockInfo) {
        let mut inner = self.inner.write().unwrap();
        inner.blocks.push(block);
    }

    /// Inserts a system config for the given block number.
    pub fn insert_system_config(&self, number: u64, config: SystemConfig) {
        let mut inner = self.inner.write().unwrap();
        inner.system_configs.insert(number, config);
    }

    /// Inserts an OpBlock.
    pub fn insert_op_block(&self, block: OpBlock) {
        let mut inner = self.inner.write().unwrap();
        inner.op_blocks.push(block);
    }
}

#[async_trait]
impl BatchValidationProvider for TestL2Provider {
    type Error = TestL2ProviderError;

    async fn l2_block_info_by_number(&mut self, number: u64) -> Result<L2BlockInfo, Self::Error> {
        let inner = self.inner.read().unwrap();
        inner
            .blocks
            .iter()
            .find(|b| b.block_info.number == number)
            .copied()
            .ok_or(TestL2ProviderError::BlockNotFound(number))
    }

    async fn block_by_number(&mut self, number: u64) -> Result<OpBlock, Self::Error> {
        let inner = self.inner.read().unwrap();
        inner
            .op_blocks
            .iter()
            .find(|b| b.header.number == number)
            .cloned()
            .ok_or(TestL2ProviderError::BlockNotFound(number))
    }
}

#[async_trait]
impl L2ChainProvider for TestL2Provider {
    type Error = TestL2ProviderError;

    async fn system_config_by_number(
        &mut self,
        number: u64,
        _rollup_config: Arc<RollupConfig>,
    ) -> Result<SystemConfig, <Self as L2ChainProvider>::Error> {
        let inner = self.inner.read().unwrap();
        inner
            .system_configs
            .get(&number)
            .cloned()
            .ok_or(TestL2ProviderError::SystemConfigNotFound(number))
    }
}
