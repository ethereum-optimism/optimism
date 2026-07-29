//! Client for polling Derivation Delegate sync status.

use async_trait::async_trait;
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

/// Polls sync status from an external OP Stack CL node acting as the derivation delegate.
///
/// Abstracted as a trait so [`DelegateDerivationActor`](super::DelegateDerivationActor) can be
/// constructed with a mock in tests instead of a live HTTP client.
#[async_trait]
pub trait DerivationDelegateProvider: Send + Sync {
    /// Fetches the current sync status from the derivation delegate.
    async fn fetch_sync_status(&self) -> Result<SyncStatus, DerivationDelegateClientError>;
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
}

#[async_trait]
impl DerivationDelegateProvider for DerivationDelegateClient {
    /// Calls `optimism_syncStatus` RPC method.
    async fn fetch_sync_status(&self) -> Result<SyncStatus, DerivationDelegateClientError> {
        Ok(self.derivation_client.op_sync_status().await?)
    }
}
