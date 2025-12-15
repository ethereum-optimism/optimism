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
use derive_more::Constructor;
use futures_util::TryStreamExt;
use reth_execution_types::Chain;
use reth_exex::{ExExContext, ExExEvent, ExExNotification};
use reth_node_api::{FullNodeComponents, NodePrimitives};
use reth_node_types::NodeTypes;
use reth_optimism_trie::{
    live::LiveTrieCollector, OpProofStoragePrunerTask, OpProofsStorage, OpProofsStore,
};
use reth_provider::{BlockReader, TransactionVariant};
use reth_trie::{updates::TrieUpdatesSorted, HashedPostStateSorted};
use std::{sync::Arc, time::Duration};
use tracing::{debug, info};

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
/// use reth_optimism_node::{args::RollupArgs, OpNode};
/// use reth_optimism_trie::{db::MdbxProofsStorage, InMemoryProofsStorage, OpProofsStorage};
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
/// // Can also use install_exex_if along with a boolean flag
/// // Set this based on your configuration or CLI args
/// let _builder = NodeBuilder::new(config)
///     .with_database(db)
///     .with_types_and_provider::<OpNode, BlockchainProvider<NodeTypesWithDBAdapter<OpNode, _>>>()
///     .with_components(op_node.components())
///     .install_exex("proofs-history", move |exex_context| async move {
///         Ok(OpProofsExEx::new(
///             exex_context,
///             storage_exec,
///             proofs_history_window,
///             proofs_history_prune_interval,
///         )
///         .run()
///         .boxed())
///     })
///     .on_node_started(|_full_node| Ok(()))
///     .check_launch();
/// ```
#[derive(Debug, Constructor)]
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
}

impl<Node, Storage, Primitives> OpProofsExEx<Node, Storage>
where
    Node: FullNodeComponents<Types: NodeTypes<Primitives = Primitives>>,
    Primitives: NodePrimitives,
    Storage: OpProofsStore + Clone + 'static,
{
    /// Main execution loop for the ExEx
    pub async fn run(mut self) -> eyre::Result<()> {
        self.ensure_initialized().await?;

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
            self.handle_notification(notification, &collector).await?;
        }

        Ok(())
    }

    /// Ensure proofs storage is initialized
    async fn ensure_initialized(&self) -> eyre::Result<()> {
        // Check if proofs storage is initialized
        #[cfg_attr(not(feature = "metrics"), expect(unused_variables))]
        let earliest_block_number = match self.storage.get_earliest_block_number().await? {
            Some((n, _)) => n,
            None => {
                return Err(eyre::eyre!(
                    "Proofs storage not initialized. Please run 'op-reth initialize-op-proofs --proofs-history.storage-path <PATH>' first."
                ));
            }
        };

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

    async fn handle_notification(
        &self,
        notification: ExExNotification<Primitives>,
        collector: &LiveTrieCollector<'_, Node::Evm, Node::Provider, Storage>,
    ) -> eyre::Result<()> {
        let latest_stored = match self.storage.get_latest_block_number().await? {
            Some((n, _)) => n,
            None => {
                return Err(eyre::eyre!("No blocks stored in proofs storage"));
            }
        };

        match &notification {
            ExExNotification::ChainCommitted { new } => {
                self.handle_chain_committed(new.clone(), latest_stored, collector).await?
            }
            ExExNotification::ChainReorged { old, new } => {
                self.handle_chain_reorged(old.clone(), new.clone(), latest_stored, collector)
                    .await?
            }
            ExExNotification::ChainReverted { old } => {
                self.handle_chain_reverted(old.clone(), latest_stored, collector).await?
            }
        }

        if let Some(committed_chain) = notification.committed_chain() {
            self.ctx.events.send(ExExEvent::FinishedHeight(committed_chain.tip().num_hash()))?;
        }

        Ok(())
    }

    async fn handle_chain_committed(
        &self,
        new: Arc<Chain<Primitives>>,
        latest_stored: u64,
        collector: &LiveTrieCollector<'_, Node::Evm, Node::Provider, Storage>,
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

        // Process each block from latest_stored + 1 to tip
        let start = latest_stored.saturating_add(1);
        for block_number in start..=new.tip().number() {
            self.process_block(block_number, &new, collector).await?;
        }

        Ok(())
    }

    /// Process a single block - either from chain or provider
    async fn process_block(
        &self,
        block_number: u64,
        chain: &Chain<Primitives>,
        collector: &LiveTrieCollector<'_, Node::Evm, Node::Provider, Storage>,
    ) -> eyre::Result<()> {
        // Try to get block data from the chain first
        // 1. Fast Path: Try to use pre-computed state from the notification
        if let Some(block) = chain.blocks().get(&block_number) {
            // Check if we have BOTH trie updates and hashed state.
            // If either is missing, we fall back to execution to ensure data integrity.
            if let (Some(trie_updates), Some(hashed_state)) =
                (chain.trie_updates_at(block_number), chain.hashed_state_at(block_number))
            {
                debug!(
                    target: "optimism::exex",
                    block_number,
                    "Using pre-computed state updates from notification"
                );

                collector
                    .store_block_updates(
                        block.block_with_parent(),
                        (**trie_updates).clone(),
                        (**hashed_state).clone(),
                    )
                    .await?;

                return Ok(());
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

        collector.execute_and_store_block_updates(&block).await?;
        Ok(())
    }

    async fn handle_chain_reorged(
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
            let trie_updates = new.trie_updates_at(*block_number).ok_or_else(|| {
                eyre::eyre!("Missing Trie updates for block {} in new chain", block_number)
            })?;
            let hashed_state = new.hashed_state_at(*block_number).ok_or_else(|| {
                eyre::eyre!("Missing Hashed state for block {} in new chain", block_number)
            })?;

            block_updates.push((
                block.block_with_parent(),
                trie_updates.clone(),
                hashed_state.clone(),
            ));
        }

        collector.unwind_and_store_block_updates(block_updates).await?;

        Ok(())
    }

    async fn handle_chain_reverted(
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

        collector.unwind_history(old.first().block_with_parent()).await?;
        Ok(())
    }
}
