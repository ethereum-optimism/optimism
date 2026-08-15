//! Safe-chain service backed by a trusted derivation delegate.

use crate::{
    engine::{ENGINE_RETRY_DELAY, EngineClient, EngineError},
    safe_chain::{
        SafeChainControlError, SafeChainHandle, SafeChainServiceError, control::ResetRequest,
    },
    unsafe_chain::SequencerHandle,
};
use alloy_primitives::BlockHash;
use async_trait::async_trait;
use jsonrpsee::{
    core::ClientError,
    http_client::{HttpClient, HttpClientBuilder},
};
use kona_derive::ChainProvider;
use kona_protocol::SyncStatus;
use kona_rpc::RollupNodeApiClient;
use std::time::Duration;
use tokio::{sync::mpsc, time};
use tokio_util::sync::CancellationToken;
use url::Url;

const POLL_INTERVAL: Duration = Duration::from_secs(4);
const REQUEST_TIMEOUT: Duration = Duration::from_secs(5);

/// Source of delegated derivation status.
#[async_trait]
pub trait DerivationDelegateProvider: core::fmt::Debug + Send + Sync {
    /// Fetches the delegate's current rollup synchronization status.
    async fn fetch_sync_status(&self) -> Result<SyncStatus, DerivationDelegateError>;
}

/// HTTP derivation-delegate client.
#[derive(Debug, Clone)]
pub struct DerivationDelegateClient {
    client: HttpClient,
}

impl DerivationDelegateClient {
    /// Creates an HTTP delegate client.
    pub fn new(url: Url) -> Result<Self, DerivationDelegateError> {
        let client = HttpClientBuilder::default()
            .request_timeout(REQUEST_TIMEOUT)
            .build(url)
            .map_err(|error| DerivationDelegateError::Build(error.to_string()))?;
        Ok(Self { client })
    }
}

#[async_trait]
impl DerivationDelegateProvider for DerivationDelegateClient {
    async fn fetch_sync_status(&self) -> Result<SyncStatus, DerivationDelegateError> {
        self.client.op_sync_status().await.map_err(Into::into)
    }
}

/// Long-running delegated safe-chain service.
#[derive(Debug)]
pub struct DelegatedSafeChainService<Delegate, L1> {
    delegate: Delegate,
    l1: L1,
    engine: EngineClient,
    sequencer: Option<SequencerHandle>,
    reset_rx: mpsc::Receiver<ResetRequest>,
}

impl<Delegate, L1> DelegatedSafeChainService<Delegate, L1>
where
    Delegate: DerivationDelegateProvider + 'static,
    L1: ChainProvider + Send + 'static,
{
    /// Creates a delegated service and its reset/refresh handle.
    pub fn new(
        delegate: Delegate,
        l1: L1,
        engine: EngineClient,
        sequencer: Option<SequencerHandle>,
    ) -> (Self, SafeChainHandle) {
        let (handle, reset_rx) = SafeChainHandle::channel();
        (Self { delegate, l1, engine, sequencer, reset_rx }, handle)
    }

    /// Polls, validates, and applies delegated safe/finalized progress.
    pub async fn run(mut self, shutdown: CancellationToken) -> Result<(), SafeChainServiceError> {
        self.engine.wait_ready().await?;
        let mut poll = time::interval(POLL_INTERVAL);
        poll.set_missed_tick_behavior(time::MissedTickBehavior::Skip);

        loop {
            tokio::select! {
                biased;
                _ = shutdown.cancelled() => return Ok(()),
                request = self.reset_rx.recv() => {
                    let request = request.ok_or(SafeChainServiceError::ControlChannelClosed)?;
                    let result = self.poll_once().await.map_err(|error| {
                        SafeChainControlError::Reset(error.to_string())
                    });
                    let _ = request.response.send(result);
                }
                _ = poll.tick() => {
                    if let Err(error) = self.poll_once().await {
                        warn!(target: "safe_chain", ?error, "Delegated derivation poll yielded");
                    }
                }
            }
        }
    }

    async fn poll_once(&mut self) -> Result<(), SafeChainServiceError> {
        let status = self
            .delegate
            .fetch_sync_status()
            .await
            .map_err(|error| SafeChainServiceError::Critical(error.to_string()))?;
        if !self.validate(&status).await {
            return Ok(());
        }

        loop {
            match self.engine.update_safe(status.safe_l2).await {
                Ok(_) => break,
                Err(EngineError::Temporary(_) | EngineError::ResponseDropped) => {
                    tokio::time::sleep(ENGINE_RETRY_DELAY).await;
                }
                Err(EngineError::ResetRequired(_)) => {
                    if self.sequencer.as_ref().is_some_and(SequencerHandle::is_active) {
                        return Err(EngineError::RecoveryWhileSequencing.into());
                    }
                    self.engine.recover().await?;
                }
                Err(error) => return Err(error.into()),
            }
        }

        loop {
            match self.engine.update_finalized(status.finalized_l2).await {
                Ok(()) => return Ok(()),
                Err(EngineError::Temporary(_) | EngineError::ResponseDropped) => {
                    tokio::time::sleep(ENGINE_RETRY_DELAY).await;
                }
                Err(error) => return Err(error.into()),
            }
        }
    }

    async fn validate(&mut self, status: &SyncStatus) -> bool {
        let checks = [
            ("safe L2 origin", status.safe_l2.l1_origin.number, status.safe_l2.l1_origin.hash),
            (
                "finalized L2 origin",
                status.finalized_l2.l1_origin.number,
                status.finalized_l2.l1_origin.hash,
            ),
            ("current L1", status.current_l1.number, status.current_l1.hash),
        ];
        for (context, number, expected) in checks {
            match self.l1.block_info_by_number(number).await {
                Ok(block) if block.hash == expected => {}
                Ok(block) => {
                    warn!(target: "safe_chain", context, number, %expected, actual = %block.hash, "Rejected non-canonical delegated status");
                    return false;
                }
                Err(error) => {
                    warn!(target: "safe_chain", context, number, error = %error, "Could not validate delegated status");
                    return false;
                }
            }
        }
        true
    }
}

/// Derivation delegate client failure.
#[derive(Debug, thiserror::Error)]
pub enum DerivationDelegateError {
    /// HTTP client construction failed.
    #[error("failed to build derivation delegate client: {0}")]
    Build(String),
    /// Delegate RPC failed.
    #[error("derivation delegate RPC failed: {0}")]
    Rpc(#[from] ClientError),
    /// Canonical L1 hash did not match the delegate.
    #[error("delegated L1 mismatch at {number}: expected {expected}, got {actual}")]
    Canonicality {
        /// L1 height.
        number: u64,
        /// Delegate hash.
        expected: BlockHash,
        /// Local canonical hash.
        actual: BlockHash,
    },
}
