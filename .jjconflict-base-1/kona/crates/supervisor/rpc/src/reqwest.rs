//! RPC API implementation using `reqwest`

#[cfg(feature = "reqwest")]
use alloy_primitives::B256;
#[cfg(feature = "reqwest")]
use alloy_rpc_client::ReqwestClient;
#[cfg(feature = "reqwest")]
use derive_more::Constructor;
#[cfg(feature = "reqwest")]
use kona_interop::{ExecutingDescriptor, SafetyLevel};

/// Error types for supervisor RPC interactions
#[cfg(feature = "reqwest")]
#[derive(Debug, thiserror::Error)]
pub enum SupervisorClientError {
    /// RPC client error
    #[error("RPC client error: {0}")]
    Client(Box<dyn std::error::Error + Send + Sync>),
}

#[cfg(feature = "reqwest")]
impl SupervisorClientError {
    /// Creates a new client error
    pub fn client(err: impl std::error::Error + Send + Sync + 'static) -> Self {
        Self::Client(Box::new(err))
    }
}

/// Subset of `op-supervisor` API, used for validating interop events.
#[cfg(feature = "reqwest")]
pub trait CheckAccessListClient {
    /// Returns if the messages meet the minimum safety level.
    fn check_access_list(
        &self,
        inbox_entries: &[B256],
        min_safety: SafetyLevel,
        executing_descriptor: ExecutingDescriptor,
    ) -> impl std::future::Future<Output = Result<(), SupervisorClientError>> + Send;
}

/// A supervisor client.
#[cfg(feature = "reqwest")]
#[derive(Debug, Clone, Constructor)]
pub struct SupervisorClient {
    /// The inner RPC client.
    client: ReqwestClient,
}

#[cfg(feature = "reqwest")]
impl CheckAccessListClient for SupervisorClient {
    async fn check_access_list(
        &self,
        inbox_entries: &[B256],
        min_safety: SafetyLevel,
        executing_descriptor: ExecutingDescriptor,
    ) -> Result<(), SupervisorClientError> {
        self.client
            .request(
                "supervisor_checkAccessList",
                (inbox_entries, min_safety, executing_descriptor),
            )
            .await
            .map_err(SupervisorClientError::client)
    }
}
