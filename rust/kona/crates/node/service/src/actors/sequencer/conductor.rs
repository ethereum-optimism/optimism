use alloy_rpc_client::ReqwestClient;
use alloy_transport::{RpcError, TransportErrorKind};
use async_trait::async_trait;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{fmt::Debug, time::Duration};
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
    timeout: Duration,
}

#[async_trait]
impl Conductor for ConductorClient {
    /// Commit an unsafe payload to the conductor.
    async fn commit_unsafe_payload(
        &self,
        payload: &OpExecutionPayloadEnvelope,
    ) -> Result<(), ConductorError> {
        tokio::time::timeout(
            self.timeout,
            self.rpc.request("conductor_commitUnsafePayload", [payload]),
        )
        .await?
        .map_err(Into::into)
    }

    /// Override the leader of the conductor.
    async fn override_leader(&self) -> Result<(), ConductorError> {
        tokio::time::timeout(self.timeout, self.rpc.request("conductor_overrideLeader", ()))
            .await?
            .map_err(Into::into)
    }
}

impl ConductorClient {
    /// Creates a new conductor client using HTTP transport
    pub fn new_http(url: Url) -> Self {
        Self::new_http_with_timeout(url, Duration::from_secs(1))
    }

    /// Creates a client with a deadline for each complete RPC exchange.
    pub fn new_http_with_timeout(url: Url, timeout: Duration) -> Self {
        Self { rpc: ReqwestClient::new_http(url), timeout }
    }

    /// Check if the node is a leader of the conductor.
    pub async fn leader(&self) -> Result<bool, ConductorError> {
        tokio::time::timeout(self.timeout, self.rpc.request("conductor_leader", ()))
            .await?
            .map_err(Into::into)
    }

    /// Check if the conductor is active.
    pub async fn conductor_active(&self) -> Result<bool, ConductorError> {
        tokio::time::timeout(self.timeout, self.rpc.request("conductor_active", ()))
            .await?
            .map_err(Into::into)
    }
}

/// Error type for conductor operations
#[derive(Debug, thiserror::Error)]
pub enum ConductorError {
    /// The configured conductor deadline elapsed; acceptance is ambiguous.
    #[error("Conductor RPC timed out: {0}")]
    Timeout(#[from] tokio::time::error::Elapsed),
    /// An error occurred while making an RPC call to the conductor.
    #[error("RPC error: {0}")]
    Rpc(#[from] RpcError<TransportErrorKind>),
}

#[cfg(test)]
mod tests {
    use super::*;
    #[tokio::test]
    async fn leader_request_honors_the_configured_deadline() {
        let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
        let address = listener.local_addr().unwrap();
        let stalled = tokio::spawn(async move {
            let (_stream, _) = listener.accept().await.unwrap();
            std::future::pending::<()>().await;
        });
        let client = ConductorClient::new_http_with_timeout(
            format!("http://{address}").parse().unwrap(),
            Duration::from_millis(50),
        );
        let result = tokio::time::timeout(Duration::from_secs(2), client.leader()).await.unwrap();
        assert!(matches!(result, Err(ConductorError::Timeout(_))));
        stalled.abort();
    }
}
