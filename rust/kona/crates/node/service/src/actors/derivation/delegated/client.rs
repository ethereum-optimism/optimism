//! Client for polling Derivation Delegate sync status.

use jsonrpsee::{
    core::ClientError,
    http_client::{HttpClient, HttpClientBuilder},
};
use kona_protocol::SyncStatus;
use kona_rpc::RollupNodeApiClient;
use std::time::Duration;
use thiserror::Error;
use url::Url;

/// Default timeout for follow client requests in milliseconds.
const DEFAULT_FOLLOW_TIMEOUT: u64 = 5000;

/// Error type for Derivation Delegate client operations.
#[derive(Debug, Error)]
pub enum DerivationDelegateClientError {
    /// Failed to fetch sync status from Derivation Delegate.
    #[error("Failed to fetch sync status: {0}")]
    FetchFailed(String),

    /// RPC error from Derivation Delegate.
    #[error("RPC error: {0}")]
    RpcError(#[from] ClientError),

    /// Failed to create HTTP client.
    #[error("HTTP client build failed: {0}")]
    HttpClientBuild(String),
}

/// Client for fetching sync status from an external OP Stack CL node.
#[derive(Debug, Clone)]
pub struct DerivationDelegateClient {
    /// The RPC client for the Derivation Delegate.
    derivation_client: HttpClient,
}

impl DerivationDelegateClient {
    /// Creates a new Derivation Delegate client.
    pub fn new(derivation_client_url: Url) -> Result<Self, DerivationDelegateClientError> {
        let derivation_client = HttpClientBuilder::default()
            .request_timeout(Duration::from_millis(DEFAULT_FOLLOW_TIMEOUT))
            .build(derivation_client_url)
            .map_err(|e| DerivationDelegateClientError::HttpClientBuild(e.to_string()))?;

        Ok(Self { derivation_client })
    }

    /// Fetches the current sync status from the Derivation Delegate.
    ///
    /// Calls `optimism_syncStatus` RPC method.
    pub async fn fetch_sync_status(&self) -> Result<SyncStatus, DerivationDelegateClientError> {
        Ok(self.derivation_client.op_sync_status().await?)
    }
}
