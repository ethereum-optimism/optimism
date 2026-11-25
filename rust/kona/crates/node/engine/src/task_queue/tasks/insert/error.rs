//! Contains the error types for the [`InsertTask`](crate::InsertTask).

use crate::{
    EngineTaskError, SynchronizeTaskError, task_queue::tasks::task::EngineTaskErrorSeverity,
};
use alloy_rpc_types_engine::PayloadStatusEnum;
use alloy_transport::{RpcError, TransportErrorKind};
use kona_protocol::FromBlockError;
use op_alloy_rpc_types_engine::OpPayloadError;

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
    /// An unsafe payload at or below the local-safe head is not imported.
    #[error("Unsafe payload {number} is at or below the local-safe head {local_safe}")]
    AtOrBelowLocalSafe {
        /// The payload's block number.
        number: u64,
        /// The local-safe head's block number.
        local_safe: u64,
    },
    /// An unsafe payload is not imported while a cross-chain verifier denied a block above the
    /// finalized head.
    #[error(
        "Unsafe payloads are held off until finality passes the denied block {highest_denied}, finalized is {finalized}"
    )]
    DeniedChainRecovery {
        /// The height of the highest denied block.
        highest_denied: u64,
        /// The height of the finalized head.
        finalized: u64,
    },
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
            Self::AtOrBelowLocalSafe { .. } | Self::DeniedChainRecovery { .. } => {
                EngineTaskErrorSeverity::Skip
            }
        }
    }
}
