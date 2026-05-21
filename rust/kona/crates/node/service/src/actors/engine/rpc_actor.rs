use crate::{EngineError, EngineRpcRequest, NodeActor};
use async_trait::async_trait;
use kona_engine::{EngineClient, EngineState};
use kona_genesis::RollupConfig;
use std::sync::Arc;
use tokio::sync::{mpsc, watch};

/// Handles [`EngineRpcRequest`]s by reading engine state and queue length via watches and
/// dispatching the request through an [`EngineClient`].
#[derive(Debug)]
pub struct EngineRpcActor<EngineClient_: EngineClient> {
    /// An [`EngineClient`] used for handling engine queries.
    engine_client: Arc<EngineClient_>,
    /// The [`RollupConfig`] used to handle queries.
    rollup_config: Arc<RollupConfig>,
    /// Receiver for [`EngineState`] updates.
    engine_state_receiver: watch::Receiver<EngineState>,
    /// Receiver for engine queue length updates.
    engine_queue_length_receiver: watch::Receiver<usize>,
    /// The inbound request channel.
    inbound_request_rx: mpsc::Receiver<EngineRpcRequest>,
}

impl<EngineClient_: EngineClient> EngineRpcActor<EngineClient_> {
    /// Constructs a new [`EngineRpcActor`].
    pub const fn new(
        engine_client: Arc<EngineClient_>,
        rollup_config: Arc<RollupConfig>,
        engine_state_receiver: watch::Receiver<EngineState>,
        engine_queue_length_receiver: watch::Receiver<usize>,
        inbound_request_rx: mpsc::Receiver<EngineRpcRequest>,
    ) -> Self {
        Self {
            engine_client,
            rollup_config,
            engine_state_receiver,
            engine_queue_length_receiver,
            inbound_request_rx,
        }
    }

    async fn handle_rpc_request(&self, request: EngineRpcRequest) -> Result<(), EngineError> {
        let EngineRpcRequest(req) = request;
        trace!(target: "engine", ?req, "Received engine query.");

        if let Err(e) = req
            .handle(
                &self.engine_state_receiver,
                &self.engine_queue_length_receiver,
                &self.engine_client,
                &self.rollup_config,
            )
            .await
        {
            warn!(target: "engine", err = ?e, "Failed to handle engine query.");
        }

        Ok(())
    }
}

#[async_trait]
impl<EngineClient_> NodeActor for EngineRpcActor<EngineClient_>
where
    EngineClient_: EngineClient + 'static,
{
    type Error = EngineError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        let query = self.inbound_request_rx.recv().await.ok_or_else(|| {
            error!(target: "engine", "Engine rpc request receiver closed unexpectedly");
            EngineError::ChannelClosed
        })?;
        self.handle_rpc_request(query).await
    }
}
