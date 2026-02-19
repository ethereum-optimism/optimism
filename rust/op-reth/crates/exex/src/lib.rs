//! ExEx unique for OP-Reth. See also [`reth_exex`] for more op-reth execution extensions.

#![doc(
    html_logo_url = "https://raw.githubusercontent.com/paradigmxyz/reth/main/assets/reth-docs.png",
    html_favicon_url = "https://avatars0.githubusercontent.com/u/97369466?s=256",
    issue_tracker_base_url = "https://github.com/paradigmxyz/reth/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]

use alloy_consensus::BlockHeader;
use alloy_eips::eip1898::BlockWithParent;
use futures_util::TryStreamExt;
use reth_execution_types::Chain;
use reth_exex::{ExExContext, ExExEvent, ExExNotification};
use reth_node_api::{FullNodeComponents, NodePrimitives, NodeTypes};
use reth_optimism_trie::{
    OpProofStoragePrunerTask, OpProofsStorage, OpProofsStore, live::LiveTrieCollector,
};
use reth_provider::{BlockNumReader, BlockReader, TransactionVariant};
use reth_trie::{HashedPostStateSorted, SortedTrieData, updates::TrieUpdatesSorted};
use std::{sync::Arc, time::Duration};
use tokio::{sync::watch, task, time};
use tracing::{debug, error, info};

// Safety threshold for maximum blocks to prune automatically on startup.
// If the required prune exceeds this, the node will error out and require manual pruning. Default
// is 1000 blocks.
const MAX_PRUNE_BLOCKS_STARTUP: u64 = 1000;

/// How many blocks to process in a single batch before yielding. Default is 50 blocks.
const SYNC_BLOCKS_BATCH_SIZE: usize = 50;

/// How close to tip before we process blocks in real-time vs batch. Default is 1024 blocks.
const REAL_TIME_BLOCKS_THRESHOLD: u64 = 1024;

/// How long to sleep when sync task is caught up. Default is 5 seconds.
const SYNC_IDLE_SLEEP_SECS: u64 = 5;

/// Default proofs history window: 1 month of blocks at 2s block time
const DEFAULT_PROOFS_HISTORY_WINDOW: u64 = 1_296_000;

/// Default interval between proof-storage prune runs. Default is 15 seconds.
const DEFAULT_PRUNE_INTERVAL: Duration = Duration::from_secs(15);

/// Default verification interval: disabled
const DEFAULT_VERIFICATION_INTERVAL: u64 = 0; // disabled

/// Builder for [`OpProofsExEx`].
#[derive(Debug)]
pub struct OpProofsExExBuilder<Node, Storage>
where
    Node: FullNodeComponents,
{
    ctx: ExExContext<Node>,
    storage: OpProofsStorage<Storage>,
    proofs_history_window: u64,
    proofs_history_prune_interval: Duration,
    verification_interval: u64,
}

impl<Node, Storage> OpProofsExExBuilder<Node, Storage>
where
    Node: FullNodeComponents,
{
    /// Create a new builder with required parameters and defaults.
    pub const fn new(ctx: ExExContext<Node>, storage: OpProofsStorage<Storage>) -> Self {
        Self {
            ctx,
            storage,
            proofs_history_window: DEFAULT_PROOFS_HISTORY_WINDOW,
            proofs_history_prune_interval: DEFAULT_PRUNE_INTERVAL,
            verification_interval: DEFAULT_VERIFICATION_INTERVAL,
        }
    }

    /// Sets the window to span blocks for proofs history.
    pub const fn with_proofs_history_window(mut self, window: u64) -> Self {
        self.proofs_history_window = window;
        self
    }

    /// Sets the interval between proof-storage prune runs.
    pub const fn with_proofs_history_prune_interval(mut self, interval: Duration) -> Self {
        self.proofs_history_prune_interval = interval;
        self
    }

    /// Sets the verification interval.
    pub const fn with_verification_interval(mut self, interval: u64) -> Self {
        self.verification_interval = interval;
        self
    }

    /// Builds the [`OpProofsExEx`].
    pub fn build(self) -> OpProofsExEx<Node, Storage> {
        OpProofsExEx {
            ctx: self.ctx,
            storage: self.storage,
            proofs_history_window: self.proofs_history_window,
            proofs_history_prune_interval: self.proofs_history_prune_interval,
            verification_interval: self.verification_interval,
        }
    }
}

/// OP Proofs ExEx - processes blocks and tracks state changes within fault proof window.
///
/// Saves and serves trie nodes to make proofs faster. This handles the process of
/// saving the current state, new blocks as they're added, and serving proof RPCs
/// based on the saved data.
///
/// # Examples
///
/// The following example shows how to install the ExEx with either in-memory or persistent storage.
/// This can be used when launching an OP-Reth node via a binary.
/// We are currently using it in optimism/bin/src/main.rs.
///
/// ```
/// use futures_util::FutureExt;
/// use reth_db::test_utils::create_test_rw_db;
/// use reth_node_api::NodeTypesWithDBAdapter;
/// use reth_node_builder::{NodeBuilder, NodeConfig};
/// use reth_optimism_chainspec::BASE_MAINNET;
/// use reth_optimism_exex::OpProofsExEx;
/// use reth_optimism_node::{OpNode, args::RollupArgs};
/// use reth_optimism_trie::{InMemoryProofsStorage, OpProofsStorage, db::MdbxProofsStorage};
/// use reth_provider::providers::BlockchainProvider;
/// use std::{sync::Arc, time::Duration};
///
/// let config = NodeConfig::new(BASE_MAINNET.clone());
/// let db = create_test_rw_db();
/// let args = RollupArgs::default();
/// let op_node = OpNode::new(args);
///
/// // Create in-memory or persistent storage
/// let storage: OpProofsStorage<Arc<InMemoryProofsStorage>> =
///     Arc::new(InMemoryProofsStorage::new()).into();
///
/// // Example for creating persistent storage
/// # let temp_dir = tempfile::tempdir().expect("Failed to create temp dir");
/// # let storage_path = temp_dir.path().join("proofs_storage");
///
/// # let storage: OpProofsStorage<Arc<MdbxProofsStorage>> = Arc::new(
/// #    MdbxProofsStorage::new(&storage_path).expect("Failed to create MdbxProofsStorage"),
/// # ).into();
///
/// let storage_exec = storage.clone();
/// let proofs_history_window = 1_296_000u64;
/// let proofs_history_prune_interval = Duration::from_secs(3600);
///
/// // Verification interval: perform full execution every N blocks
/// let verification_interval = 0; // 0 = disabled, 100 = verify every 100 blocks
///
/// // Can also use install_exex_if along with a boolean flag
/// // Set this based on your configuration or CLI args
/// let _builder = NodeBuilder::new(config)
///     .with_database(db)
///     .with_types_and_provider::<OpNode, BlockchainProvider<NodeTypesWithDBAdapter<OpNode, _>>>()
///     .with_components(op_node.components())
///     .install_exex("proofs-history", move |exex_context| async move {
///         Ok(OpProofsExEx::builder(exex_context, storage_exec)
///             .with_proofs_history_window(proofs_history_window)
///             .with_proofs_history_prune_interval(proofs_history_prune_interval)
///             .with_verification_interval(verification_interval)
///             .build()
///             .run()
///             .boxed())
///     })
///     .on_node_started(|_full_node| Ok(()))
///     .check_launch();
/// ```
#[derive(Debug)]
pub struct OpProofsExEx<Node, Storage>
where
    Node: FullNodeComponents,
{
    /// The ExEx context containing the node related utilities e.g. provider, notifications,
    /// events.
    ctx: ExExContext<Node>,
    /// The type of storage DB.
    storage: OpProofsStorage<Storage>,
    /// The window to span blocks for proofs history. Value is the number of blocks, received as
    /// cli arg.
    proofs_history_window: u64,
    /// Interval between proof-storage prune runs
    proofs_history_prune_interval: Duration,
    /// Verification interval: perform full block execution every N blocks for data integrity.
    /// If 0, verification is disabled (always use fast path when available).
    /// If 1, verification is always enabled (always execute blocks).
    verification_interval: u64,
}

impl<Node, Storage> OpProofsExEx<Node, Storage>
where
    Node: FullNodeComponents,
{
    /// Create a new `OpProofsExEx` instance.
    pub fn new(ctx: ExExContext<Node>, storage: OpProofsStorage<Storage>) -> Self {
        OpProofsExExBuilder::new(ctx, storage).build()
    }

    /// Create a new builder for `OpProofsExEx`.
    pub const fn builder(
        ctx: ExExContext<Node>,
        storage: OpProofsStorage<Storage>,
    ) -> OpProofsExExBuilder<Node, Storage> {
        OpProofsExExBuilder::new(ctx, storage)
    }
}

impl<Node, Storage, Primitives> OpProofsExEx<Node, Storage>
where
    Node: FullNodeComponents<Types: NodeTypes<Primitives = Primitives>>,
    Primitives: NodePrimitives,
    Storage: OpProofsStore + Clone + 'static,
{
    /// Main execution loop for the ExEx
    pub async fn run(mut self) -> eyre::Result<()> {
        self.ensure_initialized()?;
        let sync_target_tx = self.spawn_sync_task();

        let prune_task = OpProofStoragePrunerTask::new(
            self.storage.clone(),
            self.ctx.provider().clone(),
            self.proofs_history_window,
            self.proofs_history_prune_interval,
        );
        self.ctx
            .task_executor()
            .spawn_with_graceful_shutdown_signal(|signal| Box::pin(prune_task.run(signal)));

        let collector = LiveTrieCollector::new(
            self.ctx.evm_config().clone(),
            self.ctx.provider().clone(),
            &self.storage,
        );

        while let Some(notification) = self.ctx.notifications.try_next().await? {
            self.handle_notification(notification, &collector, &sync_target_tx)?;
        }

        Ok(())
    }

    /// Ensure proofs storage is initialized
    fn ensure_initialized(&self) -> eyre::Result<()> {
        // Check if proofs storage is initialized
        let earliest_block_number = match self.storage.get_earliest_block_number()? {
            Some((n, _)) => n,
            None => {
                return Err(eyre::eyre!(
                    "Proofs storage not initialized. Please run 'op-reth initialize-op-proofs --proofs-history.storage-path <PATH>' first."
                ));
            }
        };

        let latest_block_number = match self.storage.get_latest_block_number()? {
            Some((n, _)) => n,
            None => {
                return Err(eyre::eyre!(
                    "Proofs storage not initialized. Please run 'op-reth initialize-op-proofs --proofs-history.storage-path <PATH>' first."
                ));
            }
        };

        // Check if we have accumulated too much history for the configured window.
        // If the gap between what we have and what we want to keep is too large, the auto-pruner
        // will stall the node.
        let target_earliest = latest_block_number.saturating_sub(self.proofs_history_window);
        if target_earliest > earliest_block_number {
            let blocks_to_prune = target_earliest - earliest_block_number;
            if blocks_to_prune > MAX_PRUNE_BLOCKS_STARTUP {
                return Err(eyre::eyre!(
                    "Configuration requires pruning {} blocks, which exceeds the safety threshold of {}. \
                     Huge prune operations can stall the node. \
                     Please run 'op-reth proofs prune' manually before starting the node.",
                    blocks_to_prune,
                    MAX_PRUNE_BLOCKS_STARTUP
                ));
            }
        }

        // Need to update the earliest block metric on startup as this is not called frequently and
        // can show outdated info. When metrics are disabled, this is a no-op.
        #[cfg(feature = "metrics")]
        {
            self.storage
                .metrics()
                .block_metrics()
                .earliest_number
                .set(earliest_block_number as f64);
        }

        Ok(())
    }

    /// Spawn the background sync task and return the target sender
    fn spawn_sync_task(&self) -> watch::Sender<u64> {
        let (sync_target_tx, sync_target_rx) = watch::channel(0u64);

        let task_storage = self.storage.clone();
        let task_provider = self.ctx.provider().clone();
        let task_evm_config = self.ctx.evm_config().clone();

        self.ctx.task_executor().spawn_critical_task(
            "optimism::exex::proofs_storage_sync_loop",
            async move {
                let storage = task_storage.clone();
                let task_collector =
                    LiveTrieCollector::new(task_evm_config, task_provider.clone(), &storage);
                Self::sync_loop(sync_target_rx, task_storage, task_provider, &task_collector).await;
            },
        );

        sync_target_tx
    }

    /// Background sync loop that processes blocks up to the target
    async fn sync_loop(
        mut sync_target_rx: watch::Receiver<u64>,
        storage: OpProofsStorage<Storage>,
        provider: Node::Provider,
        collector: &LiveTrieCollector<'_, Node::Evm, Node::Provider, Storage>,
    ) {
        debug!(target: "optimism::exex", "Starting proofs storage sync loop");

        loop {
            let target = *sync_target_rx.borrow_and_update();
            let latest = match storage.get_latest_block_number() {
                Ok(Some((n, _))) => n,
                Ok(None) => {
                    error!(target: "optimism::exex", "No blocks stored in proofs storage during sync loop");
                    continue;
                }
                Err(e) => {
                    error!(target: "optimism::exex", error = ?e, "Failed to get latest block");
                    continue;
                }
            };

            if latest >= target {
                time::sleep(Duration::from_secs(SYNC_IDLE_SLEEP_SECS)).await;
                continue;
            }

            // Process one batch
            if let Err(e) =
                Self::process_batch(latest, target, &provider, collector, SYNC_BLOCKS_BATCH_SIZE)
            {
                error!(target: "optimism::exex", error = ?e, "Batch processing failed");
            }

            // Yield to allow other tasks to run
            debug!(target: "optimism::exex", latest_stored = latest, target, "Batch processed, yielding");
            task::yield_now().await;
        }
    }

    /// Process a batch of blocks from start to target (up to `batch_size`)
    fn process_batch(
        start: u64,
        target: u64,
        provider: &Node::Provider,
        collector: &LiveTrieCollector<'_, Node::Evm, Node::Provider, Storage>,
        batch_size: usize,
    ) -> eyre::Result<()> {
        let end = (start + batch_size as u64).min(target);
        debug!(
            target: "optimism::exex",
            start,
            end,
            "Processing proofs storage sync batch"
        );

        for block_num in (start + 1)..=end {
            let block = provider
                .recovered_block(block_num.into(), TransactionVariant::NoHash)?
                .ok_or_else(|| eyre::eyre!("Missing block {}", block_num))?;

            collector.execute_and_store_block_updates(&block)?;
        }

        Ok(())
    }

    fn handle_notification(
        &self,
        notification: ExExNotification<Primitives>,
        collector: &LiveTrieCollector<'_, Node::Evm, Node::Provider, Storage>,
        sync_target_tx: &watch::Sender<u64>,
    ) -> eyre::Result<()> {
        let latest_stored = match self.storage.get_latest_block_number()? {
            Some((n, _)) => n,
            None => {
                return Err(eyre::eyre!("No blocks stored in proofs storage"));
            }
        };

        match &notification {
            ExExNotification::ChainCommitted { new } => {
                self.handle_chain_committed(new.clone(), latest_stored, collector, sync_target_tx)?
            }
            ExExNotification::ChainReorged { old, new } => {
                self.handle_chain_reorged(old.clone(), new.clone(), latest_stored, collector)?
            }
            ExExNotification::ChainReverted { old } => {
                self.handle_chain_reverted(old.clone(), latest_stored, collector)?
            }
        }

        if let Some(committed_chain) = notification.committed_chain() {
            self.ctx.events.send(ExExEvent::FinishedHeight(committed_chain.tip().num_hash()))?;
        }

        Ok(())
    }

    fn handle_chain_committed(
        &self,
        new: Arc<Chain<Primitives>>,
        latest_stored: u64,
        collector: &LiveTrieCollector<'_, Node::Evm, Node::Provider, Storage>,
        sync_target_tx: &watch::Sender<u64>,
    ) -> eyre::Result<()> {
        debug!(
            target: "optimism::exex",
            block_number = new.tip().number(),
            block_hash = ?new.tip().hash(),
            "ChainCommitted notification received",
        );

        // If tip is not newer than what we have, nothing to do.
        if new.tip().number() <= latest_stored {
            debug!(
                target: "optimism::exex",
                block_number = new.tip().number(),
                latest_stored,
                "Already processed, skipping"
            );
            return Ok(());
        }

        let best_block = self.ctx.provider().best_block_number()?;
        let is_sequential = new.tip().number() == latest_stored + 1;
        let is_near_tip =
            best_block.saturating_sub(new.tip().number()) < REAL_TIME_BLOCKS_THRESHOLD;

        if is_sequential && is_near_tip {
            debug!(
                target: "optimism::exex",
                block_number = new.tip().number(),
                latest_stored,
                best_block,
                "Processing in real-time"
            );

            // Process each block from latest_stored + 1 to tip
            let start = latest_stored.saturating_add(1);
            for block_number in start..=new.tip().number() {
                self.process_block(block_number, &new, collector)?;
            }
        } else {
            debug!(
                target: "optimism::exex",
                block_number = new.tip().number(),
                latest_stored,
                best_block,
                is_sequential,
                is_near_tip,
                "Scheduling batch processing via sync task"
            );

            // Update the sync target to the new tip
            sync_target_tx.send(new.tip().number())?;
        }

        Ok(())
    }

    /// Process a single block - either from chain or provider
    fn process_block(
        &self,
        block_number: u64,
        chain: &Chain<Primitives>,
        collector: &LiveTrieCollector<'_, Node::Evm, Node::Provider, Storage>,
    ) -> eyre::Result<()> {
        // Check if this block should be verified via full execution
        let should_verify = self.verification_interval > 0 &&
            block_number.is_multiple_of(self.verification_interval);

        // Try to get block data from the chain first
        // 1. Fast Path: Try to use pre-computed state from the notification
        if let Some(block) = chain.blocks().get(&block_number) {
            // Check if we have BOTH trie updates and hashed state.
            // If either is missing, we fall back to execution to ensure data integrity.
            if let Some((trie_updates, hashed_state)) = chain.trie_data_at(block_number).map(|d| {
                let SortedTrieData { hashed_state, trie_updates } = d.get();
                (trie_updates, hashed_state)
            }) {
                // Use fast path only if we're not scheduled to verify this block
                if !should_verify {
                    debug!(
                        target: "optimism::exex",
                        block_number,
                        "Using pre-computed state updates from notification"
                    );

                    collector.store_block_updates(
                        block.block_with_parent(),
                        (**trie_updates).clone(),
                        (**hashed_state).clone(),
                    )?;

                    return Ok(());
                }

                info!(
                    target: "optimism::exex",
                    block_number,
                    verification_interval = self.verification_interval,
                    "Periodic verification: performing full block execution"
                );
            }

            debug!(
                target: "optimism::exex",
                block_number,
                "Block present in notification but state updates missing, falling back to execution"
            );
        }

        // 2. Slow Path: Block not in chain (or state missing), fetch from provider and execute
        debug!(
            target: "optimism::exex",
            block_number,
            "Fetching block from provider for execution",
        );

        let block = self
            .ctx
            .provider()
            .recovered_block(block_number.into(), TransactionVariant::NoHash)?
            .ok_or_else(|| eyre::eyre!("Missing block {} in provider", block_number))?;

        collector.execute_and_store_block_updates(&block)?;
        Ok(())
    }

    fn handle_chain_reorged(
        &self,
        old: Arc<Chain<Primitives>>,
        new: Arc<Chain<Primitives>>,
        latest_stored: u64,
        collector: &LiveTrieCollector<'_, Node::Evm, Node::Provider, Storage>,
    ) -> eyre::Result<()> {
        info!(
            old_block_number = old.tip().number(),
            old_block_hash = ?old.tip().hash(),
            new_block_number = new.tip().number(),
            new_block_hash = ?new.tip().hash(),
            "ChainReorged notification received",
        );

        if old.first().number() > latest_stored {
            debug!(target: "optimism::exex", "Reorg beyond stored blocks, skipping");
            return Ok(());
        }

        // find the common ancestor
        let mut block_updates: Vec<(
            BlockWithParent,
            Arc<TrieUpdatesSorted>,
            Arc<HashedPostStateSorted>,
        )> = Vec::with_capacity(new.len());
        for block_number in new.blocks().keys() {
            // verify if the fork point matches
            if old.fork_block() != new.fork_block() {
                return Err(eyre::eyre!(
                    "Fork blocks do not match: old fork block {:?}, new fork block {:?}",
                    old.fork_block(),
                    new.fork_block()
                ));
            }

            let block = new
                .blocks()
                .get(block_number)
                .ok_or_else(|| eyre::eyre!("Missing block {} in new chain", block_number))?;
            let trie_data = new
                .trie_data_at(*block_number)
                .ok_or_else(|| {
                    eyre::eyre!("Missing Trie data for block {} in new chain", block_number)
                })?
                .get();
            let trie_updates = &trie_data.trie_updates;
            let hashed_state = &trie_data.hashed_state;

            block_updates.push((
                block.block_with_parent(),
                trie_updates.clone(),
                hashed_state.clone(),
            ));
        }

        collector.unwind_and_store_block_updates(block_updates)?;

        Ok(())
    }

    fn handle_chain_reverted(
        &self,
        old: Arc<Chain<Primitives>>,
        latest_stored: u64,
        collector: &LiveTrieCollector<'_, Node::Evm, Node::Provider, Storage>,
    ) -> eyre::Result<()> {
        info!(
            target: "optimism::exex",
            old_block_number = old.tip().number(),
            old_block_hash = ?old.tip().hash(),
            "ChainReverted notification received",
        );

        if old.first().number() > latest_stored {
            debug!(
                target: "optimism::exex",
                first_block_number = old.first().number(),
                latest_stored = latest_stored,
                "Fork block number is greater than latest stored, skipping",
            );
            return Ok(());
        }

        collector.unwind_history(old.first().block_with_parent())?;
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_consensus::private::alloy_primitives::B256;
    use alloy_eips::{BlockNumHash, NumHash, eip1898::BlockWithParent};
    use reth_db::test_utils::tempdir_path;
    use reth_ethereum_primitives::{Block, Receipt};
    use reth_execution_types::{Chain, ExecutionOutcome};
    use reth_optimism_trie::{
        BlockStateDiff, OpProofsStorage, OpProofsStore, db::MdbxProofsStorage,
    };
    use reth_primitives_traits::RecoveredBlock;
    use reth_trie::{HashedPostStateSorted, LazyTrieData, updates::TrieUpdatesSorted};
    use std::{collections::BTreeMap, default::Default, sync::Arc, time::Duration};

    // -------------------------------------------------------------------------
    // Helpers: deterministic blocks and deterministic Chain with precomputed updates
    // -------------------------------------------------------------------------
    fn b256(byte: u8) -> B256 {
        B256::new([byte; 32])
    }

    // deterministic hash from block number: 0 -> 0x00.., 1 -> 0x01.., etc.
    fn hash_for_num(num: u64) -> B256 {
        // if you only care about small test numbers, this is enough:
        b256(num as u8)

        // If you want to avoid wrapping when num > 255, use something like:
        // let mut out = [0u8; 32];
        // out[0..8].copy_from_slice(&num.to_be_bytes());
        // B256::new(out)
    }

    fn mk_block(num: u64) -> RecoveredBlock<Block> {
        let mut b: RecoveredBlock<Block> = Default::default();
        b.set_block_number(num);
        b.set_hash(hash_for_num(num));
        b.set_parent_hash(hash_for_num(num - 1));
        b
    }

    fn mk_chain_with_updates(
        from: u64,
        to: u64,
        hash_override: Option<B256>,
    ) -> Chain<reth_ethereum_primitives::EthPrimitives> {
        let mut blocks: Vec<RecoveredBlock<Block>> = Vec::new();
        let mut trie_data = BTreeMap::new();

        for n in from..=to {
            let mut b = mk_block(n);
            if let Some(hash) = hash_override {
                b.set_hash(hash);
            }
            blocks.push(b);

            let data = LazyTrieData::ready(
                Arc::new(HashedPostStateSorted::default()),
                Arc::new(TrieUpdatesSorted::default()),
            );
            trie_data.insert(n, data);
        }

        let execution_outcome: ExecutionOutcome<Receipt> = ExecutionOutcome {
            bundle: Default::default(),
            receipts: Vec::new(),
            requests: Vec::new(),
            first_block: from,
        };

        Chain::new(blocks, execution_outcome, trie_data)
    }

    // Init_storage to the genesis block
    fn init_storage<S: OpProofsStore>(storage: OpProofsStorage<S>) {
        let genesis_block = NumHash::new(0, b256(0x00));
        storage
            .set_earliest_block_number(genesis_block.number, genesis_block.hash)
            .expect("set earliest");
        storage
            .store_trie_updates(
                BlockWithParent::new(genesis_block.hash, genesis_block),
                BlockStateDiff::default(),
            )
            .expect("store trie update");
    }

    // Initialize exex with config
    fn build_test_exex<NodeT, Store>(
        ctx: ExExContext<NodeT>,
        storage: OpProofsStorage<Store>,
    ) -> OpProofsExEx<NodeT, Store>
    where
        NodeT: FullNodeComponents,
        Store: OpProofsStore + Clone + 'static,
    {
        OpProofsExEx::builder(ctx, storage)
            .with_proofs_history_window(20)
            .with_proofs_history_prune_interval(Duration::from_secs(3600))
            .with_verification_interval(1000)
            .build()
    }

    #[tokio::test]
    async fn handle_notification_chain_committed() {
        // MDBX proofs storage
        let dir = tempdir_path();
        let store = Arc::new(MdbxProofsStorage::new(dir.as_path()).expect("env"));
        let proofs: OpProofsStorage<Arc<MdbxProofsStorage>> = store.clone().into();

        init_storage(proofs.clone());

        let (ctx, _handle) =
            reth_exex_test_utils::test_exex_context().await.expect("exex test context");

        let collector = LiveTrieCollector::new(
            ctx.components.components.evm_config.clone(),
            ctx.components.provider.clone(),
            &proofs,
        );
        let exex = build_test_exex(ctx, proofs.clone());

        // Notification: chain committed 1..5
        let new_chain = Arc::new(mk_chain_with_updates(1, 1, None));
        let notif = ExExNotification::ChainCommitted { new: new_chain };

        let (sync_target_tx, _) = tokio::sync::watch::channel(0u64);

        exex.handle_notification(notif, &collector, &sync_target_tx).expect("handle chain commit");

        let latest = proofs.get_latest_block_number().expect("get latest block").expect("ok").0;
        assert_eq!(latest, 1);
    }

    #[tokio::test]
    async fn handle_notification_chain_committed_skips_already_processed() {
        // MDBX proofs storage
        let dir = tempdir_path();
        let store = Arc::new(MdbxProofsStorage::new(dir.as_path()).expect("env"));
        let proofs: OpProofsStorage<Arc<MdbxProofsStorage>> = store.clone().into();

        init_storage(proofs.clone());

        let (ctx, _handle) =
            reth_exex_test_utils::test_exex_context().await.expect("exex test context");

        let collector = LiveTrieCollector::new(
            ctx.components.components.evm_config.clone(),
            ctx.components.provider.clone(),
            &proofs,
        );

        let exex = build_test_exex(ctx, proofs.clone());

        let (sync_target_tx, _) = tokio::sync::watch::channel(0u64);
        // Process blocks 1..5 sequentially to trigger real-time path (synchronous)
        for i in 1..=5 {
            let new_chain = Arc::new(mk_chain_with_updates(i, i, None));
            let notif = ExExNotification::ChainCommitted { new: new_chain };
            exex.handle_notification(notif, &collector, &sync_target_tx)
                .expect("handle chain commit");
        }

        let latest = proofs.get_latest_block_number().expect("get latest block").expect("ok").0;
        assert_eq!(latest, 5);

        // Try to handle already processed notification
        let new_chain = Arc::new(mk_chain_with_updates(5, 5, Some(hash_for_num(10))));
        let notif = ExExNotification::ChainCommitted { new: new_chain };
        exex.handle_notification(notif, &collector, &sync_target_tx).expect("handle chain commit");
        let latest = proofs.get_latest_block_number().expect("get latest block").expect("ok");
        assert_eq!(latest.0, 5);
        assert_eq!(latest.1, hash_for_num(5)); // block was not updated
    }

    #[tokio::test]
    async fn handle_notification_chain_reorged() {
        // MDBX proofs storage
        let dir = tempdir_path();
        let store = Arc::new(MdbxProofsStorage::new(dir.as_path()).expect("env"));
        let proofs: OpProofsStorage<Arc<MdbxProofsStorage>> = store.clone().into();

        init_storage(proofs.clone());

        let (ctx, _handle) =
            reth_exex_test_utils::test_exex_context().await.expect("exex test context");

        let collector = LiveTrieCollector::new(
            ctx.components.components.evm_config.clone(),
            ctx.components.provider.clone(),
            &proofs,
        );

        let exex = build_test_exex(ctx, proofs.clone());

        let (sync_target_tx, _) = tokio::sync::watch::channel(0u64);

        for i in 1..=10 {
            let new_chain = Arc::new(mk_chain_with_updates(i, i, None));
            let notif = ExExNotification::ChainCommitted { new: new_chain };
            exex.handle_notification(notif, &collector, &sync_target_tx)
                .expect("handle chain commit");
        }

        let latest = proofs.get_latest_block_number().expect("get latest block").expect("ok").0;
        assert_eq!(latest, 10);

        // Now the tip is 10, and we want to reorg from block 6..12
        let old_chain = Arc::new(mk_chain_with_updates(6, 10, None));
        let new_chain = Arc::new(mk_chain_with_updates(6, 12, None));

        // Notification: chain reorged 6..12
        let notif = ExExNotification::ChainReorged { new: new_chain, old: old_chain };

        exex.handle_notification(notif, &collector, &sync_target_tx)
            .expect("handle chain re-orged");
        let latest = proofs.get_latest_block_number().expect("get latest block").expect("ok").0;
        assert_eq!(latest, 12);
    }

    #[tokio::test]
    async fn handle_notification_chain_reorged_skips_beyond_stored_blocks() {
        // MDBX proofs storage
        let dir = tempdir_path();
        let store = Arc::new(MdbxProofsStorage::new(dir.as_path()).expect("env"));
        let proofs: OpProofsStorage<Arc<MdbxProofsStorage>> = store.clone().into();

        init_storage(proofs.clone());

        let (ctx, _handle) =
            reth_exex_test_utils::test_exex_context().await.expect("exex test context");

        let collector = LiveTrieCollector::new(
            ctx.components.components.evm_config.clone(),
            ctx.components.provider.clone(),
            &proofs,
        );

        let exex = build_test_exex(ctx, proofs.clone());

        let (sync_target_tx, _) = tokio::sync::watch::channel(0u64);

        for i in 1..=10 {
            let new_chain = Arc::new(mk_chain_with_updates(i, i, None));
            let notif = ExExNotification::ChainCommitted { new: new_chain };

            exex.handle_notification(notif, &collector, &sync_target_tx)
                .expect("handle chain commit");
        }

        let latest = proofs.get_latest_block_number().expect("get latest block").expect("ok").0;
        assert_eq!(latest, 10);

        // Now the tip is 10, and we want to reorg from block 12..15
        let old_chain = Arc::new(mk_chain_with_updates(12, 15, None));
        let new_chain = Arc::new(mk_chain_with_updates(10, 20, None));

        // Notification: chain reorged 12..15
        let notif = ExExNotification::ChainReorged { new: new_chain, old: old_chain };

        exex.handle_notification(notif, &collector, &sync_target_tx)
            .expect("handle chain re-orged");
        let latest = proofs.get_latest_block_number().expect("get latest block").expect("ok").0;
        assert_eq!(latest, 10);
    }

    #[tokio::test]
    async fn handle_notification_chain_reverted() {
        // MDBX proofs storage
        let dir = tempdir_path();
        let store = Arc::new(MdbxProofsStorage::new(dir.as_path()).expect("env"));
        let proofs: OpProofsStorage<Arc<MdbxProofsStorage>> = store.clone().into();

        init_storage(proofs.clone());

        let (ctx, _handle) =
            reth_exex_test_utils::test_exex_context().await.expect("exex test context");

        let collector = LiveTrieCollector::new(
            ctx.components.components.evm_config.clone(),
            ctx.components.provider.clone(),
            &proofs,
        );

        let exex = build_test_exex(ctx, proofs.clone());

        let (sync_target_tx, _) = tokio::sync::watch::channel(0u64);

        for i in 1..=10 {
            let new_chain = Arc::new(mk_chain_with_updates(i, i, None));
            let notif = ExExNotification::ChainCommitted { new: new_chain };

            exex.handle_notification(notif, &collector, &sync_target_tx)
                .expect("handle chain commit");
        }

        let latest = proofs.get_latest_block_number().expect("get latest block").expect("ok").0;
        assert_eq!(latest, 10);

        // Now the tip is 10, and we want to revert from block 9..10
        let old_chain = Arc::new(mk_chain_with_updates(9, 10, None));

        // Notification: chain reverted 9..10
        let notif = ExExNotification::ChainReverted { old: old_chain };

        exex.handle_notification(notif, &collector, &sync_target_tx)
            .expect("handle chain reverted");
        let latest = proofs.get_latest_block_number().expect("get latest block").expect("ok").0;
        assert_eq!(latest, 8);
    }

    #[tokio::test]
    async fn handle_notification_chain_reverted_skips_beyond_stored_blocks() {
        // MDBX proofs storage
        let dir = tempdir_path();
        let store = Arc::new(MdbxProofsStorage::new(dir.as_path()).expect("env"));
        let proofs: OpProofsStorage<Arc<MdbxProofsStorage>> = store.clone().into();

        init_storage(proofs.clone());

        let (ctx, _handle) =
            reth_exex_test_utils::test_exex_context().await.expect("exex test context");

        let collector = LiveTrieCollector::new(
            ctx.components.components.evm_config.clone(),
            ctx.components.provider.clone(),
            &proofs,
        );

        let exex = build_test_exex(ctx, proofs.clone());

        let (sync_target_tx, _) = tokio::sync::watch::channel(0u64);

        for i in 1..=5 {
            let new_chain = Arc::new(mk_chain_with_updates(i, i, None));
            let notif = ExExNotification::ChainCommitted { new: new_chain };

            exex.handle_notification(notif, &collector, &sync_target_tx)
                .expect("handle chain commit");
        }

        let latest = proofs.get_latest_block_number().expect("get latest block").expect("ok").0;
        assert_eq!(latest, 5);

        // Now the tip is 10, and we want to revert from block 9..10
        let old_chain = Arc::new(mk_chain_with_updates(9, 10, None));

        // Notification: chain reverted 9..10
        let notif = ExExNotification::ChainReverted { old: old_chain };

        exex.handle_notification(notif, &collector, &sync_target_tx)
            .expect("handle chain reverted");
        let latest = proofs.get_latest_block_number().expect("get latest block").expect("ok").0;
        assert_eq!(latest, 5);
    }

    #[tokio::test]
    async fn ensure_initialized_errors_on_storage_not_initialized() {
        // MDBX proofs storage
        let dir = tempdir_path();
        let store = Arc::new(MdbxProofsStorage::new(dir.as_path()).expect("env"));
        let proofs: OpProofsStorage<Arc<MdbxProofsStorage>> = store.clone().into();

        let (ctx, _handle) =
            reth_exex_test_utils::test_exex_context().await.expect("exex test context");

        let exex = build_test_exex(ctx, proofs.clone());
        let _ = exex.ensure_initialized().expect_err("should return error");
    }

    #[tokio::test]
    async fn ensure_initialized_errors_when_prune_exceeds_threshold() {
        // MDBX proofs storage
        let dir = tempdir_path();
        let store = Arc::new(MdbxProofsStorage::new(dir.as_path()).expect("env"));
        let proofs: OpProofsStorage<Arc<MdbxProofsStorage>> = store.clone().into();

        init_storage(proofs.clone());

        for i in 1..1100 {
            proofs
                .store_trie_updates(
                    BlockWithParent::new(
                        hash_for_num(i - 1),
                        BlockNumHash::new(i, hash_for_num(i)),
                    ),
                    BlockStateDiff::default(),
                )
                .expect("store trie update");
        }

        let (ctx, _handle) =
            reth_exex_test_utils::test_exex_context().await.expect("exex test context");

        let exex = build_test_exex(ctx, proofs.clone());
        let _ = exex.ensure_initialized().expect_err("should return error");
    }

    #[tokio::test]
    async fn ensure_initialized_succeeds() {
        // MDBX proofs storage
        let dir = tempdir_path();
        let store = Arc::new(MdbxProofsStorage::new(dir.as_path()).expect("env"));
        let proofs: OpProofsStorage<Arc<MdbxProofsStorage>> = store.clone().into();

        init_storage(proofs.clone());

        let (ctx, _handle) =
            reth_exex_test_utils::test_exex_context().await.expect("exex test context");

        let exex = build_test_exex(ctx, proofs.clone());
        exex.ensure_initialized().expect("should not return error");
    }

    #[tokio::test]
    async fn handle_notification_errors_on_empty_storage() {
        // MDBX proofs storage
        let dir = tempdir_path();
        let store = Arc::new(MdbxProofsStorage::new(dir.as_path()).expect("env"));
        let proofs: OpProofsStorage<Arc<MdbxProofsStorage>> = store.clone().into();

        let (ctx, _handle) =
            reth_exex_test_utils::test_exex_context().await.expect("exex test context");

        let collector = LiveTrieCollector::new(
            ctx.components.components.evm_config.clone(),
            ctx.components.provider.clone(),
            &proofs,
        );

        let exex = build_test_exex(ctx, proofs.clone());

        // Any notification will do
        let new_chain = Arc::new(mk_chain_with_updates(1, 5, None));
        let notif = ExExNotification::ChainCommitted { new: new_chain };

        let (sync_target_tx, _) = tokio::sync::watch::channel(0u64);
        let err = exex.handle_notification(notif, &collector, &sync_target_tx).unwrap_err();
        assert_eq!(err.to_string(), "No blocks stored in proofs storage");
    }

    #[tokio::test]
    async fn handle_notification_schedules_async_on_gap() {
        // MDBX proofs storage
        let dir = tempdir_path();
        let store = Arc::new(MdbxProofsStorage::new(dir.as_path()).expect("env"));
        let proofs: OpProofsStorage<Arc<MdbxProofsStorage>> = store.clone().into();

        init_storage(proofs.clone());

        let (ctx, _handle) =
            reth_exex_test_utils::test_exex_context().await.expect("exex test context");

        let collector = LiveTrieCollector::new(
            ctx.components.components.evm_config.clone(),
            ctx.components.provider.clone(),
            &proofs,
        );
        let exex = build_test_exex(ctx, proofs.clone());

        // Notification: chain committed 5..10 (Blocks 1,2,3,4 are missing from storage)
        let new_chain = Arc::new(mk_chain_with_updates(5, 10, None));
        let notif = ExExNotification::ChainCommitted { new: new_chain };

        let (sync_target_tx, mut sync_target_rx) = tokio::sync::watch::channel(0u64);

        // Process notification
        exex.handle_notification(notif, &collector, &sync_target_tx)
            .expect("handle chain commit should return ok immediately");

        // Verify async signal was sent
        // The target in the channel should now be 10 (the tip of the new chain)
        assert_eq!(
            *sync_target_rx.borrow_and_update(),
            10,
            "Should have scheduled sync to block 10"
        );

        // Verify Main Thread did NOT process it
        // Because we didn't spawn the actual worker thread in this test, storage should still be at
        // 0. This proves the 'handle_notification' returned instantly without doing the
        // heavy lifting.
        let latest = proofs.get_latest_block_number().expect("get").expect("ok").0;
        assert_eq!(latest, 0, "Main thread should not have processed the blocks synchronously");
    }
}
