//! Per-chain state served by the [`L1WatcherActor`](super::L1WatcherActor).

use super::{L1WatcherActorError, L1WatcherDerivationClient};
use alloy_primitives::Address;
use alloy_provider::Provider;
use kona_genesis::RollupConfig;
use kona_protocol::BlockInfo;
use kona_rpc::L1WatcherQueries;
use std::{future::IntoFuture, sync::Arc};
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
    /// Runs independent forwarding, configuration, and query loops for this chain. A closed
    /// channel tears down only this worker; slow RPCs cannot hold up shared L1 observation.
    pub(super) async fn run(
        self,
        provider: impl Provider,
        head: tokio::sync::watch::Receiver<Option<BlockInfo>>,
        finalized: tokio::sync::watch::Receiver<Option<BlockInfo>>,
    ) -> Result<(), L1WatcherActorError<BlockInfo>> {
        let chain_id = self.chain_id();
        let Self { rollup_config, derivation_client, block_signer_sender, mut inbound_queries } =
            self;
        let mut heads = head.clone();
        let mut finalized = finalized;
        let forward = async {
            loop {
                tokio::select! {
                    result = heads.changed() => {
                        result.map_err(|_| L1WatcherActorError::StreamEnded)?;
                        let block = *heads.borrow_and_update();
                        if let Some(block) = block {
                            derivation_client.send_new_l1_head(block).await.map_err(|source| L1WatcherActorError::DerivationClientError {chain_id, source})?;
                        }
                    }
                    result = finalized.changed() => {
                        result.map_err(|_| L1WatcherActorError::StreamEnded)?;
                        let block = *finalized.borrow_and_update();
                        if let Some(block) = block {
                            derivation_client.send_finalized_l1_block(block).await.map_err(|source| L1WatcherActorError::DerivationClientError {chain_id, source})?;
                        }
                    }
                }
            }
            #[allow(unreachable_code)]
            Ok::<(), L1WatcherActorError<BlockInfo>>(())
        };
        let signer =
            Self::refresh_signer(&provider, &rollup_config, &block_signer_sender, head.clone());
        let queries = Self::serve_queries(&provider, &rollup_config, head, &mut inbound_queries);
        tokio::try_join!(forward, signer, queries)?;
        Ok(())
    }

    async fn refresh_signer(
        provider: &impl Provider,
        config: &RollupConfig,
        signer_tx: &mpsc::Sender<Address>,
        mut head: tokio::sync::watch::Receiver<Option<BlockInfo>>,
    ) -> Result<(), L1WatcherActorError<BlockInfo>> {
        use tokio::time::{self, Duration};
        let mut retry = time::interval(Duration::from_secs(10));
        retry.set_missed_tick_behavior(time::MissedTickBehavior::Skip);
        let mut last_signer = None;
        loop {
            tokio::select! {
                result = head.changed() => result.map_err(|_| L1WatcherActorError::StreamEnded)?,
                _ = retry.tick() => {},
                _ = signer_tx.closed() => return Err(L1WatcherActorError::StreamEnded),
            }
            let Some(target) = *head.borrow_and_update() else { continue };
            let read = kona_providers_alloy::unsafe_block_signer(
                provider,
                config.l1_system_config_address,
                target.hash,
            );
            match time::timeout(Duration::from_secs(10), read).await {
                Ok(Ok(signer)) => {
                    // Head changes while a read is outstanding supersede that read, including
                    // same-height reorgs. Never install a snapshot for an obsolete branch.
                    if *head.borrow() != Some(target) || last_signer == Some(signer) {
                        continue;
                    }
                    let permit =
                        signer_tx.reserve().await.map_err(|_| L1WatcherActorError::StreamEnded)?;
                    if *head.borrow() == Some(target) {
                        permit.send(signer);
                        last_signer = Some(signer);
                    }
                }
                result => {
                    warn!(target: "l1_watcher", chain_id = config.l2_chain_id.id(), ?result, "Failed to refresh unsafe block signer; retaining previous value")
                }
            }
        }
    }

    async fn serve_queries(
        provider: &impl Provider,
        config: &RollupConfig,
        head: tokio::sync::watch::Receiver<Option<BlockInfo>>,
        queries: &mut mpsc::Receiver<L1WatcherQueries>,
    ) -> Result<(), L1WatcherActorError<BlockInfo>> {
        while let Some(query) = queries.recv().await {
            match query {
                L1WatcherQueries::Config(sender) => {
                    let _ = sender.send(config.clone());
                }
                L1WatcherQueries::L1State(sender) => {
                    let current = *head.borrow();
                    let _ = sender.send(Self::l1_state(provider, current).await);
                }
            }
        }
        Err(L1WatcherActorError::StreamEnded)
    }

    async fn l1_state(
        provider: &impl Provider,
        current_l1: Option<BlockInfo>,
    ) -> kona_rpc::L1State {
        use alloy_eips::BlockId;
        use tokio::time::{Duration, timeout};
        let read = |id| async move {
            match timeout(Duration::from_secs(10), provider.get_block(id).into_future()).await {
                Ok(Ok(block)) => block.map(|block| block.into_consensus().into()),
                result => {
                    warn!(target: "l1_watcher", ?result, "L1 state query failed");
                    None
                }
            }
        };
        let (head_l1, finalized_l1, safe_l1) = tokio::join!(
            read(BlockId::latest()),
            read(BlockId::finalized()),
            read(BlockId::safe())
        );
        kona_rpc::L1State {
            current_l1,
            current_l1_finalized: finalized_l1,
            head_l1,
            finalized_l1,
            safe_l1,
        }
    }
}
