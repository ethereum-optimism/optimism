//! Contains error types for the [`crate::DenyTask`].

use crate::{
    EngineTaskError, SynchronizeTaskError, task_queue::tasks::task::EngineTaskErrorSeverity,
};
use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use alloy_transport::{RpcError, TransportErrorKind};
use kona_protocol::FromBlockError;
use thiserror::Error;

/// An error that occurs when running the [`crate::DenyTask`], or its outcome.
///
/// A failed lookup skips the task rather than retry it ahead of derivation: the verifier's denied
/// blocks are polled again, and the next poll carries a fresh task.
#[derive(Debug, Error)]
pub enum DenyTaskError {
    /// The local-safe head moved back to `local_safe`, the parent of a denied block. Derivation
    /// restarts from it and replaces the denied block as it derives it again.
    #[error("Moved the local-safe head back to {local_safe:?}, below denied block {denied:?}")]
    Rewound {
        /// The denied block.
        denied: BlockNumHash,
        /// Its parent, the new local-safe head.
        local_safe: BlockNumHash,
    },
    /// A denied block is at or below the finalized head, so it is not replaced.
    #[error(
        "Refusing to replace denied block {denied:?} at or below the finalized head {finalized:?}"
    )]
    Finalized {
        /// The denied block.
        denied: BlockNumHash,
        /// The finalized head.
        finalized: BlockNumHash,
    },
    /// The engine does not have the parent of a denied block.
    #[error("The engine has no parent {0} of a denied block")]
    ParentNotFound(B256),
    /// A block lookup failed.
    #[error(transparent)]
    Lookup(#[from] RpcError<TransportErrorKind>),
    /// An error occurred while transforming the RPC block into [`L2BlockInfo`].
    ///
    /// [`L2BlockInfo`]: kona_protocol::L2BlockInfo
    #[error(transparent)]
    FromBlock(#[from] FromBlockError),
    /// The forkchoice update after moving the local-safe head failed.
    #[error(transparent)]
    ForkchoiceUpdateFailed(#[from] SynchronizeTaskError),
}

impl EngineTaskError for DenyTaskError {
    fn severity(&self) -> EngineTaskErrorSeverity {
        match self {
            Self::Rewound { .. } => EngineTaskErrorSeverity::Rederive,
            Self::Finalized { .. } |
            Self::ParentNotFound(_) |
            Self::Lookup(_) |
            Self::FromBlock(_) => EngineTaskErrorSeverity::Skip,
            Self::ForkchoiceUpdateFailed(inner) => inner.severity(),
        }
    }
}
