use alloy_rpc_client::ReqwestClient;
use alloy_transport::{RpcError, TransportErrorKind};
use async_trait::async_trait;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{
    fmt::Debug,
    sync::{
        Arc,
        atomic::{AtomicBool, Ordering},
    },
};
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

    /// Locally override conductor leadership for disaster recovery.
    async fn override_leader(&self) -> Result<(), ConductorError>;
}

/// A client for communicating with the conductor service via RPC
#[derive(Debug, Clone)]
pub struct ConductorClient {
    /// The inner RPC provider
    rpc: ReqwestClient,
    /// Bypass conductor interactions after a local disaster-recovery override.
    override_leader: Arc<AtomicBool>,
}

#[async_trait]
impl Conductor for ConductorClient {
    /// Commit an unsafe payload to the conductor.
    async fn commit_unsafe_payload(
        &self,
        payload: &OpExecutionPayloadEnvelope,
    ) -> Result<(), ConductorError> {
        if self.override_leader.load(Ordering::Relaxed) {
            return Ok(());
        }
        self.rpc.request("conductor_commitUnsafePayload", [payload]).await.map_err(Into::into)
    }

    /// Override conductor interactions locally, matching op-node's recovery behavior.
    async fn override_leader(&self) -> Result<(), ConductorError> {
        self.override_leader.store(true, Ordering::Relaxed);
        Ok(())
    }
}

impl ConductorClient {
    /// Creates a new conductor client using HTTP transport
    pub fn new_http(url: Url) -> Self {
        let rpc = ReqwestClient::new_http(url);
        Self { rpc, override_leader: Arc::new(AtomicBool::new(false)) }
    }

    /// Check if the node is a leader of the conductor.
    pub async fn leader(&self) -> Result<bool, ConductorError> {
        if self.override_leader.load(Ordering::Relaxed) {
            return Ok(true);
        }
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
    use alloy_consensus::{Block, TxEnvelope};
    use alloy_rpc_types_engine::ExecutionPayloadV1;

    #[tokio::test]
    async fn local_override_bypasses_unavailable_conductor_for_all_clones() {
        let client = ConductorClient::new_http(Url::parse("http://127.0.0.1:0").unwrap());
        let clone = client.clone();

        client.override_leader().await.unwrap();
        let payload = OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(
            &Block::<TxEnvelope>::default(),
        ));

        for client in [client, clone] {
            assert!(client.leader().await.unwrap());
            client.commit_unsafe_payload(&payload).await.unwrap();
        }
    }
}
