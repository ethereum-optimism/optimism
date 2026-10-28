//! Contains error types for the [`crate::BuildTask`].

use crate::{EngineTaskError, task_queue::tasks::task::EngineTaskErrorSeverity};
use alloy_rpc_types_engine::{
    INVALID_FORK_CHOICE_STATE_ERROR, INVALID_PAYLOAD_ATTRIBUTES_ERROR, PayloadStatusEnum,
};
use alloy_transport::{RpcError, TransportErrorKind};
use std::sync::Arc;
use thiserror::Error;

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
#[derive(Debug, Clone, Error)]
pub enum EngineBuildError {
    /// The finalized head is ahead of the unsafe head.
    #[error("Finalized head is ahead of unsafe head")]
    FinalizedAheadOfUnsafe(u64, u64),
    /// The forkchoice update call to the engine api failed.
    #[error("Failed to build payload attributes in the engine. Forkchoice RPC error: {0}")]
    AttributesInsertionFailed(#[source] Arc<RpcError<TransportErrorKind>>),
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

impl EngineBuildError {
    /// Whether forkchoiceUpdated rejected the attributes with -38003.
    ///
    /// A payload-status INVALID rejects the forkchoice head, before attributes are processed.
    /// Replacing the attributes cannot repair that head; it requires an engine reset.
    pub fn is_invalid_attributes(&self) -> bool {
        match self {
            Self::AttributesInsertionFailed(err) => err
                .as_error_resp()
                .is_some_and(|resp| resp.code == i64::from(INVALID_PAYLOAD_ATTRIBUTES_ERROR)),
            _ => false,
        }
    }
}

/// An error that occurs when running the [`crate::BuildTask`].
#[derive(Debug, Clone, Error)]
pub enum BuildTaskError {
    /// An error occurred when building the payload attributes in the engine.
    #[error(transparent)]
    EngineBuildError(#[from] EngineBuildError),
    /// The sequencer build no longer extends the current unsafe head.
    #[error("Sequencer build parent is stale")]
    StaleBuild,
}

impl BuildTaskError {
    /// Whether forkchoiceUpdated rejected the attributes with -38003.
    pub fn is_invalid_attributes(&self) -> bool {
        match self {
            Self::EngineBuildError(err) => err.is_invalid_attributes(),
            Self::StaleBuild => false,
        }
    }
}

impl EngineTaskError for BuildTaskError {
    fn severity(&self) -> EngineTaskErrorSeverity {
        match self {
            Self::EngineBuildError(EngineBuildError::AttributesInsertionFailed(err))
                if err.as_error_resp().is_some_and(|resp| {
                    resp.code == i64::from(INVALID_FORK_CHOICE_STATE_ERROR)
                }) =>
            {
                EngineTaskErrorSeverity::Reset
            }
            Self::EngineBuildError(EngineBuildError::FinalizedAheadOfUnsafe(_, _)) => {
                EngineTaskErrorSeverity::Critical
            }
            // Retrying the same invalid head cannot recover. Reconcile with the EL's chain.
            Self::EngineBuildError(EngineBuildError::InvalidPayload(_)) => {
                EngineTaskErrorSeverity::Reset
            }
            Self::EngineBuildError(
                EngineBuildError::AttributesInsertionFailed(_) |
                EngineBuildError::UnexpectedPayloadStatus(_) |
                EngineBuildError::MissingPayloadId |
                EngineBuildError::EngineSyncing,
            ) |
            Self::StaleBuild => EngineTaskErrorSeverity::Temporary,
        }
    }
}
