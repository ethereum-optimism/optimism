use async_trait::async_trait;
use derive_more::Constructor;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::fmt::Debug;
use thiserror::Error;
use tokio::sync::mpsc;

/// Client used to schedule unsafe [`OpExecutionPayloadEnvelope`] to be gossiped.
#[cfg_attr(test, mockall::automock)]
#[async_trait]
pub trait UnsafePayloadGossipClient: Send + Sync + Debug {
    /// This is a fire-and-forget function that schedules the provided
    /// [`OpExecutionPayloadEnvelope`] to be gossiped. The implementation should return as
    /// quickly as possible and offers no guarantees that the payload actually was gossiped
    /// successfully.
    async fn schedule_execution_payload_gossip(
        &self,
        payload: OpExecutionPayloadEnvelope,
    ) -> Result<(), UnsafePayloadGossipClientError>;
}

/// Errors that can occur when using the [`UnsafePayloadGossipClient`].
#[derive(Debug, Error)]
pub enum UnsafePayloadGossipClientError {
    /// Error sending request.
    #[error("Error sending request: {0}")]
    RequestError(String),
}

/// Queued implementation of [`UnsafePayloadGossipClient`] that handles requests by sending them
/// to a handler via the contained sender.
#[derive(Debug, Clone, Constructor)]
pub struct QueuedUnsafePayloadGossipClient {
    /// Queue used to relay unsafe payloads to gossip.
    request_tx: mpsc::Sender<OpExecutionPayloadEnvelope>,
}

#[async_trait]
impl UnsafePayloadGossipClient for QueuedUnsafePayloadGossipClient {
    async fn schedule_execution_payload_gossip(
        &self,
        payload: OpExecutionPayloadEnvelope,
    ) -> Result<(), UnsafePayloadGossipClientError> {
        self.request_tx
            .send(payload.clone())
            .await
            .map_err(|_| UnsafePayloadGossipClientError::RequestError("request channel closed".to_string()))
            .inspect_err(|err| error!(target: "gossip_client", ?payload, ?err, "failed to request to gossip payload."))
    }
}
