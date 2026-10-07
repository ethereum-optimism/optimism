use crate::{EngineError, EngineRpcRequest, NodeActor};
use async_trait::async_trait;
use kona_engine::{EngineQueryClient, EngineState};
use kona_genesis::RollupConfig;
use std::sync::Arc;
use tokio::sync::{mpsc, watch};

/// Handles [`EngineRpcRequest`]s by reading engine state via a watch and
/// dispatching the request through an [`EngineQueryClient`].
///
/// The [`EngineQueryClient`] exposes only execution-layer reads, so this actor cannot call
/// Engine API mutations.
#[derive(Debug)]
pub struct EngineRpcActor {
    /// An [`EngineQueryClient`] used for handling engine queries.
    engine_rpc_client: EngineQueryClient,
    /// The [`RollupConfig`] used to handle queries.
    rollup_config: Arc<RollupConfig>,
    /// Receiver for [`EngineState`] updates.
    engine_state_receiver: watch::Receiver<EngineState>,
    /// The inbound request channel.
    inbound_request_rx: mpsc::Receiver<EngineRpcRequest>,
}

impl EngineRpcActor {
    /// Constructs a new [`EngineRpcActor`].
    pub const fn new(
        engine_rpc_client: EngineQueryClient,
        rollup_config: Arc<RollupConfig>,
        engine_state_receiver: watch::Receiver<EngineState>,
        inbound_request_rx: mpsc::Receiver<EngineRpcRequest>,
    ) -> Self {
        Self { engine_rpc_client, rollup_config, engine_state_receiver, inbound_request_rx }
    }

    async fn handle_rpc_request(&self, request: EngineRpcRequest) -> Result<(), EngineError> {
        let EngineRpcRequest(req) = request;
        trace!(target: "engine", ?req, "Received engine query.");

        if let Err(e) = req
            .handle(&self.engine_state_receiver, &self.engine_rpc_client, &self.rollup_config)
            .await
        {
            warn!(target: "engine", err = ?e, "Failed to handle engine query.");
        }

        Ok(())
    }
}

#[async_trait]
impl NodeActor for EngineRpcActor {
    type Error = EngineError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        let query = self.inbound_request_rx.recv().await.ok_or_else(|| {
            error!(target: "engine", "Engine rpc request receiver closed unexpectedly");
            EngineError::ChannelClosed
        })?;
        self.handle_rpc_request(query).await
    }
}
