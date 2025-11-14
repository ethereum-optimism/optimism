//! ExEx unique for OP-Reth. See also [`reth_exex`] for more op-reth execution extensions.

#![doc(
    html_logo_url = "https://raw.githubusercontent.com/paradigmxyz/reth/main/assets/reth-docs.png",
    html_favicon_url = "https://avatars0.githubusercontent.com/u/97369466?s=256",
    issue_tracker_base_url = "https://github.com/paradigmxyz/reth/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]

use alloy_consensus::BlockHeader;
use derive_more::Constructor;
use futures_util::TryStreamExt;
use reth_chainspec::ChainInfo;
use reth_exex::{ExExContext, ExExEvent, ExExNotification};
use reth_node_api::{FullNodeComponents, NodePrimitives};
use reth_node_types::NodeTypes;
use reth_optimism_trie::{live::LiveTrieCollector, BackfillJob, OpProofsStorage, OpProofsStore};
use reth_primitives_traits::{BlockTy, RecoveredBlock};
use reth_provider::{
    BlockNumReader, BlockReader, DBProvider, DatabaseProviderFactory, TransactionVariant,
};
use tracing::{debug, error, info};

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
/// use std::sync::Arc;
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
/// // Can also use install_exex_if along with a boolean flag
/// // Set this based on your configuration or CLI args
/// let _builder = NodeBuilder::new(config)
///     .with_database(db)
///     .with_types_and_provider::<OpNode, BlockchainProvider<NodeTypesWithDBAdapter<OpNode, _>>>()
///     .with_components(op_node.components())
///     .install_exex("proofs-history", move |exex_context| async move {
///         Ok(OpProofsExEx::new(exex_context, storage_exec, proofs_history_window).run().boxed())
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
    #[expect(dead_code)]
    proofs_history_window: u64,
}

impl<Node, Storage, Primitives> OpProofsExEx<Node, Storage>
where
    Node: FullNodeComponents<Types: NodeTypes<Primitives = Primitives>>,
    Primitives: NodePrimitives,
    Storage: OpProofsStore + Clone + 'static,
{
    /// Main execution loop for the ExEx
    pub async fn run(mut self) -> eyre::Result<()> {
        {
            let db_provider =
                self.ctx.provider().database_provider_ro()?.disable_long_read_transaction_safety();
            let db_tx = db_provider.into_tx();
            let ChainInfo { best_number, best_hash } = self.ctx.provider().chain_info()?;
            BackfillJob::new(self.storage.clone(), &db_tx).run(best_number, best_hash).await?;
        }

        let collector = LiveTrieCollector::new(
            self.ctx.evm_config().clone(),
            self.ctx.provider().clone(),
            &self.storage,
        );

        while let Some(notification) = self.ctx.notifications.try_next().await? {
            match &notification {
                ExExNotification::ChainCommitted { new } => {
                    debug!(
                        target: "optimism::exex",
                        block_number = new.tip().number(),
                        block_hash = ?new.tip().hash(),
                        "ChainCommitted notification received",
                    );

                    // Get latest stored number (ignore stored hash for now)
                    let latest_stored_block_number =
                        match self.storage.get_latest_block_number().await? {
                            Some((n, _)) => n,
                            None => {
                                return Err(eyre::eyre!("No blocks stored in proofs storage"));
                            }
                        };

                    // If tip is not newer than what we have, nothing to do.
                    if new.tip().number() <= latest_stored_block_number {
                        debug!(
                            target: "optimism::exex",
                            block_number = new.tip().number(),
                            latest_stored = latest_stored_block_number,
                            "Tip number is less than or equal to latest stored, skipping"
                        );
                        continue;
                    }

                    // Start from the next block after the latest stored one.
                    let start = latest_stored_block_number.saturating_add(1);
                    debug!(
                        target: "optimism::exex",
                        start,
                        end = new.tip().number(),
                        "Applying updates for blocks in committed chain"
                    );
                    for block_number in start..=new.tip().number() {
                        match self
                            .ctx
                            .provider()
                            .recovered_block(block_number.into(), TransactionVariant::NoHash)?
                        {
                            Some(block) => {
                                collector.execute_and_store_block_updates(&block).await?;
                            }
                            None => {
                                error!(
                                    block_number,
                                    "Missing block in committed chain, stopping incremental application",
                                );
                                return Err(eyre::eyre!(
                                    "Missing block {} in committed chain",
                                    block_number
                                ));
                            }
                        }
                    }
                }
                ExExNotification::ChainReorged { old, new } => {
                    info!(
                        old_block_number = old.tip().number(),
                        old_block_hash = ?old.tip().hash(),
                        new_block_number = new.tip().number(),
                        new_block_hash = ?new.tip().hash(),
                        "ChainReorged notification received",
                    );

                    // find the common ancestor
                    let mut new_blocks: Vec<&RecoveredBlock<BlockTy<Primitives>>> =
                        Vec::with_capacity(new.len());
                    for block_number in new.blocks().keys().rev() {
                        let old_block = old.blocks().get(block_number);
                        let new_block = new.blocks().get(block_number);
                        match (new_block, old_block) {
                            (Some(new_block), Some(old_block)) => {
                                if new_block.hash() == old_block.hash() {
                                    break;
                                }

                                new_blocks.push(new_block);
                                if new_block.parent_hash() == old_block.parent_hash() {
                                    break;
                                }
                            }
                            (Some(new_block), None) => {
                                // Block only exists in new chain, collect it
                                new_blocks.push(new_block);
                            }
                            _ => {
                                error!(
                                    block_number,
                                    "Missing block in new chain during reorg detection",
                                );
                                return Err(eyre::eyre!(
                                    "Missing block {} in new chain during reorg detection",
                                    block_number
                                ));
                            }
                        }
                    }

                    // reverse to get the new blocks in the correct order
                    new_blocks.reverse();
                    collector.unwind_and_store_block_updates(new_blocks).await?;
                }
                ExExNotification::ChainReverted { old } => {
                    debug!(
                        target: "optimism::exex",
                        old_block_number = old.tip().number(),
                        old_block_hash = ?old.tip().hash(),
                        "ChainReverted notification received",
                    );
                    unimplemented!("Chain revert handling not yet implemented");
                }
            };

            if let Some(committed_chain) = notification.committed_chain() {
                self.ctx
                    .events
                    .send(ExExEvent::FinishedHeight(committed_chain.tip().num_hash()))?;
            }
        }

        Ok(())
    }
}
