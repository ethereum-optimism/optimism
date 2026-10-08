//! Derivation delegate provider.

use async_trait::async_trait;
use jsonrpsee::core::ClientError;
use kona_protocol::SyncStatus;
use thiserror::Error;

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
