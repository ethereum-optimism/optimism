//! [`NodeActor`] implementation for an L1 chain watcher that polls for L1 block updates over HTTP
//! RPC.

use crate::{
    NodeActor,
    actors::{CancellableContext, l1_watcher::error::L1WatcherActorError},
};
use alloy_eips::BlockId;
use alloy_primitives::Address;
use alloy_provider::Provider;
use async_trait::async_trait;
use futures::{Stream, StreamExt};
use kona_genesis::{RollupConfig, SystemConfigLog, SystemConfigUpdate, UnsafeBlockSignerUpdate};
use kona_protocol::BlockInfo;
use kona_rpc::{L1State, L1WatcherQueries};
use std::sync::Arc;
use tokio::{
    select,
    sync::{
        mpsc::{self},
        watch,
    },
};
use tokio_util::sync::{CancellationToken, WaitForCancellationFuture};

use super::L1WatcherDerivationClient;

/// An L1 chain watcher that checks for L1 block updates over RPC.
#[derive(Debug)]
pub struct L1WatcherActor<BlockStream, L1Provider, L1WatcherDerivationClient_>
where
    BlockStream: Stream<Item = BlockInfo> + Unpin + Send,
    L1Provider: Provider,
    L1WatcherDerivationClient_: L1WatcherDerivationClient,
{
    /// The [`RollupConfig`] to tell if ecotone is active.
    /// This is used to determine if the L1 watcher should check for unsafe block signer updates.
    rollup_config: Arc<RollupConfig>,
    /// The L1 provider.
    l1_provider: L1Provider,
    /// The inbound queries to the L1 watcher.
    inbound_queries: mpsc::Receiver<L1WatcherQueries>,
    /// The latest L1 head block.
    latest_head: watch::Sender<Option<BlockInfo>>,
    /// Client used to interact with the [`crate::DerivationActor`].
    derivation_client: L1WatcherDerivationClient_,
    /// The block signer sender.
    block_signer_sender: mpsc::Sender<Address>,
    /// The cancellation token, shared between all tasks.
    cancellation: CancellationToken,
    /// A stream over the latest head.
    head_stream: BlockStream,
    /// A stream over the finalized block accepted as canonical.
    finalized_stream: BlockStream,
}
impl<BlockStream, L1Provider, L1WatcherDerivationClient_>
    L1WatcherActor<BlockStream, L1Provider, L1WatcherDerivationClient_>
where
    BlockStream: Stream<Item = BlockInfo> + Unpin + Send,
    L1Provider: Provider,
    L1WatcherDerivationClient_: L1WatcherDerivationClient,
{
    /// Instantiate a new [`L1WatcherActor`].
    #[allow(clippy::too_many_arguments)]
    pub const fn new(
        rollup_config: Arc<RollupConfig>,
        l1_provider: L1Provider,
        l1_query_rx: mpsc::Receiver<L1WatcherQueries>,
        l1_head_updates_tx: watch::Sender<Option<BlockInfo>>,
        derivation_client: L1WatcherDerivationClient_,
        signer: mpsc::Sender<Address>,
        cancellation: CancellationToken,
        head_stream: BlockStream,
        finalized_stream: BlockStream,
    ) -> Self {
        Self {
            rollup_config,
            l1_provider,
            inbound_queries: l1_query_rx,
            latest_head: l1_head_updates_tx,
            derivation_client,
            block_signer_sender: signer,
            cancellation,
            head_stream,
            finalized_stream,
        }
    }
}

#[async_trait]
impl<BlockStream, L1Provider, L1WatcherDerivationClient_> NodeActor
    for L1WatcherActor<BlockStream, L1Provider, L1WatcherDerivationClient_>
where
    BlockStream: Stream<Item = BlockInfo> + Unpin + Send + 'static,
    L1Provider: Provider + 'static,
    L1WatcherDerivationClient_: L1WatcherDerivationClient + 'static,
{
    type Error = L1WatcherActorError<BlockInfo>;
    type StartData = ();

    /// Start the main processing loop.
    async fn start(mut self, _: Self::StartData) -> Result<(), Self::Error> {
        let cancel = self.cancellation.clone();
        let latest_head = self.latest_head.subscribe();

        loop {
            select! {
                _ = cancel.cancelled() => {
                    // Exit the task on cancellation.
                    info!(
                        target: "l1_watcher",
                        "Received shutdown signal. Exiting L1 watcher task."
                    );

                    return Ok(());
                },
                new_head = self.head_stream.next() => match new_head {
                    None => {
                        return Err(L1WatcherActorError::StreamEnded);
                    }
                    Some(head_block_info) => {
                        // Send the head update event to all consumers.
                        self.latest_head.send_replace(Some(head_block_info));
                        self.derivation_client.send_new_l1_head(head_block_info).await.map_err(|e| {
                            warn!(target: "l1_watcher", "Error sending l1 head update to derivation actor: {e}");
                            L1WatcherActorError::DerivationClientError(e)
                        })?;

                        // For each log, attempt to construct a [`SystemConfigLog`].
                        // Build the [`SystemConfigUpdate`] from the log.
                        // If the update is an Unsafe block signer update, send the address
                        // to the block signer sender.
                        let filter_address =  self.rollup_config.l1_system_config_address;
                        let logs = self.l1_provider .get_logs(&alloy_rpc_types_eth::Filter::new().address(filter_address).select(head_block_info.hash)).await?;
                        let ecotone_active = self.rollup_config.is_ecotone_active(head_block_info.timestamp);
                        for log in logs {
                            let sys_cfg_log = SystemConfigLog::new(log.into(), ecotone_active);
                            if let Ok(SystemConfigUpdate::UnsafeBlockSigner(UnsafeBlockSignerUpdate { unsafe_block_signer })) = sys_cfg_log.build() {
                                info!(
                                    target: "l1_watcher",
                                    "Unsafe block signer update: {unsafe_block_signer}"
                                );
                                if let Err(e) = self.block_signer_sender.send(unsafe_block_signer).await {
                                    error!(
                                        target: "l1_watcher",
                                        "Error sending unsafe block signer update: {e}"
                                    );
                                }
                            }
                        }
                    },
                },
                new_finalized = self.finalized_stream.next() => match new_finalized {
                    None => {
                        return Err(L1WatcherActorError::StreamEnded);
                    }
                    Some(finalized_block_info) => {
                        self.derivation_client.send_finalized_l1_block(finalized_block_info).await.map_err(|e| {
                            warn!(target: "l1_watcher", "Error sending finalized l1 block update to derivation actor: {e}");
                            L1WatcherActorError::DerivationClientError(e)
                        })?;
                    }
                },
                inbound_query = self.inbound_queries.recv() => match inbound_query {
                Some(query) => {
                    match query {
                        L1WatcherQueries::Config(sender) => {
                            if let Err(e) = sender.send((*self.rollup_config).clone()) {
                                warn!(target: "l1_watcher", error = ?e, "Failed to send L1 config to the query sender");
                            }
                        }
                        L1WatcherQueries::L1State(sender) => {
                            let current_l1 = *latest_head.borrow();

                            let head_l1 = match self.l1_provider.get_block(BlockId::latest()).await {
                                    Ok(block) => block,
                                    Err(e) => {
                                        warn!(target: "l1_watcher", error = ?e, "failed to query l1 provider for latest head block");
                                        None
                                    }}.map(|block| block.into_consensus().into());

                            let finalized_l1 = match self.l1_provider.get_block(BlockId::finalized()).await {
                                    Ok(block) => block,
                                    Err(e) => {
                                        warn!(target: "l1_watcher", error = ?e, "failed to query l1 provider for latest finalized block");
                                        None
                                    }}.map(|block| block.into_consensus().into());

                            let safe_l1 = match self.l1_provider.get_block(BlockId::safe()).await {
                                    Ok(block) => block,
                                    Err(e) => {
                                        warn!(target: "l1_watcher", error = ?e, "failed to query l1 provider for latest safe block");
                                        None
                                    }}.map(|block| block.into_consensus().into());

                            if let Err(e) = sender.send(L1State {
                                current_l1,
                                current_l1_finalized: finalized_l1,
                                head_l1,
                                safe_l1,
                                finalized_l1,
                            }) {
                                warn!(target: "l1_watcher", error = ?e, "Failed to send L1 state to the query sender");
                            }
                        }
                    }
                },
                None => {
                    error!(target: "l1_watcher", "L1 watcher query channel closed unexpectedly, exiting query processor task.");
                    return Err(L1WatcherActorError::StreamEnded)
                }
            }
            }
        }
    }
}

impl<BlockStream, L1Provider, L1WatcherDerivationClient_> CancellableContext
    for L1WatcherActor<BlockStream, L1Provider, L1WatcherDerivationClient_>
where
    BlockStream: Stream<Item = BlockInfo> + Unpin + Send + 'static,
    L1Provider: Provider,
    L1WatcherDerivationClient_: L1WatcherDerivationClient + 'static,
{
    fn cancelled(&self) -> WaitForCancellationFuture<'_> {
        self.cancellation.cancelled()
    }
}
