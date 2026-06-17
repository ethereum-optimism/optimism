//! Support for maintaining the state of the transaction pool

/// Offset before deadline expiry at which a tx becomes "stale" and triggers revalidation.
const OFFSET_TIME: u64 = 60;
/// Maximum number of transactions revalidated against the interop filter concurrently. Each
/// transaction issues up to one request per configured endpoint, so total in-flight interop
/// requests can reach `MAX_INTEROP_QUERIES * <number of endpoints>`. The bound is on
/// transactions (per-endpoint load is unchanged by fan-out).
const MAX_INTEROP_QUERIES: usize = 10;
/// Interval at which a heartbeat warning is re-logged while failsafe stays enabled, so a
/// long-lived failsafe keeps surfacing in recent logs rather than only at the transition. Also
/// reused to rate-limit the degraded-quorum log in [`InteropFilterClient`].
pub(crate) const FAILSAFE_HEARTBEAT_INTERVAL: Duration = Duration::from_secs(60);

use crate::{
    conditional::MaybeConditionalTransaction,
    interop::{MaybeInteropTransaction, is_interop_tx, is_stale_interop, is_valid_interop},
    interop_filter::InteropFilterClient,
    validator::CHECK_ACCESS_LIST_TIMEOUT_SECS,
};
use alloy_consensus::{BlockHeader, conditional::BlockConditionalAttributes};
use futures_util::{FutureExt, Stream, StreamExt, future::BoxFuture};
use metrics::{Gauge, Histogram};
use reth_chain_state::CanonStateNotification;
use reth_metrics::{Metrics, metrics::Counter};
use reth_primitives_traits::NodePrimitives;
use reth_transaction_pool::{PoolTransaction, TransactionPool};
use std::time::{Duration, Instant};
use tracing::{info, warn};

/// Transaction pool maintenance metrics
#[derive(Metrics)]
#[metrics(scope = "transaction_pool")]
struct MaintainPoolConditionalMetrics {
    /// Counter indicating the number of conditional transactions removed from
    /// the pool because of exceeded block attributes.
    removed_tx_conditional: Counter,
}

impl MaintainPoolConditionalMetrics {
    #[inline]
    fn inc_removed_tx_conditional(&self, count: usize) {
        self.removed_tx_conditional.increment(count as u64);
    }
}

/// Transaction pool maintenance metrics
#[derive(Metrics)]
#[metrics(scope = "transaction_pool")]
struct MaintainPoolInteropMetrics {
    /// Counter indicating the number of conditional transactions removed from
    /// the pool because of exceeded block attributes.
    removed_tx_interop: Counter,
    /// Number of interop transactions currently in the pool
    pooled_interop_transactions: Gauge,

    /// Counter for interop transactions that became stale and need revalidation
    stale_interop_transactions: Counter,
    // TODO: we also should add metric for (hash, counter) to check number of validation per tx
    /// Histogram for measuring interop revalidation duration (congestion metric).
    interop_revalidation_duration_seconds: Histogram,
}

impl MaintainPoolInteropMetrics {
    #[inline]
    fn inc_removed_tx_interop(&self, count: usize) {
        self.removed_tx_interop.increment(count as u64);
    }
    #[inline]
    fn set_interop_txs_in_pool(&self, count: usize) {
        self.pooled_interop_transactions.set(count as f64);
    }

    #[inline]
    fn inc_stale_tx_interop(&self, count: usize) {
        self.stale_interop_transactions.increment(count as u64);
    }

    /// Records interop revalidation duration.
    #[inline]
    fn record_interop_duration(&self, duration: std::time::Duration) {
        self.interop_revalidation_duration_seconds.record(duration.as_secs_f64());
    }
}
/// Returns a spawnable future for maintaining the state of the conditional txs in the transaction
/// pool.
pub fn maintain_transaction_pool_conditional_future<N, Pool, St>(
    pool: Pool,
    events: St,
) -> BoxFuture<'static, ()>
where
    N: NodePrimitives,
    Pool: TransactionPool + 'static,
    Pool::Transaction: MaybeConditionalTransaction,
    St: Stream<Item = CanonStateNotification<N>> + Send + Unpin + 'static,
{
    async move {
        maintain_transaction_pool_conditional(pool, events).await;
    }
    .boxed()
}

/// Maintains the state of the conditional tx in the transaction pool by handling new blocks and
/// reorgs.
///
/// This listens for any new blocks and reorgs and updates the conditional txs in the
/// transaction pool's state accordingly
pub async fn maintain_transaction_pool_conditional<N, Pool, St>(pool: Pool, mut events: St)
where
    N: NodePrimitives,
    Pool: TransactionPool,
    Pool::Transaction: MaybeConditionalTransaction,
    St: Stream<Item = CanonStateNotification<N>> + Send + Unpin + 'static,
{
    let metrics = MaintainPoolConditionalMetrics::default();
    loop {
        let Some(event) = events.next().await else { break };
        if let CanonStateNotification::Commit { new } = event {
            let block_attr = BlockConditionalAttributes {
                number: new.tip().number(),
                timestamp: new.tip().timestamp(),
            };
            let mut to_remove = Vec::new();
            for tx in &pool.pooled_transactions() {
                if tx.transaction.has_exceeded_block_attributes(&block_attr) {
                    to_remove.push(*tx.hash());
                }
            }
            if !to_remove.is_empty() {
                let removed = pool.remove_transactions(to_remove);
                metrics.inc_removed_tx_conditional(removed.len());
            }
        }
    }
}

/// Returns a spawnable future for maintaining the state of the interop tx in the transaction pool.
pub fn maintain_transaction_pool_interop_future<N, Pool, St>(
    pool: Pool,
    events: St,
    interop_client: InteropFilterClient,
) -> BoxFuture<'static, ()>
where
    N: NodePrimitives,
    Pool: TransactionPool + 'static,
    Pool::Transaction: MaybeInteropTransaction,
    St: Stream<Item = CanonStateNotification<N>> + Send + Unpin + 'static,
{
    async move {
        maintain_transaction_pool_interop(pool, events, interop_client).await;
    }
    .boxed()
}

/// Maintains the state of the interop tx in the transaction pool by handling new blocks and reorgs.
///
/// This listens for any new blocks and reorgs and updates the interop tx in the transaction pool's
/// state accordingly
pub async fn maintain_transaction_pool_interop<N, Pool, St>(
    pool: Pool,
    mut events: St,
    interop_client: InteropFilterClient,
) where
    N: NodePrimitives,
    Pool: TransactionPool,
    Pool::Transaction: MaybeInteropTransaction,
    St: Stream<Item = CanonStateNotification<N>> + Send + Unpin + 'static,
{
    let metrics = MaintainPoolInteropMetrics::default();

    loop {
        let Some(event) = events.next().await else { break };
        match event {
            // A reorg can invalidate the initiating message a pooled executing tx depends on, and
            // source-chain reorgs are invisible here, so drop every interop tx.
            CanonStateNotification::Reorg { .. } => {
                let evicted = evict_all_interop_txs(&pool, &metrics);
                if evicted > 0 {
                    info!(
                        target: "txpool::interop",
                        count = evicted,
                        "reorg detected: evicting all interop transactions"
                    );
                }
            }
            CanonStateNotification::Commit { new } => {
                if interop_client.is_failsafe_enabled() {
                    let evicted = evict_all_interop_txs(&pool, &metrics);
                    if evicted > 0 {
                        info!(
                            target: "txpool::interop",
                            count = evicted,
                            "failsafe active on block event: evicting all interop transactions"
                        );
                    }
                } else {
                    revalidate_stale_interop_txs(
                        &pool,
                        &interop_client,
                        new.tip().timestamp(),
                        &metrics,
                    )
                    .await;
                }
            }
        }
    }
}

/// Evicts every interop tx from the pool, returning the number removed. Scans `all_transactions()`
/// so non-propagatable interop txs (`Private` origin), hidden from `pooled_transactions()` but
/// still buildable, are reached too.
fn evict_all_interop_txs<Pool>(pool: &Pool, metrics: &MaintainPoolInteropMetrics) -> usize
where
    Pool: TransactionPool,
    Pool::Transaction: MaybeInteropTransaction,
{
    let interop_hashes: Vec<_> = pool
        .all_transactions()
        .iter()
        .filter(|tx| is_interop_tx(&tx.transaction))
        .map(|tx| *tx.hash())
        .collect();
    if interop_hashes.is_empty() {
        return 0;
    }
    let removed = pool.remove_transactions(interop_hashes).len();
    metrics.inc_removed_tx_interop(removed);
    removed
}

/// Revalidates the stale interop txs in the pool against the interop filter for a single block and
/// evicts those that are no longer admissible. Split out of the event loop so the eviction
/// decision can be exercised against a real pool in tests.
///
/// A pooled interop tx is evicted when its deadline has passed, when it is no longer a cross-chain
/// tx (`None`), or when revalidation returns a decisive invalid verdict
/// ([`is_now_invalid`](crate::InvalidCrossTx::is_now_invalid)). A still-valid tx has its deadline
/// refreshed; a
/// transient/non-decisive verdict leaves the tx in place so a flapping or unreachable interop
/// filter cannot drain the pool.
async fn revalidate_stale_interop_txs<Pool>(
    pool: &Pool,
    interop_client: &InteropFilterClient,
    timestamp: u64,
    metrics: &MaintainPoolInteropMetrics,
) where
    Pool: TransactionPool,
    Pool::Transaction: MaybeInteropTransaction,
{
    let mut to_remove = Vec::new();
    let mut to_revalidate = Vec::new();
    let mut interop_count = 0;

    // Scan all transactions, not just propagatable ones: non-propagatable interop txs (e.g.
    // `Private`-origin conditional txs) are still selected by the builder but are hidden from
    // `pooled_transactions()`, so revalidation/eviction must iterate the whole pool.
    // `all_transactions()` covers the pending/basefee/queued subpools; the blob subpool is excluded
    // but irrelevant here since interop txs are never type-3 (EIP-4844 is rejected at ingress).
    for pooled_tx in pool.all_transactions() {
        if let Some(interop_deadline_val) = pooled_tx.transaction.interop_deadline() {
            interop_count += 1;
            if !is_valid_interop(interop_deadline_val, timestamp) {
                to_remove.push(*pooled_tx.transaction.hash());
            } else if is_stale_interop(interop_deadline_val, timestamp, OFFSET_TIME) {
                to_revalidate.push(pooled_tx.transaction.clone());
            }
        }
    }

    metrics.set_interop_txs_in_pool(interop_count);

    if !to_revalidate.is_empty() {
        metrics.inc_stale_tx_interop(to_revalidate.len());

        let revalidation_start = Instant::now();
        let revalidation_stream = interop_client.revalidate_interop_txs_stream(
            to_revalidate,
            timestamp,
            CHECK_ACCESS_LIST_TIMEOUT_SECS,
            MAX_INTEROP_QUERIES,
        );

        futures_util::pin_mut!(revalidation_stream);

        while let Some((tx_item_from_stream, validation_result)) = revalidation_stream.next().await
        {
            match validation_result {
                Some(Ok(())) => {
                    tx_item_from_stream
                        .set_interop_deadline(timestamp + CHECK_ACCESS_LIST_TIMEOUT_SECS);
                }
                // Evict only on a decisive invalid verdict; transient or non-decisive results
                // keep the tx so an unreachable interop filter cannot drain the pool.
                Some(Err(err)) => {
                    if err.is_now_invalid() {
                        to_remove.push(*tx_item_from_stream.hash());
                    }
                }
                None => {
                    warn!(
                        target: "txpool",
                        hash = %tx_item_from_stream.hash(),
                        "Interop transaction no longer considered cross-chain during revalidation; removing."
                    );
                    to_remove.push(*tx_item_from_stream.hash());
                }
            }
        }

        metrics.record_interop_duration(revalidation_start.elapsed());
    }

    if !to_remove.is_empty() {
        let removed = pool.remove_transactions(to_remove);
        metrics.inc_removed_tx_interop(removed.len());
    }
}

/// Background task that polls the interop filter for failsafe state every second.
/// When failsafe transitions from disabled to enabled, evicts all interop txs
/// from the pool immediately (does not wait for the next block event).
/// Matches op-geth's `startBackgroundInteropFailsafeDetection` (miner/miner.go:140-165).
pub async fn poll_failsafe<Pool>(interop_client: InteropFilterClient, pool: Pool)
where
    Pool: TransactionPool,
    Pool::Transaction: MaybeInteropTransaction,
{
    let metrics = MaintainPoolInteropMetrics::default();
    let mut interval = tokio::time::interval(Duration::from_secs(1));
    let mut was_enabled = false;
    let mut last_heartbeat = Instant::now();
    loop {
        interval.tick().await;
        match interop_client.query_failsafe().await {
            Ok(enabled) => {
                if enabled && !was_enabled {
                    // Transition to enabled: evict all interop txs immediately and log
                    // unconditionally, so the state change is visible even when no interop txs
                    // are pooled.
                    let evicted = evict_all_interop_txs(&pool, &metrics);
                    warn!(
                        target: "txpool::interop",
                        evicted,
                        "interop failsafe enabled: rejecting all interop transactions; admission resumes automatically (within the ~1s poll interval) once the interop filter clears failsafe"
                    );
                    last_heartbeat = Instant::now();
                } else if enabled && last_heartbeat.elapsed() >= FAILSAFE_HEARTBEAT_INTERVAL {
                    // Heartbeat: keep a long-lived failsafe visible in recent logs.
                    warn!(
                        target: "txpool::interop",
                        "interop failsafe still active: all interop transactions are being rejected; admission resumes automatically once the interop filter clears failsafe"
                    );
                    last_heartbeat = Instant::now();
                } else if !enabled && was_enabled {
                    info!(
                        target: "txpool::interop",
                        "interop failsafe cleared: resuming interop transaction processing"
                    );
                }
                was_enabled = enabled;
            }
            Err(err) => {
                warn!(
                    target: "txpool::interop",
                    %err,
                    "failed to query failsafe state"
                );
            }
        }
    }
}

/// Creates a boxed future for the failsafe polling task.
pub fn poll_failsafe_future<Pool>(
    interop_client: InteropFilterClient,
    pool: Pool,
) -> BoxFuture<'static, ()>
where
    Pool: TransactionPool + 'static,
    Pool::Transaction: MaybeInteropTransaction,
{
    Box::pin(poll_failsafe(interop_client, pool))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{OpPooledTransaction, interop_filter::CROSS_L2_INBOX_ADDRESS};
    use alloy_consensus::{SignableTransaction, TxEip1559, transaction::Recovered};
    use alloy_eips::eip2930::{AccessList, AccessListItem};
    use alloy_primitives::{Address, B256, Signature, TxKind, U256};
    use jsonrpsee::types::ErrorObjectOwned;
    use jsonrpsee_server::{RpcModule, ServerBuilder, ServerHandle};
    use op_alloy_rpc_types::SuperchainDAError;
    use reth_execution_types::{Chain, ExecutionOutcome};
    use reth_optimism_primitives::{OpBlock, OpPrimitives};
    use reth_primitives_traits::RecoveredBlock;
    use reth_transaction_pool::{
        CoinbaseTipOrdering, Pool, PoolConfig, TransactionOrigin, blobstore::InMemoryBlobStore,
        noop::MockTransactionValidator,
    };
    use std::{collections::BTreeMap, net::SocketAddr, sync::Arc};

    /// A pool of [`OpPooledTransaction`] backed by the always-valid mock validator, so a built tx
    /// lands in the pending subpool and is visible to the maintenance sweep.
    type TestOpPool = Pool<
        MockTransactionValidator<OpPooledTransaction>,
        CoinbaseTipOrdering<OpPooledTransaction>,
        InMemoryBlobStore,
    >;

    fn build_pool() -> TestOpPool {
        Pool::new(
            MockTransactionValidator::default(),
            CoinbaseTipOrdering::default(),
            InMemoryBlobStore::default(),
            PoolConfig::default(),
        )
    }

    /// Builds an interop [`OpPooledTransaction`]: an EIP-1559 tx whose access list targets the
    /// cross-L2 inbox. The signature is a dummy; the mock validator does not verify it and the
    /// sender is set explicitly.
    fn interop_pooled_tx() -> OpPooledTransaction {
        let tx = TxEip1559 {
            chain_id: 10,
            nonce: 0,
            gas_limit: 21_000,
            max_fee_per_gas: 20_000_000_000,
            max_priority_fee_per_gas: 1,
            to: TxKind::Call(Address::ZERO),
            value: U256::ZERO,
            access_list: AccessList(vec![AccessListItem {
                address: CROSS_L2_INBOX_ADDRESS,
                storage_keys: vec![B256::ZERO],
            }]),
            input: Default::default(),
        };
        let signed = tx.into_signed(Signature::new(U256::from(1u64), U256::from(1u64), false));
        OpPooledTransaction::from_pooled(Recovered::new_unchecked(
            op_alloy_consensus::OpPooledTransaction::Eip1559(signed),
            Address::with_last_byte(1),
        ))
    }

    /// Builds a non-interop EIP-1559 [`OpPooledTransaction`]: same shape as [`interop_pooled_tx`]
    /// but with an empty access list, so `is_interop_tx` is false and the reorg sweep must leave it
    /// in place. Uses a distinct sender so it shares no nonce sequence with the interop tx.
    fn non_interop_pooled_tx() -> OpPooledTransaction {
        let tx = TxEip1559 {
            chain_id: 10,
            nonce: 0,
            gas_limit: 21_000,
            max_fee_per_gas: 20_000_000_000,
            max_priority_fee_per_gas: 1,
            to: TxKind::Call(Address::ZERO),
            value: U256::ZERO,
            access_list: AccessList::default(),
            input: Default::default(),
        };
        let signed = tx.into_signed(Signature::new(U256::from(1u64), U256::from(1u64), false));
        OpPooledTransaction::from_pooled(Recovered::new_unchecked(
            op_alloy_consensus::OpPooledTransaction::Eip1559(signed),
            Address::with_last_byte(2),
        ))
    }

    /// A minimal reorg notification. The interop reorg sweep evicts unconditionally and never
    /// inspects the reverted/committed chains, so a single default block on both sides suffices.
    fn reorg_event() -> CanonStateNotification<OpPrimitives> {
        let block: RecoveredBlock<OpBlock> = Default::default();
        let chain = Arc::new(Chain::new([block], ExecutionOutcome::default(), BTreeMap::new()));
        CanonStateNotification::Reorg { old: chain.clone(), new: chain }
    }

    /// A minimal commit notification with a single default block.
    fn commit_event() -> CanonStateNotification<OpPrimitives> {
        let block: RecoveredBlock<OpBlock> = Default::default();
        let chain = Arc::new(Chain::new([block], ExecutionOutcome::default(), BTreeMap::new()));
        CanonStateNotification::Commit { new: chain }
    }

    struct MockFilter {
        url: String,
        _handle: ServerHandle,
    }

    /// A single mock interop-filter endpoint whose `interop_checkAccessList` always answers with
    /// the given JSON-RPC error code/message.
    async fn mock_filter(code: i32, message: &'static str) -> MockFilter {
        let server = ServerBuilder::default()
            .build("127.0.0.1:0".parse::<SocketAddr>().unwrap())
            .await
            .unwrap();
        let addr = server.local_addr().unwrap();
        let mut module = RpcModule::new(());
        module
            .register_async_method("interop_checkAccessList", move |_p, _c, _| async move {
                Err::<(), _>(ErrorObjectOwned::owned(code, message, None::<()>))
            })
            .unwrap();
        module
            .register_async_method("admin_getFailsafeEnabled", |_p, _c, _| async move {
                Ok::<bool, ErrorObjectOwned>(false)
            })
            .unwrap();
        let handle = server.start(module);
        MockFilter { url: format!("http://{addr}"), _handle: handle }
    }

    async fn client_for(filter: &MockFilter) -> InteropFilterClient {
        InteropFilterClient::builder(vec![filter.url.clone()], 10)
            .timeout(Duration::from_millis(300))
            .build()
            .await
    }

    /// Adds a stale-but-still-valid interop tx with the given `origin` to the pool and runs the
    /// sweep against `filter`, returning whether the tx survived.
    async fn stale_interop_tx_survives_sweep(
        filter: &MockFilter,
        origin: TransactionOrigin,
    ) -> bool {
        let pool = build_pool();
        let tx = interop_pooled_tx();
        let hash = *tx.hash();
        let timestamp = 1_000;
        // Deadline in (timestamp, timestamp + OFFSET_TIME]: stale (triggers revalidation) yet still
        // valid (not expired), so the sweep revalidates rather than expiry-evicts it.
        tx.set_interop_deadline(timestamp + 30);
        pool.add_transaction(origin, tx).await.unwrap();
        assert!(pool.get(&hash).is_some(), "tx should be pooled before the sweep");

        let client = client_for(filter).await;
        revalidate_stale_interop_txs(
            &pool,
            &client,
            timestamp,
            &MaintainPoolInteropMetrics::default(),
        )
        .await;

        pool.get(&hash).is_some()
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn sweep_evicts_definitive_invalid_tx() {
        // The filter returns a definitive invalid verdict (InvalidEntry). The sweep must remove the
        // tx from the pool. This fails if eviction is gated on `is_bad_transaction`.
        let filter =
            mock_filter(SuperchainDAError::ConflictingData as i32, "conflicting data").await;
        assert!(
            !stale_interop_tx_survives_sweep(&filter, TransactionOrigin::External).await,
            "a definitive invalid verdict must be evicted from the pool by the sweep"
        );
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn sweep_evicts_non_propagatable_invalid_tx() {
        // A `Private`-origin tx (e.g. one submitted via eth_sendRawTransactionConditional) is
        // marked non-propagatable, so it is invisible to `pooled_transactions()`. The builder can
        // still include it, so the sweep must revalidate and evict it on a definitive invalid
        // verdict. This fails if the scan iterates only propagatable transactions.
        let filter =
            mock_filter(SuperchainDAError::ConflictingData as i32, "conflicting data").await;
        assert!(
            !stale_interop_tx_survives_sweep(&filter, TransactionOrigin::Private).await,
            "a non-propagatable invalid interop tx must be evicted from the pool by the sweep"
        );
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn reorg_evicts_pooled_interop_txs() {
        // The filter answers transient errors, so passing proves eviction is independent of
        // revalidation: the interop tx must go, the plain tx must stay.
        let pool = build_pool();
        let interop = interop_pooled_tx();
        let interop_hash = *interop.hash();
        let plain = non_interop_pooled_tx();
        let plain_hash = *plain.hash();
        pool.add_transaction(TransactionOrigin::External, interop).await.unwrap();
        pool.add_transaction(TransactionOrigin::External, plain).await.unwrap();
        assert!(pool.get(&interop_hash).is_some(), "interop tx should be pooled before the reorg");
        assert!(pool.get(&plain_hash).is_some(), "plain tx should be pooled before the reorg");

        let filter = mock_filter(-32603, "internal error").await;
        let client = client_for(&filter).await;
        let events = futures_util::stream::iter(vec![reorg_event()]);

        maintain_transaction_pool_interop(pool.clone(), events, client).await;

        assert!(pool.get(&interop_hash).is_none(), "interop tx must be evicted on reorg");
        assert!(pool.get(&plain_hash).is_some(), "non-interop tx must survive the reorg");
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn reorg_evicts_non_propagatable_interop_tx() {
        // A `Private`-origin interop tx is hidden from `pooled_transactions()` but still buildable,
        // so the reorg sweep must reach it via `all_transactions()` and evict it too.
        let pool = build_pool();
        let interop = interop_pooled_tx();
        let interop_hash = *interop.hash();
        pool.add_transaction(TransactionOrigin::Private, interop).await.unwrap();
        assert!(pool.get(&interop_hash).is_some(), "interop tx should be pooled before the reorg");

        let filter = mock_filter(-32603, "internal error").await;
        let client = client_for(&filter).await;
        let events = futures_util::stream::iter(vec![reorg_event()]);

        maintain_transaction_pool_interop(pool.clone(), events, client).await;

        assert!(
            pool.get(&interop_hash).is_none(),
            "non-propagatable interop tx must be evicted on reorg"
        );
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn commit_with_failsafe_evicts_all_interop_txs() {
        // With failsafe active, a block commit must evict every interop tx, not revalidate. The
        // interop tx carries no deadline, so the revalidation path would leave it untouched;
        // requiring it to be gone proves the commit arm took the failsafe branch.
        let pool = build_pool();
        let interop = interop_pooled_tx();
        let interop_hash = *interop.hash();
        let plain = non_interop_pooled_tx();
        let plain_hash = *plain.hash();
        pool.add_transaction(TransactionOrigin::External, interop).await.unwrap();
        pool.add_transaction(TransactionOrigin::External, plain).await.unwrap();

        let filter = mock_filter(-32603, "internal error").await;
        let client = client_for(&filter).await;
        client.apply_failsafe_state(true);
        let events = futures_util::stream::iter(vec![commit_event()]);

        maintain_transaction_pool_interop(pool.clone(), events, client).await;

        assert!(pool.get(&interop_hash).is_none(), "failsafe commit must evict the interop tx");
        assert!(pool.get(&plain_hash).is_some(), "non-interop tx must survive the failsafe commit");
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn sweep_retains_tx_on_transient_failure() {
        // A JSON-RPC internal error (-32603) is a non-response: with a single endpoint the quorum
        // is not reached. A flapping/unreachable interop filter must not drain the pool, so the
        // tx stays.
        let filter = mock_filter(-32603, "internal error").await;
        assert!(
            stale_interop_tx_survives_sweep(&filter, TransactionOrigin::External).await,
            "a transient/non-decisive verdict must NOT evict the tx"
        );
    }
}
