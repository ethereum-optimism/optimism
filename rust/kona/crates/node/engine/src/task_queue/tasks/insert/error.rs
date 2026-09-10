//! Contains the error types for the [`InsertTask`](crate::InsertTask).

use crate::{
    EngineTaskError, SynchronizeTaskError, task_queue::tasks::task::EngineTaskErrorSeverity,
};
use alloy_primitives::B256;
use alloy_rpc_types_engine::PayloadStatusEnum;
use alloy_transport::{RpcError, TransportErrorKind};
use kona_protocol::{FromBlockError, L2BlockInfo};
use op_alloy_rpc_types_engine::OpPayloadError;
use tokio::sync::mpsc;

/// An error that occurs when running the [`InsertTask`](crate::InsertTask).
#[derive(Debug, thiserror::Error)]
pub enum InsertTaskError {
    /// Error converting a payload into a block.
    #[error(transparent)]
    FromBlockError(#[from] OpPayloadError),
    /// A sequencer payload no longer extends the current unsafe head.
    #[error("Payload parent {parent} does not match current unsafe head {unsafe_head}")]
    StalePayload {
        /// The parent hash of the payload being inserted.
        parent: B256,
        /// The current unsafe head hash.
        unsafe_head: B256,
    },
    /// Failed to walk the current unsafe chain while classifying an older payload.
    #[error("Failed to walk unsafe chain ancestry: {0}")]
    AncestryLookupFailed(RpcError<TransportErrorKind>),
    /// A block needed to walk the current unsafe chain was unavailable.
    #[error("Unsafe chain block {hash} at height {number} is unavailable")]
    AncestorBlockNotFound {
        /// Expected block hash.
        hash: B256,
        /// Expected block number.
        number: u64,
    },
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
    /// Failed to send the insertion result to the waiting caller.
    #[error("Failed to send insertion result")]
    MpscSend(#[from] Box<mpsc::error::SendError<Result<L2BlockInfo, Self>>>),
}

impl EngineTaskError for InsertTaskError {
    fn severity(&self) -> EngineTaskErrorSeverity {
        match self {
            Self::FromBlockError(_) | Self::L2BlockInfoConstruction(_) => {
                EngineTaskErrorSeverity::Critical
            }
            Self::StalePayload { .. } |
            Self::AncestryLookupFailed(_) |
            Self::AncestorBlockNotFound { .. } |
            Self::InsertFailed(_) |
            Self::UnexpectedPayloadStatus(_) => EngineTaskErrorSeverity::Temporary,
            Self::ForkchoiceUpdateFailed(inner) => inner.severity(),
            Self::MpscSend(_) => EngineTaskErrorSeverity::Critical,
        }
    }
}
