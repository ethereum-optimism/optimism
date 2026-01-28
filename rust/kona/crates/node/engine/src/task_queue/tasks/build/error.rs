//! Contains error types for the [crate::SynchronizeTask].

use crate::{EngineTaskError, task_queue::tasks::task::EngineTaskErrorSeverity};
use alloy_rpc_types_engine::{PayloadId, PayloadStatusEnum};
use alloy_transport::{RpcError, TransportErrorKind};
use thiserror::Error;
use tokio::sync::mpsc;

/// An error that occurs during payload building within the engine.
///
/// This error type is specific to the block building process and represents failures
/// that can occur during the automatic forkchoice update phase of [`BuildTask`].
/// Unlike [`BuildTaskError`], which handles higher-level build orchestration errors,
/// `EngineBuildError` focuses on low-level engine API communication failures.
///
/// ## Error Categories
///
/// - **State Validation**: Errors related to inconsistent chain state
/// - **Engine Communication**: RPC failures during forkchoice updates
/// - **Payload Validation**: Invalid payload status responses from the execution layer
///
/// [`BuildTask`]: crate::BuildTask
#[derive(Debug, Error)]
pub enum EngineBuildError {
    /// The finalized head is ahead of the unsafe head.
    #[error("Finalized head is ahead of unsafe head")]
    FinalizedAheadOfUnsafe(u64, u64),
    /// The forkchoice update call to the engine api failed.
    #[error("Failed to build payload attributes in the engine. Forkchoice RPC error: {0}")]
    AttributesInsertionFailed(#[from] RpcError<TransportErrorKind>),
    /// The inserted payload is invalid.
    #[error("The inserted payload is invalid: {0}")]
    InvalidPayload(String),
    /// The inserted payload status is unexpected.
    #[error("The inserted payload status is unexpected: {0}")]
    UnexpectedPayloadStatus(PayloadStatusEnum),
    /// The payload ID is missing.
    #[error("The inserted payload ID is missing")]
    MissingPayloadId,
    /// The engine is syncing.
    #[error("The engine is syncing")]
    EngineSyncing,
}

/// An error that occurs when running the [crate::BuildTask].
#[derive(Debug, Error)]
pub enum BuildTaskError {
    /// An error occurred when building the payload attributes in the engine.
    #[error("An error occurred when building the payload attributes to the engine.")]
    EngineBuildError(EngineBuildError),
    /// Error sending the built payload envelope.
    #[error(transparent)]
    MpscSend(#[from] Box<mpsc::error::SendError<PayloadId>>),
}

impl EngineTaskError for BuildTaskError {
    fn severity(&self) -> EngineTaskErrorSeverity {
        match self {
            Self::EngineBuildError(EngineBuildError::FinalizedAheadOfUnsafe(_, _)) => {
                EngineTaskErrorSeverity::Critical
            }
            Self::EngineBuildError(EngineBuildError::AttributesInsertionFailed(_)) => {
                EngineTaskErrorSeverity::Temporary
            }
            Self::EngineBuildError(EngineBuildError::InvalidPayload(_)) => {
                EngineTaskErrorSeverity::Temporary
            }
            Self::EngineBuildError(EngineBuildError::UnexpectedPayloadStatus(_)) => {
                EngineTaskErrorSeverity::Temporary
            }
            Self::EngineBuildError(EngineBuildError::MissingPayloadId) => {
                EngineTaskErrorSeverity::Temporary
            }
            Self::EngineBuildError(EngineBuildError::EngineSyncing) => {
                EngineTaskErrorSeverity::Temporary
            }
            Self::MpscSend(_) => EngineTaskErrorSeverity::Critical,
        }
    }
}
