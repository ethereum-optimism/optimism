//! Support for maintaining the state of the transaction pool

/// The interval for which we check transaction against supervisor, 10 min.
const TRANSACTION_VALIDITY_WINDOW: u64 = 600;
/// Interval in seconds at which the transaction should be revalidated.
const OFFSET_TIME: u64 = 60;
/// Maximum number of supervisor requests at the same time
const MAX_SUPERVISOR_QUERIES: usize = 10;

use crate::{
    conditional::MaybeConditionalTransaction,
    interop::{MaybeInteropTransaction, is_stale_interop, is_valid_interop},
    supervisor::{SupervisorClient, parse_access_list_items_to_inbox_entries},
};
use alloy_consensus::{
    BlockHeader, Transaction, Typed2718, conditional::BlockConditionalAttributes,
    transaction::TxHashRef,
};
use alloy_eips::{BlockNumberOrTag, eip2930::AccessList, eip7594::BlobTransactionSidecarVariant};
use alloy_primitives::{
    Address, BlockHash, BlockNumber,
    map::{AddressSet, HashSet},
};
use futures_util::{
    FutureExt, Stream, StreamExt,
    future::{BoxFuture, Fuse, FusedFuture},
};
use metrics::{Gauge, Histogram};
use reth_chain_state::CanonStateNotification;
use reth_chainspec::{ChainSpecProvider, EthChainSpec, EthereumHardforks};
use reth_execution_types::ChangedAccount;
use reth_metrics::{Metrics, metrics::Counter};
use reth_optimism_forks::OpHardforks;
use reth_primitives_traits::{NodePrimitives, SealedHeader};
use reth_storage_api::{BlockReaderIdExt, StateProviderFactory, errors::provider::ProviderError};
use reth_tasks::Runtime;
use reth_transaction_pool::{
    AllPoolTransactions, BlockInfo, EthPoolTransaction, PoolTransaction, PoolUpdateKind,
    TransactionPool, TransactionPoolExt,
    blobstore::{BlobSidecarConverter, BlobStoreCanonTracker, BlobStoreUpdates},
    error::PoolTransactionError,
    maintain::MaintainPoolConfig,
    metrics::MaintainPoolMetrics,
};
use std::{
    borrow::Borrow,
    hash::{Hash, Hasher},
    sync::Arc,
    time::Instant,
};
use tokio::{
    sync::oneshot,
    time::{self, Duration},
};
use tracing::{debug, trace, warn};

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
    /// Histogram for measuring supervisor revalidation duration (congestion metric)
    supervisor_revalidation_duration_seconds: Histogram,
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

    /// Record supervisor revalidation duration
    #[inline]
    fn record_supervisor_duration(&self, duration: std::time::Duration) {
        self.supervisor_revalidation_duration_seconds.record(duration.as_secs_f64());
    }
}

/// Returns a spawnable future for maintaining the state of the transaction pool.
///
/// This is intentionally copied from upstream reth's maintainer and kept in sync, with the only
/// behavior change being interop filtering for transactions reinjected after reorgs.
pub fn op_maintain_transaction_pool_future<N, Client, P, St>(
    client: Client,
    pool: P,
    events: St,
    task_spawner: Runtime,
    config: MaintainPoolConfig,
) -> BoxFuture<'static, ()>
where
    N: NodePrimitives,
    Client: StateProviderFactory
        + BlockReaderIdExt<Header = N::BlockHeader>
        + ChainSpecProvider<
            ChainSpec: EthChainSpec<Header = N::BlockHeader> + EthereumHardforks + OpHardforks,
        > + Clone
        + 'static,
    P: TransactionPoolExt<Transaction: PoolTransaction<Consensus = N::SignedTx>, Block = N::Block>
        + 'static,
    St: Stream<Item = CanonStateNotification<N>> + Send + Unpin + 'static,
{
    async move {
        op_maintain_transaction_pool::<N, Client, P, St>(
            client,
            pool,
            events,
            task_spawner,
            config,
        )
        .await;
    }
    .boxed()
}

/// Maintains the state of the transaction pool by handling new blocks and reorgs.
///
/// This is intentionally copied from upstream reth's maintainer and kept in sync, with the only
/// behavior change being interop filtering for transactions reinjected after reorgs.
pub async fn op_maintain_transaction_pool<N, Client, P, St>(
    client: Client,
    pool: P,
    mut events: St,
    task_spawner: Runtime,
    config: MaintainPoolConfig,
) where
    N: NodePrimitives,
    Client: StateProviderFactory
        + BlockReaderIdExt<Header = N::BlockHeader>
        + ChainSpecProvider<
            ChainSpec: EthChainSpec<Header = N::BlockHeader> + EthereumHardforks + OpHardforks,
        > + Clone
        + 'static,
    P: TransactionPoolExt<Transaction: PoolTransaction<Consensus = N::SignedTx>, Block = N::Block>
        + 'static,
    St: Stream<Item = CanonStateNotification<N>> + Send + Unpin + 'static,
{
    let metrics = MaintainPoolMetrics::default();
    let MaintainPoolConfig { max_update_depth, max_reload_accounts, .. } = config;
    if let Ok(Some(latest)) = client.header_by_number_or_tag(BlockNumberOrTag::Latest) {
        let latest = SealedHeader::seal_slow(latest);
        let chain_spec = client.chain_spec();
        let info = BlockInfo {
            block_gas_limit: latest.gas_limit(),
            last_seen_block_hash: latest.hash(),
            last_seen_block_number: latest.number(),
            pending_basefee: chain_spec
                .next_block_base_fee(latest.header(), latest.timestamp())
                .unwrap_or_default(),
            pending_blob_fee: latest
                .maybe_next_block_blob_fee(chain_spec.blob_params_at_timestamp(latest.timestamp())),
        };
        pool.set_block_info(info);
    }

    let mut blob_store_tracker = BlobStoreCanonTracker::default();
    let mut last_finalized_block =
        FinalizedBlockTracker::new(client.finalized_block_number().ok().flatten());
    let mut dirty_addresses = HashSet::default();
    let mut maintained_state = MaintainedPoolState::InSync;
    let mut reload_accounts_fut = Fuse::terminated();
    let mut stale_eviction_interval = time::interval(config.max_tx_lifetime);
    let mut first_event = true;

    loop {
        trace!(target: "txpool", state=?maintained_state, "awaiting new block or reorg");

        metrics.set_dirty_accounts_len(dirty_addresses.len());
        let pool_info = pool.block_info();

        if maintained_state.is_drifted() {
            metrics.inc_drift();
            dirty_addresses = pool.unique_senders();
            maintained_state = MaintainedPoolState::InSync;
        }

        if !dirty_addresses.is_empty() && reload_accounts_fut.is_terminated() {
            let (tx, rx) = oneshot::channel();
            let c = client.clone();
            let at = pool_info.last_seen_block_hash;
            let fut = if dirty_addresses.len() > max_reload_accounts {
                let accs_to_reload =
                    dirty_addresses.iter().copied().take(max_reload_accounts).collect::<Vec<_>>();
                for acc in &accs_to_reload {
                    dirty_addresses.remove(acc);
                }
                async move {
                    let res = load_accounts(c, at, accs_to_reload);
                    let _ = tx.send(res);
                }
                .boxed()
            } else {
                let accs_to_reload = std::mem::take(&mut dirty_addresses);
                async move {
                    let res = load_accounts(c, at, accs_to_reload);
                    let _ = tx.send(res);
                }
                .boxed()
            };
            reload_accounts_fut = rx.fuse();
            task_spawner.spawn_blocking_task(fut);
        }

        if let Some(finalized) =
            last_finalized_block.update(client.finalized_block_number().ok().flatten()) &&
            let BlobStoreUpdates::Finalized(blobs) =
                blob_store_tracker.on_finalized_block(finalized)
        {
            metrics.inc_deleted_tracked_blobs(blobs.len());
            pool.delete_blobs(blobs);
            let pool = pool.clone();
            task_spawner.spawn_blocking_task(async move {
                debug!(target: "txpool", finalized_block = %finalized, "cleaning up blob store");
                pool.cleanup_blobs();
            });
        }

        let mut event = None;
        let mut reloaded = None;

        tokio::select! {
            res = &mut reload_accounts_fut =>  {
                reloaded = Some(res);
            }
            ev = events.next() =>  {
                 if ev.is_none() {
                    break;
                }
                event = ev;
                if first_event {
                    maintained_state = MaintainedPoolState::Drifted;
                    first_event = false
                }
            }
            _ = stale_eviction_interval.tick() => {
                let queued = pool.queued_transactions();
                let mut stale_blobs = Vec::new();
                let now = std::time::Instant::now();
                let stale_txs: Vec<_> = queued
                    .into_iter()
                    .filter(|tx| {
                        (tx.origin.is_external() || config.no_local_exemptions) &&
                            now - tx.timestamp > config.max_tx_lifetime
                    })
                    .map(|tx| {
                        if tx.is_eip4844() {
                            stale_blobs.push(*tx.hash());
                        }
                        *tx.hash()
                    })
                    .collect();
                debug!(target: "txpool", count=%stale_txs.len(), "removing stale transactions");
                pool.remove_transactions(stale_txs);
                pool.delete_blobs(stale_blobs);
            }
        }

        match reloaded {
            Some(Ok(Ok(LoadedAccounts { accounts, failed_to_load }))) => {
                dirty_addresses.extend(failed_to_load);
                pool.update_accounts(accounts);
            }
            Some(Ok(Err(res))) => {
                let (accs, err) = *res;
                debug!(target: "txpool", %err, "failed to load accounts");
                dirty_addresses.extend(accs);
            }
            Some(Err(_)) => {
                maintained_state = MaintainedPoolState::Drifted;
            }
            None => {}
        }

        let Some(event) = event else { continue };
        match event {
            CanonStateNotification::Reorg { old, new } => {
                let (old_blocks, old_state) = old.inner();
                let (new_blocks, new_state) = new.inner();
                let new_tip = new_blocks.tip();
                let new_first = new_blocks.first();
                let old_first = old_blocks.first();

                if !(old_first.parent_hash() == pool_info.last_seen_block_hash ||
                    new_first.parent_hash() == pool_info.last_seen_block_hash)
                {
                    maintained_state = MaintainedPoolState::Drifted;
                }

                let chain_spec = client.chain_spec();
                let pending_block_base_fee = chain_spec
                    .next_block_base_fee(new_tip.header(), new_tip.timestamp())
                    .unwrap_or_default();
                let pending_block_blob_fee = new_tip.header().maybe_next_block_blob_fee(
                    chain_spec.blob_params_at_timestamp(new_tip.timestamp()),
                );

                let new_changed_accounts: HashSet<_> =
                    new_state.changed_accounts().map(ChangedAccountEntry).collect();

                let missing_changed_acc = old_state
                    .accounts_iter()
                    .map(|(a, _)| a)
                    .filter(|addr| !new_changed_accounts.contains(addr));

                let mut changed_accounts =
                    match load_accounts(client.clone(), new_tip.hash(), missing_changed_acc) {
                        Ok(LoadedAccounts { accounts, failed_to_load }) => {
                            dirty_addresses.extend(failed_to_load);
                            accounts
                        }
                        Err(err) => {
                            let (addresses, err) = *err;
                            debug!(
                                target: "txpool",
                                %err,
                                "failed to load missing changed accounts at new tip: {:?}",
                                new_tip.hash()
                            );
                            dirty_addresses.extend(addresses);
                            vec![]
                        }
                    };

                changed_accounts.extend(new_changed_accounts.into_iter().map(|entry| entry.0));

                let new_mined_transactions: HashSet<_> = new_blocks.transaction_hashes().collect();

                let pruned_old_transactions = old_blocks
                    .transactions_ecrecovered()
                    .filter(|tx| !new_mined_transactions.contains(tx.tx_hash()))
                    .filter_map(|tx| {
                        if tx.is_eip4844() {
                            pool.get_blob(*tx.tx_hash())
                                .ok()
                                .flatten()
                                .map(Arc::unwrap_or_clone)
                                .and_then(|sidecar| {
                                    <P as TransactionPool>::Transaction::try_from_eip4844(
                                        tx, sidecar,
                                    )
                                })
                        } else {
                            <P as TransactionPool>::Transaction::try_from_consensus(tx).ok()
                        }
                    })
                    .collect::<Vec<_>>();
                let interop_active = chain_spec.is_interop_active_at_timestamp(new_tip.timestamp());
                let pruned_old_transactions = pruned_old_transactions
                    .into_iter()
                    .filter(|tx| {
                        if should_drop_reorged_interop_transaction(tx.access_list(), interop_active)
                        {
                            warn!(
                                target: "txpool",
                                hash = %tx.hash(),
                                "Dropping interop transaction during reorg"
                            );
                            return false;
                        }
                        true
                    })
                    .collect::<Vec<_>>();

                let update = reth_transaction_pool::CanonicalStateUpdate {
                    new_tip: new_tip.sealed_block(),
                    pending_block_base_fee,
                    pending_block_blob_fee,
                    changed_accounts,
                    mined_transactions: new_blocks.transaction_hashes().collect(),
                    update_kind: PoolUpdateKind::Reorg,
                };
                pool.on_canonical_state_change(update);

                metrics.inc_reinserted_transactions(pruned_old_transactions.len());
                let _ = pool.add_external_transactions(pruned_old_transactions).await;

                blob_store_tracker.add_new_chain_blocks(&new_blocks);
            }
            CanonStateNotification::Commit { new } => {
                let (blocks, state) = new.inner();
                let tip = blocks.tip();
                let chain_spec = client.chain_spec();
                let pending_block_base_fee = chain_spec
                    .next_block_base_fee(tip.header(), tip.timestamp())
                    .unwrap_or_default();
                let pending_block_blob_fee = tip.header().maybe_next_block_blob_fee(
                    chain_spec.blob_params_at_timestamp(tip.timestamp()),
                );

                let first_block = blocks.first();
                trace!(
                    target: "txpool",
                    first = first_block.number(),
                    tip = tip.number(),
                    pool_block = pool_info.last_seen_block_number,
                    "update pool on new commit"
                );

                let depth = tip.number().abs_diff(pool_info.last_seen_block_number);
                if depth > max_update_depth {
                    maintained_state = MaintainedPoolState::Drifted;
                    debug!(target: "txpool", ?depth, "skipping deep canonical update");
                    let info = BlockInfo {
                        block_gas_limit: tip.header().gas_limit(),
                        last_seen_block_hash: tip.hash(),
                        last_seen_block_number: tip.number(),
                        pending_basefee: pending_block_base_fee,
                        pending_blob_fee: pending_block_blob_fee,
                    };
                    pool.set_block_info(info);
                    blob_store_tracker.add_new_chain_blocks(&blocks);
                    continue;
                }

                let mut changed_accounts = Vec::with_capacity(state.state().len());
                for acc in state.changed_accounts() {
                    dirty_addresses.remove(&acc.address);
                    changed_accounts.push(acc);
                }

                let mined_transactions = blocks.transaction_hashes().collect();

                if first_block.parent_hash() != pool_info.last_seen_block_hash {
                    maintained_state = MaintainedPoolState::Drifted;
                }

                let update = reth_transaction_pool::CanonicalStateUpdate {
                    new_tip: tip.sealed_block(),
                    pending_block_base_fee,
                    pending_block_blob_fee,
                    changed_accounts,
                    mined_transactions,
                    update_kind: PoolUpdateKind::Commit,
                };
                pool.on_canonical_state_change(update);

                blob_store_tracker.add_new_chain_blocks(&blocks);

                if !chain_spec.is_osaka_active_at_timestamp(tip.timestamp()) &&
                    !chain_spec.is_osaka_active_at_timestamp(tip.timestamp().saturating_add(12)) &&
                    chain_spec.is_osaka_active_at_timestamp(tip.timestamp().saturating_add(24))
                {
                    let pool = pool.clone();
                    let spawner = task_spawner.clone();
                    let client = client.clone();
                    task_spawner.spawn_task(async move {
                        tokio::time::sleep(Duration::from_secs(4)).await;

                        let mut interval = tokio::time::interval(Duration::from_secs(1));
                        loop {
                            let last_iteration =
                                client.latest_header().ok().flatten().is_none_or(|header| {
                                    client
                                        .chain_spec()
                                        .is_osaka_active_at_timestamp(header.timestamp())
                                });

                            let AllPoolTransactions { pending, queued } = pool.all_transactions();
                            for tx in pending.into_iter().chain(queued).filter(|tx| tx.is_eip4844())
                            {
                                let tx_hash = *tx.hash();
                                let Ok(Some(sidecar)) = pool.get_blob(tx_hash) else {
                                    continue;
                                };
                                if !sidecar.is_eip4844() {
                                    continue;
                                }
                                let Some(tx) = pool.remove_transactions(vec![tx_hash]).pop() else {
                                    continue;
                                };
                                pool.delete_blob(tx_hash);

                                let BlobTransactionSidecarVariant::Eip4844(sidecar) =
                                    Arc::unwrap_or_clone(sidecar)
                                else {
                                    continue;
                                };

                                let converter = BlobSidecarConverter::new();
                                let pool = pool.clone();
                                spawner.spawn_task(async move {
                                    let Some(sidecar) = converter.convert(sidecar).await else {
                                        return;
                                    };

                                    let origin = tx.origin;
                                    let Some(tx) =
                                        reth_transaction_pool::EthPoolTransaction::try_from_eip4844(
                                            tx.to_consensus(),
                                            sidecar.into(),
                                        )
                                    else {
                                        return;
                                    };
                                    let _ = pool.add_transaction(origin, tx).await;
                                });
                            }

                            if last_iteration {
                                break;
                            }

                            interval.tick().await;
                        }
                    });
                }
            }
        }
    }
}

fn should_drop_reorged_interop_transaction(
    access_list: Option<&AccessList>,
    interop_active: bool,
) -> bool {
    interop_active &&
        access_list.is_some_and(|access_list| {
            parse_access_list_items_to_inbox_entries(access_list.iter()).next().is_some()
        })
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

    loop {
        let Some(event) = events.next().await else { break };
        if let CanonStateNotification::Commit { new } = event {
            let timestamp = new.tip().timestamp();
            let mut to_remove = Vec::new();
            let mut to_revalidate = Vec::new();
            let mut interop_count = 0;

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
                let revalidation_stream = supervisor_client.revalidate_interop_txs_stream(
                    to_revalidate,
                    timestamp,
                    TRANSACTION_VALIDITY_WINDOW,
                    MAX_SUPERVISOR_QUERIES,
                );

                futures_util::pin_mut!(revalidation_stream);

                while let Some((tx_item_from_stream, validation_result)) =
                    revalidation_stream.next().await
                {
                    match validation_result {
                        Some(Ok(())) => {
                            tx_item_from_stream
                                .set_interop_deadline(timestamp + TRANSACTION_VALIDITY_WINDOW);
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

                metrics.record_supervisor_duration(revalidation_start.elapsed());
            }

            if !to_remove.is_empty() {
                let removed = pool.remove_transactions(to_remove);
                metrics.inc_removed_tx_interop(removed.len());
            }
        }
    }
}

struct FinalizedBlockTracker {
    last_finalized_block: Option<BlockNumber>,
}

impl FinalizedBlockTracker {
    const fn new(last_finalized_block: Option<BlockNumber>) -> Self {
        Self { last_finalized_block }
    }

    fn update(&mut self, finalized_block: Option<BlockNumber>) -> Option<BlockNumber> {
        let finalized = finalized_block?;
        self.last_finalized_block.is_none_or(|last| last < finalized).then(|| {
            self.last_finalized_block = Some(finalized);
            finalized
        })
    }
}

#[derive(Debug, PartialEq, Eq)]
enum MaintainedPoolState {
    InSync,
    Drifted,
}

impl MaintainedPoolState {
    #[inline]
    const fn is_drifted(&self) -> bool {
        matches!(self, Self::Drifted)
    }
}

#[derive(Eq)]
struct ChangedAccountEntry(ChangedAccount);

impl PartialEq for ChangedAccountEntry {
    fn eq(&self, other: &Self) -> bool {
        self.0.address == other.0.address
    }
}

impl Hash for ChangedAccountEntry {
    fn hash<H: Hasher>(&self, state: &mut H) {
        self.0.address.hash(state);
    }
}

impl Borrow<Address> for ChangedAccountEntry {
    fn borrow(&self) -> &Address {
        &self.0.address
    }
}

#[derive(Default)]
struct LoadedAccounts {
    accounts: Vec<ChangedAccount>,
    failed_to_load: Vec<Address>,
}

fn load_accounts<Client, I>(
    client: Client,
    at: BlockHash,
    addresses: I,
) -> Result<LoadedAccounts, Box<(AddressSet, ProviderError)>>
where
    I: IntoIterator<Item = Address>,
    Client: StateProviderFactory,
{
    let addresses = addresses.into_iter();
    let mut res = LoadedAccounts::default();
    let state = match client.history_by_block_hash(at) {
        Ok(state) => state,
        Err(err) => return Err(Box::new((addresses.collect(), err))),
    };
    for addr in addresses {
        if let Ok(maybe_acc) = state.basic_account(&addr) {
            let acc = maybe_acc
                .map(|acc| ChangedAccount { address: addr, nonce: acc.nonce, balance: acc.balance })
                .unwrap_or_else(|| ChangedAccount::empty(addr));
            res.accounts.push(acc)
        } else {
            res.failed_to_load.push(addr);
        }
    }
    Ok(res)
}

#[cfg(test)]
mod tests {
    use super::should_drop_reorged_interop_transaction;
    use crate::supervisor::CROSS_L2_INBOX_ADDRESS;
    use alloy_eips::eip2930::{AccessList, AccessListItem};
    use alloy_primitives::{Address, B256};

    #[test]
    fn keeps_reorged_transactions_when_interop_is_inactive() {
        let interop_access_list = AccessList::from(vec![AccessListItem {
            address: CROSS_L2_INBOX_ADDRESS,
            storage_keys: vec![B256::ZERO],
        }]);

        assert!(!should_drop_reorged_interop_transaction(None, false));
        assert!(!should_drop_reorged_interop_transaction(Some(&interop_access_list), false));
    }

    #[test]
    fn drops_only_cross_l2_inbox_transactions_during_interop() {
        let interop_access_list = AccessList::from(vec![AccessListItem {
            address: CROSS_L2_INBOX_ADDRESS,
            storage_keys: vec![B256::ZERO],
        }]);
        let non_interop_access_list = AccessList::from(vec![AccessListItem {
            address: Address::repeat_byte(0x11),
            storage_keys: vec![B256::ZERO],
        }]);

        assert!(should_drop_reorged_interop_transaction(Some(&interop_access_list), true));
        assert!(!should_drop_reorged_interop_transaction(Some(&non_interop_access_list), true));
        assert!(!should_drop_reorged_interop_transaction(None, true));
    }
}
