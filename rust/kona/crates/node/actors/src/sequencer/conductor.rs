use alloy_rpc_client::RpcClient;
use alloy_transport::{RpcError, TransportErrorKind};
use async_trait::async_trait;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::fmt::Debug;

/// Trait for interacting with the conductor service.
///
/// The conductor service is responsible for coordinating sequencer behavior
/// in a high-availability setup with leader election.
#[cfg_attr(test, mockall::automock)]
#[async_trait]
pub trait Conductor: Debug + Send + Sync {
    /// Commit an unsafe payload to the conductor.
    async fn commit_unsafe_payload(
        &self,
        payload: &OpExecutionPayloadEnvelope,
    ) -> Result<(), ConductorError>;

    /// Returns whether the conductor leads the cluster.
    async fn leader(&self) -> Result<bool, ConductorError>;
}

/// A client for communicating with the conductor service via RPC
#[derive(Debug, Clone)]
pub struct ConductorClient {
    /// The inner RPC provider
    rpc: RpcClient,
}

#[async_trait]
impl Conductor for ConductorClient {
    /// Commit an unsafe payload to the conductor.
    async fn commit_unsafe_payload(
        &self,
        payload: &OpExecutionPayloadEnvelope,
    ) -> Result<(), ConductorError> {
        Ok(self.rpc.request("conductor_commitUnsafePayload", [payload]).await?)
    }

    /// Check if the node is a leader of the conductor.
    async fn leader(&self) -> Result<bool, ConductorError> {
        Ok(self.rpc.request("conductor_leader", ()).await?)
    }
}

impl ConductorClient {
    /// Creates a conductor client that sends its requests through `rpc`, which carries their
    /// metrics and deadline, as [`kona_sources::conductor_rpc`] builds it.
    pub const fn new(rpc: RpcClient) -> Self {
        Self { rpc }
    }

    /// Check if the conductor is active.
    pub async fn conductor_active(&self) -> Result<bool, ConductorError> {
        Ok(self.rpc.request("conductor_active", ()).await?)
    }
}

/// Error type for conductor operations
#[derive(Debug, thiserror::Error)]
pub enum ConductorError {
    /// An error occurred while making an RPC call to the conductor.
    #[error("RPC error: {0}")]
    Rpc(#[from] RpcError<TransportErrorKind>),
}
