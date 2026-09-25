use alloy_rpc_client::ReqwestClient;
use alloy_transport::{RpcError, TransportErrorKind};
use async_trait::async_trait;
use backon::{ConstantBuilder, Retryable};
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

/// Number of conductor RPC retries after the initial attempt, matching op-node's two attempts.
const CONDUCTOR_RPC_MAX_RETRIES: usize = 1;
/// Delay between conductor RPC attempts, matching op-node.
const CONDUCTOR_RPC_RETRY_DELAY: Duration = Duration::from_millis(50);

fn conductor_rpc_backoff() -> ConstantBuilder {
    ConstantBuilder::default()
        .with_delay(CONDUCTOR_RPC_RETRY_DELAY)
        .with_max_times(CONDUCTOR_RPC_MAX_RETRIES)
}

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
            (|| self.rpc.request("conductor_commitUnsafePayload", [payload]))
                .retry(conductor_rpc_backoff()),
        )
        .await
        .map_err(|_| ConductorError::Timeout(self.rpc_timeout))?
        .map_err(Into::into)
    }

    async fn leader(&self) -> Result<bool, ConductorError> {
        if self.override_leader.load(Ordering::Relaxed) {
            return Ok(true);
        }
        tokio::time::timeout(
            self.rpc_timeout,
            (|| self.rpc.request("conductor_leader", ())).retry(conductor_rpc_backoff()),
        )
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
        tokio::time::timeout(
            self.rpc_timeout,
            (|| self.rpc.request("conductor_active", ())).retry(conductor_rpc_backoff()),
        )
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
    use jsonrpsee::{RpcModule, server::ServerBuilder, types::ErrorObjectOwned};
    use std::{
        io::Read,
        net::TcpListener,
        sync::{atomic::AtomicUsize, mpsc},
        thread,
    };

    #[tokio::test]
    async fn conductor_request_retries_once() {
        let attempts = Arc::new(AtomicUsize::new(0));
        let server = ServerBuilder::default().build("127.0.0.1:0").await.unwrap();
        let addr = server.local_addr().unwrap();
        let mut module = RpcModule::new(attempts.clone());
        module
            .register_async_method("conductor_leader", |_, attempts, _| async move {
                if attempts.fetch_add(1, Ordering::Relaxed) == 0 {
                    Err(ErrorObjectOwned::owned(-32000, "temporary failure", None::<()>))
                } else {
                    Ok(true)
                }
            })
            .unwrap();
        let handle = server.start(module);

        let client = ConductorClient::new_http_with_timeout(
            Url::parse(&format!("http://{addr}")).unwrap(),
            Duration::from_secs(1),
        );
        assert!(client.leader().await.unwrap());
        assert_eq!(attempts.load(Ordering::Relaxed), 2);

        handle.stop().unwrap();
    }

    #[tokio::test]
    async fn conductor_request_stops_after_two_attempts() {
        let attempts = Arc::new(AtomicUsize::new(0));
        let server = ServerBuilder::default().build("127.0.0.1:0").await.unwrap();
        let addr = server.local_addr().unwrap();
        let mut module = RpcModule::new(attempts.clone());
        module
            .register_async_method("conductor_leader", |_, attempts, _| async move {
                attempts.fetch_add(1, Ordering::Relaxed);
                Err::<bool, _>(ErrorObjectOwned::owned(-32000, "persistent failure", None::<()>))
            })
            .unwrap();
        let handle = server.start(module);

        let client = ConductorClient::new_http_with_timeout(
            Url::parse(&format!("http://{addr}")).unwrap(),
            Duration::from_secs(1),
        );
        assert!(client.leader().await.is_err());
        assert_eq!(attempts.load(Ordering::Relaxed), 2);

        handle.stop().unwrap();
    }

    #[tokio::test]
    async fn conductor_request_honors_timeout() {
        let listener = TcpListener::bind("127.0.0.1:0").unwrap();
        let addr = listener.local_addr().unwrap();
        let (release_tx, release_rx) = mpsc::channel::<()>();
        let server = thread::spawn(move || {
            let (mut stream, _) = listener.accept().unwrap();
            let mut request = [0_u8; 1024];
            let _ = stream.read(&mut request);
            let _ = release_rx.recv();
        });

        let client = ConductorClient::new_http_with_timeout(
            Url::parse(&format!("http://{addr}")).unwrap(),
            Duration::from_millis(20),
        );
        let err = client.leader().await.unwrap_err();
        assert!(
            matches!(err, ConductorError::Timeout(duration) if duration == Duration::from_millis(20))
        );
        drop(release_tx);
        server.join().unwrap();
    }
}
