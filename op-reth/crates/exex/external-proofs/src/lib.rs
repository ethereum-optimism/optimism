//! External Proofs ExEx - processes blocks and tracks state changes

use futures_util::TryStreamExt;
use reth_node_api::{FullNodeComponents, NodePrimitives};
use reth_node_types::NodeTypes;
use reth_provider::StateReader;

use reth_exex::{ExExContext, ExExEvent};

/// Saves and serves trie nodes to make proofs faster. This handles the process of
/// saving the current state, new blocks as they're added, and serving proof RPCs
/// based on the saved data.
#[derive(Debug)]
pub struct ExternalProofExEx<Node>
where
    Node: FullNodeComponents,
    Node::Provider: StateReader,
{
    ctx: ExExContext<Node>,
}

impl<Node, Primitives> ExternalProofExEx<Node>
where
    Node: FullNodeComponents<Types: NodeTypes<Primitives = Primitives>>,
    Primitives: NodePrimitives,
{
    /// Create a new `ExternalProofExEx` instance
    pub const fn new(ctx: ExExContext<Node>) -> Self {
        Self { ctx }
    }

    /// Main execution loop for the ExEx
    pub async fn run(mut self) -> eyre::Result<()> {
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
