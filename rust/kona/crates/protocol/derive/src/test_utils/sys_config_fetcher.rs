//! Implements a mock [`L2ChainProvider`] and [`BatchValidationProvider`] for testing.

use crate::{
    errors::{PipelineError, PipelineErrorKind},
    traits::L2ChainProvider,
};
use alloc::{boxed::Box, string::ToString, sync::Arc};
use alloy_primitives::{B256, map::HashMap};
use async_trait::async_trait;
use kona_genesis::{RollupConfig, SystemConfig};
use kona_protocol::{BatchValidationProvider, L2BlockInfo};
use op_alloy_consensus::OpBlock;
use thiserror::Error;

/// A mock implementation of the [`L2ChainProvider`] and [`BatchValidationProvider`] for testing.
#[derive(Debug, Default)]
pub struct TestSystemConfigL2Fetcher {
    /// A map from L2 block hash to a [`SystemConfig`].
    pub system_configs: HashMap<B256, SystemConfig>,
}

impl TestSystemConfigL2Fetcher {
    /// Inserts a new system config into the mock fetcher with the given block hash.
    pub fn insert(&mut self, hash: B256, config: SystemConfig) {
        self.system_configs.insert(hash, config);
    }

    /// Clears all system configs from the mock fetcher.
    pub fn clear(&mut self) {
        self.system_configs.clear();
    }
}

/// An error returned by the [`TestSystemConfigL2Fetcher`].
#[derive(Error, Debug, PartialEq, Eq)]
pub enum TestSystemConfigL2FetcherError {
    /// The system config was not found.
    #[error("system config not found: {0}")]
    NotFound(B256),
}

impl From<TestSystemConfigL2FetcherError> for PipelineErrorKind {
    fn from(val: TestSystemConfigL2FetcherError) -> Self {
        PipelineError::Provider(val.to_string()).temp()
    }
}

#[async_trait]
impl BatchValidationProvider for TestSystemConfigL2Fetcher {
    type Error = TestSystemConfigL2FetcherError;

    async fn block_by_number(&mut self, _: u64) -> Result<Arc<OpBlock>, Self::Error> {
        unimplemented!()
    }

    async fn l2_block_info_by_number(&mut self, _: u64) -> Result<L2BlockInfo, Self::Error> {
        unimplemented!()
    }
}

#[async_trait]
impl L2ChainProvider for TestSystemConfigL2Fetcher {
    type Error = TestSystemConfigL2FetcherError;

    async fn system_config_by_l2_hash(
        &mut self,
        hash: B256,
        _: Arc<RollupConfig>,
    ) -> Result<SystemConfig, <Self as L2ChainProvider>::Error> {
        self.system_configs
            .get(&hash)
            .copied()
            .ok_or(TestSystemConfigL2FetcherError::NotFound(hash))
    }
}
