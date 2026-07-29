//! Chain state buffer implementation for handling L2 chain events and reorgs.
//!
//! This module provides the core caching infrastructure for the buffered provider.
//! It manages an LRU cache of blocks indexed by both hash and number, handles
//! chain reorganizations, and maintains the canonical chain state.

use alloy_primitives::B256;
use kona_protocol::L2BlockInfo;
use lru::LruCache;
use op_alloy_consensus::OpBlock;
use std::num::NonZeroUsize;
use tokio::sync::RwLock;

/// Events that can affect chain state
#[derive(Debug, Clone)]
pub enum ChainStateEvent {
    /// New blocks have been committed to the canonical chain
    ChainCommitted {
        /// The new chain head
        new_head: B256,
        /// The blocks that were committed
        committed: Vec<B256>,
    },
    /// Chain reorganization occurred
    ChainReorged {
        /// The old chain head before reorg
        old_head: B256,
        /// The new chain head after reorg
        new_head: B256,
        /// The depth of the reorg (how many blocks were reverted)
        depth: u64,
    },
    /// Chain was reverted to a previous state
    ChainReverted {
        /// The old chain head before revert
        old_head: B256,
        /// The new chain head after revert
        new_head: B256,
        /// The blocks that were reverted
        reverted: Vec<B256>,
    },
}

/// Cached block data containing full block information.
///
/// This structure stores a complete OP block along with its derived L2 block info.
/// This allows the buffered provider to serve all queries without needing to
/// recompute or fetch data from external sources.
#[derive(Debug, Clone)]
pub struct CachedBlock {
    /// Full OP block data including header and body
    pub block: OpBlock,
    /// L2 block info derived from the block
    pub l2_block_info: L2BlockInfo,
    /// Whether this block is part of the canonical chain
    pub canonical: bool,
}

impl CachedBlock {
    /// Create a new cached block
    pub const fn new(block: OpBlock, l2_block_info: L2BlockInfo) -> Self {
        Self { block, l2_block_info, canonical: true }
    }

    /// Mark this block as non-canonical
    pub const fn mark_non_canonical(mut self) -> Self {
        self.canonical = false;
        self
    }

    /// Get the block hash
    pub const fn hash(&self) -> B256 {
        self.l2_block_info.block_info.hash
    }

    /// Get the block number
    pub const fn number(&self) -> u64 {
        self.l2_block_info.block_info.number
    }
}

/// Buffer for managing chain state with LRU caching and reorg handling.
///
/// This buffer maintains two indexes for efficient block lookups:
/// - By hash: Direct access to blocks by their hash
/// - By number: Maps block numbers to hashes for numbered queries
///
/// The buffer also tracks the canonical chain head and handles reorganizations
/// up to a configurable depth. Deep reorgs beyond the configured limit will
/// trigger a cache clear to maintain consistency.
#[derive(Debug)]
pub struct ChainStateBuffer {
    /// LRU cache for blocks by hash
    blocks_by_hash: RwLock<LruCache<B256, CachedBlock>>,
    /// LRU cache for blocks by number
    blocks_by_number: RwLock<LruCache<u64, B256>>,
    /// Current canonical chain head
    canonical_head: RwLock<Option<B256>>,
    /// Maximum reorg depth to support
    max_reorg_depth: u64,
    /// Cache capacity
    capacity: usize,
}

impl ChainStateBuffer {
    /// Create a new chain state buffer.
    ///
    /// # Arguments
    /// * `capacity` - Maximum number of blocks to cache (affects memory usage)
    /// * `max_reorg_depth` - Maximum reorg depth to handle before clearing cache
    pub fn new(capacity: usize, max_reorg_depth: u64) -> Self {
        Self {
            blocks_by_hash: RwLock::new(LruCache::new(NonZeroUsize::new(capacity).unwrap())),
            blocks_by_number: RwLock::new(LruCache::new(NonZeroUsize::new(capacity).unwrap())),
            canonical_head: RwLock::new(None),
            max_reorg_depth,
            capacity,
        }
    }

    /// Get block by hash from cache
    pub async fn get_block_by_hash(&self, hash: B256) -> Option<CachedBlock> {
        let cache = self.blocks_by_hash.read().await;
        cache.peek(&hash).cloned()
    }

    /// Get block by number from cache
    pub async fn get_block_by_number(&self, number: u64) -> Option<CachedBlock> {
        let blocks_by_number = self.blocks_by_number.read().await;
        if let Some(hash) = blocks_by_number.peek(&number) {
            let hash = *hash;
            drop(blocks_by_number);
            self.get_block_by_hash(hash).await
        } else {
            None
        }
    }

    /// Insert a block into the cache
    pub async fn insert_block(&self, block: CachedBlock) {
        let hash = block.hash();
        let number = block.number();

        let mut blocks_by_hash = self.blocks_by_hash.write().await;
        let mut blocks_by_number = self.blocks_by_number.write().await;

        blocks_by_hash.put(hash, block);
        blocks_by_number.put(number, hash);

        #[cfg(feature = "metrics")]
        {
            use crate::Metrics;
            kona_macros::set!(
                gauge,
                Metrics::CACHE_ENTRIES,
                "cache",
                "blocks_by_hash",
                blocks_by_hash.len() as f64
            );
            kona_macros::set!(
                gauge,
                Metrics::CACHE_ENTRIES,
                "cache",
                "blocks_by_number",
                blocks_by_number.len() as f64
            );
        }
    }

    /// Handle a chain state event
    pub async fn handle_event(&self, event: ChainStateEvent) -> Result<(), ChainBufferError> {
        match event {
            ChainStateEvent::ChainCommitted { new_head, committed } => {
                self.handle_chain_committed(new_head, committed).await
            }
            ChainStateEvent::ChainReorged { old_head, new_head, depth } => {
                self.handle_chain_reorged(old_head, new_head, depth).await
            }
            ChainStateEvent::ChainReverted { old_head, new_head, reverted } => {
                self.handle_chain_reverted(old_head, new_head, reverted).await
            }
        }
    }

    /// Handle chain committed event
    async fn handle_chain_committed(
        &self,
        new_head: B256,
        committed: Vec<B256>,
    ) -> Result<(), ChainBufferError> {
        // Update canonical head
        let mut canonical_head = self.canonical_head.write().await;
        *canonical_head = Some(new_head);

        // Mark all committed blocks as canonical
        let mut blocks_by_hash = self.blocks_by_hash.write().await;
        for hash in committed {
            if let Some(block) = blocks_by_hash.get_mut(&hash) {
                block.canonical = true;
            }
        }

        Ok(())
    }

    /// Handle chain reorged event
    async fn handle_chain_reorged(
        &self,
        _old_head: B256,
        new_head: B256,
        depth: u64,
    ) -> Result<(), ChainBufferError> {
        if depth > self.max_reorg_depth {
            return Err(ChainBufferError::ReorgTooDeep { depth, max_depth: self.max_reorg_depth });
        }

        #[cfg(feature = "metrics")]
        {
            use crate::Metrics;
            kona_macros::set!(gauge, Metrics::REORG_DEPTH, depth as f64);
        }

        // Update canonical head
        let mut canonical_head = self.canonical_head.write().await;
        *canonical_head = Some(new_head);

        // We need to invalidate cached blocks that are no longer canonical
        // For now, we'll clear the entire cache on deep reorgs
        if depth > 10 {
            let mut blocks_by_hash = self.blocks_by_hash.write().await;
            let mut blocks_by_number = self.blocks_by_number.write().await;
            blocks_by_hash.clear();
            blocks_by_number.clear();

            #[cfg(feature = "metrics")]
            {
                use crate::Metrics;
                kona_macros::inc!(gauge, Metrics::CACHE_CLEARS);
            }
        }

        Ok(())
    }

    /// Handle chain reverted event
    async fn handle_chain_reverted(
        &self,
        _old_head: B256,
        new_head: B256,
        reverted: Vec<B256>,
    ) -> Result<(), ChainBufferError> {
        // Update canonical head
        let mut canonical_head = self.canonical_head.write().await;
        *canonical_head = Some(new_head);

        // Mark reverted blocks as non-canonical and remove from cache
        let mut blocks_by_hash = self.blocks_by_hash.write().await;
        for hash in reverted {
            blocks_by_hash.pop(&hash);
        }

        Ok(())
    }

    /// Get the current canonical head
    pub async fn canonical_head(&self) -> Option<B256> {
        let canonical_head = self.canonical_head.read().await;
        *canonical_head
    }

    /// Get cache statistics
    pub async fn cache_stats(&self) -> CacheStats {
        let blocks_by_hash = self.blocks_by_hash.read().await;
        let blocks_by_number = self.blocks_by_number.read().await;

        CacheStats {
            blocks_by_hash_len: blocks_by_hash.len(),
            blocks_by_number_len: blocks_by_number.len(),
            capacity: self.capacity,
            max_reorg_depth: self.max_reorg_depth,
        }
    }

    /// Clear the entire cache
    pub async fn clear(&self) {
        let mut blocks_by_hash = self.blocks_by_hash.write().await;
        let mut blocks_by_number = self.blocks_by_number.write().await;
        let mut canonical_head = self.canonical_head.write().await;

        blocks_by_hash.clear();
        blocks_by_number.clear();
        *canonical_head = None;

        #[cfg(feature = "metrics")]
        {
            use crate::Metrics;
            kona_macros::inc!(gauge, Metrics::CACHE_CLEARS);
            kona_macros::set!(gauge, Metrics::CACHE_ENTRIES, "cache", "blocks_by_hash", 0);
            kona_macros::set!(gauge, Metrics::CACHE_ENTRIES, "cache", "blocks_by_number", 0);
        }
    }
}

/// Cache statistics
#[derive(Debug, Clone)]
pub struct CacheStats {
    /// Number of blocks cached by hash
    pub blocks_by_hash_len: usize,
    /// Number of blocks cached by number
    pub blocks_by_number_len: usize,
    /// Total cache capacity
    pub capacity: usize,
    /// Maximum reorg depth supported
    pub max_reorg_depth: u64,
}

/// Errors that can occur in the chain buffer
#[derive(Debug, thiserror::Error)]
pub enum ChainBufferError {
    /// Reorg is too deep to handle
    #[error("Reorg depth {depth} exceeds maximum supported depth {max_depth}")]
    ReorgTooDeep {
        /// The depth of the reorg attempted
        depth: u64,
        /// The maximum supported reorg depth
        max_depth: u64,
    },
    /// Block not found in cache
    #[error("Block not found in cache: {hash}")]
    BlockNotFound {
        /// The hash of the block that was not found
        hash: B256,
    },
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_consensus::Header;
    use alloy_eips::BlockNumHash;
    use alloy_primitives::{FixedBytes, U256};
    use kona_protocol::BlockInfo;
    use op_alloy_consensus::OpTxEnvelope;

    fn create_test_block(number: u64, _hash: B256, parent_hash: B256) -> (OpBlock, L2BlockInfo) {
        let header = Header {
            number,
            parent_hash,
            timestamp: 1234567890,
            gas_limit: 8000000,
            gas_used: 5000000,
            base_fee_per_gas: Some(20_000_000_000u64),
            difficulty: U256::ZERO,
            nonce: FixedBytes::ZERO,
            ..Default::default()
        };

        let block = OpBlock {
            header: header.clone(),
            body: alloy_consensus::BlockBody {
                transactions: Vec::<OpTxEnvelope>::new(),
                ommers: Vec::new(),
                withdrawals: None,
            },
        };

        let l2_block_info = L2BlockInfo {
            block_info: BlockInfo {
                hash: header.hash_slow(),
                number: header.number,
                parent_hash: header.parent_hash,
                timestamp: header.timestamp,
            },
            l1_origin: BlockNumHash { number: 1, hash: B256::ZERO },
            seq_num: 0,
        };

        (block, l2_block_info)
    }

    #[tokio::test]
    async fn test_buffer_basic_operations() {
        let buffer = ChainStateBuffer::new(100, 10);

        let (block, l2_info) = create_test_block(1, B256::ZERO, B256::ZERO);
        let cached_block = CachedBlock::new(block, l2_info);
        let computed_hash = cached_block.hash();

        // Insert block
        buffer.insert_block(cached_block.clone()).await;

        // Retrieve by hash
        let retrieved = buffer.get_block_by_hash(computed_hash).await;
        assert!(retrieved.is_some());
        assert_eq!(retrieved.unwrap().hash(), computed_hash);

        // Retrieve by number
        let retrieved = buffer.get_block_by_number(1).await;
        assert!(retrieved.is_some());
        assert_eq!(retrieved.unwrap().number(), 1);
    }

    #[tokio::test]
    async fn test_chain_committed_event() {
        let buffer = ChainStateBuffer::new(100, 10);

        let hash1 = B256::with_last_byte(1);
        let hash2 = B256::with_last_byte(2);
        let (block1, l2_info1) = create_test_block(1, hash1, B256::ZERO);
        let (block2, l2_info2) = create_test_block(2, hash2, hash1);

        let cached_block1 = CachedBlock::new(block1, l2_info1);
        let cached_block2 = CachedBlock::new(block2, l2_info2);

        buffer.insert_block(cached_block1).await;
        buffer.insert_block(cached_block2).await;

        // Handle committed event
        let event =
            ChainStateEvent::ChainCommitted { new_head: hash2, committed: vec![hash1, hash2] };

        buffer.handle_event(event).await.unwrap();

        // Check canonical head is updated
        assert_eq!(buffer.canonical_head().await, Some(hash2));
    }

    #[tokio::test]
    async fn test_reorg_too_deep() {
        let buffer = ChainStateBuffer::new(100, 5);

        let hash1 = B256::with_last_byte(1);
        let hash2 = B256::with_last_byte(2);

        let event = ChainStateEvent::ChainReorged {
            old_head: hash1,
            new_head: hash2,
            depth: 10, // Exceeds max depth of 5
        };

        let result = buffer.handle_event(event).await;
        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), ChainBufferError::ReorgTooDeep { .. }));
    }

    #[tokio::test]
    async fn test_cache_stats() {
        let buffer = ChainStateBuffer::new(100, 10);

        let hash1 = B256::with_last_byte(1);
        let (block, l2_info) = create_test_block(1, hash1, B256::ZERO);
        let cached_block = CachedBlock::new(block, l2_info);

        buffer.insert_block(cached_block).await;

        let stats = buffer.cache_stats().await;
        assert_eq!(stats.blocks_by_hash_len, 1);
        assert_eq!(stats.blocks_by_number_len, 1);
        assert_eq!(stats.capacity, 100);
        assert_eq!(stats.max_reorg_depth, 10);
    }

    #[tokio::test]
    async fn test_clear_cache() {
        let buffer = ChainStateBuffer::new(100, 10);

        let (block, l2_info) = create_test_block(1, B256::ZERO, B256::ZERO);
        let cached_block = CachedBlock::new(block, l2_info);
        let computed_hash = cached_block.hash();

        buffer.insert_block(cached_block).await;
        assert!(buffer.get_block_by_hash(computed_hash).await.is_some());

        buffer.clear().await;
        assert!(buffer.get_block_by_hash(computed_hash).await.is_none());
        assert_eq!(buffer.canonical_head().await, None);
    }
}
