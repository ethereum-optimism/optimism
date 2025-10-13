//! External Proofs ExEx - processes blocks and tracks state changes

use crate::{backfill::BackfillJob, in_memory::InMemoryExternalStorage};
use futures_util::TryStreamExt;
use reth_chainspec::ChainInfo;
use reth_exex::{ExExContext, ExExEvent};
use reth_node_api::{FullNodeComponents, NodePrimitives};
use reth_node_types::NodeTypes;
use reth_provider::{BlockNumReader, DBProvider, DatabaseProviderFactory};
use std::sync::Arc;

pub mod backfill;
pub mod in_memory;
pub mod storage;

#[cfg(test)]
mod storage_tests;

/// Saves and serves trie nodes to make proofs faster. This handles the process of
/// saving the current state, new blocks as they're added, and serving proof RPCs
/// based on the saved data.
#[derive(Debug)]
pub struct ExternalProofExEx<Node>
where
    Node: FullNodeComponents,
{
    ctx: ExExContext<Node>,
    storage: Arc<InMemoryExternalStorage>,
}

impl<Node, Primitives> ExternalProofExEx<Node>
where
    Node: FullNodeComponents<Types: NodeTypes<Primitives = Primitives>>,
    Primitives: NodePrimitives,
{
    /// Create a new `ExternalProofExEx` instance
    pub fn new(ctx: ExExContext<Node>) -> Self {
        Self { ctx, storage: Arc::new(InMemoryExternalStorage::new()) }
    }

    /// Main execution loop for the ExEx
    pub async fn run(mut self) -> eyre::Result<()> {
        let db_provider =
            self.ctx.provider().database_provider_ro()?.disable_long_read_transaction_safety();
        let db_tx = db_provider.into_tx();
        let ChainInfo { best_number, best_hash } = self.ctx.provider().chain_info()?;
        BackfillJob::new(self.storage.clone(), &db_tx).run(best_number, best_hash).await?;

        while let Some(notification) = self.ctx.notifications.try_next().await? {
            // match &notification {
            //     _ => {}
            // };

            if let Some(committed_chain) = notification.committed_chain() {
                self.ctx
                    .events
                    .send(ExExEvent::FinishedHeight(committed_chain.tip().num_hash()))?;
            }
        }

        Ok(())
    }
}
