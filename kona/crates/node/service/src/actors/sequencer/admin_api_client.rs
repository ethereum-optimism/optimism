//! The RPC server for the sequencer actor.
//! Mostly handles queries from the admin rpc.

use alloy_primitives::B256;
use async_trait::async_trait;
use derive_more::Constructor;
use kona_rpc::{SequencerAdminAPIClient, SequencerAdminAPIError};
use tokio::sync::{mpsc, oneshot};

/// Queued implementation of [`SequencerAdminAPIClient`] that handles requests by sending them to
/// a handler via the contained sender.
#[derive(Debug, Clone, Constructor)]
pub struct QueuedSequencerAdminAPIClient {
    /// Queue used to relay admin queries
    request_tx: mpsc::Sender<SequencerAdminQuery>,
}

/// The query types to the sequencer actor for the admin api.
#[derive(Debug)]
pub enum SequencerAdminQuery {
    /// A query to check if the sequencer is active.
    SequencerActive(oneshot::Sender<Result<bool, SequencerAdminAPIError>>),
    /// A query to start the sequencer.
    StartSequencer(oneshot::Sender<Result<(), SequencerAdminAPIError>>),
    /// A query to stop the sequencer.
    StopSequencer(oneshot::Sender<Result<B256, SequencerAdminAPIError>>),
    /// A query to check if the conductor is enabled.
    ConductorEnabled(oneshot::Sender<Result<bool, SequencerAdminAPIError>>),
    /// A query to check if the sequencer is in recovery mode.
    RecoveryMode(oneshot::Sender<Result<bool, SequencerAdminAPIError>>),
    /// A query to set the recovery mode.
    SetRecoveryMode(bool, oneshot::Sender<Result<(), SequencerAdminAPIError>>),
    /// A query to override the leader.
    OverrideLeader(oneshot::Sender<Result<(), SequencerAdminAPIError>>),
    /// A query to reset the derivation pipeline.
    ResetDerivationPipeline(oneshot::Sender<Result<(), SequencerAdminAPIError>>),
}

#[async_trait]
impl SequencerAdminAPIClient for QueuedSequencerAdminAPIClient {
    async fn is_sequencer_active(&self) -> Result<bool, SequencerAdminAPIError> {
        let (tx, rx) = oneshot::channel();

        self.request_tx.send(SequencerAdminQuery::SequencerActive(tx)).await.map_err(|_| {
            SequencerAdminAPIError::RequestError("request channel closed".to_string())
        })?;
        rx.await.map_err(|_| {
            SequencerAdminAPIError::ResponseError("response channel closed".to_string())
        })?
    }

    async fn is_conductor_enabled(&self) -> Result<bool, SequencerAdminAPIError> {
        let (tx, rx) = oneshot::channel();

        self.request_tx.send(SequencerAdminQuery::ConductorEnabled(tx)).await.map_err(|_| {
            SequencerAdminAPIError::RequestError("request channel closed".to_string())
        })?;
        rx.await.map_err(|_| {
            SequencerAdminAPIError::ResponseError("response channel closed".to_string())
        })?
    }

    async fn is_recovery_mode(&self) -> Result<bool, SequencerAdminAPIError> {
        let (tx, rx) = oneshot::channel();

        self.request_tx.send(SequencerAdminQuery::RecoveryMode(tx)).await.map_err(|_| {
            SequencerAdminAPIError::RequestError("request channel closed".to_string())
        })?;
        rx.await.map_err(|_| {
            SequencerAdminAPIError::ResponseError("response channel closed".to_string())
        })?
    }

    async fn start_sequencer(&self) -> Result<(), SequencerAdminAPIError> {
        let (tx, rx) = oneshot::channel();

        self.request_tx.send(SequencerAdminQuery::StartSequencer(tx)).await.map_err(|_| {
            SequencerAdminAPIError::RequestError("request channel closed".to_string())
        })?;
        rx.await.map_err(|_| {
            SequencerAdminAPIError::ResponseError("response channel closed".to_string())
        })?
    }

    async fn stop_sequencer(&self) -> Result<B256, SequencerAdminAPIError> {
        let (tx, rx) = oneshot::channel();

        self.request_tx.send(SequencerAdminQuery::StopSequencer(tx)).await.map_err(|_| {
            SequencerAdminAPIError::RequestError("request channel closed".to_string())
        })?;
        rx.await.map_err(|_| {
            SequencerAdminAPIError::ResponseError("response channel closed".to_string())
        })?
    }

    async fn set_recovery_mode(&self, mode: bool) -> Result<(), SequencerAdminAPIError> {
        let (tx, rx) = oneshot::channel();

        self.request_tx.send(SequencerAdminQuery::SetRecoveryMode(mode, tx)).await.map_err(
            |_| SequencerAdminAPIError::RequestError("request channel closed".to_string()),
        )?;
        rx.await.map_err(|_| {
            SequencerAdminAPIError::ResponseError("response channel closed".to_string())
        })?
    }

    async fn override_leader(&self) -> Result<(), SequencerAdminAPIError> {
        let (tx, rx) = oneshot::channel();

        self.request_tx.send(SequencerAdminQuery::OverrideLeader(tx)).await.map_err(|_| {
            SequencerAdminAPIError::RequestError("request channel closed".to_string())
        })?;
        rx.await.map_err(|_| {
            SequencerAdminAPIError::ResponseError("response channel closed".to_string())
        })?
    }

    async fn reset_derivation_pipeline(&self) -> Result<(), SequencerAdminAPIError> {
        let (tx, rx) = oneshot::channel();

        self.request_tx.send(SequencerAdminQuery::ResetDerivationPipeline(tx)).await.map_err(
            |_| SequencerAdminAPIError::RequestError("request channel closed".to_string()),
        )?;
        rx.await.map_err(|_| {
            SequencerAdminAPIError::ResponseError("response channel closed".to_string())
        })?
    }
}
