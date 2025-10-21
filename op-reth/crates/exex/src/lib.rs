//! ExEx unique for OP-Reth. See also [`reth_exex`] for more op-reth execution extensions.

#![doc(
    html_logo_url = "https://raw.githubusercontent.com/paradigmxyz/reth/main/assets/reth-docs.png",
    html_favicon_url = "https://avatars0.githubusercontent.com/u/97369466?s=256",
    issue_tracker_base_url = "https://github.com/paradigmxyz/reth/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]

use derive_more::Constructor;
use futures_util::TryStreamExt;
use reth_chainspec::ChainInfo;
use reth_exex::{ExExContext, ExExEvent};
use reth_node_api::{FullNodeComponents, NodePrimitives};
use reth_node_types::NodeTypes;
use reth_optimism_trie::{BackfillJob, OpProofsStore};
use reth_provider::{BlockNumReader, DBProvider, DatabaseProviderFactory};

/// OP Proofs ExEx - processes blocks and tracks state changes within fault proof window.
///
/// Saves and serves trie nodes to make proofs faster. This handles the process of
/// saving the current state, new blocks as they're added, and serving proof RPCs
/// based on the saved data.
#[derive(Debug, Constructor)]
pub struct OpProofsExEx<Node, S>
where
    Node: FullNodeComponents,
    S: OpProofsStore + Clone,
{
    /// The ExEx context containing the node related utilities e.g. provider, notifications,
    /// events.
    ctx: ExExContext<Node>,
    /// The type of storage DB.
    storage: S,
    /// The window to span blocks for proofs history. Value is the number of blocks, received as
    /// cli arg.
    #[expect(dead_code)]
    proofs_history_window: u64,
}

impl<Node, S, Primitives> OpProofsExEx<Node, S>
where
    Node: FullNodeComponents<Types: NodeTypes<Primitives = Primitives>>,
    Primitives: NodePrimitives,
    S: OpProofsStore + Clone,
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
