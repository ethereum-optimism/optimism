//! Error types for inserting a payload.

use crate::{
    EngineTaskError, SynchronizeTaskError, task_queue::tasks::task::EngineTaskErrorSeverity,
};
use alloy_rpc_types_engine::PayloadStatusEnum;
use alloy_transport::{RpcError, TransportErrorKind};
use kona_protocol::FromBlockError;
use op_alloy_rpc_types_engine::OpPayloadError;

/// An error inserting a payload, from [`Engine::insert_unsafe`] or while sealing a block.
///
/// [`Engine::insert_unsafe`]: crate::Engine::insert_unsafe
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
