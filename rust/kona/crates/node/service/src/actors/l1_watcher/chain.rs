//! Per-chain state served by the [`L1WatcherActor`](super::L1WatcherActor).

use super::{L1WatcherActorError, L1WatcherDerivationClient};
use crate::DerivationClientError;
use alloy_primitives::{Address, B256, b256};
use alloy_provider::Provider;
use kona_genesis::{RollupConfig, SystemConfigLog, SystemConfigUpdate, UnsafeBlockSignerUpdate};
use kona_protocol::BlockInfo;
use kona_rpc::L1WatcherQueries;
use std::sync::Arc;
use tokio::sync::mpsc;

/// The storage slot of the unsafe block signer address in the L1 `SystemConfig` contract:
/// `keccak256("systemconfig.unsafeblocksigner")`.
///
/// Mirrors `UNSAFE_BLOCK_SIGNER_SLOT` in `SystemConfig.sol` and op-node's
/// `UnsafeBlockSignerAddressSystemConfigStorageSlot` (`op-node/node/runcfg/runtime_config.go`).
const UNSAFE_BLOCK_SIGNER_SLOT: B256 =
    b256!("0x65a7ed542fb37fe237fdfbdd70b31598523fe5b32879e307bae27a0bd9581c08");

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

    /// Reads this chain's system config logs from the given L1 head block and forwards any unsafe
    /// block signer update they carry to this chain's network actor.
    ///
    /// Never fails: a failed log fetch is logged and this head's logs are skipped for this chain,
    /// so a transient L1 RPC error neither stops the watcher (and with it the node) nor the other
    /// chains' updates. This mirrors op-node, whose runtime-config reloader treats a failed
    /// unsafe-block-signer read from L1 as a warning and waits for the next reload interval —
    /// missing an update is explicitly not critical there. A signer update skipped here (or in a
    /// head the poll-based head stream never yielded) is healed by the watcher's periodic
    /// [`reconcile_unsafe_block_signer`](Self::reconcile_unsafe_block_signer) pass.
    pub(super) async fn process_system_config_logs(
        &self,
        l1_provider: &impl Provider,
        head_block_info: BlockInfo,
    ) {
        let filter_address = self.rollup_config.l1_system_config_address;
        let logs = match l1_provider
            .get_logs(
                &alloy_rpc_types_eth::Filter::new()
                    .address(filter_address)
                    .select(head_block_info.hash),
            )
            .await
        {
            Ok(logs) => logs,
            Err(e) => {
                warn!(
                    target: "l1_watcher",
                    chain_id = self.chain_id(),
                    "Error fetching system config logs, skipping this head's logs: {e}"
                );
                return;
            }
        };
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
    }

    /// Reads this chain's current unsafe block signer straight from the L1 `SystemConfig`
    /// contract's storage and forwards it to this chain's network actor.
    ///
    /// This is the periodic reconciliation behind the per-head
    /// [`process_system_config_logs`](Self::process_system_config_logs) fast path: the slot read
    /// is idempotent latest-state, so it converges on the current signer no matter which update
    /// events were missed — a failed log fetch, a head the poll-based head stream skipped, or a
    /// rotation that predates this process. It mirrors op-node's periodic runtime-config reload,
    /// which is op-node's only signer refresh mechanism. The read is a plain `eth_getStorageAt`,
    /// matching the trust the watcher already places in this provider for headers and logs.
    ///
    /// Never fails: like a failed log fetch, a failed slot read is logged and skipped — the next
    /// reconciliation pass heals it.
    pub(super) async fn reconcile_unsafe_block_signer(
        &self,
        l1_provider: &impl Provider,
        head_block_info: BlockInfo,
    ) {
        // Pinned to the latest head the watcher has processed rather than the `latest` tag, so
        // the read cannot race ahead of the watcher's own view onto a block (or a reorged
        // sibling) it has not seen.
        let value = match l1_provider
            .get_storage_at(
                self.rollup_config.l1_system_config_address,
                UNSAFE_BLOCK_SIGNER_SLOT.into(),
            )
            .hash(head_block_info.hash)
            .await
        {
            Ok(value) => value,
            Err(e) => {
                warn!(
                    target: "l1_watcher",
                    chain_id = self.chain_id(),
                    "Error reading the unsafe block signer slot, skipping this reconciliation: {e}"
                );
                return;
            }
        };

        // The slot holds an `address`: the value's last 20 bytes.
        let unsafe_block_signer = Address::from_word(B256::from(value));
        debug!(
            target: "l1_watcher",
            chain_id = self.chain_id(),
            "Reconciled unsafe block signer: {unsafe_block_signer}"
        );
        if let Err(e) = self.block_signer_sender.send(unsafe_block_signer).await {
            error!(
                target: "l1_watcher",
                chain_id = self.chain_id(),
                "Error sending reconciled unsafe block signer: {e}"
            );
        }
    }
}
