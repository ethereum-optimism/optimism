//! Derivation forwarding and signer refresh.

use super::{ActorError, Derivation, State};
use alloy_primitives::Address;
use alloy_provider::Provider;
use kona_genesis::RollupConfig;
use std::sync::Arc;
use tokio::sync::watch;

#[derive(Debug)]
pub(super) struct Worker<L1Provider, Client> {
    l1_provider: L1Provider,
    /// The configuration and `SystemConfig` address of this chain.
    rollup_config: Arc<RollupConfig>,
    /// Client used to interact with this chain's [`crate::DerivationActor`].
    derivation_client: Client,
    /// The source of truth for this chain's unsafe block signer, read from `SystemConfig`. Gossip
    /// validation and block signing subscribe to it.
    block_signer_sender: watch::Sender<Address>,
}

impl<L1Provider, Client> Worker<L1Provider, Client>
where
    L1Provider: Provider,
    Client: Derivation,
{
    pub(super) const fn new(
        l1_provider: L1Provider,
        rollup_config: Arc<RollupConfig>,
        derivation_client: Client,
        block_signer_sender: watch::Sender<Address>,
    ) -> Self {
        Self { l1_provider, rollup_config, derivation_client, block_signer_sender }
    }

    /// Runs forwarding and signer refresh independently of L1 observation.
    pub(super) async fn run(self, state: watch::Receiver<State>) -> Result<(), ActorError> {
        let Self { l1_provider, rollup_config, derivation_client, block_signer_sender } = self;
        let mut updates = state.clone();
        let forward = async {
            let mut previous = State::default();
            loop {
                let current = *updates.borrow_and_update();
                if current.head_l1 != previous.head_l1 &&
                    let Some(block) = current.head_l1
                {
                    derivation_client
                        .send_new_l1_head(block)
                        .await
                        .map_err(|error| ActorError::Derivation(error.to_string()))?;
                }
                if current.finalized_l1 != previous.finalized_l1 &&
                    let Some(block) = current.finalized_l1
                {
                    derivation_client
                        .send_finalized_l1_block(block)
                        .await
                        .map_err(|error| ActorError::Derivation(error.to_string()))?;
                }
                previous = current;
                updates.changed().await.map_err(|_| ActorError::ChannelClosed)?;
            }
            #[allow(unreachable_code)]
            Ok::<(), ActorError>(())
        };
        let signer =
            Self::refresh_signer(&l1_provider, &rollup_config, &block_signer_sender, state);
        tokio::try_join!(forward, signer)?;
        Ok(())
    }

    async fn refresh_signer(
        provider: &impl Provider,
        config: &RollupConfig,
        signer_tx: &watch::Sender<Address>,
        mut state: watch::Receiver<State>,
    ) -> Result<(), ActorError> {
        use tokio::time::{self, Duration};
        let mut retry = time::interval(Duration::from_secs(10));
        retry.set_missed_tick_behavior(time::MissedTickBehavior::Skip);
        let mut last_head = None;
        loop {
            let state_changed = tokio::select! {
                result = state.changed() => {
                    result.map_err(|_| ActorError::ChannelClosed)?;
                    true
                }
                _ = retry.tick() => false,
                _ = signer_tx.closed() => return Err(ActorError::ChannelClosed),
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
                    warn!(target: "l1_watcher", chain_id = config.l2_chain_id.id(), ?result, "unsafe block signer refresh unavailable; retaining previous value")
                }
            }
        }
    }
}
