use alloy_primitives::Signature;
use async_trait::async_trait;
use derive_more::Constructor;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::fmt::Debug;
use thiserror::Error;
use tokio::sync::mpsc;

/// A payload scheduled for gossip, with or without a signature already over it.
///
/// The sequencer schedules unsigned payloads and the network actor signs them with its own block
/// signer at publish time. `opstack_publishBlockV1` mirrors op-node's `PublishBlock`
/// (`op-node/node/node.go`), which publishes the signature the caller supplied — so a payload can
/// also arrive here signed, and the network actor then publishes those exact bytes instead of
/// signing again.
#[derive(Debug, Clone)]
pub enum PayloadToPublish {
    /// Signed by the network actor's own block signer at publish time.
    Unsigned(OpExecutionPayloadEnvelope),
    /// Already signed by the caller; published with the given signature.
    Signed(OpExecutionPayloadEnvelope, Signature),
}

impl PayloadToPublish {
    /// The payload being published.
    pub const fn payload(&self) -> &OpExecutionPayloadEnvelope {
        match self {
            Self::Unsigned(payload) | Self::Signed(payload, _) => payload,
        }
    }
}

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
    request_tx: mpsc::Sender<PayloadToPublish>,
}

#[async_trait]
impl UnsafePayloadGossipClient for QueuedUnsafePayloadGossipClient {
    async fn schedule_execution_payload_gossip(
        &self,
        payload: OpExecutionPayloadEnvelope,
    ) -> Result<(), UnsafePayloadGossipClientError> {
        self.request_tx
            .send(PayloadToPublish::Unsigned(payload.clone()))
            .await
            .map_err(|_| UnsafePayloadGossipClientError::RequestError("request channel closed".to_string()))
            .inspect_err(|err| error!(target: "gossip_client", ?payload, ?err, "failed to request to gossip payload."))
    }
}
