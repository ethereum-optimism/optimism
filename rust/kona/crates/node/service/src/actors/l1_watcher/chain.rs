//! Per-chain state served by the [`L1WatcherActor`](super::L1WatcherActor).

use super::{L1WatcherActorError, L1WatcherDerivationClient};
use alloy_primitives::Address;
use alloy_provider::Provider;
use kona_genesis::{RollupConfig, SystemConfigLog, SystemConfigUpdate, UnsafeBlockSignerUpdate};
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
    /// The [`RollupConfig`] of this chain. Used for the chain's system config address filter and
    /// to tell if ecotone is active, which decides how its system config logs are read.
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
    ///
    /// Recorded on this chain's log lines and errors, so that with N chains an operator can tell
    /// which chain a message came from.
    pub(super) fn chain_id(&self) -> u64 {
        self.rollup_config.l2_chain_id.id()
    }
}

impl<L1WatcherDerivationClient_> L1WatcherChain<L1WatcherDerivationClient_>
where
    L1WatcherDerivationClient_: L1WatcherDerivationClient,
{
    /// Sends a new L1 head to this chain's derivation actor.
    pub(super) async fn send_new_l1_head(
        &self,
        block: BlockInfo,
    ) -> Result<(), L1WatcherActorError<BlockInfo>> {
        self.derivation_client.send_new_l1_head(block).await.map_err(|e| {
            warn!(target: "l1_watcher", chain_id = self.chain_id(), "Error sending l1 head update to derivation actor: {e}");
            L1WatcherActorError::DerivationClientError { chain_id: self.chain_id(), source: e }
        })
    }

    /// Sends a new finalized L1 block to this chain's derivation actor.
    pub(super) async fn send_finalized_l1_block(
        &self,
        block: BlockInfo,
    ) -> Result<(), L1WatcherActorError<BlockInfo>> {
        self.derivation_client.send_finalized_l1_block(block).await.map_err(|e| {
            warn!(target: "l1_watcher", chain_id = self.chain_id(), "Error sending finalized l1 block update to derivation actor: {e}");
            L1WatcherActorError::DerivationClientError { chain_id: self.chain_id(), source: e }
        })
    }

    /// Reads this chain's system config logs from the given L1 head block and forwards any unsafe
    /// block signer update they carry to this chain's network actor.
    ///
    /// Each chain is queried separately rather than through one filter over all system config
    /// addresses, so that a single-chain watcher issues exactly the request it always has.
    pub(super) async fn process_system_config_logs(
        &self,
        l1_provider: &impl Provider,
        head_block_info: BlockInfo,
    ) -> Result<(), L1WatcherActorError<BlockInfo>> {
        // For each log, attempt to construct a [`SystemConfigLog`].
        // Build the [`SystemConfigUpdate`] from the log.
        // If the update is an Unsafe block signer update, send the address
        // to the block signer sender.
        let filter_address = self.rollup_config.l1_system_config_address;
        let logs = l1_provider
            .get_logs(
                &alloy_rpc_types_eth::Filter::new()
                    .address(filter_address)
                    .select(head_block_info.hash),
            )
            .await?;
        let ecotone_active = self.rollup_config.is_ecotone_active(head_block_info.timestamp);
        for log in logs {
            let sys_cfg_log = SystemConfigLog::new(log.into(), ecotone_active);
            if let Ok(SystemConfigUpdate::UnsafeBlockSigner(UnsafeBlockSignerUpdate {
                unsafe_block_signer,
            })) = sys_cfg_log.build()
            {
                info!(
                    target: "l1_watcher",
                    chain_id = self.chain_id(),
                    "Unsafe block signer update: {unsafe_block_signer}"
                );
                if let Err(e) = self.block_signer_sender.send(unsafe_block_signer).await {
                    error!(
                        target: "l1_watcher",
                        chain_id = self.chain_id(),
                        "Error sending unsafe block signer update: {e}"
                    );
                }
            }
        }

        Ok(())
    }
}
