//! Contains error types for the [`crate::PromoteTask`].

use crate::{
    EngineTaskError, SynchronizeTaskError, task_queue::tasks::task::EngineTaskErrorSeverity,
};
use alloy_transport::{RpcError, TransportErrorKind};
use kona_protocol::FromBlockError;
use thiserror::Error;

/// An error that occurs when running the [`crate::PromoteTask`].
///
/// A failed lookup skips the task rather than retry it ahead of derivation: the verifier's head is
/// polled again, and the next poll carries a fresh task.
#[derive(Debug, Error)]
pub enum PromoteTaskError {
    /// The engine has no canonical block at a height at or below the local-safe head.
    #[error("The engine has no block {0} below the local-safe head")]
    BlockNotFound(u64),
    /// An error occurred while transforming the RPC block into [`L2BlockInfo`].
    ///
    /// [`L2BlockInfo`]: kona_protocol::L2BlockInfo
    #[error(transparent)]
    FromBlock(#[from] FromBlockError),
    /// The block lookup failed.
    #[error(transparent)]
    TransportError(#[from] RpcError<TransportErrorKind>),
    /// The forkchoice update moving the safe label failed.
    #[error(transparent)]
    ForkchoiceUpdateFailed(#[from] SynchronizeTaskError),
}

impl EngineTaskError for PromoteTaskError {
    fn severity(&self) -> EngineTaskErrorSeverity {
        match self {
            Self::BlockNotFound(_) | Self::FromBlock(_) | Self::TransportError(_) => {
                EngineTaskErrorSeverity::Skip
            }
            Self::ForkchoiceUpdateFailed(inner) => inner.severity(),
        }
    }
}
