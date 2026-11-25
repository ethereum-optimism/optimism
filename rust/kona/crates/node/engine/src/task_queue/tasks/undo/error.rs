//! Contains error types for the [`crate::UndoTask`].

use crate::{
    EngineTaskError, SynchronizeTaskError, task_queue::tasks::task::EngineTaskErrorSeverity,
};
use alloy_eips::BlockNumHash;
use thiserror::Error;

/// An error that occurs when running the [`crate::UndoTask`], or its outcome.
#[derive(Debug, Error)]
pub enum UndoTaskError {
    /// The local-safe head and the safe label moved back to the finalized head `local_safe`.
    /// Derivation restarts from it.
    #[error(
        "Moved the local-safe head and the safe label back to the finalized head {local_safe:?}"
    )]
    Undone {
        /// The finalized head, the new local-safe head.
        local_safe: BlockNumHash,
    },
    /// The engine has no finalized head yet, as before its first reset.
    #[error("The engine has no finalized head yet")]
    NoFinalizedHead,
    /// The forkchoice update moving the labels back failed.
    #[error(transparent)]
    ForkchoiceUpdateFailed(#[from] SynchronizeTaskError),
}

impl EngineTaskError for UndoTaskError {
    fn severity(&self) -> EngineTaskErrorSeverity {
        match self {
            Self::Undone { .. } => EngineTaskErrorSeverity::Rederive,
            Self::NoFinalizedHead => EngineTaskErrorSeverity::Temporary,
            Self::ForkchoiceUpdateFailed(inner) => inner.severity(),
        }
    }
}
