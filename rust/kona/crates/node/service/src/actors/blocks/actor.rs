use super::{
    BlocksClient, BlocksClientConfig, BlocksClientError, BlocksClientLocalProvider,
    BlocksClientLocalProviderError,
};
#[cfg(feature = "metrics")]
use crate::Metrics;
use crate::{EngineClientError, NetworkEngineClient, NodeActor};
use alloy_eips::BlockNumHash;
use async_trait::async_trait;
use backon::{BackoffBuilder, ExponentialBackoff, ExponentialBuilder};
use kona_engine::EngineState;
use std::time::Duration;
use thiserror::Error;
use tokio::sync::watch;
use tokio_tungstenite::tungstenite::http::StatusCode;

const DEFAULT_RECONNECT_MAX_DELAY: Duration = Duration::from_secs(30);

/// Actor that forwards canonical unsafe blocks from a blocks stream to Kona's engine actor.
#[derive(Debug)]
pub struct BlocksClientActor<EngineClient, LocalProvider> {
    config: BlocksClientConfig,
    engine_client: EngineClient,
    local_provider: LocalProvider,
    engine_state_rx: watch::Receiver<EngineState>,
    genesis_number: u64,
    blocks_client: Option<BlocksClient>,
    backoff_builder: ExponentialBuilder,
    reconnect_backoff: ExponentialBackoff,
}

impl<EngineClient, LocalProvider> BlocksClientActor<EngineClient, LocalProvider> {
    /// Creates a blocks client actor.
    pub fn new(
        config: BlocksClientConfig,
        engine_client: EngineClient,
        local_provider: LocalProvider,
        engine_state_rx: watch::Receiver<EngineState>,
        genesis_number: u64,
    ) -> Self {
        let backoff_builder = default_reconnect_backoff();
        let reconnect_backoff = backoff_builder.build();
        Self {
            config,
            engine_client,
            local_provider,
            engine_state_rx,
            genesis_number,
            blocks_client: None,
            backoff_builder,
            reconnect_backoff,
        }
    }

    #[cfg(test)]
    pub(super) fn with_backoff(mut self, backoff_builder: ExponentialBuilder) -> Self {
        self.reconnect_backoff = backoff_builder.build();
        self.backoff_builder = backoff_builder;
        self
    }

    fn reset_backoff(&mut self) {
        self.reconnect_backoff = self.backoff_builder.build();
    }
}

impl<EngineClient, LocalProvider> BlocksClientActor<EngineClient, LocalProvider>
where
    EngineClient: NetworkEngineClient,
    LocalProvider: BlocksClientLocalProvider,
{
    async fn establish_connection(&mut self) -> Result<BlocksClient, EstablishConnectionError> {
        let local_head = self.local_provider.latest_block().await?;
        let safe_head = self.local_provider.safe_block().await?.unwrap_or_else(|| BlockNumHash {
            number: self.genesis_number,
            hash: Default::default(),
        });
        let safe_number = safe_head.number.clamp(self.genesis_number, local_head.number);
        let mut anchor_number = local_head.number;

        loop {
            let mut client = match BlocksClient::connect(
                self.config.endpoint.clone(),
                anchor_number,
            )
            .await
            {
                Ok(client) => client,
                Err(error)
                    if error.http_status() == Some(StatusCode::RANGE_NOT_SATISFIABLE) &&
                        anchor_number > safe_number =>
                {
                    warn!(
                        target: "blocks_client",
                        anchor_number,
                        safe_number,
                        "Sequencer blocks server is behind the local unsafe head; anchoring from the local safe head"
                    );
                    anchor_number = safe_number;
                    continue;
                }
                Err(error) if error.http_status() == Some(StatusCode::RANGE_NOT_SATISFIABLE) => {
                    return Err(BlocksClientActorError::ServerBehindSafe { safe_number }.into());
                }
                Err(error) if error.http_status() == Some(StatusCode::GONE) => {
                    return Err(BlocksClientActorError::HistoryUnavailable {
                        block_number: anchor_number,
                    }
                    .into());
                }
                Err(error) if error.is_retryable() => {
                    return Err(EstablishConnectionError::Retry(error));
                }
                Err(error) => return Err(BlocksClientActorError::Client(error).into()),
            };

            let remote_anchor = match client.next_block().await {
                Ok(block) => block,
                Err(error) if error.is_retryable() => {
                    return Err(EstablishConnectionError::Retry(error));
                }
                Err(error) => return Err(BlocksClientActorError::Client(error).into()),
            };
            let remote_number = remote_anchor.execution_payload.block_number();
            let remote_hash = remote_anchor.execution_payload.block_hash();
            if remote_number < anchor_number {
                let replacement_parent = remote_number.saturating_sub(1);
                if replacement_parent < safe_number {
                    return Err(BlocksClientActorError::ReorgBelowSafe {
                        safe_number,
                        replacement_number: remote_number,
                    }
                    .into());
                }
                warn!(
                    target: "blocks_client",
                    requested = anchor_number,
                    received = remote_number,
                    replacement_parent,
                    "Sequencer reorged while anchoring the blocks stream; restarting from the replacement parent"
                );
                anchor_number = replacement_parent;
                continue;
            }
            if remote_number > anchor_number {
                return Err(BlocksClientActorError::UnexpectedAnchor {
                    requested: anchor_number,
                    received: remote_number,
                }
                .into());
            }

            let local_anchor = self
                .local_provider
                .block_by_number(anchor_number)
                .await?
                .ok_or(BlocksClientActorError::LocalBlockUnavailable(anchor_number))?;
            if local_anchor.hash == remote_hash {
                info!(
                    target: "blocks_client",
                    endpoint = %self.config.endpoint,
                    number = anchor_number,
                    hash = %remote_hash,
                    "Anchored sequencer blocks stream to local L2 chain"
                );
                kona_macros::set!(gauge, Metrics::BLOCKS_CLIENT_CONNECTED, 1.0);
                kona_macros::inc!(counter, Metrics::BLOCKS_CLIENT_CONNECTIONS);
                self.reset_backoff();
                return Ok(client);
            }

            warn!(
                target: "blocks_client",
                number = anchor_number,
                local_hash = %local_anchor.hash,
                remote_hash = %remote_hash,
                "Local unsafe chain differs from sequencer blocks stream; searching for common ancestor"
            );
            if anchor_number <= safe_number {
                return Err(BlocksClientActorError::NoCommonAncestor {
                    safe_number,
                    local_hash: local_anchor.hash,
                    remote_hash,
                }
                .into());
            }
            anchor_number = anchor_number.saturating_sub(1);
        }
    }

    async fn wait_until_applied(
        &mut self,
        expected: BlockNumHash,
    ) -> Result<(), BlocksClientActorError> {
        loop {
            let unsafe_head = self.engine_state_rx.borrow().sync_state.unsafe_head().block_info;
            if unsafe_head.number == expected.number && unsafe_head.hash == expected.hash {
                return Ok(());
            }

            let local_head = self.local_provider.latest_block().await?;
            if local_head.number >= expected.number &&
                self.local_provider
                    .block_by_number(expected.number)
                    .await?
                    .is_some_and(|block| block.hash == expected.hash)
            {
                return Ok(());
            }

            self.engine_state_rx
                .changed()
                .await
                .map_err(|_| BlocksClientActorError::EngineStateClosed)?;
        }
    }

    async fn reconnect_after(&mut self, error: &BlocksClientError) {
        self.blocks_client = None;
        kona_macros::set!(gauge, Metrics::BLOCKS_CLIENT_CONNECTED, 0.0);
        kona_macros::inc!(counter, Metrics::BLOCKS_CLIENT_RECONNECTS);
        let delay = self.reconnect_backoff.next().unwrap_or(DEFAULT_RECONNECT_MAX_DELAY);
        warn!(
            target: "blocks_client",
            %error,
            ?delay,
            "Sequencer blocks stream disconnected; reconnecting from applied local head"
        );
        tokio::time::sleep(delay).await;
    }

    async fn forward_next_block(&mut self) -> Result<(), BlocksClientActorError> {
        let result = self
            .blocks_client
            .as_mut()
            .expect("connection established before forwarding")
            .next_block()
            .await;
        let block = match result {
            Ok(block) => block,
            Err(error) if error.is_retryable() => {
                self.reconnect_after(&error).await;
                return Ok(());
            }
            Err(error) => return Err(error.into()),
        };

        let number = block.execution_payload.block_number();
        let hash = block.execution_payload.block_hash();
        let previous_head = self.local_provider.latest_block().await?;
        kona_macros::set!(gauge, Metrics::BLOCKS_CLIENT_RECEIVED_BLOCK, number as f64);
        if number <= previous_head.number &&
            self.local_provider
                .block_by_number(number)
                .await?
                .is_some_and(|local| local.hash == hash)
        {
            trace!(
                target: "blocks_client",
                number,
                %hash,
                "Skipping blocks stream payload already canonical in the local execution layer"
            );
            kona_macros::set!(gauge, Metrics::BLOCKS_CLIENT_APPLIED_BLOCK, number as f64);
            return Ok(());
        }

        trace!(
            target: "blocks_client",
            number,
            %hash,
            "Forwarding unsafe block from sequencer blocks stream"
        );

        // Mark the current state version as observed before submitting the payload. The subsequent
        // `changed` notification then acts as an acknowledgement that the engine actor drained a
        // task after this request was queued. The local provider check also handles coalesced watch
        // updates and descendants of the submitted block.
        drop(self.engine_state_rx.borrow_and_update());
        self.engine_client.send_unsafe_block(block).await?;
        self.wait_until_applied(BlockNumHash { number, hash }).await?;

        kona_macros::set!(gauge, Metrics::BLOCKS_CLIENT_APPLIED_BLOCK, number as f64);
        if number <= previous_head.number && hash != previous_head.hash {
            let depth = previous_head.number.saturating_sub(number).saturating_add(1);
            kona_macros::inc!(counter, Metrics::BLOCKS_CLIENT_REORGS);
            kona_macros::set!(gauge, Metrics::BLOCKS_CLIENT_REORG_DEPTH, depth as f64);
            info!(
                target: "blocks_client",
                number,
                %hash,
                depth,
                "Applied replacement block from sequencer blocks stream"
            );
        }
        Ok(())
    }
}

#[async_trait]
impl<EngineClient, LocalProvider> NodeActor for BlocksClientActor<EngineClient, LocalProvider>
where
    EngineClient: NetworkEngineClient + 'static,
    LocalProvider: BlocksClientLocalProvider + 'static,
{
    type Error = BlocksClientActorError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        loop {
            if self.blocks_client.is_none() {
                match self.establish_connection().await {
                    Ok(client) => self.blocks_client = Some(client),
                    Err(EstablishConnectionError::Retry(error)) => {
                        self.reconnect_after(&error).await;
                        continue;
                    }
                    Err(EstablishConnectionError::Fatal(error)) => {
                        kona_macros::inc!(counter, Metrics::BLOCKS_CLIENT_FAILURES);
                        return Err(error);
                    }
                }
            }

            if let Err(error) = self.forward_next_block().await {
                kona_macros::inc!(counter, Metrics::BLOCKS_CLIENT_FAILURES);
                return Err(error);
            }
            if self.blocks_client.is_some() {
                return Ok(());
            }
        }
    }
}

fn default_reconnect_backoff() -> ExponentialBuilder {
    ExponentialBuilder::default()
        .with_jitter()
        .with_max_delay(DEFAULT_RECONNECT_MAX_DELAY)
        .without_max_times()
}

#[derive(Debug)]
enum EstablishConnectionError {
    Retry(BlocksClientError),
    Fatal(BlocksClientActorError),
}

impl From<BlocksClientActorError> for EstablishConnectionError {
    fn from(error: BlocksClientActorError) -> Self {
        Self::Fatal(error)
    }
}

impl From<BlocksClientLocalProviderError> for EstablishConnectionError {
    fn from(error: BlocksClientLocalProviderError) -> Self {
        Self::Fatal(error.into())
    }
}

/// An error forwarding blocks stream payloads to Kona's engine actor.
#[derive(Debug, Error)]
pub enum BlocksClientActorError {
    /// The blocks stream client failed permanently.
    #[error(transparent)]
    Client(#[from] BlocksClientError),
    /// The payload could not be forwarded to the engine actor.
    #[error(transparent)]
    Engine(#[from] EngineClientError),
    /// The local L2 provider failed.
    #[error(transparent)]
    LocalProvider(#[from] BlocksClientLocalProviderError),
    /// The local block required for anchoring is unavailable.
    #[error("local L2 block {0} is unavailable while anchoring the blocks stream")]
    LocalBlockUnavailable(u64),
    /// The blocks server returned a block other than the requested anchor.
    #[error("blocks server returned anchor block {received}, requested {requested}")]
    UnexpectedAnchor {
        /// Requested anchor block number.
        requested: u64,
        /// Received block number.
        received: u64,
    },
    /// The local and sequencer chains disagree at the local safe head.
    #[error(
        "no common blocks-stream ancestor at safe block {safe_number}: local {local_hash}, remote {remote_hash}"
    )]
    NoCommonAncestor {
        /// Local safe block number.
        safe_number: u64,
        /// Local hash at the safe block number.
        local_hash: alloy_primitives::B256,
        /// Remote hash at the safe block number.
        remote_hash: alloy_primitives::B256,
    },
    /// The server replayed a replacement block below the local safe head.
    #[error(
        "sequencer blocks stream replacement block {replacement_number} reorgs below local safe block {safe_number}"
    )]
    ReorgBelowSafe {
        /// Local safe block number.
        safe_number: u64,
        /// First replacement block number sent by the server.
        replacement_number: u64,
    },
    /// The sequencer server cannot serve the local safe head.
    #[error("sequencer blocks server is behind local safe block {safe_number}")]
    ServerBehindSafe {
        /// Local safe block number.
        safe_number: u64,
    },
    /// The blocks server no longer has required historical block data.
    #[error("sequencer blocks server cannot serve required historical block {block_number}")]
    HistoryUnavailable {
        /// Required block number.
        block_number: u64,
    },
    /// The engine state watch closed before the submitted block was applied.
    #[error("engine state watch closed while waiting for blocks stream payload application")]
    EngineStateClosed,
}
