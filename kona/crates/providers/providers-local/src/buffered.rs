//! Buffered L2 Provider implementation that serves data from in-memory chain state.
//!
//! This provider operates as a pure in-memory cache without any dependency on external RPC
//! providers. It stores complete blocks with their L2 block information and serves all queries
//! directly from this cached state. Chain updates are provided through the `add_block` and
//! `handle_chain_event` methods.

use alloy_primitives::B256;
use async_trait::async_trait;
use kona_derive::{L2ChainProvider, PipelineError, PipelineErrorKind};
use kona_genesis::{ChainGenesis, RollupConfig, SystemConfig};
use kona_protocol::{BatchValidationProvider, L2BlockInfo, to_system_config};
use op_alloy_consensus::OpBlock;
use std::sync::Arc;
use tokio::sync::RwLock;

use crate::{CachedBlock, ChainBufferError, ChainStateBuffer, ChainStateEvent};

/// A buffered L2 provider that serves data from in-memory chain state.
///
/// This provider maintains an in-memory cache of L2 blocks and serves all queries
/// from this cache. It does not depend on any external RPC provider. Instead, blocks
/// must be explicitly added to the cache using the `add_block` method, typically
/// when processing chain events from execution extensions or other sources.
///
/// # Features
/// - Pure in-memory operation without external dependencies
/// - LRU caching with configurable size
/// - Reorg handling up to a configurable depth
/// - Efficient block and L2 block info queries
/// - System config extraction from cached blocks
#[derive(Debug)]
pub struct BufferedL2Provider {
    /// Chain state buffer for storing blocks
    buffer: Arc<ChainStateBuffer>,
    /// Current chain head we're tracking
    current_head: RwLock<Option<B256>>,
    /// Rollup configuration
    rollup_config: Arc<RollupConfig>,
    /// Genesis information
    genesis: ChainGenesis,
}

impl BufferedL2Provider {
    /// Create a new buffered L2 provider.
    ///
    /// # Arguments
    /// * `rollup_config` - The rollup configuration containing genesis and chain parameters
    /// * `cache_size` - Maximum number of blocks to keep in the LRU cache
    /// * `max_reorg_depth` - Maximum reorg depth to handle before clearing the cache
    pub fn new(rollup_config: Arc<RollupConfig>, cache_size: usize, max_reorg_depth: u64) -> Self {
        let genesis = rollup_config.genesis;
        Self {
            buffer: Arc::new(ChainStateBuffer::new(cache_size, max_reorg_depth)),
            current_head: RwLock::new(None),
            rollup_config,
            genesis,
        }
    }

    /// Process a chain state event.
    ///
    /// This method should be called when receiving chain state notifications from
    /// execution extensions or other chain event sources. It updates the internal
    /// state and cache based on the event type (commit, reorg, or revert).
    ///
    /// # Arguments
    /// * `event` - The chain state event to process
    pub async fn handle_chain_event(
        &self,
        event: ChainStateEvent,
    ) -> Result<(), BufferedProviderError> {
        // Track metrics for chain events
        #[cfg(feature = "metrics")]
        let event_type = match &event {
            ChainStateEvent::ChainCommitted { .. } => "committed",
            ChainStateEvent::ChainReorged { .. } => "reorged",
            ChainStateEvent::ChainReverted { .. } => "reverted",
        };

        // Update our tracked head based on the event
        match &event {
            ChainStateEvent::ChainCommitted { new_head, .. } |
            ChainStateEvent::ChainReorged { new_head, .. } |
            ChainStateEvent::ChainReverted { new_head, .. } => {
                let mut current_head = self.current_head.write().await;
                *current_head = Some(*new_head);
            }
        }

        // Handle the event in the buffer
        let result = self.buffer.handle_event(event).await.map_err(BufferedProviderError::Buffer);

        #[cfg(feature = "metrics")]
        {
            use crate::Metrics;
            if result.is_ok() {
                kona_macros::inc!(gauge, Metrics::CHAIN_EVENTS_PROCESSED, "event" => event_type);
            } else {
                kona_macros::inc!(gauge, Metrics::CHAIN_EVENT_ERRORS, "event" => event_type);
            }
        }

        result
    }

    /// Add a block to the buffer.
    ///
    /// This is the primary method for populating the provider with block data.
    /// The block and its associated L2 block info are stored in the cache and
    /// can be queried using the various provider methods.
    ///
    /// # Arguments
    /// * `block` - The OP block to add
    /// * `l2_block_info` - The L2 block information associated with the block
    pub async fn add_block(
        &self,
        block: OpBlock,
        l2_block_info: L2BlockInfo,
    ) -> Result<(), BufferedProviderError> {
        let cached_block = CachedBlock::new(block, l2_block_info);
        self.buffer.insert_block(cached_block).await;

        #[cfg(feature = "metrics")]
        {
            use crate::Metrics;
            kona_macros::inc!(gauge, Metrics::BLOCKS_ADDED);
        }

        Ok(())
    }

    /// Get the current chain head
    pub async fn current_head(&self) -> Option<B256> {
        let current_head = self.current_head.read().await;
        *current_head
    }

    /// Get cache statistics
    pub async fn cache_stats(&self) -> crate::buffer::CacheStats {
        self.buffer.cache_stats().await
    }

    /// Clear the cache
    pub async fn clear_cache(&self) {
        self.buffer.clear().await;
    }
}

/// Clone implementation for BufferedL2Provider
impl Clone for BufferedL2Provider {
    fn clone(&self) -> Self {
        Self {
            buffer: self.buffer.clone(),
            current_head: RwLock::new(None),
            rollup_config: self.rollup_config.clone(),
            genesis: self.genesis,
        }
    }
}

#[async_trait]
impl L2ChainProvider for BufferedL2Provider {
    type Error = BufferedProviderError;

    async fn system_config_by_number(
        &mut self,
        number: u64,
        rollup_config: Arc<RollupConfig>,
    ) -> Result<SystemConfig, <Self as L2ChainProvider>::Error> {
        // Check if this is the genesis block
        if number == self.genesis.l2.number {
            return self.genesis.system_config.ok_or(BufferedProviderError::SystemConfigMissing);
        }

        // Get the block from cache
        let cached_block = self.buffer.get_block_by_number(number).await;

        #[cfg(feature = "metrics")]
        {
            use crate::Metrics;
            if cached_block.is_some() {
                kona_macros::inc!(gauge, Metrics::BUFFERED_PROVIDER_CACHE_HITS, "method" => "system_config");
            } else {
                kona_macros::inc!(gauge, Metrics::BUFFERED_PROVIDER_CACHE_MISSES, "method" => "system_config");
            }
        }

        let cached_block = cached_block.ok_or(BufferedProviderError::BlockNotFound(number))?;

        // Extract system config from the block
        to_system_config(&cached_block.block, &rollup_config)
            .map_err(|_| BufferedProviderError::SystemConfigConversion(number))
    }
}

#[async_trait]
impl BatchValidationProvider for BufferedL2Provider {
    type Error = BufferedProviderError;

    async fn block_by_number(&mut self, number: u64) -> Result<OpBlock, Self::Error> {
        // Get the block from cache
        let cached_block = self.buffer.get_block_by_number(number).await;

        #[cfg(feature = "metrics")]
        {
            use crate::Metrics;
            if cached_block.is_some() {
                kona_macros::inc!(gauge, Metrics::BUFFERED_PROVIDER_CACHE_HITS, "method" => "block_by_number");
            } else {
                kona_macros::inc!(gauge, Metrics::BUFFERED_PROVIDER_CACHE_MISSES, "method" => "block_by_number");
            }
        }

        let cached_block = cached_block.ok_or(BufferedProviderError::BlockNotFound(number))?;

        Ok(cached_block.block)
    }

    async fn l2_block_info_by_number(&mut self, number: u64) -> Result<L2BlockInfo, Self::Error> {
        // Check if this is the genesis block
        if number == self.genesis.l2.number {
            return L2BlockInfo::from_block_and_genesis(
                &OpBlock::default(), // Genesis doesn't need full block data
                &self.genesis,
            )
            .map_err(|_| BufferedProviderError::L2BlockInfoConstruction(number));
        }

        // Get the block from cache
        let cached_block = self.buffer.get_block_by_number(number).await;

        #[cfg(feature = "metrics")]
        {
            use crate::Metrics;
            if cached_block.is_some() {
                kona_macros::inc!(gauge, Metrics::BUFFERED_PROVIDER_CACHE_HITS, "method" => "l2_block_info");
            } else {
                kona_macros::inc!(gauge, Metrics::BUFFERED_PROVIDER_CACHE_MISSES, "method" => "l2_block_info");
            }
        }

        let cached_block = cached_block.ok_or(BufferedProviderError::BlockNotFound(number))?;

        Ok(cached_block.l2_block_info)
    }
}

/// Errors that can occur in the buffered provider
#[derive(Debug, thiserror::Error)]
pub enum BufferedProviderError {
    /// Error from the chain buffer
    #[error("Buffer error: {0}")]
    Buffer(#[from] ChainBufferError),
    /// Block not found in cache
    #[error("Block {0} not found in cache")]
    BlockNotFound(u64),
    /// Failed to construct L2BlockInfo
    #[error("Failed to construct L2BlockInfo for block {0}")]
    L2BlockInfoConstruction(u64),
    /// Failed to convert block to SystemConfig
    #[error("Failed to convert block {0} to SystemConfig")]
    SystemConfigConversion(u64),
    /// System config missing from genesis
    #[error("System config missing from genesis")]
    SystemConfigMissing,
}

impl From<BufferedProviderError> for PipelineErrorKind {
    fn from(e: BufferedProviderError) -> Self {
        match e {
            BufferedProviderError::Buffer(ChainBufferError::ReorgTooDeep { depth, max_depth }) => {
                Self::Critical(PipelineError::Provider(format!(
                    "Reorg too deep: {depth} > {max_depth}"
                )))
            }
            BufferedProviderError::Buffer(ChainBufferError::BlockNotFound { hash }) => {
                Self::Temporary(PipelineError::Provider(format!(
                    "Block not found in cache: {hash}"
                )))
            }
            BufferedProviderError::BlockNotFound(number) => Self::Temporary(
                PipelineError::Provider(format!("Block {number} not found in cache")),
            ),
            BufferedProviderError::L2BlockInfoConstruction(number) => {
                Self::Temporary(PipelineError::Provider(format!(
                    "Failed to construct L2BlockInfo for block {number}"
                )))
            }
            BufferedProviderError::SystemConfigConversion(number) => {
                Self::Temporary(PipelineError::Provider(format!(
                    "Failed to convert block {number} to SystemConfig"
                )))
            }
            BufferedProviderError::SystemConfigMissing => Self::Critical(PipelineError::Provider(
                "System config missing from genesis".to_string(),
            )),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_consensus::Header;
    use alloy_eips::BlockNumHash;
    use alloy_primitives::B256;
    use kona_genesis::RollupConfig;
    use kona_protocol::BlockInfo;

    async fn create_test_provider() -> BufferedL2Provider {
        let mut rollup_config = RollupConfig::default();
        rollup_config.genesis.l2 = BlockNumHash { number: 0, hash: B256::ZERO };
        rollup_config.genesis.l1 = BlockNumHash { number: 0, hash: B256::ZERO };
        rollup_config.genesis.system_config = Some(SystemConfig::default());
        let rollup_config = Arc::new(rollup_config);

        BufferedL2Provider::new(rollup_config, 100, 10)
    }

    #[tokio::test]
    async fn test_provider_creation() {
        let provider = create_test_provider().await;
        assert!(provider.current_head().await.is_none());
    }

    #[tokio::test]
    async fn test_chain_event_handling() {
        let provider = create_test_provider().await;

        let hash1 = B256::with_last_byte(1);
        let hash2 = B256::with_last_byte(2);

        // First add some blocks to the cache
        let header1 = Header {
            number: 1,
            parent_hash: B256::ZERO,
            timestamp: 1234567890,
            ..Default::default()
        };
        let block1 = OpBlock { header: header1, body: Default::default() };
        let l2_info1 = L2BlockInfo {
            block_info: BlockInfo {
                hash: hash1,
                number: 1,
                parent_hash: B256::ZERO,
                timestamp: 1234567890,
            },
            l1_origin: BlockNumHash { number: 1, hash: B256::ZERO },
            seq_num: 0,
        };
        provider.add_block(block1, l2_info1).await.unwrap();

        let event =
            ChainStateEvent::ChainCommitted { new_head: hash2, committed: vec![hash1, hash2] };

        let result = provider.handle_chain_event(event).await;
        assert!(result.is_ok());

        assert_eq!(provider.current_head().await, Some(hash2));
    }

    #[tokio::test]
    async fn test_cache_stats() {
        let provider = create_test_provider().await;
        let stats = provider.cache_stats().await;

        assert_eq!(stats.capacity, 100);
        assert_eq!(stats.max_reorg_depth, 10);
        assert_eq!(stats.blocks_by_hash_len, 0);
        assert_eq!(stats.blocks_by_number_len, 0);
    }

    #[tokio::test]
    async fn test_clear_cache() {
        let provider = create_test_provider().await;

        // Clear should work even on empty cache
        provider.clear_cache().await;

        let stats = provider.cache_stats().await;
        assert_eq!(stats.blocks_by_hash_len, 0);
        assert_eq!(stats.blocks_by_number_len, 0);
    }

    #[tokio::test]
    async fn test_add_and_retrieve_block() {
        let mut provider = create_test_provider().await;

        let hash = B256::with_last_byte(1);
        let header = Header {
            number: 1,
            parent_hash: B256::ZERO,
            timestamp: 1234567890,
            ..Default::default()
        };
        let block = OpBlock { header, body: Default::default() };
        let l2_info = L2BlockInfo {
            block_info: BlockInfo {
                hash,
                number: 1,
                parent_hash: B256::ZERO,
                timestamp: 1234567890,
            },
            l1_origin: BlockNumHash { number: 1, hash: B256::ZERO },
            seq_num: 0,
        };

        // Add block to the provider
        provider.add_block(block.clone(), l2_info).await.unwrap();

        // Retrieve block by number
        let retrieved_block = provider.block_by_number(1).await.unwrap();
        assert_eq!(retrieved_block.header.number, 1);

        // Retrieve L2 block info by number
        let retrieved_info = provider.l2_block_info_by_number(1).await.unwrap();
        assert_eq!(retrieved_info.block_info.number, 1);
    }
}
