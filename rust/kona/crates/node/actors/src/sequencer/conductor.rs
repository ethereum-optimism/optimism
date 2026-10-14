use alloy_rpc_client::{ClientBuilder, RpcClient};
use alloy_transport::{RpcError, TransportErrorKind};
use async_trait::async_trait;
use kona_sources::RpcClientLayer;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{fmt::Debug, future::IntoFuture, time::Duration};
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

    /// Returns whether the conductor leads the cluster.
    async fn leader(&self) -> Result<bool, ConductorError>;
}

/// A client for communicating with the conductor service via RPC
#[derive(Debug, Clone)]
pub struct ConductorClient {
    /// The inner RPC provider
    rpc: RpcClient,
    /// How long a request may take before it fails.
    timeout: Duration,
}

#[async_trait]
impl Conductor for ConductorClient {
    /// Commit an unsafe payload to the conductor.
    async fn commit_unsafe_payload(
        &self,
        payload: &OpExecutionPayloadEnvelope,
    ) -> Result<(), ConductorError> {
        self.timed(self.rpc.request("conductor_commitUnsafePayload", [payload])).await
    }

    /// Check if the node is a leader of the conductor.
    async fn leader(&self) -> Result<bool, ConductorError> {
        self.timed(self.rpc.request("conductor_leader", ())).await
    }
}

impl ConductorClient {
    /// Creates a new conductor client using HTTP transport. Its requests are recorded as the
    /// client `conductor`.
    ///
    /// A request that takes longer than `timeout` fails, so that an unresponsive conductor cannot
    /// hold up the sequencer.
    pub fn new_http(url: Url, timeout: Duration) -> Self {
        let rpc = ClientBuilder::default().layer(RpcClientLayer::new("conductor")).http(url);
        Self { rpc, timeout }
    }

    /// Check if the conductor is active.
    pub async fn conductor_active(&self) -> Result<bool, ConductorError> {
        self.timed(self.rpc.request("conductor_active", ())).await
    }

    /// Awaits a request, failing it once it takes longer than the timeout.
    async fn timed<T>(
        &self,
        request: impl IntoFuture<Output = Result<T, RpcError<TransportErrorKind>>>,
    ) -> Result<T, ConductorError> {
        tokio::time::timeout(self.timeout, request.into_future())
            .await
            .map_err(|_| ConductorError::Timeout(self.timeout))?
            .map_err(Into::into)
    }
}

/// Error type for conductor operations
#[derive(Debug, thiserror::Error)]
pub enum ConductorError {
    /// An error occurred while making an RPC call to the conductor.
    #[error("RPC error: {0}")]
    Rpc(#[from] RpcError<TransportErrorKind>),
    /// The conductor did not answer in time.
    #[error("the conductor did not answer within {0:?}")]
    Timeout(Duration),
}

#[cfg(test)]
mod tests {
    use super::*;
    use tokio::net::TcpListener;

    /// A conductor that accepts connections but never answers fails the request after the timeout,
    /// rather than holding up the sequencer.
    #[tokio::test]
    async fn an_unresponsive_conductor_times_out() {
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let url = format!("http://{}", listener.local_addr().unwrap()).parse().unwrap();
        // Hold the accepted connection open without answering.
        let server = tokio::spawn(async move {
            let (_connection, _) = listener.accept().await.unwrap();
            std::future::pending::<()>().await;
        });
        let conductor = ConductorClient::new_http(url, Duration::from_millis(100));

        assert!(matches!(conductor.leader().await, Err(ConductorError::Timeout(_))));
        server.abort();
    }
}
