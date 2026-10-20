use alloy_rpc_client::ReqwestClient;
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
    /// Whether this sequencer is the conductor's leader.
    async fn leader(&self) -> Result<bool, ConductorError>;

    /// Commit an unsafe payload to the conductor.
    async fn commit_unsafe_payload(
        &self,
        payload: &OpExecutionPayloadEnvelope,
    ) -> Result<(), ConductorError>;
}

/// A client for communicating with the conductor service via RPC
#[derive(Debug, Clone)]
pub struct ConductorClient {
    /// The inner RPC provider
    rpc: ReqwestClient,
}

#[async_trait]
impl Conductor for ConductorClient {
    async fn leader(&self) -> Result<bool, ConductorError> {
        self.rpc.request("conductor_leader", ()).await.map_err(Into::into)
    }

    /// Commit an unsafe payload to the conductor.
    async fn commit_unsafe_payload(
        &self,
        payload: &OpExecutionPayloadEnvelope,
    ) -> Result<(), ConductorError> {
        self.rpc.request("conductor_commitUnsafePayload", [payload]).await.map_err(Into::into)
    }
}

impl ConductorClient {
    /// Creates a conductor client with the node's configured transport deadline.
    pub const fn new(rpc: ReqwestClient) -> Self {
        Self { rpc }
    }

    /// Check if the conductor is active.
    pub async fn conductor_active(&self) -> Result<bool, ConductorError> {
        self.rpc.request("conductor_active", ()).await.map_err(Into::into)
    }
}

/// Error type for conductor operations
#[derive(Debug, thiserror::Error)]
pub enum ConductorError {
    /// An error occurred while making an RPC call to the conductor.
    #[error("RPC error: {0}")]
    Rpc(#[from] RpcError<TransportErrorKind>),
}
