//! HA conductor publication authorization.

use alloy_rpc_client::ReqwestClient;
use alloy_transport::{RpcError, TransportErrorKind};
use async_trait::async_trait;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use url::Url;

/// Publication gate used by a high-availability sequencer deployment.
#[cfg_attr(test, mockall::automock)]
#[async_trait]
pub trait Conductor: core::fmt::Debug + Send + Sync {
    /// Commits an exact unsafe payload before it may be published or canonicalized.
    async fn commit_unsafe_payload(
        &self,
        payload: &OpExecutionPayloadEnvelope,
    ) -> Result<(), ConductorError>;

    /// Overrides conductor leadership.
    async fn override_leader(&self) -> Result<(), ConductorError>;
}

/// HTTP conductor client.
#[derive(Debug, Clone)]
pub struct ConductorClient {
    rpc: ReqwestClient,
}

impl ConductorClient {
    /// Creates a conductor client using HTTP transport.
    pub fn new_http(url: Url) -> Self {
        Self { rpc: ReqwestClient::new_http(url) }
    }
}

#[async_trait]
impl Conductor for ConductorClient {
    async fn commit_unsafe_payload(
        &self,
        payload: &OpExecutionPayloadEnvelope,
    ) -> Result<(), ConductorError> {
        self.rpc.request("conductor_commitUnsafePayload", [payload]).await.map_err(Into::into)
    }

    async fn override_leader(&self) -> Result<(), ConductorError> {
        self.rpc.request("conductor_overrideLeader", ()).await.map_err(Into::into)
    }
}

/// Conductor transport failure.
#[derive(Debug, thiserror::Error)]
pub enum ConductorError {
    /// JSON-RPC request failed.
    #[error("conductor RPC failed: {0}")]
    Rpc(#[from] RpcError<TransportErrorKind>),
}
