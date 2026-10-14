//! Derivation delegate provider.

use alloy_rpc_client::{ClientBuilder, RpcClient};
use alloy_transport::TransportError;
use alloy_transport_http::{Http, reqwest};
use async_trait::async_trait;
use kona_protocol::SyncStatus;
use kona_sources::RpcClientLayer;
use std::time::Duration;
use thiserror::Error;
use url::Url;

/// Deadline for each request to the derivation delegate, including reading its response.
const REQUEST_TIMEOUT: Duration = Duration::from_secs(5);

/// Error type for Derivation Delegate client operations.
#[derive(Debug, Error)]
pub enum DerivationDelegateClientError {
    /// Failed to fetch sync status from Derivation Delegate.
    #[error("Failed to fetch sync status: {0}")]
    FetchFailed(String),

    /// RPC error from Derivation Delegate.
    #[error("RPC error: {0}")]
    RpcError(#[from] TransportError),

    /// Failed to create HTTP client.
    #[error("HTTP client build failed: {0}")]
    HttpClientBuild(#[source] reqwest::Error),
}

/// Polls sync status from an external OP Stack CL node acting as the derivation delegate.
///
/// Abstracted as a trait so the delegated derivation actor can be built with a mock in tests
/// instead of a live HTTP client.
#[async_trait]
pub trait DerivationDelegateProvider: Send + Sync {
    /// Fetches the current sync status from the derivation delegate.
    async fn fetch_sync_status(&self) -> Result<SyncStatus, DerivationDelegateClientError>;
}

/// Client for fetching sync status from an external OP Stack CL node. Its requests are recorded
/// as the client `derivation_delegate`.
#[derive(Debug, Clone)]
pub struct DerivationDelegateClient {
    rpc: RpcClient,
}

impl DerivationDelegateClient {
    /// Creates a client for the CL node at `url`. Each request times out after five seconds.
    pub fn new(url: Url) -> Result<Self, DerivationDelegateClientError> {
        let http = reqwest::Client::builder()
            .timeout(REQUEST_TIMEOUT)
            .build()
            .map_err(DerivationDelegateClientError::HttpClientBuild)?;
        let rpc = ClientBuilder::default()
            .layer(RpcClientLayer::new("derivation_delegate"))
            .transport(Http::with_client(http, url), false);
        Ok(Self { rpc })
    }
}

#[async_trait]
impl DerivationDelegateProvider for DerivationDelegateClient {
    /// Calls `optimism_syncStatus`.
    async fn fetch_sync_status(&self) -> Result<SyncStatus, DerivationDelegateClientError> {
        Ok(self.rpc.request_noparams("optimism_syncStatus").await?)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use httpmock::{Method::POST, MockServer};
    use kona_protocol::{BlockInfo, L2BlockInfo};
    use serde_json::json;

    #[tokio::test]
    async fn fetches_the_sync_status() {
        let server = MockServer::start_async().await;
        let block = BlockInfo { number: 7, ..Default::default() };
        let l2_block = L2BlockInfo { block_info: block, ..Default::default() };
        let status = SyncStatus {
            current_l1: block,
            current_l1_finalized: block,
            head_l1: block,
            safe_l1: block,
            finalized_l1: block,
            unsafe_l2: l2_block,
            safe_l2: l2_block,
            finalized_l2: l2_block,
            local_safe_l2: l2_block,
        };
        let mock = server
            .mock_async(|when, then| {
                when.method(POST).json_body_includes(r#"{"method":"optimism_syncStatus"}"#);
                then.status(200).json_body(json!({"jsonrpc": "2.0", "id": 0, "result": status}));
            })
            .await;

        let client = DerivationDelegateClient::new(server.base_url().parse().unwrap()).unwrap();

        assert_eq!(client.fetch_sync_status().await.unwrap(), status);
        mock.assert_async().await;
    }
}
