//! Contains the error types for the [`InsertTask`](crate::InsertTask).

use crate::{
    EngineTaskError, SynchronizeTaskError, task_queue::tasks::task::EngineTaskErrorSeverity,
};
use alloy_json_rpc::ErrorPayload;
use alloy_rpc_types_engine::PayloadStatusEnum;
use alloy_transport::{RpcError, TransportErrorKind};
use kona_protocol::FromBlockError;
use op_alloy_rpc_types_engine::OpPayloadError;

/// Engine API request too large error code.
const TOO_LARGE_REQUEST_ERROR: i64 = -38004;
/// Engine API unsupported fork error code.
const UNSUPPORTED_FORK_ERROR: i64 = -38005;

/// An error that occurs when running the [`InsertTask`](crate::InsertTask).
#[derive(Debug, thiserror::Error)]
pub enum InsertTaskError {
    /// Error converting a payload into a block.
    #[error(transparent)]
    FromBlockError(#[from] OpPayloadError),
    /// Failed to insert new payload.
    #[error("Failed to insert new payload: {0}")]
    InsertFailed(RpcError<TransportErrorKind>),
    /// Unexpected payload status
    #[error("Unexpected payload status: {0}")]
    UnexpectedPayloadStatus(PayloadStatusEnum),
    /// Error converting the payload + chain genesis into an L2 block info.
    #[error(transparent)]
    L2BlockInfoConstruction(#[from] FromBlockError),
    /// The forkchoice update call to consolidate the block into the engine state failed.
    #[error(transparent)]
    ForkchoiceUpdateFailed(#[from] SynchronizeTaskError),
}

impl InsertTaskError {
    /// Returns whether this error is terminal for an externally supplied unsafe payload.
    ///
    /// [`SealTask`](crate::SealTask) executes [`InsertTask`](crate::InsertTask) directly and
    /// retains the stricter inner severity for locally produced payloads.
    pub(crate) fn is_terminal_unsafe_payload_error(&self) -> bool {
        let Self::InsertFailed(error) = self else { return false };
        match error {
            RpcError::ErrorResp(response) => {
                response.code == ErrorPayload::<()>::invalid_request().code ||
                    response.code == ErrorPayload::<()>::method_not_found().code ||
                    response.code == ErrorPayload::<()>::invalid_params().code ||
                    matches!(response.code, TOO_LARGE_REQUEST_ERROR | UNSUPPORTED_FORK_ERROR)
            }
            RpcError::UnsupportedFeature(_) |
            RpcError::LocalUsageError(_) |
            RpcError::SerError(_) => true,
            RpcError::NullResp | RpcError::DeserError { .. } | RpcError::Transport(_) => false,
        }
    }
}

impl EngineTaskError for InsertTaskError {
    fn severity(&self) -> EngineTaskErrorSeverity {
        match self {
            Self::FromBlockError(_) | Self::L2BlockInfoConstruction(_) => {
                EngineTaskErrorSeverity::Critical
            }
            Self::InsertFailed(_) | Self::UnexpectedPayloadStatus(_) => {
                EngineTaskErrorSeverity::Temporary
            }
            Self::ForkchoiceUpdateFailed(inner) => inner.severity(),
        }
    }
}
