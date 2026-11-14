//! Contains error types for the [`crate::FinalizeTask`].

use crate::{
    EngineTaskError, SynchronizeTaskError, task_queue::tasks::task::EngineTaskErrorSeverity,
};
use alloy_eips::BlockNumHash;
use alloy_transport::{RpcError, TransportErrorKind};
use kona_protocol::FromBlockError;
use thiserror::Error;

/// An error that occurs when running the [`crate::FinalizeTask`].
#[derive(Debug, Error)]
pub enum FinalizeTaskError {
    /// The target is below the finalized head, and finality never moves backward.
    #[error("Finalization target {target} is behind the finalized head {finalized}")]
    BehindFinalizedHead {
        /// The number of the finalized head.
        finalized: u64,
        /// The number of the finalization target.
        target: u64,
    },
    /// The block to finalize was not found.
    #[error("The block to finalize was not found: Number {0}")]
    BlockNotFound(u64),
    /// The block is not safe yet, and therefore cannot be finalized. A reset can lower the safe
    /// head below a target that was safe when it was requested, and a delegate can run ahead of
    /// the local safe head.
    #[error("Finalization target {target} is above the safe head {safe}")]
    BlockNotSafe {
        /// The number of the safe head.
        safe: u64,
        /// The number of the finalization target.
        target: u64,
    },
    /// The engine returned a block at a different height than the one requested.
    #[error("Requested block {requested} to finalize, but the engine returned block {returned}")]
    BlockNumberMismatch {
        /// The requested block number.
        requested: u64,
        /// The number of the block the engine returned.
        returned: u64,
    },
    /// The target is a different block at the finalized head's height.
    #[error("Finalization target {target:?} conflicts with the finalized head {finalized:?}")]
    ConflictsWithFinalizedHead {
        /// The finalized head.
        finalized: BlockNumHash,
        /// The finalization target.
        target: BlockNumHash,
    },
    /// The forkchoice update call to finalize the block failed.
    #[error(transparent)]
    ForkchoiceUpdateFailed(#[from] SynchronizeTaskError),
    /// An error occurred while transforming the RPC block into [`L2BlockInfo`].
    ///
    /// [`L2BlockInfo`]: kona_protocol::L2BlockInfo
    #[error(transparent)]
    FromBlock(#[from] FromBlockError),
    /// The pinned target is not the engine's canonical block at its height.
    #[error("Finalization target {target:?} is not canonical; the engine has {canonical:?}")]
    NotCanonical {
        /// The engine's canonical block at the target's height.
        canonical: BlockNumHash,
        /// The finalization target.
        target: BlockNumHash,
    },
    /// The engine has no block at the pinned target's height yet, for example while it catches up.
    #[error("The engine does not have finalization target {target:?} yet")]
    NotYetAvailable {
        /// The finalization target.
        target: BlockNumHash,
    },
    /// A temporary RPC failure.
    #[error(transparent)]
    TransportError(#[from] RpcError<TransportErrorKind>),
}

impl EngineTaskError for FinalizeTaskError {
    fn severity(&self) -> EngineTaskErrorSeverity {
        match self {
            Self::BehindFinalizedHead { .. } |
            Self::BlockNotSafe { .. } |
            Self::ConflictsWithFinalizedHead { .. } |
            Self::NotCanonical { .. } |
            Self::NotYetAvailable { .. } => EngineTaskErrorSeverity::Skip,
            Self::BlockNotFound(_) | Self::BlockNumberMismatch { .. } | Self::FromBlock(_) => {
                EngineTaskErrorSeverity::Critical
            }
            Self::ForkchoiceUpdateFailed(inner) => inner.severity(),
            Self::TransportError(_) => EngineTaskErrorSeverity::Temporary,
        }
    }
}
