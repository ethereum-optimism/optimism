//! OP-specific transaction pool wrapper
//!
//! See [`OpPool`] for details.

use std::{
    fmt,
    sync::{
        Arc, RwLock,
        atomic::{AtomicBool, Ordering},
    },
    time::{Duration, Instant},
};

use alloy_consensus::Transaction;
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
use tracing::debug;

use crate::supervisor::CROSS_L2_INBOX_ADDRESS;

/// Duration after a reorg during which all interop transactions are filtered
/// from `add_external_transactions`.
/// It is safe to drop the interop transactions during reorg while the chain
/// stabilizes, since those transactions are system-generated and can be
/// reinjected after the chain recovers.
/// This window exists to prevent potential race conditions that could lead to
/// interop transactions not being filtered after a reorg, which could cause
/// the supernode to enter a rewind loop.
const REORG_WINDOW: Duration = Duration::from_secs(30);

/// Optimism-specific transaction pool wrapper. Delegates most operations
/// directly to the inner transaction pool.
///
/// This type derives [`Clone`] and is expected to be cloned across tasks. Clones
/// share underlying state via `Arc` internally. Make sure any mutable state
/// added to this struct is shared across clones (e.g. via `Arc`).
///
/// Currently, the only difference is a transaction filter that stops
/// interop transactions from being added to the pool after a reorg.
#[derive(Clone)]
pub struct OpPool<P>
where
    P: TransactionPool,
    P::Transaction: Transaction,
{
    /// The wrapped inner pool.
    inner: P,
    /// Shared state for reorg tracking. Wrapped in Arc so clones share state.
    /// If the interop filter is disabled at pool creation, this is `None`.
    reorg_state: Option<Arc<ReorgFilterState>>,
    /// Metrics for reorg filtering.
    metrics: OpPoolMetrics,
}

/// Reorg filter state, shared across `OpPool` clones.
///
/// On each reorg, the wrapper records the reorg time and arms a one-shot
/// fallback. Every `add_external_transactions` call within the `REORG_WINDOW`
/// filters interop transactions. After at least one filtered batch, we clear
/// the one-shot atomic flag.
/// still allows one best-effort filter pass if it was not already consumed.
#[derive(Debug)]
struct ReorgFilterState {
    /// Timestamp of the most recent reorg observed via `on_canonical_state_change`.
    last_reorg_at: RwLock<Option<Instant>>,
    /// One-shot post-reorg filter flag. Set to `false` (armed) on each reorg,
    /// set to `true` (consumed) by the first `add_external_transactions` call
    /// after the active reorg window has expired. This is intentionally
    /// consumed even if the batch contains no interop txs, because a given
    /// reorg may legitimately have nothing to filter.
    ///
    /// This is safe because we know there will always be at least one
    /// `add_external_transactions` call after a reorg inside reth's
    /// transaction pool maintenance task.
    filter_armed: AtomicBool,
}

impl ReorgFilterState {
    /// Atomically consumes the one-shot reorg fallback filter.
    ///
    /// Returns `true` exactly once per reorg (until re-armed in
    /// `on_canonical_state_change`), even under concurrency.
    fn consume_filter(&self) -> bool {
        self.filter_armed.compare_exchange(true, false, Ordering::AcqRel, Ordering::Acquire).is_ok()
    }

    fn in_reorg_window(&self) -> bool {
        self.last_reorg_at.read().unwrap().is_some_and(|reorg_at| reorg_at.elapsed() < REORG_WINDOW)
    }
}

/// Metrics for the [`OpPool`] reorg interop filter.
#[derive(Metrics, Clone)]
#[metrics(scope = "transaction_pool")]
struct OpPoolMetrics {
    /// Number of interop transactions filtered by the reorg filter
    reorg_interop_txs_filtered: Counter,
}

impl<P> fmt::Debug for OpPool<P>
where
    P: fmt::Debug + TransactionPool,
    P::Transaction: Transaction,
{
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("OpPool")
            .field("inner", &self.inner)
            .field("reorg_state", &self.reorg_state)
            .finish()
    }
}

/// Returns true if the transaction's access list targets `CROSS_L2_INBOX_ADDRESS`
/// with at least one storage key.
fn is_interop_tx<T>(tx: &T) -> bool
where
    T: PoolTransaction + Transaction,
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
    P::Transaction: Transaction,
{
    /// Wraps an inner pool with interop reorg filtering.
    ///
    /// When `interop_enabled` is `false`, the wrapper is fully transparent.
    pub fn new(inner: P, interop_enabled: bool) -> Self {
        let reorg_state = interop_enabled.then(|| {
            Arc::new(ReorgFilterState {
                last_reorg_at: RwLock::new(None),
                // Not armed at construction time.
                filter_armed: AtomicBool::new(false),
            })
        });
        Self { inner, reorg_state, metrics: OpPoolMetrics::default() }
    }

    /// Returns true if interop filtering should fire on this
    /// `add_external_transactions` call.
    ///
    /// Filtering stays active for the full `REORG_WINDOW` after a reorg.
    /// Outside the window, a one-shot fallback still filters the first
    /// eligible batch if the fallback has not been consumed yet.
    fn should_filter_interop(&self) -> bool {
        if let Some(state) = &self.reorg_state {
            if state.in_reorg_window() {
                // Prevent an extra post-window filter pass after the window has
                // already covered the normal reinsertion flow.
                let _ = state.consume_filter();
                return true;
            }

            state.consume_filter()
        } else {
            false
        }
    }

    /// Filters interop transactions from the batch, logging each removal.
    fn filter_interop_txs(&self, txs: Vec<P::Transaction>) -> Vec<P::Transaction> {
        let before = txs.len();
        let filtered: Vec<_> = txs
            .into_iter()
            .filter(|tx| {
                if is_interop_tx(tx) {
                    debug!(
                        target: "txpool",
                        hash = %tx.hash(),
                        "Filtering interop tx from pool"
                    );
                    false
                } else {
                    true
                }
            })
            .collect();
        let removed = before - filtered.len();
        debug!(
            target: "txpool",
            batch_size = before,
            interop_filtered = removed,
            forwarded = filtered.len(),
            "add_external_transactions: reorg filter applied"
        );
        if removed > 0 {
            self.metrics.reorg_interop_txs_filtered.increment(removed as u64);
        }
        filtered
    }
}

/// Convenience macro for delegating methods to the inner pool.
macro_rules! delegate {
    // No-arg methods returning a value.
    (fn $name:ident(&self) -> $ret:ty) => {
        fn $name(&self) -> $ret {
            self.inner.$name()
        }
    };
    // Single-arg methods returning a value.
    (fn $name:ident(&self, $arg:ident : $arg_ty:ty) -> $ret:ty) => {
        fn $name(&self, $arg: $arg_ty) -> $ret {
            self.inner.$name($arg)
        }
    };
    // Two-arg methods returning a value.
    (fn $name:ident(&self, $a1:ident : $a1_ty:ty, $a2:ident : $a2_ty:ty) -> $ret:ty) => {
        fn $name(&self, $a1: $a1_ty, $a2: $a2_ty) -> $ret {
            self.inner.$name($a1, $a2)
        }
    };
    // Three-arg methods returning a value.
    (
        fn
        $name:ident(&self, $a1:ident : $a1_ty:ty, $a2:ident : $a2_ty:ty, $a3:ident : $a3_ty:ty) ->
        $ret:ty
    ) => {
        fn $name(&self, $a1: $a1_ty, $a2: $a2_ty, $a3: $a3_ty) -> $ret {
            self.inner.$name($a1, $a2, $a3)
        }
    };
    // No-arg void methods.
    (fn $name:ident(&self)) => {
        fn $name(&self) {
            self.inner.$name()
        }
    };
    // Single-arg void methods.
    (fn $name:ident(&self, $arg:ident : $arg_ty:ty)) => {
        fn $name(&self, $arg: $arg_ty) {
            self.inner.$name($arg)
        }
    };
    // Async two-arg methods.
    (async fn $name:ident(&self, $a1:ident : $a1_ty:ty, $a2:ident : $a2_ty:ty) -> $ret:ty) => {
        async fn $name(&self, $a1: $a1_ty, $a2: $a2_ty) -> $ret {
            self.inner.$name($a1, $a2).await
        }
    };
}

impl<P> TransactionPool for OpPool<P>
where
    P: TransactionPool,
    P::Transaction: Transaction,
{
    type Transaction = P::Transaction;

    async fn add_external_transactions(
        &self,
        transactions: Vec<Self::Transaction>,
    ) -> Vec<PoolResult<reth_transaction_pool::pool::AddedTransactionOutcome>> {
        let txs = if self.should_filter_interop() {
            self.filter_interop_txs(transactions)
        } else {
            tracing::trace!(
                target: "txpool",
                batch_size = transactions.len(),
                "add_external_transactions: reorg filter inactive, passing through"
            );
            transactions
        };
        self.inner.add_external_transactions(txs).await
    }

    // Delegated methods below

    delegate!(async fn add_transaction_and_subscribe(&self, origin: TransactionOrigin, transaction: Self::Transaction) -> PoolResult<TransactionEvents>);
    delegate!(async fn add_transaction(&self, origin: TransactionOrigin, transaction: Self::Transaction) -> PoolResult<reth_transaction_pool::pool::AddedTransactionOutcome>);
    delegate!(async fn add_transactions(&self, origin: TransactionOrigin, transactions: Vec<Self::Transaction>) -> Vec<PoolResult<reth_transaction_pool::pool::AddedTransactionOutcome>>);

    // Cannot delegate via macro: `impl IntoIterator` arg not matchable by macro `:ty`.
    async fn add_transactions_with_origins(
        &self,
        transactions: impl IntoIterator<Item = (TransactionOrigin, Self::Transaction)> + Send,
    ) -> Vec<PoolResult<reth_transaction_pool::pool::AddedTransactionOutcome>> {
        self.inner.add_transactions_with_origins(transactions).await
    }

    delegate!(fn pool_size(&self) -> PoolSize);
    delegate!(fn block_info(&self) -> BlockInfo);
    delegate!(fn transaction_event_listener(&self, tx_hash: TxHash) -> Option<TransactionEvents>);
    delegate!(fn all_transactions_event_listener(&self) -> AllTransactionsEvents<Self::Transaction>);
    delegate!(fn pending_transactions_listener_for(&self, kind: TransactionListenerKind) -> Receiver<TxHash>);
    delegate!(fn blob_transaction_sidecars_listener(&self) -> Receiver<reth_transaction_pool::NewBlobSidecar>);
    delegate!(fn new_transactions_listener_for(&self, kind: TransactionListenerKind) -> Receiver<NewTransactionEvent<Self::Transaction>>);
    delegate!(fn pooled_transaction_hashes(&self) -> Vec<TxHash>);
    delegate!(fn pooled_transaction_hashes_max(&self, max: usize) -> Vec<TxHash>);
    delegate!(fn pooled_transactions(&self) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>>);
    delegate!(fn pooled_transactions_max(&self, max: usize) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>>);
    delegate!(fn get_pooled_transaction_elements(&self, tx_hashes: Vec<TxHash>, limit: GetPooledTransactionLimit) -> Vec<<Self::Transaction as PoolTransaction>::Pooled>);
    delegate!(fn get_pooled_transaction_element(&self, tx_hash: TxHash) -> Option<reth_primitives_traits::Recovered<<Self::Transaction as PoolTransaction>::Pooled>>);
    delegate!(fn best_transactions(&self) -> Box<dyn BestTransactions<Item = Arc<ValidPoolTransaction<Self::Transaction>>>>);
    delegate!(fn best_transactions_with_attributes(&self, best_transactions_attributes: BestTransactionsAttributes) -> Box<dyn BestTransactions<Item = Arc<ValidPoolTransaction<Self::Transaction>>>>);
    delegate!(fn pending_transactions(&self) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>>);
    delegate!(fn pending_transactions_max(&self, max: usize) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>>);
    delegate!(fn queued_transactions(&self) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>>);
    delegate!(fn pending_and_queued_txn_count(&self) -> (usize, usize));
    delegate!(fn all_transactions(&self) -> AllPoolTransactions<Self::Transaction>);
    delegate!(fn all_transaction_hashes(&self) -> Vec<TxHash>);
    delegate!(fn remove_transactions(&self, hashes: Vec<TxHash>) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>>);
    delegate!(fn remove_transactions_and_descendants(&self, hashes: Vec<TxHash>) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>>);
    delegate!(fn remove_transactions_by_sender(&self, sender: Address) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>>);
    delegate!(fn prune_transactions(&self, hashes: Vec<TxHash>) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>>);

    // Cannot delegate via macro: generic type parameter `<A>` with where clause.
    fn retain_unknown<A>(&self, announcement: &mut A)
    where
        A: reth_eth_wire_types::HandleMempoolData,
    {
        self.inner.retain_unknown(announcement)
    }

    delegate!(fn get(&self, tx_hash: &TxHash) -> Option<Arc<ValidPoolTransaction<Self::Transaction>>>);
    delegate!(fn get_all(&self, txs: Vec<TxHash>) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>>);
    delegate!(fn on_propagated(&self, txs: PropagatedTransactions));
    delegate!(fn get_transactions_by_sender(&self, sender: Address) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>>);

    // Cannot delegate via macro: `impl FnMut` arg not matchable by macro `:ty`.
    fn get_pending_transactions_with_predicate(
        &self,
        predicate: impl FnMut(&ValidPoolTransaction<Self::Transaction>) -> bool,
    ) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>> {
        self.inner.get_pending_transactions_with_predicate(predicate)
    }

    delegate!(fn get_pending_transactions_by_sender(&self, sender: Address) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>>);
    delegate!(fn get_queued_transactions_by_sender(&self, sender: Address) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>>);
    delegate!(fn get_highest_transaction_by_sender(&self, sender: Address) -> Option<Arc<ValidPoolTransaction<Self::Transaction>>>);
    delegate!(fn get_highest_consecutive_transaction_by_sender(&self, sender: Address, on_chain_nonce: u64) -> Option<Arc<ValidPoolTransaction<Self::Transaction>>>);
    delegate!(fn get_transaction_by_sender_and_nonce(&self, sender: Address, nonce: u64) -> Option<Arc<ValidPoolTransaction<Self::Transaction>>>);
    delegate!(fn get_transactions_by_origin(&self, origin: TransactionOrigin) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>>);
    delegate!(fn get_pending_transactions_by_origin(&self, origin: TransactionOrigin) -> Vec<Arc<ValidPoolTransaction<Self::Transaction>>>);
    delegate!(fn unique_senders(&self) -> alloy_primitives::map::AddressSet);
    delegate!(fn get_blob(&self, tx_hash: TxHash) -> Result<Option<Arc<BlobTransactionSidecarVariant>>, BlobStoreError>);
    delegate!(fn get_all_blobs(&self, tx_hashes: Vec<TxHash>) -> Result<Vec<(TxHash, Arc<BlobTransactionSidecarVariant>)>, BlobStoreError>);
    delegate!(fn get_all_blobs_exact(&self, tx_hashes: Vec<TxHash>) -> Result<Vec<Arc<BlobTransactionSidecarVariant>>, BlobStoreError>);
    delegate!(fn get_blobs_for_versioned_hashes_v1(&self, versioned_hashes: &[B256]) -> Result<Vec<Option<alloy_eips::eip4844::BlobAndProofV1>>, BlobStoreError>);
    delegate!(fn get_blobs_for_versioned_hashes_v2(&self, versioned_hashes: &[B256]) -> Result<Option<Vec<alloy_eips::eip4844::BlobAndProofV2>>, BlobStoreError>);
    delegate!(fn get_blobs_for_versioned_hashes_v3(&self, versioned_hashes: &[B256]) -> Result<Vec<Option<alloy_eips::eip4844::BlobAndProofV2>>, BlobStoreError>);
}

impl<P> TransactionPoolExt for OpPool<P>
where
    P: TransactionPoolExt,
    P::Transaction: Transaction,
{
    type Block = P::Block;

    fn on_canonical_state_change(
        &self,
        update: reth_transaction_pool::CanonicalStateUpdate<'_, Self::Block>,
    ) {
        if let Some(reorg_state) = &self.reorg_state &&
            update.update_kind == PoolUpdateKind::Reorg
        {
            *reorg_state.last_reorg_at.write().unwrap() = Some(Instant::now());
            reorg_state.filter_armed.store(true, Ordering::Release);
            debug!(
                target: "txpool",
                "reorg detected, filtering interop txs from the pool"
            );
        }
        self.inner.on_canonical_state_change(update);
    }

    // Delegated methods below

    delegate!(fn set_block_info(&self, info: BlockInfo));
    delegate!(fn update_accounts(&self, accounts: Vec<reth_execution_types::ChangedAccount>));
    delegate!(fn delete_blob(&self, tx: B256));
    delegate!(fn delete_blobs(&self, txs: Vec<B256>));
    delegate!(fn cleanup_blobs(&self));
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

    // is_interop_tx unit tests

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

    // OpPool filtering tests

    /// Helper to simulate a reorg without waiting for wall-clock time.
    fn mark_reorg(
        pool: &OpPool<reth_transaction_pool::noop::NoopTransactionPool<MockTransaction>>,
    ) {
        if let Some(state) = &pool.reorg_state {
            *state.last_reorg_at.write().unwrap() = Some(Instant::now());
            state.filter_armed.store(true, Ordering::Release);
        }
    }

    fn expire_reorg_window(
        pool: &OpPool<reth_transaction_pool::noop::NoopTransactionPool<MockTransaction>>,
    ) {
        if let Some(state) = &pool.reorg_state {
            *state.last_reorg_at.write().unwrap() =
                Some(Instant::now() - REORG_WINDOW - Duration::from_secs(1));
        }
    }

    #[test]
    fn test_should_filter_fallback_fires_once_after_window_expires() {
        let pool: OpPool<reth_transaction_pool::noop::NoopTransactionPool<MockTransaction>> =
            OpPool::new(reth_transaction_pool::noop::NoopTransactionPool::new(), true);

        mark_reorg(&pool);
        expire_reorg_window(&pool);

        assert!(pool.should_filter_interop());
        assert!(!pool.should_filter_interop());
    }

    #[test]
    fn test_should_filter_rearms_on_new_reorg() {
        let pool: OpPool<reth_transaction_pool::noop::NoopTransactionPool<MockTransaction>> =
            OpPool::new(reth_transaction_pool::noop::NoopTransactionPool::new(), true);

        mark_reorg(&pool);
        expire_reorg_window(&pool);
        assert!(pool.should_filter_interop());
        assert!(!pool.should_filter_interop());

        // Re-arm (new reorg).
        mark_reorg(&pool);
        assert!(pool.should_filter_interop());
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

        // All-normal batch passes through unchanged.
        assert_eq!(pool.filter_interop_txs(vec![mock_normal_tx(), mock_normal_tx()]).len(), 2);
    }

    // Async tests that call through add_external_transactions
    //
    // These use NoopTransactionPool as the inner pool. The number of results
    // returned equals the number of transactions forwarded (one Err per tx
    // since NoopTransactionPool rejects everything).

    #[tokio::test]
    async fn test_reorg_filters_interop_txs() {
        let pool: OpPool<reth_transaction_pool::noop::NoopTransactionPool<MockTransaction>> =
            OpPool::new(reth_transaction_pool::noop::NoopTransactionPool::new(), true);

        mark_reorg(&pool);

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

        mark_reorg(&pool);

        let normal = mock_normal_tx();
        let interop = mock_interop_tx();

        let results = pool.add_external_transactions(vec![normal, interop]).await;
        assert_eq!(results.len(), 2, "disabled pool should be transparent");
    }

    #[tokio::test]
    async fn test_window_filters_multiple_external_batches() {
        let pool: OpPool<reth_transaction_pool::noop::NoopTransactionPool<MockTransaction>> =
            OpPool::new(reth_transaction_pool::noop::NoopTransactionPool::new(), true);

        mark_reorg(&pool);

        // First call: within the active reorg window, interop tx is filtered.
        let results =
            pool.add_external_transactions(vec![mock_normal_tx(), mock_interop_tx()]).await;
        assert_eq!(results.len(), 1, "reorg window should filter interop tx");

        // Second call: still within the reorg window, so it should keep filtering.
        let results2 =
            pool.add_external_transactions(vec![mock_normal_tx(), mock_interop_tx()]).await;
        assert_eq!(results2.len(), 1, "reorg window should stay active across batches");
    }

    #[tokio::test]
    async fn test_expired_window_fallback_passes_through_after_being_spent() {
        let pool: OpPool<reth_transaction_pool::noop::NoopTransactionPool<MockTransaction>> =
            OpPool::new(reth_transaction_pool::noop::NoopTransactionPool::new(), true);

        mark_reorg(&pool);
        let _ = pool.add_external_transactions(vec![mock_normal_tx(), mock_interop_tx()]).await;
        expire_reorg_window(&pool);

        let results =
            pool.add_external_transactions(vec![mock_normal_tx(), mock_interop_tx()]).await;
        assert_eq!(results.len(), 2, "spent fallback should not filter after the window");
    }

    #[tokio::test]
    async fn test_add_transaction_unaffected_by_reorg() {
        let pool: OpPool<reth_transaction_pool::noop::NoopTransactionPool<MockTransaction>> =
            OpPool::new(reth_transaction_pool::noop::NoopTransactionPool::new(), true);

        mark_reorg(&pool);

        let interop = mock_interop_tx();

        // add_transaction (RPC/Local path) is never affected by the reorg filter.
        let result = pool.add_transaction(TransactionOrigin::Local, interop).await;
        assert!(result.is_err(), "NoopTransactionPool always rejects, proving tx was forwarded");
    }
}
