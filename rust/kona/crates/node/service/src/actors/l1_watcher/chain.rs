//! Per-chain state served by the [`L1WatcherActor`](super::L1WatcherActor).

use super::{L1WatcherActorError, L1WatcherDerivationClient};
use crate::DerivationClientError;
use alloy_primitives::Address;
use alloy_provider::Provider;
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
    /// The configuration and `SystemConfig` address of this chain.
    pub(super) rollup_config: Arc<RollupConfig>,
    /// Client used to interact with this chain's [`crate::DerivationActor`].
    pub(super) derivation_client: L1WatcherDerivationClient_,
    /// This chain's block signer sender.
    pub(super) block_signer_sender: mpsc::Sender<Address>,
    /// The inbound queries for this chain.
    pub(super) inbound_queries: mpsc::Receiver<L1WatcherQueries>,
}

impl<L1WatcherDerivationClient_> L1WatcherChain<L1WatcherDerivationClient_> {
    /// Instantiate a new [`L1WatcherChain`].
    pub const fn new(
        rollup_config: Arc<RollupConfig>,
        derivation_client: L1WatcherDerivationClient_,
        block_signer_sender: mpsc::Sender<Address>,
        inbound_queries: mpsc::Receiver<L1WatcherQueries>,
    ) -> Self {
        Self { rollup_config, derivation_client, block_signer_sender, inbound_queries }
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

    /// Sends a new finalized L1 block to this chain's derivation actor.
    pub(super) async fn send_finalized_l1_block(
        &self,
        block: BlockInfo,
    ) -> Result<(), L1WatcherActorError<BlockInfo>> {
        self.derivation_client
            .send_finalized_l1_block(block)
            .await
            .map_err(|e| self.client_err("finalized l1 block update", e))
    }

    /// Reconciles the signer against current state, including rotations missed between heads.
    /// Failed reads retain the last known signer and are retried by the next head or timer.
    pub(super) async fn reconcile_signer(&self, l1_provider: &impl Provider, head: BlockInfo) {
        let read = kona_providers_alloy::unsafe_block_signer(
            l1_provider,
            self.rollup_config.l1_system_config_address,
            head.hash,
        );
        match tokio::time::timeout(std::time::Duration::from_secs(10), read).await {
            Ok(Ok(signer)) => {
                if let Err(err) = self.block_signer_sender.send(signer).await {
                    warn!(target: "l1_watcher", chain_id = self.chain_id(), %err, "Signer receiver closed");
                }
            }
            result => {
                warn!(target: "l1_watcher", chain_id = self.chain_id(), ?result, "Failed to refresh unsafe block signer; retaining previous value");
            }
        }
    }
}
