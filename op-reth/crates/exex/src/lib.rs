//! ExEx unique for OP-Reth. See also [`reth_exex`] for more op-reth execution extensions.

use derive_more::Constructor;
use futures_util::TryStreamExt;
use reth_chainspec::ChainInfo;
use reth_exex::{ExExContext, ExExEvent};
use reth_node_api::{FullNodeComponents, NodePrimitives};
use reth_node_types::NodeTypes;
use reth_optimism_trie::{BackfillJob, InMemoryProofsStorage};
use reth_provider::{BlockNumReader, DBProvider, DatabaseProviderFactory};
use std::sync::Arc;

/// OP Proofs ExEx - processes blocks and tracks state changes within fault proof window.
///
/// Saves and serves trie nodes to make proofs faster. This handles the process of
/// saving the current state, new blocks as they're added, and serving proof RPCs
/// based on the saved data.
#[derive(Debug, Constructor)]
pub struct OpProofsExEx<Node>
where
    Node: FullNodeComponents,
{
    ctx: ExExContext<Node>,
    storage: Arc<InMemoryProofsStorage>,
}

impl<Node, Primitives> OpProofsExEx<Node>
where
    Node: FullNodeComponents<Types: NodeTypes<Primitives = Primitives>>,
    Primitives: NodePrimitives,
{
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
