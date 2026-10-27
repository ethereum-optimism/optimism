use alloy_rpc_client::ReqwestClient;
use alloy_transport::{RpcError, TransportErrorKind};
use alloy_transport_http::reqwest;
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
        Self::leader(self).await
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
    /// Creates an HTTP client with a deadline covering each request and its response body.
    pub fn new_http(url: Url, timeout: Duration) -> Result<Self, reqwest::Error> {
        let http = reqwest::Client::builder().timeout(timeout).build()?;
        let rpc = ReqwestClient::builder().http_with_client(http, url);
        Ok(Self { rpc })
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

#[cfg(test)]
mod tests {
    use super::*;
    use tokio::{
        io::{AsyncReadExt, AsyncWriteExt},
        net::TcpListener,
        sync::oneshot,
        time::timeout,
    };

    #[tokio::test]
    async fn deadline_covers_stalled_response_body() {
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let url = format!("http://{}", listener.local_addr().unwrap()).parse().unwrap();
        let (headers_sent, headers_received) = oneshot::channel();
        let (release, released) = oneshot::channel();
        let server = tokio::spawn(async move {
            let (mut socket, _) = listener.accept().await.unwrap();
            let mut request = Vec::new();
            while !request.windows(4).any(|window| window == b"\r\n\r\n") {
                assert_ne!(socket.read_buf(&mut request).await.unwrap(), 0);
            }
            socket.write_all(b"HTTP/1.1 200 OK\r\nContent-Length: 100\r\n\r\n{").await.unwrap();
            headers_sent.send(()).unwrap();
            released.await.unwrap();
        });

        let client = ConductorClient::new_http(url, Duration::from_millis(200)).unwrap();
        let ConductorError::Rpc(error) =
            timeout(Duration::from_secs(5), client.leader()).await.unwrap().unwrap_err();
        headers_received.await.unwrap();
        let error = error.as_transport_err().unwrap().as_custom().unwrap();
        assert!(error.downcast_ref::<reqwest::Error>().unwrap().is_timeout());

        release.send(()).unwrap();
        server.await.unwrap();
    }
}
