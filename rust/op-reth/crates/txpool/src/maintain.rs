//! Support for maintaining the state of the transaction pool

/// Offset before deadline expiry at which a tx becomes "stale" and triggers revalidation.
const OFFSET_TIME: u64 = 60;
/// Maximum number of interop filter requests at the same time.
const MAX_INTEROP_QUERIES: usize = 10;

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
use reth_transaction_pool::{PoolTransaction, TransactionPool, error::PoolTransactionError};
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
        if let CanonStateNotification::Commit { new } = event {
            let timestamp = new.tip().timestamp();
            let mut to_remove = Vec::new();
            let mut to_revalidate = Vec::new();
            let mut interop_count = 0;

            // If failsafe is active, evict ALL interop txs and skip revalidation.
            // Belt-and-suspenders with poll_failsafe: catches any tx that raced past
            // the ingress check or was added between poll_failsafe transition ticks.
            if interop_client.is_failsafe_enabled() {
                let interop_hashes: Vec<_> = pool
                    .pooled_transactions()
                    .iter()
                    .filter(|tx| is_interop_tx(&tx.transaction))
                    .map(|tx| *tx.hash())
                    .collect();
                if !interop_hashes.is_empty() {
                    info!(
                        target: "txpool::interop",
                        count = interop_hashes.len(),
                        "failsafe active on block event: evicting all interop transactions"
                    );
                    let removed = pool.remove_transactions(interop_hashes);
                    metrics.inc_removed_tx_interop(removed.len());
                }
                continue;
            }

            // scan all pooled interop transactions
            for pooled_tx in pool.pooled_transactions() {
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

                while let Some((tx_item_from_stream, validation_result)) =
                    revalidation_stream.next().await
                {
                    match validation_result {
                        Some(Ok(())) => {
                            tx_item_from_stream
                                .set_interop_deadline(timestamp + CHECK_ACCESS_LIST_TIMEOUT_SECS);
                        }
                        Some(Err(err)) => {
                            if err.is_bad_transaction() {
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
    loop {
        interval.tick().await;
        match interop_client.query_failsafe().await {
            Ok(enabled) => {
                // On transition to enabled: evict all interop txs immediately
                if enabled && !was_enabled {
                    let interop_hashes: Vec<_> = pool
                        .pooled_transactions()
                        .iter()
                        .filter(|tx| is_interop_tx(&tx.transaction))
                        .map(|tx| *tx.hash())
                        .collect();
                    if !interop_hashes.is_empty() {
                        info!(
                            target: "txpool::interop",
                            count = interop_hashes.len(),
                            "failsafe enabled: evicting all interop transactions"
                        );
                        let removed = pool.remove_transactions(interop_hashes);
                        metrics.inc_removed_tx_interop(removed.len());
                    }
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
