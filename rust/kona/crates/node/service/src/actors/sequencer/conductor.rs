use alloy_rpc_client::ReqwestClient;
use alloy_transport::{RpcError, TransportErrorKind};
use async_trait::async_trait;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::fmt::Debug;
use url::Url;

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

    /// Override the leader of the conductor.
    async fn override_leader(&self) -> Result<(), ConductorError>;
}

/// A client for communicating with the conductor service via RPC
#[derive(Debug, Clone)]
pub struct ConductorClient {
    /// The inner RPC provider
    rpc: ReqwestClient,
}

#[async_trait]
impl Conductor for ConductorClient {
    /// Commit an unsafe payload to the conductor.
    async fn commit_unsafe_payload(
        &self,
        payload: &OpExecutionPayloadEnvelope,
    ) -> Result<(), ConductorError> {
        self.rpc.request("conductor_commitUnsafePayload", [payload]).await.map_err(Into::into)
    }

    /// Override the leader of the conductor.
    async fn override_leader(&self) -> Result<(), ConductorError> {
        self.rpc.request("conductor_overrideLeader", ()).await.map_err(Into::into)
    }
}

impl ConductorClient {
    /// Creates a new conductor client using HTTP transport
    pub fn new_http(url: Url) -> Self {
        let rpc = ReqwestClient::new_http(url);
        Self { rpc }
    }

    /// Check if the node is a leader of the conductor.
    pub async fn leader(&self) -> Result<bool, ConductorError> {
        self.rpc.request("conductor_leader", ()).await.map_err(Into::into)
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
