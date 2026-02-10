//! [`NodeActor`] implementation for an L1 chain watcher that polls for L1 block updates over HTTP
//! RPC.

use crate::{NodeActor, actors::l1_watcher::error::L1WatcherActorError};
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

use super::L1WatcherDerivationClient;

/// Builder for the [`L1WatcherActor`].
pub struct L1WatcherActorBuilder<BlockStream_, L1Provider_, L1WatcherDerivationClient_>
where
    BlockStream_: Stream<Item = BlockInfo> + Unpin + Send,
    L1Provider_: Provider,
    L1WatcherDerivationClient_: L1WatcherDerivationClient,
{
    /// The [`RollupConfig`].
    pub rollup_config: Arc<RollupConfig>,
    /// The L1 provider.
    pub l1_provider: L1Provider_,
    /// Client used to interact with the [`crate::DerivationActor`].
    pub derivation_client: L1WatcherDerivationClient_,
    /// The block signer sender.
    pub block_signer_sender: mpsc::Sender<Address>,
    /// A stream over the latest head.
    pub head_stream: BlockStream_,
    /// A stream over the finalized block accepted as canonical.
    pub finalized_stream: BlockStream_,
}

impl<BlockStream_, L1Provider_, L1WatcherDerivationClient_> std::fmt::Debug
    for L1WatcherActorBuilder<BlockStream_, L1Provider_, L1WatcherDerivationClient_>
where
    BlockStream_: Stream<Item = BlockInfo> + Unpin + Send,
    L1Provider_: Provider,
    L1WatcherDerivationClient_: L1WatcherDerivationClient,
{
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("L1WatcherActorBuilder").finish()
    }
}

/// Inbound data for the [`L1WatcherActor`].
#[derive(Debug)]
pub struct L1WatcherInboundData {
    /// A channel to send queries about the state of L1.
    pub query_tx: mpsc::Sender<L1WatcherQueries>,
    /// A receiver for L1 head updates (subscribed from the actor's watch channel).
    pub l1_head_updates_rx: watch::Receiver<Option<BlockInfo>>,
}

/// An L1 chain watcher that checks for L1 block updates over RPC.
pub struct L1WatcherActor<BlockStream_, L1Provider_, L1WatcherDerivationClient_>
where
    BlockStream_: Stream<Item = BlockInfo> + Unpin + Send,
    L1Provider_: Provider,
    L1WatcherDerivationClient_: L1WatcherDerivationClient,
{
    /// The [`RollupConfig`] to tell if ecotone is active.
    rollup_config: Arc<RollupConfig>,
    /// The L1 provider.
    l1_provider: L1Provider_,
    /// The inbound queries to the L1 watcher.
    inbound_queries: mpsc::Receiver<L1WatcherQueries>,
    /// The latest L1 head block.
    latest_head: watch::Sender<Option<BlockInfo>>,
    /// Client used to interact with the [`crate::DerivationActor`].
    derivation_client: L1WatcherDerivationClient_,
    /// The block signer sender.
    block_signer_sender: mpsc::Sender<Address>,
    /// A stream over the latest head.
    head_stream: BlockStream_,
    /// A stream over the finalized block accepted as canonical.
    finalized_stream: BlockStream_,
    /// A subscriber to the latest head (for responding to queries).
    latest_head_subscriber: watch::Receiver<Option<BlockInfo>>,
}

impl<BlockStream_, L1Provider_, L1WatcherDerivationClient_> std::fmt::Debug
    for L1WatcherActor<BlockStream_, L1Provider_, L1WatcherDerivationClient_>
where
    BlockStream_: Stream<Item = BlockInfo> + Unpin + Send,
    L1Provider_: Provider,
    L1WatcherDerivationClient_: L1WatcherDerivationClient,
{
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("L1WatcherActor").finish()
    }
}

/// Fetches a block by [`BlockId`] from the L1 provider, returning `None` on error.
async fn fetch_block_info(provider: &impl Provider, block_id: BlockId) -> Option<BlockInfo> {
    match provider.get_block(block_id).await {
        Ok(block) => block,
        Err(e) => {
            warn!(target: "l1_watcher", error = ?e, "failed to query l1 provider for block");
            None
        }
    }
    .map(|block| block.into_consensus().into())
}

#[async_trait]
impl<BlockStream_, L1Provider_, L1WatcherDerivationClient_> NodeActor
    for L1WatcherActor<BlockStream_, L1Provider_, L1WatcherDerivationClient_>
where
    BlockStream_: Stream<Item = BlockInfo> + Unpin + Send + 'static,
    L1Provider_: Provider + 'static,
    L1WatcherDerivationClient_: L1WatcherDerivationClient + 'static,
{
    type Error = L1WatcherActorError<BlockInfo>;
    type Builder = L1WatcherActorBuilder<BlockStream_, L1Provider_, L1WatcherDerivationClient_>;
    type InboundData = L1WatcherInboundData;

    async fn init(builder: Self::Builder) -> Result<(Self::InboundData, Self), Self::Error> {
        let (query_tx, query_rx) = mpsc::channel(1024);
        let (latest_head_tx, latest_head_rx) = watch::channel(None);
        let latest_head_subscriber = latest_head_tx.subscribe();

        let actor = Self {
            rollup_config: builder.rollup_config,
            l1_provider: builder.l1_provider,
            inbound_queries: query_rx,
            latest_head: latest_head_tx,
            derivation_client: builder.derivation_client,
            block_signer_sender: builder.block_signer_sender,
            head_stream: builder.head_stream,
            finalized_stream: builder.finalized_stream,
            latest_head_subscriber,
        };

        let inbound = L1WatcherInboundData { query_tx, l1_head_updates_rx: latest_head_rx };

        Ok((inbound, actor))
    }

    async fn step(&mut self) -> Result<(), Self::Error> {
        select! {
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
                        let current_l1 = *self.latest_head_subscriber.borrow();
                        let head_l1 = fetch_block_info(&self.l1_provider, BlockId::latest()).await;
                        let finalized_l1 = fetch_block_info(&self.l1_provider, BlockId::finalized()).await;
                        let safe_l1 = fetch_block_info(&self.l1_provider, BlockId::safe()).await;

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

        Ok(())
    }
}
