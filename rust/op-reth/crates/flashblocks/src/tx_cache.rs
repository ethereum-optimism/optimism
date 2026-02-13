//! Transaction execution caching for flashblock building.
//!
//! When flashblocks arrive incrementally, each new flashblock triggers a rebuild of pending
//! state from all transactions in the sequence. Without caching, this means re-reading
//! state from disk for accounts/storage that were already loaded in previous builds.
//!
//! # Approach
//!
//! This module caches the cumulative bundle state from previous executions. When the next
//! flashblock arrives, if its transaction list is a continuation of the cached list, the
//! cached bundle can be used as a **prestate** for the State builder. This avoids redundant
//! disk reads for accounts/storage that were already modified.
//!
//! **Important**: Prefix transaction skipping is only safe when the incoming transaction list
//! fully extends the cached list. In that case, callers can execute only the uncached suffix
//! and stitch in the cached prefix receipts/metadata.
//!
//! The cache stores:
//! - Ordered list of executed transaction hashes (for prefix matching)
//! - Cumulative bundle state after all cached transactions (used as prestate)
//! - Cumulative receipts for all cached transactions (for future optimization)
//! - Block-level execution metadata for cached transactions (gas/requests)
//!
//! # Example
//!
//! ```text
//! Flashblock 0: txs [A, B]
//!   -> Execute A, B from scratch (cold state reads)
//!   -> Cache: txs=[A,B], bundle=state_after_AB
//!
//! Flashblock 1: txs [A, B, C]
//!   -> Prefix [A, B] matches cache
//!   -> Use cached bundle as prestate (warm state)
//!   -> Execute A, B, C (A, B hit prestate cache, faster)
//!   -> Cache: txs=[A,B,C], bundle=state_after_ABC
//!
//! Flashblock 2 (reorg): txs [A, D, E]
//!   -> Prefix [A] matches, but tx[1]=D != B
//!   -> Cached prestate may be partially useful, but diverges
//!   -> Execute A, D, E
//! ```

use alloy_eips::eip7685::Requests;
use alloy_primitives::B256;
use reth_primitives_traits::NodePrimitives;
use reth_revm::db::BundleState;

/// Cached block-level execution metadata for the stored transaction prefix.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct CachedExecutionMeta {
    /// EIP-7685 requests emitted while executing the cached prefix.
    pub requests: Requests,
    /// Total gas used by the cached prefix.
    pub gas_used: u64,
    /// Total blob/DA gas used by the cached prefix.
    pub blob_gas_used: u64,
}

/// Resumable cached state: bundle + receipts + cached prefix length.
pub(crate) type ResumableState<'a, N> =
    (&'a BundleState, &'a [<N as NodePrimitives>::Receipt], usize);

/// Resumable cached state plus execution metadata for the cached prefix.
pub(crate) type ResumableStateWithExecutionMeta<'a, N> =
    (&'a BundleState, &'a [<N as NodePrimitives>::Receipt], &'a Requests, u64, u64, usize);

/// Cache of transaction execution results for a single block.
///
/// Stores cumulative execution state that can be used as a prestate to avoid
/// redundant disk reads when re-executing transactions. The cached bundle provides
/// warm state for accounts/storage already loaded, improving execution performance.
///
/// **Note**: This cache does NOT skip transaction execution - all transactions must
/// still be executed to populate the block body. The cache only optimizes state reads.
///
/// The cache is invalidated when:
/// - A new block starts (different block number)
/// - Parent hash changes for parent-scoped lookups
/// - A reorg is detected (transaction list diverges from cached prefix)
/// - Explicitly cleared
#[derive(Debug)]
pub struct TransactionCache<N: NodePrimitives> {
    /// Block number this cache is valid for.
    block_number: u64,
    /// Parent hash this cache is valid for.
    cached_parent_hash: Option<B256>,
    /// Ordered list of transaction hashes that have been executed.
    executed_tx_hashes: Vec<B256>,
    /// Cumulative bundle state after executing all cached transactions.
    cumulative_bundle: BundleState,
    /// Receipts for all cached transactions, in execution order.
    receipts: Vec<N::Receipt>,
    /// Cached block-level execution metadata.
    execution_meta: CachedExecutionMeta,
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
            cached_parent_hash: None,
            executed_tx_hashes: Vec::new(),
            cumulative_bundle: BundleState::default(),
            receipts: Vec::new(),
            execution_meta: CachedExecutionMeta::default(),
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

    /// Returns the parent hash this cache is valid for, if tracked.
    pub const fn parent_hash(&self) -> Option<B256> {
        self.cached_parent_hash
    }

    /// Checks if this cache is valid for the given block number.
    pub const fn is_valid_for_block(&self, block_number: u64) -> bool {
        self.block_number == block_number
    }

    /// Checks if this cache is valid for the given block number and parent hash.
    pub fn is_valid_for_block_parent(&self, block_number: u64, parent_hash: B256) -> bool {
        self.block_number == block_number && self.cached_parent_hash == Some(parent_hash)
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
        self.execution_meta = CachedExecutionMeta::default();
        self.block_number = 0;
        self.cached_parent_hash = None;
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
    ) -> Option<ResumableState<'_, N>> {
        self.get_resumable_state_with_execution_meta(block_number, tx_hashes)
            .map(|(bundle, receipts, .., skip_count)| (bundle, receipts, skip_count))
    }

    /// Returns cached state and execution metadata for resuming execution if the incoming
    /// transactions have a matching prefix with the cache.
    ///
    /// Returns `Some((bundle, receipts, requests, gas_used, blob_gas_used, skip_count))` if
    /// there's a non-empty matching prefix and the entire cache matches the incoming prefix.
    pub(crate) fn get_resumable_state_with_execution_meta(
        &self,
        block_number: u64,
        tx_hashes: &[B256],
    ) -> Option<ResumableStateWithExecutionMeta<'_, N>> {
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
            &self.execution_meta.requests,
            self.execution_meta.gas_used,
            self.execution_meta.blob_gas_used,
            prefix_len,
        ))
    }

    /// Returns cached state and execution metadata for resuming execution if the incoming
    /// transactions have a matching prefix with the cache and the parent hash matches.
    ///
    /// Returns `Some((bundle, receipts, requests, gas_used, blob_gas_used, skip_count))` if
    /// there's a non-empty matching prefix, the full cache matches the incoming prefix, and the
    /// `(block_number, parent_hash)` tuple matches the cached scope.
    pub(crate) fn get_resumable_state_with_execution_meta_for_parent(
        &self,
        block_number: u64,
        parent_hash: B256,
        tx_hashes: &[B256],
    ) -> Option<ResumableStateWithExecutionMeta<'_, N>> {
        if !self.is_valid_for_block_parent(block_number, parent_hash) || self.is_empty() {
            return None;
        }

        let prefix_len = self.matching_prefix_len(tx_hashes);
        if prefix_len == 0 {
            return None;
        }

        (prefix_len == self.executed_tx_hashes.len()).then_some((
            &self.cumulative_bundle,
            self.receipts.as_slice(),
            &self.execution_meta.requests,
            self.execution_meta.gas_used,
            self.execution_meta.blob_gas_used,
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
        self.update_with_execution_meta(
            block_number,
            tx_hashes,
            bundle,
            receipts,
            CachedExecutionMeta::default(),
        );
    }

    /// Updates the cache with new execution results and block-level metadata.
    pub(crate) fn update_with_execution_meta(
        &mut self,
        block_number: u64,
        tx_hashes: Vec<B256>,
        bundle: BundleState,
        receipts: Vec<N::Receipt>,
        execution_meta: CachedExecutionMeta,
    ) {
        self.block_number = block_number;
        self.cached_parent_hash = None;
        self.executed_tx_hashes = tx_hashes;
        self.cumulative_bundle = bundle;
        self.receipts = receipts;
        self.execution_meta = execution_meta;
    }

    /// Updates the cache with new execution results and block-level metadata, scoped to the
    /// provided parent hash.
    pub(crate) fn update_with_execution_meta_for_parent(
        &mut self,
        block_number: u64,
        parent_hash: B256,
        tx_hashes: Vec<B256>,
        bundle: BundleState,
        receipts: Vec<N::Receipt>,
        execution_meta: CachedExecutionMeta,
    ) {
        self.block_number = block_number;
        self.cached_parent_hash = Some(parent_hash);
        self.executed_tx_hashes = tx_hashes;
        self.cumulative_bundle = bundle;
        self.receipts = receipts;
        self.execution_meta = execution_meta;
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
        assert!(!cache.is_valid_for_block_parent(100, B256::repeat_byte(0x11)));

        // Update for same block doesn't clear
        assert!(!cache.update_for_block(100));

        // Update for different block clears
        assert!(cache.update_for_block(101));
        assert!(cache.is_valid_for_block(101));
        assert!(cache.parent_hash().is_none());
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

    // ==================== E2E Cache Reuse Scenario Tests ====================

    /// Tests the complete E2E cache scenario: fb0 [A,B] → fb1 [A,B,C]
    /// Verifies that cached bundle can be used as prestate for the continuation.
    #[test]
    fn test_e2e_cache_reuse_continuation_scenario() {
        let mut cache = TestCache::new();

        let tx_a = B256::repeat_byte(0xAA);
        let tx_b = B256::repeat_byte(0xBB);
        let tx_c = B256::repeat_byte(0xCC);

        // Simulate fb0: execute [A, B] from scratch
        let fb0_txs = vec![tx_a, tx_b];
        assert!(cache.get_resumable_state(100, &fb0_txs).is_none());

        // After fb0 execution, update cache
        cache.update(100, fb0_txs, BundleState::default(), vec![]);
        assert_eq!(cache.len(), 2);

        // Simulate fb1: [A, B, C] - should resume from cached state
        let fb1_txs = vec![tx_a, tx_b, tx_c];
        let result = cache.get_resumable_state(100, &fb1_txs);
        assert!(result.is_some());
        let (bundle, receipts, skip) = result.unwrap();

        // skip=2 indicates 2 txs are covered by cached state (for logging)
        // Note: All transactions are still executed, skip is informational only
        assert_eq!(skip, 2);
        // Bundle is used as prestate to warm the State builder
        assert!(bundle.state.is_empty()); // Default bundle is empty in test
        assert!(receipts.is_empty()); // No receipts in this test

        // After fb1 execution, update cache with full list
        cache.update(100, fb1_txs, BundleState::default(), vec![]);
        assert_eq!(cache.len(), 3);
    }

    /// Tests reorg scenario: fb0 [A, B] → fb1 [A, D, E]
    /// Verifies that divergent tx list invalidates cache.
    #[test]
    fn test_e2e_cache_reorg_scenario() {
        let mut cache = TestCache::new();

        let tx_a = B256::repeat_byte(0xAA);
        let tx_b = B256::repeat_byte(0xBB);
        let tx_d = B256::repeat_byte(0xDD);
        let tx_e = B256::repeat_byte(0xEE);

        // fb0: execute [A, B]
        cache.update(100, vec![tx_a, tx_b], BundleState::default(), vec![]);

        // fb1 (reorg): [A, D, E] - tx[1] diverges, cannot resume
        let fb1_txs = vec![tx_a, tx_d, tx_e];
        let result = cache.get_resumable_state(100, &fb1_txs);
        assert!(result.is_none()); // Partial match means we can't use cache
    }

    /// Tests multi-flashblock progression within same block:
    /// fb0 [A] → fb1 [A,B] → fb2 [A,B,C]
    ///
    /// Each flashblock can use the previous bundle as prestate for warm state reads.
    /// Note: All transactions are still executed; skip count is for logging only.
    #[test]
    fn test_e2e_multi_flashblock_progression() {
        let mut cache = TestCache::new();

        let tx_a = B256::repeat_byte(0xAA);
        let tx_b = B256::repeat_byte(0xBB);
        let tx_c = B256::repeat_byte(0xCC);

        // fb0: [A]
        cache.update(100, vec![tx_a], BundleState::default(), vec![]);
        assert_eq!(cache.len(), 1);

        // fb1: [A, B] - cached state covers [A] (skip=1 for logging)
        let fb1_txs = vec![tx_a, tx_b];
        let result = cache.get_resumable_state(100, &fb1_txs);
        assert!(result.is_some());
        assert_eq!(result.unwrap().2, 1); // 1 tx covered by cache

        cache.update(100, fb1_txs, BundleState::default(), vec![]);
        assert_eq!(cache.len(), 2);

        // fb2: [A, B, C] - cached state covers [A, B] (skip=2 for logging)
        let fb2_txs = vec![tx_a, tx_b, tx_c];
        let result = cache.get_resumable_state(100, &fb2_txs);
        assert!(result.is_some());
        assert_eq!(result.unwrap().2, 2); // 2 txs covered by cache

        cache.update(100, fb2_txs, BundleState::default(), vec![]);
        assert_eq!(cache.len(), 3);
    }

    /// Tests that cache is invalidated on block number change.
    #[test]
    fn test_e2e_block_transition_clears_cache() {
        let mut cache = TestCache::new();

        let tx_a = B256::repeat_byte(0xAA);
        let tx_b = B256::repeat_byte(0xBB);

        // Block 100: cache [A, B]
        cache.update(100, vec![tx_a, tx_b], BundleState::default(), vec![]);
        assert_eq!(cache.len(), 2);

        // Block 101: same txs shouldn't resume (different block)
        let result = cache.get_resumable_state(101, &[tx_a, tx_b]);
        assert!(result.is_none());

        // Explicit block update clears cache
        cache.update_for_block(101);
        assert!(cache.is_empty());
    }

    /// Tests cache behavior with empty transaction list.
    #[test]
    fn test_cache_empty_transactions() {
        let mut cache = TestCache::new();

        // Empty flashblock (only system tx, no user txs)
        cache.update(100, vec![], BundleState::default(), vec![]);
        assert!(cache.is_empty());

        // Can't resume from empty cache
        let tx_a = B256::repeat_byte(0xAA);
        assert!(cache.get_resumable_state(100, &[tx_a]).is_none());
    }

    /// Documents the semantics of `skip_count`.
    ///
    /// A resumable state is only returned when the incoming transaction list fully extends the
    /// cached list. In that case, `skip_count` is the number of prefix transactions covered by
    /// cached execution output.
    #[test]
    fn test_skip_count_matches_cached_prefix_len() {
        let mut cache = TestCache::new();

        let tx_a = B256::repeat_byte(0xAA);
        let tx_b = B256::repeat_byte(0xBB);
        let tx_c = B256::repeat_byte(0xCC);

        // Cache state after executing [A, B]
        cache.update(100, vec![tx_a, tx_b], BundleState::default(), vec![]);

        // get_resumable_state returns skip=2 for prefix [A, B]
        let result = cache.get_resumable_state(100, &[tx_a, tx_b, tx_c]);
        assert!(result.is_some());
        let (bundle, _receipts, skip_count) = result.unwrap();

        // skip_count indicates cached prefix length
        assert_eq!(skip_count, 2);

        // The bundle is the important part - used as resumable prestate.
        assert!(bundle.state.is_empty()); // Default in test, real one has state
    }

    /// Tests that receipts are properly cached and returned.
    #[test]
    fn test_cache_preserves_receipts() {
        use op_alloy_consensus::OpReceipt;
        use reth_optimism_primitives::OpPrimitives;

        let mut cache: TransactionCache<OpPrimitives> = TransactionCache::new();

        let tx_a = B256::repeat_byte(0xAA);
        let tx_b = B256::repeat_byte(0xBB);

        // Create mock receipts
        let receipt_a = OpReceipt::Legacy(alloy_consensus::Receipt {
            status: alloy_consensus::Eip658Value::Eip658(true),
            cumulative_gas_used: 21000,
            logs: vec![],
        });
        let receipt_b = OpReceipt::Legacy(alloy_consensus::Receipt {
            status: alloy_consensus::Eip658Value::Eip658(true),
            cumulative_gas_used: 42000,
            logs: vec![],
        });

        cache.update(100, vec![tx_a, tx_b], BundleState::default(), vec![receipt_a, receipt_b]);

        // Verify receipts are preserved
        assert_eq!(cache.receipts().len(), 2);

        // On resumable state, receipts are returned
        let tx_c = B256::repeat_byte(0xCC);
        let result = cache.get_resumable_state(100, &[tx_a, tx_b, tx_c]);
        assert!(result.is_some());
        let (_, receipts, _) = result.unwrap();
        assert_eq!(receipts.len(), 2);
    }

    #[test]
    fn test_cache_preserves_execution_meta() {
        let mut cache = TestCache::new();

        let tx_a = B256::repeat_byte(0xAA);
        let tx_b = B256::repeat_byte(0xBB);
        let tx_c = B256::repeat_byte(0xCC);

        let mut requests = Requests::default();
        requests.push_request_with_type(0x01, [0xAA, 0xBB]);

        cache.update_with_execution_meta(
            100,
            vec![tx_a, tx_b],
            BundleState::default(),
            vec![],
            CachedExecutionMeta {
                requests: requests.clone(),
                gas_used: 42_000,
                blob_gas_used: 123,
            },
        );

        let resumable = cache.get_resumable_state_with_execution_meta(100, &[tx_a, tx_b, tx_c]);
        assert!(resumable.is_some());
        let (_, _, cached_requests, gas_used, blob_gas_used, skip_count) = resumable.unwrap();
        assert_eq!(skip_count, 2);
        assert_eq!(gas_used, 42_000);
        assert_eq!(blob_gas_used, 123);
        assert_eq!(cached_requests, &requests);
    }

    #[test]
    fn test_cache_parent_scoping() {
        let mut cache = TestCache::new();

        let tx_a = B256::repeat_byte(0xAA);
        let tx_b = B256::repeat_byte(0xBB);
        let tx_c = B256::repeat_byte(0xCC);
        let parent_a = B256::repeat_byte(0x11);
        let parent_b = B256::repeat_byte(0x22);

        cache.update_with_execution_meta_for_parent(
            100,
            parent_a,
            vec![tx_a, tx_b],
            BundleState::default(),
            vec![],
            CachedExecutionMeta {
                requests: Requests::default(),
                gas_used: 42_000,
                blob_gas_used: 0,
            },
        );

        // Matching block + parent should hit.
        let hit = cache.get_resumable_state_with_execution_meta_for_parent(
            100,
            parent_a,
            &[tx_a, tx_b, tx_c],
        );
        assert!(hit.is_some());

        // Same block but different parent should miss.
        let miss = cache.get_resumable_state_with_execution_meta_for_parent(
            100,
            parent_b,
            &[tx_a, tx_b, tx_c],
        );
        assert!(miss.is_none());
    }
}
