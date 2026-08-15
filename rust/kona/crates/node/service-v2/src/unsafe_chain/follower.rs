//! Network-following unsafe-chain workflow.

use crate::{
    engine::{EngineClient, EngineError},
    unsafe_chain::{UnsafeChainError, UnsafePayloadIngressError},
};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;

/// Default number of network payloads that may wait for engine processing.
pub const DEFAULT_UNSAFE_PAYLOAD_CAPACITY: usize = 256;

/// A cloneable network ingress for complete unsafe payloads.
#[derive(Debug, Clone)]
pub struct UnsafePayloadIngress {
    payload_tx: mpsc::Sender<OpExecutionPayloadEnvelope>,
}

impl UnsafePayloadIngress {
    /// Submits a complete unsafe payload received from the network.
    pub async fn send(
        &self,
        payload: OpExecutionPayloadEnvelope,
    ) -> Result<(), UnsafePayloadIngressError> {
        self.payload_tx.send(payload).await.map_err(|_| UnsafePayloadIngressError::Unavailable)
    }
}

/// Imports complete network payloads through the semantic engine interface.
#[derive(Debug)]
pub struct FollowerService {
    engine: EngineClient,
    payload_rx: mpsc::Receiver<OpExecutionPayloadEnvelope>,
}

impl FollowerService {
    /// Creates a follower service and its network payload ingress.
    pub fn new(engine: EngineClient) -> (Self, UnsafePayloadIngress) {
        Self::with_capacity(engine, DEFAULT_UNSAFE_PAYLOAD_CAPACITY)
    }

    /// Creates a follower service with a bounded network payload capacity.
    pub fn with_capacity(engine: EngineClient, capacity: usize) -> (Self, UnsafePayloadIngress) {
        let (payload_tx, payload_rx) = mpsc::channel(capacity);
        (Self { engine, payload_rx }, UnsafePayloadIngress { payload_tx })
    }

    /// Follows network payloads until node shutdown.
    ///
    /// Shutdown is checked between payloads. Import of a payload already removed from the queue is
    /// allowed to finish so an in-flight `newPayload` or forkchoice update is not cancelled.
    pub async fn run(mut self, shutdown: CancellationToken) -> Result<(), UnsafeChainError> {
        loop {
            let payload = tokio::select! {
                biased;
                _ = shutdown.cancelled() => return Ok(()),
                payload = self.payload_rx.recv() => {
                    payload.ok_or(UnsafeChainError::PayloadChannelClosed)?
                }
            };

            match self.engine.import_unsafe(payload).await {
                Ok(_) => {}
                Err(EngineError::InvalidUnsafePayload(error)) => {
                    tracing::warn!(target: "unsafe_chain", %error, "Dropping invalid unsafe payload");
                }
                Err(error) => return Err(error.into()),
            }
        }
    }
}
