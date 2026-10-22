use alloy_rpc_client::ReqwestClient;
use alloy_transport::{RpcError, TransportErrorKind};
use async_trait::async_trait;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{fmt::Debug, time::Duration};
use url::Url;

pub(super) const DEFAULT_CONDUCTOR_RPC_TIMEOUT: Duration = Duration::from_secs(1);

/// Trait for interacting with the conductor service.
///
/// The conductor service is responsible for coordinating sequencer behavior
/// in a high-availability setup with leader election.
#[cfg_attr(test, mockall::automock)]
#[async_trait]
pub trait Conductor: Debug + Send + Sync {
    /// Whether this node may sequence as the conductor's leader.
    async fn leader(&self) -> Result<bool, ConductorError>;

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
    /// Deadline for a conductor RPC, including response-body reads.
    rpc_timeout: Duration,
}

#[async_trait]
impl Conductor for ConductorClient {
    async fn leader(&self) -> Result<bool, ConductorError> {
        Self::leader(self).await
    }

    /// Commit an unsafe payload to the conductor.
    async fn commit_unsafe_payload(
        &self,
        payload: &OpExecutionPayloadEnvelope,
    ) -> Result<(), ConductorError> {
        tokio::time::timeout(
            self.rpc_timeout,
            self.rpc.request("conductor_commitUnsafePayload", [payload]),
        )
        .await?
        .map_err(Into::into)
    }

    /// Override the leader of the conductor.
    async fn override_leader(&self) -> Result<(), ConductorError> {
        tokio::time::timeout(self.rpc_timeout, self.rpc.request("conductor_overrideLeader", ()))
            .await?
            .map_err(Into::into)
    }
}

impl ConductorClient {
    /// Creates a new conductor client using HTTP transport
    pub fn new_http(url: Url) -> Self {
        let rpc = ReqwestClient::new_http(url);
        Self { rpc, rpc_timeout: DEFAULT_CONDUCTOR_RPC_TIMEOUT }
    }

    /// Sets the deadline applied to every conductor RPC.
    pub const fn with_timeout(mut self, timeout: Duration) -> Self {
        self.rpc_timeout = timeout;
        self
    }

    /// Check if the node is a leader of the conductor.
    pub async fn leader(&self) -> Result<bool, ConductorError> {
        tokio::time::timeout(self.rpc_timeout, self.rpc.request("conductor_leader", ()))
            .await?
            .map_err(Into::into)
    }

    /// Check if the conductor is active.
    pub async fn conductor_active(&self) -> Result<bool, ConductorError> {
        tokio::time::timeout(self.rpc_timeout, self.rpc.request("conductor_active", ()))
            .await?
            .map_err(Into::into)
    }
}

/// Error type for conductor operations
#[derive(Debug, thiserror::Error)]
pub enum ConductorError {
    /// A conductor RPC did not finish before its configured deadline.
    #[error("conductor RPC timed out: {0}")]
    Timeout(#[from] tokio::time::error::Elapsed),

    /// An error occurred while making an RPC call to the conductor.
    #[error("RPC error: {0}")]
    Rpc(#[from] RpcError<TransportErrorKind>),
}
