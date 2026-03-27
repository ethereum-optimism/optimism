//! OP-specific transaction pool wrapper that filters interop transactions during reorg
//! reinsertion.
//!
//! See [`OpPool`] for details.

use std::{
    fmt,
    sync::{
        Arc,
        atomic::{AtomicBool, Ordering},
    },
};

use alloy_eips::eip7594::BlobTransactionSidecarVariant;
use alloy_primitives::{Address, B256, TxHash};
use metrics::Counter;
use reth_metrics::Metrics;
use reth_transaction_pool::{
    AllPoolTransactions, AllTransactionsEvents, BestTransactions, BestTransactionsAttributes,
    BlobStoreError, BlockInfo, GetPooledTransactionLimit, NewTransactionEvent, PoolResult,
    PoolSize, PoolTransaction, PoolUpdateKind, PropagatedTransactions, TransactionEvents,
    TransactionListenerKind, TransactionOrigin, TransactionPool, TransactionPoolExt,
    ValidPoolTransaction,
};
use tokio::sync::mpsc::Receiver;

use crate::supervisor::CROSS_L2_INBOX_ADDRESS;

/// Transaction pool wrapper that filters interop transactions during reorg
/// reinsertion. Delegates all other operations to the inner pool.
///
/// On each reorg, the wrapper arms a one-shot filter. The next
/// `add_external_transactions` call consumes the one-shot and filters out
/// any interop transactions in that batch. Subsequent calls pass through
/// unmodified until the next reorg arms the filter again.
///
/// This wrapper is OP-specific. It assumes the inner pool's transaction type
/// supports access-list inspection (`alloy_consensus::Transaction`).
#[derive(Clone)]
pub struct OpPool<P> {
    /// The wrapped inner pool.
    inner: P,
    /// Whether the reorg interop filter is enabled for this pool instance.
    ///
    /// Set at pool construction time using the same startup-time gate as the
    /// existing interop maintenance task.
    enabled: bool,
    /// Shared state for reorg tracking. Wrapped in Arc so clones share state.
    reorg_state: Arc<ReorgFilterState>,
    /// Metrics for reorg filtering.
    metrics: OpPoolMetrics,
}

/// Reorg filter state, shared across `OpPool` clones.
struct ReorgFilterState {
    /// One-shot post-reorg filter flag. Set to `false` (armed) on each reorg,
    /// set to `true` (consumed) by the first `add_external_transactions` call
    /// after the reorg. This is intentionally consumed even if the batch
    /// contains no interop txs, because a given reorg may legitimately have
    /// nothing to filter.
    ///
    /// This is best-effort: it filters the first `add_external_transactions`
    /// call after a reorg, which is typically the maintain task's reinsertion
    /// batch. If a concurrent P2P batch races and consumes the one-shot first,
    /// that batch gets filtered instead. This is acceptable because interop txs
    /// are system-managed and can be resent.
    filter_armed: AtomicBool,
}

/// Metrics for the [`OpPool`] reorg interop filter.
///
/// `Clone` is required because `OpPool` derives Clone and contains this.
/// `Debug` is NOT derived — `metrics::Counter` does not implement `Debug`.
#[derive(Metrics, Clone)]
#[metrics(scope = "transaction_pool")]
struct OpPoolMetrics {
    /// Number of interop transactions filtered during reorg reinsertion
    reorg_interop_txs_filtered: Counter,
}

impl<P: fmt::Debug> fmt::Debug for OpPool<P> {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("OpPool")
            .field("inner", &self.inner)
            .field("enabled", &self.enabled)
            .field("filter_armed", &self.reorg_state.filter_armed.load(Ordering::Relaxed))
            .finish()
    }
}

impl<P> OpPool<P> {
    /// Wraps an inner pool with interop reorg filtering.
    ///
    /// When `enabled` is `false`, the wrapper is fully transparent — no reorg
    /// state is tracked and no filtering ever fires.
    pub fn new(inner: P, enabled: bool) -> Self {
        Self {
            inner,
            enabled,
            reorg_state: Arc::new(ReorgFilterState {
                // Not armed at construction time.
                filter_armed: AtomicBool::new(false),
            }),
            metrics: OpPoolMetrics::default(),
        }
    }
}

/// Returns true if the transaction's access list targets `CROSS_L2_INBOX_ADDRESS`
/// with at least one storage key.
///
/// Equivalent to op-geth's `len(interoptypes.TxToInteropAccessList(tx)) > 0`
/// in `core/types/interoptypes/interop.go:88-103`.
fn is_interop_tx<T>(tx: &T) -> bool
where
    T: PoolTransaction + alloy_consensus::Transaction,
{
    tx.access_list()
        .map(|al| {
            al.iter()
                .any(|item| item.address == CROSS_L2_INBOX_ADDRESS && !item.storage_keys.is_empty())
        })
        .unwrap_or(false)
}

impl<P> OpPool<P>
where
    P: TransactionPool,
    P::Transaction: alloy_consensus::Transaction,
{
    /// Atomically consumes the one-shot reorg filter.
    ///
    /// Returns `true` exactly once per reorg (until re-armed in
    /// `on_canonical_state_change`), even under concurrency.
    fn consume_filter(&self) -> bool {
        self.reorg_state
            .filter_armed
            .compare_exchange(true, false, Ordering::AcqRel, Ordering::Acquire)
            .is_ok()
    }

    /// Returns true if interop filtering should fire on this
    /// `add_external_transactions` call.
    ///
    /// The filter fires exactly once per reorg: the first
    /// `add_external_transactions` call after a reorg consumes the one-shot.
    fn should_filter(&self) -> bool {
        if !self.enabled {
            return false;
        }
        self.consume_filter()
    }

    /// Filters interop transactions from the batch, logging each removal.
    /// Equivalent to op-geth's `filterInteropTxs()` in `legacypool.go:1512-1522`.
    fn filter_interop_txs(&self, txs: Vec<P::Transaction>) -> Vec<P::Transaction> {
        let before = txs.len();
        let filtered: Vec<_> = txs
            .into_iter()
            .filter(|tx| {
                if is_interop_tx(tx) {
                    tracing::warn!(
                        target: "txpool",
                        hash = %tx.hash(),
                        "Filtering interop tx after reorg"
                    );
                    false
                } else {
                    true
                }
            })
            .collect();
        let removed = before - filtered.len();
        tracing::debug!(
            target: "txpool",
            batch_size = before,
            interop_filtered = removed,
            forwarded = filtered.len(),
            "add_external_transactions: reorg filter consumed"
        );
        if removed > 0 {
            self.metrics.reorg_interop_txs_filtered.increment(removed as u64);
        }
        filtered
    }
}

// ── TransactionPool implementation ──────────────────────────────────────────

/// Macro for delegating sync methods to the inner pool.
macro_rules! delegate {
    // No-arg methods returning a value.
    (fn $name:ident(&self) -> $ret:ty) => {
        fn $name(&self) -> $ret {
            self.inner.$name()
        }
    };
    // Single-arg methods.
    (fn $name:ident(&self, $arg:ident : $arg_ty:ty) -> $ret:ty) => {
        fn $name(&self, $arg: $arg_ty) -> $ret {
            self.inner.$name($arg)
        }
    };
    // Two-arg methods.
    (fn $name:ident(&self, $a1:ident : $a1_ty:ty, $a2:ident : $a2_ty:ty) -> $ret:ty) => {
        fn $name(&self, $a1: $a1_ty, $a2: $a2_ty) -> $ret {
            self.inner.$name($a1, $a2)
        }
    };
    // Three-arg methods.
    (
        fn
        $name:ident(&self, $a1:ident : $a1_ty:ty, $a2:ident : $a2_ty:ty, $a3:ident : $a3_ty:ty) ->
        $ret:ty
    ) => {
        fn $name(&self, $a1: $a1_ty, $a2: $a2_ty, $a3: $a3_ty) -> $ret {
            self.inner.$name($a1, $a2, $a3)
        }
    };
}

impl<P> TransactionPool for OpPool<P>
where
    P: TransactionPool,
    P::Transaction: alloy_consensus::Transaction,
{
    type Transaction = P::Transaction;

    // ── Intercepted method ──────────────────────────────────────────────

    async fn add_external_transactions(
        &self,
        transactions: Vec<Self::Transaction>,
    ) -> Vec<PoolResult<reth_transaction_pool::pool::AddedTransactionOutcome>> {
        let txs = if self.should_filter() {
            self.filter_interop_txs(transactions)
        } else {
            tracing::trace!(
                target: "txpool",
                batch_size = transactions.len(),
                "add_external_transactions: filter not armed, passing through"
            );
            transactions
        };
        self.inner.add_external_transactions(txs).await
    }

    // ── Delegated async methods ─────────────────────────────────────────

    async fn add_transaction_and_subscribe(
        &self,
        origin: TransactionOrigin,
        transaction: Self::Transaction,
    ) -> PoolResult<TransactionEvents> {
        self.inner.add_transaction_and_subscribe(origin, transaction).await
    }

    async fn add_transaction(
        &self,
        origin: TransactionOrigin,
        transaction: Self::Transaction,
    ) -> PoolResult<reth_transaction_pool::pool::AddedTransactionOutcome> {
        self.inner.add_transaction(origin, transaction).await
    }

    async fn add_transactions(
        &self,
        origin: TransactionOrigin,
        transactions: Vec<Self::Transaction>,
    ) -> Vec<PoolResult<reth_transaction_pool::pool::AddedTransactionOutcome>> {
        self.inner.add_transactions(origin, transactions).await
    }

    async fn add_transactions_with_origins(
        &self,
        transactions: impl IntoIterator<Item = (TransactionOrigin, Self::Transaction)> + Send,
    ) -> Vec<PoolResult<reth_transaction_pool::pool::AddedTransactionOutcome>> {
        self.inner.add_transactions_with_origins(transactions).await
    }

    // ── Delegated sync methods ──────────────────────────────────────────

    delegate!(fn pool_size(&self) -> PoolSize);
    delegate!(fn block_info(&self) -> BlockInfo);

    fn transaction_event_listener(&self, tx_hash: TxHash) -> Option<TransactionEvents> {
        self.inner.transaction_event_listener(tx_hash)
    }

    fn all_transactions_event_listener(&self) -> AllTransactionsEvents<Self::Transaction> {
        self.inner.all_transactions_event_listener()
    }

    fn pending_transactions_listener_for(&self, kind: TransactionListenerKind) -> Receiver<TxHash> {
        self.inner.pending_transactions_listener_for(kind)
    }

    fn blob_transaction_sidecars_listener(
        &self,
    ) -> Receiver<reth_transaction_pool::NewBlobSidecar> {
        self.inner.blob_transaction_sidecars_listener()
    }

    fn new_transactions_listener_for(
        &self,
        kind: TransactionListenerKind,
    ) -> Receiver<NewTransactionEvent<Self::Transaction>> {
        self.inner.new_transactions_listener_for(kind)
    }

    delegate!(fn pooled_transaction_hashes(&self) -> Vec<TxHash>);
    delegate!(fn pooled_transaction_hashes_max(&self, max: usize) -> Vec<TxHash>);
    delegate!(fn pooled_transactions(&self) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>>);
    delegate!(fn pooled_transactions_max(&self, max: usize) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>>);

    fn get_pooled_transaction_elements(
        &self,
        tx_hashes: Vec<TxHash>,
        limit: GetPooledTransactionLimit,
    ) -> Vec<<Self::Transaction as PoolTransaction>::Pooled> {
        self.inner.get_pooled_transaction_elements(tx_hashes, limit)
    }

    fn get_pooled_transaction_element(
        &self,
        tx_hash: TxHash,
    ) -> Option<reth_primitives_traits::Recovered<<Self::Transaction as PoolTransaction>::Pooled>>
    {
        self.inner.get_pooled_transaction_element(tx_hash)
    }

    fn best_transactions(
        &self,
    ) -> Box<dyn BestTransactions<Item = Arc<ValidPoolTransaction<Self::Transaction>>>> {
        self.inner.best_transactions()
    }

    fn best_transactions_with_attributes(
        &self,
        best_transactions_attributes: BestTransactionsAttributes,
    ) -> Box<dyn BestTransactions<Item = Arc<ValidPoolTransaction<Self::Transaction>>>> {
        self.inner.best_transactions_with_attributes(best_transactions_attributes)
    }

    delegate!(fn pending_transactions(&self) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>>);
    delegate!(fn pending_transactions_max(&self, max: usize) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>>);
    delegate!(fn queued_transactions(&self) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>>);
    delegate!(fn pending_and_queued_txn_count(&self) -> (usize, usize));
    delegate!(fn all_transactions(&self) -> AllPoolTransactions<Self::Transaction>);
    delegate!(fn all_transaction_hashes(&self) -> Vec<TxHash>);

    fn remove_transactions(
        &self,
        hashes: Vec<TxHash>,
    ) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>> {
        self.inner.remove_transactions(hashes)
    }

    fn remove_transactions_and_descendants(
        &self,
        hashes: Vec<TxHash>,
    ) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>> {
        self.inner.remove_transactions_and_descendants(hashes)
    }

    fn remove_transactions_by_sender(
        &self,
        sender: Address,
    ) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>> {
        self.inner.remove_transactions_by_sender(sender)
    }

    fn prune_transactions(
        &self,
        hashes: Vec<TxHash>,
    ) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>> {
        self.inner.prune_transactions(hashes)
    }

    fn retain_unknown<A>(&self, announcement: &mut A)
    where
        A: reth_eth_wire_types::HandleMempoolData,
    {
        self.inner.retain_unknown(announcement)
    }

    fn get(&self, tx_hash: &TxHash) -> Option<Arc<ValidPoolTransaction<Self::Transaction>>> {
        self.inner.get(tx_hash)
    }

    fn get_all(&self, txs: Vec<TxHash>) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>> {
        self.inner.get_all(txs)
    }

    fn on_propagated(&self, txs: PropagatedTransactions) {
        self.inner.on_propagated(txs)
    }

    fn get_transactions_by_sender(
        &self,
        sender: Address,
    ) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>> {
        self.inner.get_transactions_by_sender(sender)
    }

    fn get_pending_transactions_with_predicate(
        &self,
        predicate: impl FnMut(&ValidPoolTransaction<Self::Transaction>) -> bool,
    ) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>> {
        self.inner.get_pending_transactions_with_predicate(predicate)
    }

    fn get_pending_transactions_by_sender(
        &self,
        sender: Address,
    ) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>> {
        self.inner.get_pending_transactions_by_sender(sender)
    }

    fn get_queued_transactions_by_sender(
        &self,
        sender: Address,
    ) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>> {
        self.inner.get_queued_transactions_by_sender(sender)
    }

    fn get_highest_transaction_by_sender(
        &self,
        sender: Address,
    ) -> Option<Arc<ValidPoolTransaction<Self::Transaction>>> {
        self.inner.get_highest_transaction_by_sender(sender)
    }

    fn get_highest_consecutive_transaction_by_sender(
        &self,
        sender: Address,
        on_chain_nonce: u64,
    ) -> Option<Arc<ValidPoolTransaction<Self::Transaction>>> {
        self.inner.get_highest_consecutive_transaction_by_sender(sender, on_chain_nonce)
    }

    fn get_transaction_by_sender_and_nonce(
        &self,
        sender: Address,
        nonce: u64,
    ) -> Option<Arc<ValidPoolTransaction<Self::Transaction>>> {
        self.inner.get_transaction_by_sender_and_nonce(sender, nonce)
    }

    fn get_transactions_by_origin(
        &self,
        origin: TransactionOrigin,
    ) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>> {
        self.inner.get_transactions_by_origin(origin)
    }

    fn get_pending_transactions_by_origin(
        &self,
        origin: TransactionOrigin,
    ) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>> {
        self.inner.get_pending_transactions_by_origin(origin)
    }

    delegate!(fn unique_senders(&self) -> alloy_primitives::map::AddressSet);

    fn get_blob(
        &self,
        tx_hash: TxHash,
    ) -> Result<Option<Arc<BlobTransactionSidecarVariant>>, BlobStoreError> {
        self.inner.get_blob(tx_hash)
    }

    fn get_all_blobs(
        &self,
        tx_hashes: Vec<TxHash>,
    ) -> Result<Vec<(TxHash, Arc<BlobTransactionSidecarVariant>)>, BlobStoreError> {
        self.inner.get_all_blobs(tx_hashes)
    }

    fn get_all_blobs_exact(
        &self,
        tx_hashes: Vec<TxHash>,
    ) -> Result<Vec<Arc<BlobTransactionSidecarVariant>>, BlobStoreError> {
        self.inner.get_all_blobs_exact(tx_hashes)
    }

    fn get_blobs_for_versioned_hashes_v1(
        &self,
        versioned_hashes: &[B256],
    ) -> Result<Vec<Option<alloy_eips::eip4844::BlobAndProofV1>>, BlobStoreError> {
        self.inner.get_blobs_for_versioned_hashes_v1(versioned_hashes)
    }

    fn get_blobs_for_versioned_hashes_v2(
        &self,
        versioned_hashes: &[B256],
    ) -> Result<Option<Vec<alloy_eips::eip4844::BlobAndProofV2>>, BlobStoreError> {
        self.inner.get_blobs_for_versioned_hashes_v2(versioned_hashes)
    }

    fn get_blobs_for_versioned_hashes_v3(
        &self,
        versioned_hashes: &[B256],
    ) -> Result<Vec<Option<alloy_eips::eip4844::BlobAndProofV2>>, BlobStoreError> {
        self.inner.get_blobs_for_versioned_hashes_v3(versioned_hashes)
    }
}

// ── TransactionPoolExt implementation ───────────────────────────────────────

impl<P> TransactionPoolExt for OpPool<P>
where
    P: TransactionPoolExt,
    P::Transaction: alloy_consensus::Transaction,
{
    type Block = P::Block;

    fn on_canonical_state_change(
        &self,
        update: reth_transaction_pool::CanonicalStateUpdate<'_, Self::Block>,
    ) {
        if self.enabled && update.update_kind == PoolUpdateKind::Reorg {
            self.reorg_state.filter_armed.store(true, Ordering::Release);
            tracing::debug!(
                target: "txpool",
                "Reorg detected, interop filter armed"
            );
        }
        // MUST delegate to inner pool — inner pool handles fee updates,
        // mined tx removal, block info tracking, etc.
        self.inner.on_canonical_state_change(update);
    }

    fn set_block_info(&self, info: BlockInfo) {
        self.inner.set_block_info(info);
    }

    fn update_accounts(&self, accounts: Vec<reth_execution_types::ChangedAccount>) {
        self.inner.update_accounts(accounts);
    }

    fn delete_blob(&self, tx: B256) {
        self.inner.delete_blob(tx);
    }

    fn delete_blobs(&self, txs: Vec<B256>) {
        self.inner.delete_blobs(txs);
    }

    fn cleanup_blobs(&self) {
        self.inner.cleanup_blobs();
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_eips::eip2930::{AccessList, AccessListItem};
    use alloy_primitives::address;
    use reth_transaction_pool::test_utils::MockTransaction;
    use std::sync::atomic::Ordering;

    /// Creates a mock EIP-1559 transaction with the given access list.
    fn mock_tx_with_access_list(access_list: AccessList) -> MockTransaction {
        let mut tx = MockTransaction::eip1559();
        tx.set_accesslist(access_list);
        tx
    }

    /// Creates a mock interop transaction (access list targeting `CROSS_L2_INBOX_ADDRESS`
    /// with at least one storage key).
    fn mock_interop_tx() -> MockTransaction {
        mock_tx_with_access_list(AccessList(vec![AccessListItem {
            address: CROSS_L2_INBOX_ADDRESS,
            storage_keys: vec![B256::ZERO],
        }]))
    }

    /// Creates a mock normal (non-interop) transaction.
    fn mock_normal_tx() -> MockTransaction {
        MockTransaction::eip1559()
    }

    // ── is_interop_tx unit tests ────────────────────────────────────────

    #[test]
    fn test_is_interop_tx_no_access_list() {
        let tx = MockTransaction::eip1559();
        assert!(!is_interop_tx(&tx));
    }

    #[test]
    fn test_is_interop_tx_random_address() {
        let tx = mock_tx_with_access_list(AccessList(vec![AccessListItem {
            address: address!("0x1111111111111111111111111111111111111111"),
            storage_keys: vec![B256::ZERO],
        }]));
        assert!(!is_interop_tx(&tx));
    }

    #[test]
    fn test_is_interop_tx_cross_l2_inbox() {
        let tx = mock_interop_tx();
        assert!(is_interop_tx(&tx));
    }

    #[test]
    fn test_is_interop_tx_multiple_entries_one_matching() {
        let tx = mock_tx_with_access_list(AccessList(vec![
            AccessListItem {
                address: address!("0x1111111111111111111111111111111111111111"),
                storage_keys: vec![B256::ZERO],
            },
            AccessListItem { address: CROSS_L2_INBOX_ADDRESS, storage_keys: vec![B256::ZERO] },
        ]));
        assert!(is_interop_tx(&tx));
    }

    #[test]
    fn test_is_interop_tx_cross_l2_inbox_empty_storage_keys() {
        // Matches op-geth exactly: TxToInteropAccessList returns empty slice,
        // so len() > 0 is false.
        let tx = mock_tx_with_access_list(AccessList(vec![AccessListItem {
            address: CROSS_L2_INBOX_ADDRESS,
            storage_keys: vec![],
        }]));
        assert!(!is_interop_tx(&tx));
    }

    // ── OpPool filtering tests ──────────────────────────────────────────

    /// Helper to arm the one-shot filter (simulates a reorg).
    fn arm_filter(
        pool: &OpPool<reth_transaction_pool::noop::NoopTransactionPool<MockTransaction>>,
    ) {
        pool.reorg_state.filter_armed.store(true, Ordering::Release);
    }

    #[test]
    fn test_should_filter_disabled() {
        let pool: OpPool<reth_transaction_pool::noop::NoopTransactionPool<MockTransaction>> =
            OpPool::new(reth_transaction_pool::noop::NoopTransactionPool::new(), false);

        arm_filter(&pool);
        assert!(!pool.should_filter());
    }

    #[test]
    fn test_should_filter_no_reorg() {
        let pool: OpPool<reth_transaction_pool::noop::NoopTransactionPool<MockTransaction>> =
            OpPool::new(reth_transaction_pool::noop::NoopTransactionPool::new(), true);
        // Not armed — should not filter.
        assert!(!pool.should_filter());
    }

    #[test]
    fn test_should_filter_armed_fires_once() {
        let pool: OpPool<reth_transaction_pool::noop::NoopTransactionPool<MockTransaction>> =
            OpPool::new(reth_transaction_pool::noop::NoopTransactionPool::new(), true);

        arm_filter(&pool);
        assert!(pool.should_filter());
        // Second call: already consumed.
        assert!(!pool.should_filter());
    }

    #[test]
    fn test_should_filter_rearms_on_new_reorg() {
        let pool: OpPool<reth_transaction_pool::noop::NoopTransactionPool<MockTransaction>> =
            OpPool::new(reth_transaction_pool::noop::NoopTransactionPool::new(), true);

        arm_filter(&pool);
        assert!(pool.should_filter());
        assert!(!pool.should_filter());

        // Re-arm (new reorg).
        arm_filter(&pool);
        assert!(pool.should_filter());
        assert!(!pool.should_filter());
    }

    #[test]
    fn test_filter_interop_txs_filters_correctly() {
        let pool: OpPool<reth_transaction_pool::noop::NoopTransactionPool<MockTransaction>> =
            OpPool::new(reth_transaction_pool::noop::NoopTransactionPool::new(), true);

        let normal = mock_normal_tx();
        let interop = mock_interop_tx();
        let normal_hash = *normal.hash();

        let result = pool.filter_interop_txs(vec![normal, interop]);

        assert_eq!(result.len(), 1);
        assert_eq!(*result[0].hash(), normal_hash);
    }

    #[test]
    fn test_filter_interop_txs_all_normal_pass_through() {
        let pool: OpPool<reth_transaction_pool::noop::NoopTransactionPool<MockTransaction>> =
            OpPool::new(reth_transaction_pool::noop::NoopTransactionPool::new(), true);

        let tx1 = mock_normal_tx();
        let tx2 = mock_normal_tx();

        let result = pool.filter_interop_txs(vec![tx1, tx2]);
        assert_eq!(result.len(), 2);
    }

    // ── End-to-end async tests through add_external_transactions ────────
    //
    // These use NoopTransactionPool as the inner pool. The number of results
    // returned equals the number of transactions forwarded (one Err per tx
    // since NoopTransactionPool rejects everything).

    #[tokio::test]
    async fn test_reorg_filters_interop_txs() {
        let pool: OpPool<reth_transaction_pool::noop::NoopTransactionPool<MockTransaction>> =
            OpPool::new(reth_transaction_pool::noop::NoopTransactionPool::new(), true);

        arm_filter(&pool);

        let normal = mock_normal_tx();
        let interop = mock_interop_tx();

        let results = pool.add_external_transactions(vec![normal, interop]).await;
        assert_eq!(results.len(), 1, "interop tx should have been filtered");
    }

    #[tokio::test]
    async fn test_no_reorg_passes_all_txs() {
        let pool: OpPool<reth_transaction_pool::noop::NoopTransactionPool<MockTransaction>> =
            OpPool::new(reth_transaction_pool::noop::NoopTransactionPool::new(), true);

        let normal = mock_normal_tx();
        let interop = mock_interop_tx();

        let results = pool.add_external_transactions(vec![normal, interop]).await;
        assert_eq!(results.len(), 2, "no filtering without a reorg");
    }

    #[tokio::test]
    async fn test_filter_disabled_at_construction_is_transparent() {
        let pool: OpPool<reth_transaction_pool::noop::NoopTransactionPool<MockTransaction>> =
            OpPool::new(reth_transaction_pool::noop::NoopTransactionPool::new(), false);

        arm_filter(&pool);

        let normal = mock_normal_tx();
        let interop = mock_interop_tx();

        let results = pool.add_external_transactions(vec![normal, interop]).await;
        assert_eq!(results.len(), 2, "disabled pool should be transparent");
    }

    #[tokio::test]
    async fn test_one_shot_consumed_then_passes_through() {
        let pool: OpPool<reth_transaction_pool::noop::NoopTransactionPool<MockTransaction>> =
            OpPool::new(reth_transaction_pool::noop::NoopTransactionPool::new(), true);

        arm_filter(&pool);

        // First call: one-shot fires, filters interop tx.
        let results =
            pool.add_external_transactions(vec![mock_normal_tx(), mock_interop_tx()]).await;
        assert_eq!(results.len(), 1, "one-shot should filter interop tx");

        // Second call: one-shot consumed, both pass through.
        let results2 =
            pool.add_external_transactions(vec![mock_normal_tx(), mock_interop_tx()]).await;
        assert_eq!(results2.len(), 2, "after one-shot consumed, no more filtering");
    }

    #[tokio::test]
    async fn test_add_transaction_unaffected_by_reorg() {
        let pool: OpPool<reth_transaction_pool::noop::NoopTransactionPool<MockTransaction>> =
            OpPool::new(reth_transaction_pool::noop::NoopTransactionPool::new(), true);

        arm_filter(&pool);

        let interop = mock_interop_tx();

        // add_transaction (RPC/Local path) is never affected by the reorg filter.
        let result = pool.add_transaction(TransactionOrigin::Local, interop).await;
        assert!(result.is_err(), "NoopTransactionPool always rejects, proving tx was forwarded");
    }
}
