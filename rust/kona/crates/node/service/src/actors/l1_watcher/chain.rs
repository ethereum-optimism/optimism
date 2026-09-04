//! Per-chain state served by the [`L1WatcherActor`](super::L1WatcherActor).

use super::{L1WatcherActorError, L1WatcherDerivationClient};
use crate::DerivationClientError;
use kona_genesis::RollupConfig;
use kona_protocol::BlockInfo;
use kona_rpc::L1WatcherQueries;
use std::sync::Arc;
use tokio::sync::mpsc;

/// A single L2 chain served by the [`L1WatcherActor`](super::L1WatcherActor).
///
/// The watcher holds one of these per chain and fans the shared L1 updates out to all of them. A
/// standalone kona-node builds exactly one.
#[derive(Debug)]
pub struct L1WatcherChain<L1WatcherDerivationClient_> {
    /// The [`RollupConfig`] of this chain, answered to its config queries.
    pub(super) rollup_config: Arc<RollupConfig>,
    /// Client used to interact with this chain's [`crate::DerivationActor`].
    pub(super) derivation_client: L1WatcherDerivationClient_,
    /// The inbound queries for this chain.
    pub(super) inbound_queries: mpsc::Receiver<L1WatcherQueries>,
}

impl<L1WatcherDerivationClient_> L1WatcherChain<L1WatcherDerivationClient_> {
    /// Instantiate a new [`L1WatcherChain`].
    pub const fn new(
        rollup_config: Arc<RollupConfig>,
        derivation_client: L1WatcherDerivationClient_,
        inbound_queries: mpsc::Receiver<L1WatcherQueries>,
    ) -> Self {
        Self { rollup_config, derivation_client, inbound_queries }
    }

    /// The id of the L2 chain this instance serves.
    pub(super) fn chain_id(&self) -> u64 {
        self.rollup_config.l2_chain_id.id()
    }
}

impl<L1WatcherDerivationClient_> L1WatcherChain<L1WatcherDerivationClient_>
where
    L1WatcherDerivationClient_: L1WatcherDerivationClient,
{
    /// Logs a failed send to this chain's derivation actor and wraps it in the actor error.
    fn client_err(&self, what: &str, e: DerivationClientError) -> L1WatcherActorError<BlockInfo> {
        warn!(target: "l1_watcher", chain_id = self.chain_id(), "Error sending {what} to derivation actor: {e}");
        L1WatcherActorError::DerivationClientError { chain_id: self.chain_id(), source: e }
    }

    /// Sends a new L1 head to this chain's derivation actor.
    pub(super) async fn send_new_l1_head(
        &self,
        block: BlockInfo,
    ) -> Result<(), L1WatcherActorError<BlockInfo>> {
        self.derivation_client
            .send_new_l1_head(block)
            .await
            .map_err(|e| self.client_err("l1 head update", e))
    }
}
