//! The sequencer's signer dependency.

use async_trait::async_trait;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::fmt::Debug;

/// Queues payloads for signing.
#[cfg_attr(test, mockall::automock(type Error = std::io::Error;))]
#[async_trait]
pub trait Signer: Debug + Send + Sync {
    /// An error queuing a payload.
    type Error: std::error::Error + Send + Sync + 'static;

    /// Queues a payload, waiting for capacity if needed.
    ///
    /// Success means the payload was queued, not that it was signed or gossiped.
    async fn send(&self, payload: OpExecutionPayloadEnvelope) -> Result<(), Self::Error>;

    /// Returns whether the signer currently has room for another payload.
    fn has_capacity(&self) -> bool;
}
