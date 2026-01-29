//! Transaction execution caching for flashblock building.
//!
//! When flashblocks arrive incrementally, each new flashblock triggers a rebuild of pending
//! state from all transactions in the sequence. Without caching, this means re-executing
//! transactions that were already executed when previous flashblocks arrived.
//!
//! # Approach
//!
//! This module uses cumulative state caching: after executing a sequence of transactions,
//! the resulting bundle state and receipts are cached. When the next flashblock arrives,
//! if its transaction list is a continuation of the cached list, execution can resume
//! from the cached state rather than re-executing from scratch.
//!
//! The cache stores:
//! - Ordered list of executed transaction hashes
//! - Cumulative bundle state after all cached transactions
//! - Cumulative receipts for all cached transactions
//!
//! # Example
//!
//! ```text
//! Flashblock 0: txs [A, B]
//!   -> Execute A, B from scratch
//!   -> Cache: txs=[A,B], bundle=state_after_AB, receipts=[rA, rB]
//!
//! Flashblock 1: txs [A, B, C]
//!   -> Prefix [A, B] matches cache
//!   -> Initialize state with cached bundle
//!   -> Execute only C
//!   -> Cache: txs=[A,B,C], bundle=state_after_ABC, receipts=[rA, rB, rC]
//!
//! Flashblock 2 (reorg): txs [A, D, E]
//!   -> Prefix [A] matches, but tx[1]=D != B
//!   -> Clear cache, execute A, D, E from scratch
//! ```

use alloy_primitives::B256;
use reth_primitives_traits::NodePrimitives;
use reth_revm::db::BundleState;

/// Cache of transaction execution results for a single block.
///
/// Stores cumulative execution state that can be restored to skip re-executing
/// transactions that have already been processed. The cache is invalidated when:
/// - A new block starts (different block number)
/// - A reorg is detected (transaction list diverges from cached prefix)
/// - Explicitly cleared
#[derive(Debug)]
pub struct TransactionCache<N: NodePrimitives> {
    /// Block number this cache is valid for.
    block_number: u64,
    /// Ordered list of transaction hashes that have been executed.
    executed_tx_hashes: Vec<B256>,
    /// Cumulative bundle state after executing all cached transactions.
    cumulative_bundle: BundleState,
    /// Receipts for all cached transactions, in execution order.
    receipts: Vec<N::Receipt>,
}

impl<N: NodePrimitives> Default for TransactionCache<N> {
    fn default() -> Self {
        Self::new()
    }
}

impl<N: NodePrimitives> TransactionCache<N> {
    /// Creates a new empty transaction cache.
    pub fn new() -> Self {
        Self {
            block_number: 0,
            executed_tx_hashes: Vec::new(),
            cumulative_bundle: BundleState::default(),
            receipts: Vec::new(),
        }
    }

    /// Creates a new cache for a specific block number.
    pub fn for_block(block_number: u64) -> Self {
        Self { block_number, ..Self::new() }
    }

    /// Returns the block number this cache is valid for.
    pub const fn block_number(&self) -> u64 {
        self.block_number
    }

    /// Checks if this cache is valid for the given block number.
    pub const fn is_valid_for_block(&self, block_number: u64) -> bool {
        self.block_number == block_number
    }

    /// Returns the number of cached transactions.
    pub const fn len(&self) -> usize {
        self.executed_tx_hashes.len()
    }

    /// Returns true if the cache is empty.
    pub const fn is_empty(&self) -> bool {
        self.executed_tx_hashes.is_empty()
    }

    /// Returns the cached transaction hashes.
    pub fn executed_tx_hashes(&self) -> &[B256] {
        &self.executed_tx_hashes
    }

    /// Returns the cached receipts.
    pub fn receipts(&self) -> &[N::Receipt] {
        &self.receipts
    }

    /// Returns the cumulative bundle state.
    pub const fn bundle(&self) -> &BundleState {
        &self.cumulative_bundle
    }

    /// Clears the cache.
    pub fn clear(&mut self) {
        self.executed_tx_hashes.clear();
        self.cumulative_bundle = BundleState::default();
        self.receipts.clear();
        self.block_number = 0;
    }

    /// Updates the cache for a new block, clearing if the block number changed.
    ///
    /// Returns true if the cache was cleared.
    pub fn update_for_block(&mut self, block_number: u64) -> bool {
        if self.block_number == block_number {
            false
        } else {
            self.clear();
            self.block_number = block_number;
            true
        }
    }

    /// Computes the length of the matching prefix between cached transactions
    /// and the provided transaction hashes.
    ///
    /// Returns the number of transactions that can be skipped because they
    /// match the cached execution results.
    pub fn matching_prefix_len(&self, tx_hashes: &[B256]) -> usize {
        self.executed_tx_hashes
            .iter()
            .zip(tx_hashes.iter())
            .take_while(|(cached, incoming)| cached == incoming)
            .count()
    }

    /// Returns cached state for resuming execution if the incoming transactions
    /// have a matching prefix with the cache.
    ///
    /// Returns `Some((bundle, receipts, skip_count))` if there's a non-empty matching
    /// prefix, where:
    /// - `bundle` is the cumulative state after the matching prefix
    /// - `receipts` is the receipts for the matching prefix
    /// - `skip_count` is the number of transactions to skip
    ///
    /// Returns `None` if:
    /// - The cache is empty
    /// - No prefix matches (first transaction differs)
    /// - Block number doesn't match
    pub fn get_resumable_state(
        &self,
        block_number: u64,
        tx_hashes: &[B256],
    ) -> Option<(&BundleState, &[N::Receipt], usize)> {
        if !self.is_valid_for_block(block_number) || self.is_empty() {
            return None;
        }

        let prefix_len = self.matching_prefix_len(tx_hashes);
        if prefix_len == 0 {
            return None;
        }

        // Only return state if the full cache matches (partial prefix would need
        // intermediate state snapshots, which we don't currently store).
        // Partial match means incoming txs diverge from cache, need to re-execute.
        (prefix_len == self.executed_tx_hashes.len()).then_some((
            &self.cumulative_bundle,
            self.receipts.as_slice(),
            prefix_len,
        ))
    }

    /// Updates the cache with new execution results.
    ///
    /// This should be called after executing a flashblock. The provided bundle
    /// and receipts should represent the cumulative state after all transactions.
    pub fn update(
        &mut self,
        block_number: u64,
        tx_hashes: Vec<B256>,
        bundle: BundleState,
        receipts: Vec<N::Receipt>,
    ) {
        self.block_number = block_number;
        self.executed_tx_hashes = tx_hashes;
        self.cumulative_bundle = bundle;
        self.receipts = receipts;
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use reth_optimism_primitives::OpPrimitives;

    type TestCache = TransactionCache<OpPrimitives>;

    #[test]
    fn test_cache_block_validation() {
        let mut cache = TestCache::for_block(100);
        assert!(cache.is_valid_for_block(100));
        assert!(!cache.is_valid_for_block(101));

        // Update for same block doesn't clear
        assert!(!cache.update_for_block(100));

        // Update for different block clears
        assert!(cache.update_for_block(101));
        assert!(cache.is_valid_for_block(101));
    }

    #[test]
    fn test_cache_clear() {
        let mut cache = TestCache::for_block(100);
        assert_eq!(cache.block_number(), 100);

        cache.clear();
        assert_eq!(cache.block_number(), 0);
        assert!(cache.is_empty());
    }

    #[test]
    fn test_matching_prefix_len() {
        let mut cache = TestCache::for_block(100);

        let tx_a = B256::repeat_byte(0xAA);
        let tx_b = B256::repeat_byte(0xBB);
        let tx_c = B256::repeat_byte(0xCC);
        let tx_d = B256::repeat_byte(0xDD);

        // Update cache with [A, B]
        cache.update(100, vec![tx_a, tx_b], BundleState::default(), vec![]);

        // Full match
        assert_eq!(cache.matching_prefix_len(&[tx_a, tx_b]), 2);

        // Continuation
        assert_eq!(cache.matching_prefix_len(&[tx_a, tx_b, tx_c]), 2);

        // Partial match (reorg at position 1)
        assert_eq!(cache.matching_prefix_len(&[tx_a, tx_d, tx_c]), 1);

        // No match (reorg at position 0)
        assert_eq!(cache.matching_prefix_len(&[tx_d, tx_b, tx_c]), 0);

        // Empty incoming
        assert_eq!(cache.matching_prefix_len(&[]), 0);
    }

    #[test]
    fn test_get_resumable_state() {
        let mut cache = TestCache::for_block(100);

        let tx_a = B256::repeat_byte(0xAA);
        let tx_b = B256::repeat_byte(0xBB);
        let tx_c = B256::repeat_byte(0xCC);

        // Empty cache returns None
        assert!(cache.get_resumable_state(100, &[tx_a, tx_b]).is_none());

        // Update cache with [A, B]
        cache.update(100, vec![tx_a, tx_b], BundleState::default(), vec![]);

        // Wrong block number returns None
        assert!(cache.get_resumable_state(101, &[tx_a, tx_b]).is_none());

        // Exact match returns state
        let result = cache.get_resumable_state(100, &[tx_a, tx_b]);
        assert!(result.is_some());
        let (_, _, skip) = result.unwrap();
        assert_eq!(skip, 2);

        // Continuation returns state (can skip cached txs)
        let result = cache.get_resumable_state(100, &[tx_a, tx_b, tx_c]);
        assert!(result.is_some());
        let (_, _, skip) = result.unwrap();
        assert_eq!(skip, 2);

        // Partial match (reorg) returns None - can't use partial cache
        assert!(cache.get_resumable_state(100, &[tx_a, tx_c]).is_none());
    }
}
