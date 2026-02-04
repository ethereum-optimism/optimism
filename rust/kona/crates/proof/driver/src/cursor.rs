//! Contains the cursor for the derivation pipeline.
//!
//! This module provides the [`PipelineCursor`] which tracks the state of the derivation
//! pipeline including L1 origins, L2 safe heads, and caching for efficient reorg handling.

use alloc::collections::{btree_map::BTreeMap, vec_deque::VecDeque};
use alloy_consensus::{Header, Sealed};
use alloy_primitives::{B256, map::HashMap};
use kona_protocol::{BlockInfo, L2BlockInfo};

use crate::TipCursor;

/// A cursor that tracks the derivation pipeline state and progress.
///
/// The [`PipelineCursor`] maintains a cache of recent L1 origins and their corresponding
/// L2 tips to efficiently handle reorgs and provide quick access to recent derivation
/// state. It implements a capacity-bounded LRU cache to prevent unbounded memory growth.
///
/// # Cache Management
/// The cursor maintains a cache of recent L1/L2 mappings with a capacity based on the
/// channel timeout. This ensures that reorgs within the channel timeout can be handled
/// efficiently without re-deriving from genesis.
///
/// # Reorg Handling
/// When L1 reorgs occur, the cursor can reset to a previous safe state within the
/// channel timeout window, allowing derivation to continue from a known good state
/// rather than starting over.
///
/// # Memory Bounds
/// The cache size is bounded by `channel_timeout + 5` to ensure reasonable memory
/// usage while providing sufficient history for reorg recovery.
#[derive(Debug, Clone)]
pub struct PipelineCursor {
    /// The maximum number of cached L1/L2 mappings before evicting old entries.
    ///
    /// This capacity is calculated as `channel_timeout + 5` to ensure sufficient
    /// history for reorg handling while preventing unbounded memory growth.
    pub capacity: usize,
    /// The channel timeout in blocks used for reorg recovery calculations.
    ///
    /// This value determines how far back the cursor can reset during reorgs
    /// and influences the cache capacity to ensure adequate history retention.
    pub channel_timeout: u64,
    /// The current L1 origin block that the pipeline is processing.
    ///
    /// This represents the most recent L1 block from which L2 blocks are being
    /// derived. It advances as the pipeline processes new L1 data.
    pub origin: BlockInfo,
    /// Ordered list of L1 origin block numbers for cache eviction policy.
    ///
    /// This deque maintains insertion order to implement LRU eviction when the
    /// cache reaches capacity. The front contains the oldest entries.
    pub origins: VecDeque<u64>,
    /// Mapping from L1 block numbers to their corresponding [`BlockInfo`].
    ///
    /// This cache stores L1 block information for quick lookup during reorg
    /// recovery without needing to re-fetch from the data source.
    pub origin_infos: HashMap<u64, BlockInfo>,
    /// Mapping from L1 origin block numbers to their corresponding L2 tips.
    ///
    /// This is the main cache storing the relationship between L1 origins and
    /// the L2 safe head state derived from them. Used for efficient reorg recovery.
    pub tips: BTreeMap<u64, TipCursor>,
}

impl PipelineCursor {
    /// Creates a new pipeline cursor with the specified channel timeout and initial origin.
    ///
    /// The cursor is initialized with a cache capacity of `channel_timeout + 5` to ensure
    /// sufficient history for reorg handling. The initial L1 origin is added to the cache.
    ///
    /// # Arguments
    /// * `channel_timeout` - The channel timeout in blocks for reorg recovery
    /// * `origin` - The initial L1 origin block to start derivation from
    ///
    /// # Returns
    /// A new [`PipelineCursor`] initialized with the given parameters
    ///
    /// # Cache Capacity
    /// The capacity is set to `channel_timeout + 5` blocks to provide adequate
    /// history for reorg recovery while maintaining reasonable memory usage.
    /// This ensures the cursor can reset to any point within the channel timeout window.
    pub fn new(channel_timeout: u64, origin: BlockInfo) -> Self {
        // NOTE: capacity must be greater than the `channel_timeout` to allow
        // for derivation to proceed through a deep reorg.
        // Ref: <https://specs.optimism.io/protocol/derivation.html#timeouts>
        let capacity = channel_timeout as usize + 5;

        let mut origins = VecDeque::with_capacity(capacity);
        origins.push_back(origin.number);
        let mut origin_infos = HashMap::default();
        origin_infos.insert(origin.number, origin);
        Self { capacity, channel_timeout, origin, origins, origin_infos, tips: Default::default() }
    }

    /// Returns the current L1 origin block being processed by the pipeline.
    ///
    /// This is the most recent L1 block from which the pipeline is deriving L2 blocks.
    /// The origin advances as new L1 data becomes available and is processed.
    pub const fn origin(&self) -> BlockInfo {
        self.origin
    }

    /// Returns the current L2 safe head block information.
    ///
    /// The L2 safe head represents the most recent L2 block that has been successfully
    /// derived and is considered safe. This is the tip of the verified L2 chain.
    pub fn l2_safe_head(&self) -> &L2BlockInfo {
        &self.tip().l2_safe_head
    }

    /// Returns the sealed header of the current L2 safe head.
    ///
    /// The sealed header contains the complete block header with computed hash,
    /// providing access to all block metadata including parent hash, timestamp,
    /// gas limits, and other consensus-critical information.
    pub fn l2_safe_head_header(&self) -> &Sealed<Header> {
        &self.tip().l2_safe_head_header
    }

    /// Returns the output root of the current L2 safe head.
    ///
    /// The output root is a commitment to the L2 state after executing the safe head
    /// block. It's used for fraud proof verification and state root challenges.
    pub fn l2_safe_head_output_root(&self) -> &B256 {
        &self.tip().l2_safe_head_output_root
    }

    /// Returns the current L2 tip cursor containing safe head information.
    ///
    /// The tip cursor encapsulates the L2 safe head block info, header, and output root
    /// for the most recently processed L1 origin.
    ///
    /// # Panics
    /// This method panics if called before the cursor is properly initialized with at
    /// least one L1/L2 mapping. This should never happen in normal operation as the
    /// cursor is initialized with an origin in [`Self::new`].
    pub fn tip(&self) -> &TipCursor {
        if let Some((_, l2_tip)) = self.tips.last_key_value() {
            l2_tip
        } else {
            unreachable!("cursor must be initialized with one block before advancing")
        }
    }

    /// Advances the cursor to a new L1 origin and corresponding L2 tip.
    ///
    /// This method updates the cursor state with a new L1/L2 mapping, representing
    /// progress in the derivation pipeline. If the cache is at capacity, the oldest
    /// entry is evicted using LRU policy.
    ///
    /// # Arguments
    /// * `origin` - The new L1 origin block that has been processed
    /// * `l2_tip_block` - The L2 tip cursor resulting from processing this L1 origin
    ///
    /// # Cache Management
    /// - If cache is full, evicts the oldest L1/L2 mapping
    /// - Updates the current origin to the new L1 block
    /// - Adds the new mapping to the cache for future reorg recovery
    ///
    /// # Usage
    /// Called by the driver after successfully deriving an L2 block from L1 data
    /// to advance the derivation state and maintain the cursor cache.
    pub fn advance(&mut self, origin: BlockInfo, l2_tip_block: TipCursor) {
        if self.tips.len() >= self.capacity {
            let key = self.origins.pop_front().unwrap();
            self.tips.remove(&key);
        }

        self.origin = origin;
        self.origins.push_back(origin.number);
        self.origin_infos.insert(origin.number, origin);
        self.tips.insert(origin.number, l2_tip_block);
    }

    /// Resets the cursor state due to an L1 reorganization.
    ///
    /// When the L1 chain undergoes a reorg, the cursor needs to reset to a safe state
    /// that accounts for the channel timeout. This ensures that any L2 blocks that
    /// might have started derivation from invalidated L1 blocks are properly handled.
    ///
    /// # Arguments
    /// * `fork_block` - The L1 block number where the reorg was detected
    ///
    /// # Returns
    /// A tuple containing:
    /// * [`TipCursor`] - The L2 safe state to reset to
    /// * [`BlockInfo`] - The L1 origin block info corresponding to the reset state
    ///
    /// # Reset Logic
    /// The reset target is calculated as `fork_block - channel_timeout` because:
    /// 1. L2 blocks can derive from L1 data spanning the channel timeout window
    /// 2. Any L2 block that started derivation within this window might be affected
    /// 3. Resetting before this window ensures a clean slate for re-derivation
    ///
    /// # Cache Lookup Strategy
    /// 1. **Exact match**: If `channel_start` block is cached, use it directly
    /// 2. **Fallback**: Find the most recent cached block before `channel_start`
    ///
    /// # Panics
    /// This method panics if no suitable reset target is found in the cache,
    /// which should never happen if the cache capacity is properly sized relative
    /// to the channel timeout.
    ///
    /// # Usage
    /// Called automatically when the pipeline detects an L1 reorg to ensure
    /// derivation continues from a safe, unaffected state.
    pub fn reset(&mut self, fork_block: u64) -> (TipCursor, BlockInfo) {
        let channel_start = fork_block - self.channel_timeout;

        match self.tips.get(&channel_start) {
            Some(l2_safe_tip) => {
                // The channel start block is in the cache, we can use it to reset the cursor.
                (l2_safe_tip.clone(), self.origin_infos[&channel_start])
            }
            None => {
                // If the channel start block is not in the cache, we reset the cursor
                // to the closest known L1 block for which we have a corresponding L2 block.
                let (last_l1_known_tip, l2_known_tip) = self
                    .tips
                    .range(..=channel_start)
                    .next_back()
                    .expect("walked back to genesis without finding anchor origin block");

                (l2_known_tip.clone(), self.origin_infos[last_l1_known_tip])
            }
        }
    }
}
