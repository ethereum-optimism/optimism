//! Per-chain state served by the [`L1WatcherActor`](super::L1WatcherActor).

use super::{L1WatcherActorError, L1WatcherDerivationClient};
use alloy_primitives::Address;
use alloy_provider::Provider;
use kona_genesis::RollupConfig;
use kona_protocol::BlockInfo;
use kona_rpc::L1State;
use std::sync::Arc;
use tokio::sync::watch;

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
    /// The source of truth for this chain's unsafe block signer, read from `SystemConfig`. Gossip
    /// validation and block signing subscribe to it.
    pub(super) block_signer_sender: watch::Sender<Address>,
}

impl<L1WatcherDerivationClient_> L1WatcherChain<L1WatcherDerivationClient_> {
    /// Instantiate a new [`L1WatcherChain`].
    pub const fn new(
        rollup_config: Arc<RollupConfig>,
        derivation_client: L1WatcherDerivationClient_,
        block_signer_sender: watch::Sender<Address>,
    ) -> Self {
        Self { rollup_config, derivation_client, block_signer_sender }
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
    /// Runs independent forwarding and signer-refresh loops for this chain. A closed
    /// channel tears down only this worker; slow RPCs cannot hold up shared L1 observation.
    pub(super) async fn run(
        self,
        provider: impl Provider,
        state: watch::Receiver<L1State>,
    ) -> Result<(), L1WatcherActorError<BlockInfo>> {
        let chain_id = self.chain_id();
        let Self { rollup_config, derivation_client, block_signer_sender } = self;
        let mut updates = state.clone();
        let forward = async {
            let mut previous = L1State::default();
            loop {
                let current = *updates.borrow_and_update();
                if current.head_l1 != previous.head_l1 &&
                    let Some(block) = current.head_l1
                {
                    derivation_client.send_new_l1_head(block).await.map_err(|source| {
                        L1WatcherActorError::DerivationClientError { chain_id, source }
                    })?;
                }
                if current.finalized_l1 != previous.finalized_l1 &&
                    let Some(block) = current.finalized_l1
                {
                    derivation_client.send_finalized_l1_block(block).await.map_err(|source| {
                        L1WatcherActorError::DerivationClientError { chain_id, source }
                    })?;
                }
                previous = current;
                updates.changed().await.map_err(|_| L1WatcherActorError::StreamEnded)?;
            }
            #[allow(unreachable_code)]
            Ok::<(), L1WatcherActorError<BlockInfo>>(())
        };
        let signer = Self::refresh_signer(&provider, &rollup_config, &block_signer_sender, state);
        tokio::try_join!(forward, signer)?;
        Ok(())
    }

    async fn refresh_signer(
        provider: &impl Provider,
        config: &RollupConfig,
        signer_tx: &watch::Sender<Address>,
        mut state: watch::Receiver<L1State>,
    ) -> Result<(), L1WatcherActorError<BlockInfo>> {
        use tokio::time::{self, Duration};
        let mut retry = time::interval(Duration::from_secs(10));
        retry.set_missed_tick_behavior(time::MissedTickBehavior::Skip);
        let mut last_head = None;
        loop {
            let state_changed = tokio::select! {
                result = state.changed() => {
                    result.map_err(|_| L1WatcherActorError::StreamEnded)?;
                    true
                }
                _ = retry.tick() => false,
                _ = signer_tx.closed() => return Err(L1WatcherActorError::StreamEnded),
            };
            let Some(target) = state.borrow_and_update().head_l1 else { continue };
            if state_changed && last_head == Some(target) {
                continue;
            }
            last_head = Some(target);
            let read = kona_providers_alloy::unsafe_block_signer(
                provider,
                config.l1_system_config_address,
                target.hash,
            );
            match time::timeout(Duration::from_secs(10), read).await {
                // Head changes while a read is outstanding supersede that read, including
                // same-height reorgs. Never install a snapshot for an obsolete branch.
                Ok(Ok(signer)) if state.borrow().head_l1 == Some(target) => {
                    // Subscribers are only notified when the signer actually changes.
                    signer_tx
                        .send_if_modified(|current| std::mem::replace(current, signer) != signer);
                }
                Ok(Ok(_)) => {}
                result => {
                    warn!(target: "l1_watcher", chain_id = config.l2_chain_id.id(), ?result, "Failed to refresh unsafe block signer; retaining previous value")
                }
            }
        }
    }
}
