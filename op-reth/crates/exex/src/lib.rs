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
use reth_provider::{BlockNumReader, DBProvider, DatabaseProviderFactory};
use tracing::{debug, error};

/// OP Proofs ExEx - processes blocks and tracks state changes within fault proof window.
///
/// Saves and serves trie nodes to make proofs faster. This handles the process of
/// saving the current state, new blocks as they're added, and serving proof RPCs
/// based on the saved data.
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
        let db_provider =
            self.ctx.provider().database_provider_ro()?.disable_long_read_transaction_safety();
        let db_tx = db_provider.into_tx();
        let ChainInfo { best_number, best_hash } = self.ctx.provider().chain_info()?;
        BackfillJob::new(self.storage.clone(), &db_tx).run(best_number, best_hash).await?;

        let collector = LiveTrieCollector::new(
            self.ctx.evm_config().clone(),
            self.ctx.provider().clone(),
            &self.storage,
        );

        while let Some(notification) = self.ctx.notifications.try_next().await? {
            match &notification {
                ExExNotification::ChainCommitted { new } => {
                    debug!(
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
                            block_number = new.tip().number(),
                            latest_stored = latest_stored_block_number,
                            "Tip number is less than or equal to latest stored, skipping"
                        );
                        continue;
                    }

                    // Start from the next block after the latest stored one.
                    let start = latest_stored_block_number.saturating_add(1);
                    debug!(
                        start,
                        end = new.tip().number(),
                        "Applying updates for blocks in committed chain"
                    );
                    for block_number in start..=new.tip().number() {
                        match new.blocks().get(&block_number) {
                            Some(block) => {
                                collector.execute_and_store_block_updates(block).await?;
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
                    debug!(
                        old_block_number = old.tip().number(),
                        old_block_hash = ?old.tip().hash(),
                        new_block_number = new.tip().number(),
                        new_block_hash = ?new.tip().hash(),
                        "ChainReorged notification received",
                    );
                    unimplemented!("Chain reorg handling not yet implemented in OpProofsExEx");
                }
                ExExNotification::ChainReverted { old } => {
                    debug!(
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
