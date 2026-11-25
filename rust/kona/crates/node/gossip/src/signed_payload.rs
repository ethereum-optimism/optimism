//! A payload with the signature to gossip it with.

use alloy_primitives::Signature;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;

/// A payload and the signature to gossip it with.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SignedPayload {
    /// The payload to gossip.
    pub payload: OpExecutionPayloadEnvelope,
    /// The signature over the payload hash.
    pub signature: Signature,
}
