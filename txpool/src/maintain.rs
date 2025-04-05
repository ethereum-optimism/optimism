//! Support for maintaining the state of the transaction pool

/// The interval for which we check transaction against supervisor, 10 min.
const TRANSACTION_VALIDITY_WINDOW: u64 = 600;
/// Interval in seconds at which the transaction should be revalidated.
const OFFSET_TIME: u64 = 60;
/// Maximum number of supervisor requests at the same time
const MAX_SUPERVISOR_QUERIES: usize = 10;

use crate::{
    conditional::MaybeConditionalTransaction,
    interop::{is_stale_interop, is_valid_interop, MaybeInteropTransaction},
    supervisor::SupervisorClient,
};
use alloy_consensus::{conditional::BlockConditionalAttributes, BlockHeader, Transaction};
use futures_util::{future::BoxFuture, FutureExt, Stream, StreamExt};
use metrics::Gauge;
use reth_chain_state::CanonStateNotification;
use reth_metrics::{metrics::Counter, Metrics};
use reth_primitives_traits::NodePrimitives;
use reth_transaction_pool::{error::PoolTransactionError, PoolTransaction, TransactionPool};
use std::sync::Arc;

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
    // TODO: we should add some timing metric in here to check supervisor congestion
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
    supervisor_client: SupervisorClient,
) -> BoxFuture<'static, ()>
where
    N: NodePrimitives,
    Pool: TransactionPool + 'static,
    Pool::Transaction: MaybeInteropTransaction,
    St: Stream<Item = CanonStateNotification<N>> + Send + Unpin + 'static,
{
    async move {
        maintain_transaction_pool_interop(pool, events, supervisor_client).await;
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
    supervisor_client: SupervisorClient,
) where
    N: NodePrimitives,
    Pool: TransactionPool,
    Pool::Transaction: MaybeInteropTransaction,
    St: Stream<Item = CanonStateNotification<N>> + Send + Unpin + 'static,
{
    let metrics = MaintainPoolInteropMetrics::default();
    let supervisor_client = Arc::new(supervisor_client);
    loop {
        let Some(event) = events.next().await else { break };
        if let CanonStateNotification::Commit { new } = event {
            let timestamp = new.tip().timestamp();
            let mut to_remove = Vec::new();
            let mut to_revalidate = Vec::new();
            let mut interop_count = 0;
            for tx in &pool.pooled_transactions() {
                // Only interop txs have this field set
                if let Some(interop) = tx.transaction.interop_deadline() {
                    interop_count += 1;
                    if !is_valid_interop(interop, timestamp) {
                        // That means tx didn't revalidated during [`OFFSET_TIME`] time
                        // We could assume that it won't be validated at all and remove it
                        to_remove.push(*tx.hash());
                    } else if is_stale_interop(interop, timestamp, OFFSET_TIME) {
                        // If tx has less then [`OFFSET_TIME`] of valid time we revalidate it
                        to_revalidate.push(tx.clone())
                    }
                }
            }

            metrics.set_interop_txs_in_pool(interop_count);
            if !to_revalidate.is_empty() {
                metrics.inc_stale_tx_interop(to_revalidate.len());
                let checks_stream =
                    futures_util::stream::iter(to_revalidate.into_iter().map(|tx| {
                        let supervisor_client = supervisor_client.clone();
                        async move {
                            let check = supervisor_client
                                .is_valid_cross_tx(
                                    tx.transaction.access_list(),
                                    tx.transaction.hash(),
                                    timestamp,
                                    Some(TRANSACTION_VALIDITY_WINDOW),
                                    // We could assume that interop is enabled, because
                                    // tx.transaction.interop() would be set only in
                                    // this case
                                    true,
                                )
                                .await;
                            (tx.clone(), check)
                        }
                    }))
                    .buffered(MAX_SUPERVISOR_QUERIES);
                futures_util::pin_mut!(checks_stream);
                while let Some((tx, check)) = checks_stream.next().await {
                    if let Some(Err(err)) = check {
                        // We remove only bad transaction. If error caused by supervisor instability
                        // or other fixable issues transaction would be validated on next state
                        // change, so we ignore it
                        if err.is_bad_transaction() {
                            to_remove.push(*tx.transaction.hash());
                        }
                    } else {
                        tx.transaction.set_interop_deadline(timestamp + TRANSACTION_VALIDITY_WINDOW)
                    }
                }
            }
            if !to_remove.is_empty() {
                let removed = pool.remove_transactions(to_remove);
                metrics.inc_removed_tx_interop(removed.len());
            }
        }
    }
}
