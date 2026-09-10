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
    time::Duration,
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

    /// Check if this node is the conductor leader.
    async fn leader(&self) -> Result<bool, ConductorError>;

    /// Locally override conductor leadership for disaster recovery.
    async fn override_leader(&self) -> Result<(), ConductorError>;
}

/// A client for communicating with the conductor service via RPC
#[derive(Debug, Clone)]
pub struct ConductorClient {
    /// The inner RPC provider.
    rpc: ReqwestClient,
    /// Local disaster-recovery override. When set, conductor interactions are bypassed.
    override_leader: Arc<AtomicBool>,
    /// Maximum duration of an RPC request to the conductor.
    rpc_timeout: Duration,
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
        tokio::time::timeout(
            self.rpc_timeout,
            self.rpc.request("conductor_commitUnsafePayload", [payload]),
        )
        .await
        .map_err(|_| ConductorError::Timeout(self.rpc_timeout))?
        .map_err(Into::into)
    }

    async fn leader(&self) -> Result<bool, ConductorError> {
        if self.override_leader.load(Ordering::Relaxed) {
            return Ok(true);
        }
        tokio::time::timeout(self.rpc_timeout, self.rpc.request("conductor_leader", ()))
            .await
            .map_err(|_| ConductorError::Timeout(self.rpc_timeout))?
            .map_err(Into::into)
    }

    /// Override conductor interactions locally, matching op-node's disaster-recovery behavior.
    async fn override_leader(&self) -> Result<(), ConductorError> {
        self.override_leader.store(true, Ordering::Relaxed);
        Ok(())
    }
}

impl ConductorClient {
    /// Creates a new conductor client using HTTP transport and the default timeout.
    pub fn new_http(url: Url) -> Self {
        Self::new_http_with_timeout(url, Duration::from_secs(1))
    }

    /// Creates a new conductor client using HTTP transport and the provided timeout.
    pub fn new_http_with_timeout(url: Url, rpc_timeout: Duration) -> Self {
        let rpc = ReqwestClient::new_http(url);
        Self { rpc, override_leader: Arc::new(AtomicBool::new(false)), rpc_timeout }
    }

    /// Check if the node is a leader of the conductor.
    pub async fn leader(&self) -> Result<bool, ConductorError> {
        Conductor::leader(self).await
    }

    /// Check if the conductor is active.
    pub async fn conductor_active(&self) -> Result<bool, ConductorError> {
        tokio::time::timeout(self.rpc_timeout, self.rpc.request("conductor_active", ()))
            .await
            .map_err(|_| ConductorError::Timeout(self.rpc_timeout))?
            .map_err(Into::into)
    }
}

/// Error type for conductor operations
#[derive(Debug, thiserror::Error)]
pub enum ConductorError {
    /// An error occurred while making an RPC call to the conductor.
    #[error("RPC error: {0}")]
    Rpc(#[from] RpcError<TransportErrorKind>),
    /// A conductor RPC request exceeded its configured timeout.
    #[error("conductor RPC request timed out after {0:?}")]
    Timeout(Duration),
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::{io::Read, net::TcpListener, thread};

    #[tokio::test]
    async fn conductor_request_honors_timeout() {
        let listener = TcpListener::bind("127.0.0.1:0").unwrap();
        let addr = listener.local_addr().unwrap();
        thread::spawn(move || {
            let (mut stream, _) = listener.accept().unwrap();
            let mut request = [0_u8; 1024];
            let _ = stream.read(&mut request);
            thread::park_timeout(Duration::from_secs(1));
        });

        let client = ConductorClient::new_http_with_timeout(
            Url::parse(&format!("http://{addr}")).unwrap(),
            Duration::from_millis(20),
        );
        let err = client.leader().await.unwrap_err();
        assert!(
            matches!(err, ConductorError::Timeout(duration) if duration == Duration::from_millis(20))
        );
    }
}
