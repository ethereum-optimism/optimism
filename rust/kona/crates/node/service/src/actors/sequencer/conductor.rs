use super::config::DEFAULT_CONDUCTOR_RPC_TIMEOUT;
use alloy_rpc_client::ReqwestClient;
use alloy_transport::{RpcError, TransportErrorKind};
use async_trait::async_trait;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{fmt::Debug, future::Future, time::Duration};
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

    /// Override the leader of the conductor.
    async fn override_leader(&self) -> Result<(), ConductorError>;
}

/// A client for communicating with the conductor service via RPC
#[derive(Debug, Clone)]
pub struct ConductorClient {
    /// The inner RPC provider
    rpc: ReqwestClient,
    /// The timeout applied to each RPC request.
    rpc_timeout: Duration,
}

#[async_trait]
impl Conductor for ConductorClient {
    /// Commit an unsafe payload to the conductor.
    async fn commit_unsafe_payload(
        &self,
        payload: &OpExecutionPayloadEnvelope,
    ) -> Result<(), ConductorError> {
        self.execute_with_timeout(self.rpc.request("conductor_commitUnsafePayload", [payload]))
            .await
    }

    /// Override the leader of the conductor.
    async fn override_leader(&self) -> Result<(), ConductorError> {
        self.execute_with_timeout(self.rpc.request("conductor_overrideLeader", ())).await
    }
}

impl ConductorClient {
    /// Creates a new conductor client using HTTP transport
    pub fn new_http(url: Url) -> Self {
        Self::new_http_with_timeout(url, DEFAULT_CONDUCTOR_RPC_TIMEOUT)
    }

    /// Creates a new conductor client using HTTP transport with the given request timeout.
    pub fn new_http_with_timeout(url: Url, rpc_timeout: Duration) -> Self {
        let rpc = ReqwestClient::new_http(url);
        Self { rpc, rpc_timeout }
    }

    /// Check if the node is a leader of the conductor.
    pub async fn leader(&self) -> Result<bool, ConductorError> {
        self.execute_with_timeout(self.rpc.request("conductor_leader", ())).await
    }

    /// Check if the conductor is active.
    pub async fn conductor_active(&self) -> Result<bool, ConductorError> {
        self.execute_with_timeout(self.rpc.request("conductor_active", ())).await
    }

    async fn execute_with_timeout<T>(
        &self,
        request: impl Future<Output = Result<T, RpcError<TransportErrorKind>>>,
    ) -> Result<T, ConductorError> {
        tokio::time::timeout(self.rpc_timeout, request)
            .await
            .map_err(|_| ConductorError::Timeout(self.rpc_timeout))?
            .map_err(Into::into)
    }
}

/// Error type for conductor operations
#[derive(Debug, thiserror::Error)]
pub enum ConductorError {
    /// A conductor RPC request exceeded its configured timeout.
    #[error("RPC request timed out after {0:?}")]
    Timeout(Duration),
    /// An error occurred while making an RPC call to the conductor.
    #[error("RPC error: {0}")]
    Rpc(#[from] RpcError<TransportErrorKind>),
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_consensus::Block;
    use alloy_rpc_types_engine::ExecutionPayloadV1;
    use futures::poll;
    use jsonrpsee::{RpcModule, server::ServerBuilder, types::ErrorObjectOwned};
    use op_alloy_consensus::OpTxEnvelope;
    use rstest::rstest;
    use std::{future::pending, task::Poll};
    use tokio::time::{Instant, sleep_until};

    async fn delayed_response<T>(response: T) -> Result<T, ErrorObjectOwned> {
        tokio::time::sleep(Duration::from_secs(60)).await;
        Ok(response)
    }

    #[tokio::test]
    async fn conductor_requests_use_configured_timeout() {
        let server = ServerBuilder::default().build("127.0.0.1:0").await.unwrap();
        let addr = server.local_addr().unwrap();
        let mut module = RpcModule::new(());
        module
            .register_async_method("conductor_commitUnsafePayload", |_, _, _| delayed_response(()))
            .unwrap();
        module
            .register_async_method("conductor_overrideLeader", |_, _, _| delayed_response(()))
            .unwrap();
        module.register_async_method("conductor_leader", |_, _, _| delayed_response(true)).unwrap();
        module.register_async_method("conductor_active", |_, _, _| delayed_response(true)).unwrap();
        let _handle = server.start(module);

        let timeout = Duration::from_millis(10);
        let client = ConductorClient::new_http_with_timeout(
            Url::parse(&format!("http://{addr}")).unwrap(),
            timeout,
        );
        let payload =
            OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(&Block::<
                OpTxEnvelope,
            >::default(
            )));

        let (commit, override_leader, leader, active) = tokio::join!(
            client.commit_unsafe_payload(&payload),
            client.override_leader(),
            client.leader(),
            client.conductor_active(),
        );

        for result in [commit, override_leader] {
            assert!(matches!(result, Err(ConductorError::Timeout(actual)) if actual == timeout));
        }
        for result in [leader, active] {
            assert!(matches!(result, Err(ConductorError::Timeout(actual)) if actual == timeout));
        }
    }

    #[rstest]
    #[case::below_default(Duration::from_millis(10))]
    #[case::above_default(Duration::from_secs(2))]
    #[tokio::test(start_paused = true)]
    async fn request_expires_at_configured_deadline(#[case] timeout: Duration) {
        let client = ConductorClient::new_http_with_timeout(
            Url::parse("http://127.0.0.1:8545").unwrap(),
            timeout,
        );
        // A never-ready request can only complete through the client's timeout.
        let request =
            client.execute_with_timeout(pending::<Result<(), RpcError<TransportErrorKind>>>());
        tokio::pin!(request);

        let deadline = Instant::now() + timeout;
        // Arm the timeout without awaiting it, which would auto-advance the paused clock.
        assert!(poll!(&mut request).is_pending());

        sleep_until(deadline - Duration::from_millis(1)).await;
        assert!(poll!(&mut request).is_pending(), "request timed out before its deadline");

        sleep_until(deadline).await;
        // Poll once so an incorrectly longer timeout cannot advance time and pass the test.
        assert!(matches!(
            poll!(&mut request),
            Poll::Ready(Err(ConductorError::Timeout(actual))) if actual == timeout
        ));
    }
}
