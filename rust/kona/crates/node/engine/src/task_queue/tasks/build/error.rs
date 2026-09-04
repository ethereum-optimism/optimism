//! Contains error types for the [`crate::SynchronizeTask`].

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
    /// The forkchoice state sent alongside the payload attributes was rejected by the engine as
    /// inconsistent, i.e. the safe or finalized block is not an ancestor of the parent block the
    /// attributes build on.
    #[error("Invalid forkchoice state")]
    InvalidForkchoiceState,
}

/// An error that occurs when running the [`crate::BuildTask`].
#[derive(Debug, Error)]
pub enum BuildTaskError {
    /// An error occurred when building the payload attributes in the engine.
    #[error(transparent)]
    EngineBuildError(#[from] EngineBuildError),
    /// Error sending the payload id.
    #[error(transparent)]
    MpscSend(#[from] Box<mpsc::error::SendError<Result<PayloadId, Self>>>),
    /// The unsafe head moved between the caller reading it and this build reaching the engine, so
    /// the attributes no longer extend the unsafe chain. Building them anyway would send the
    /// execution layer a forkchoice state whose safe block is not an ancestor of the head.
    ///
    /// The caller is expected to re-build on the new unsafe head. The task queue drops the job
    /// rather than retrying it, since the forkchoice update it would send is one the execution
    /// layer is guaranteed to reject.
    #[error("Unsafe head changed before the build started")]
    UnsafeHeadChangedSinceBuild,
}

impl EngineTaskError for BuildTaskError {
    fn severity(&self) -> EngineTaskErrorSeverity {
        match self {
            Self::EngineBuildError(EngineBuildError::InvalidForkchoiceState) => {
                EngineTaskErrorSeverity::Reset
            }
            Self::EngineBuildError(
                EngineBuildError::AttributesInsertionFailed(_) |
                EngineBuildError::InvalidPayload(_) |
                EngineBuildError::UnexpectedPayloadStatus(_) |
                EngineBuildError::MissingPayloadId |
                EngineBuildError::EngineSyncing,
            ) => EngineTaskErrorSeverity::Temporary,
            Self::EngineBuildError(EngineBuildError::FinalizedAheadOfUnsafe(_, _)) |
            Self::MpscSend(_) |
            Self::UnsafeHeadChangedSinceBuild => EngineTaskErrorSeverity::Critical,
        }
    }
}
