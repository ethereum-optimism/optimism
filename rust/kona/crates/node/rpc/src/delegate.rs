//! Derivation delegate RPC client.

use crate::RollupNodeApiClient;
use async_trait::async_trait;
use jsonrpsee::http_client::{HttpClient, HttpClientBuilder};
use kona_node_actors::{DerivationDelegateClientError, DerivationDelegateProvider};
use kona_protocol::SyncStatus;
use std::time::Duration;
use url::Url;

/// Default request timeout in milliseconds.
const DEFAULT_FOLLOW_TIMEOUT: u64 = 5000;

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
