//! Contains error types for the [`crate::ConsolidateTask`].

use crate::{
    BuildTaskError, EngineTaskError, SealTaskError, SynchronizeTaskError,
    task_queue::tasks::{BuildAndSealError, task::EngineTaskErrorSeverity},
};
use thiserror::Error;

/// An error that occurs when running the [`crate::ConsolidateTask`].
#[derive(Debug, Error)]
pub enum ConsolidateTaskError {
    /// The unsafe L2 block is missing.
    #[error("Unsafe L2 block is missing {0}")]
    MissingUnsafeL2Block(u64),
    /// Failed to fetch the unsafe L2 block.
    #[error("Failed to fetch the unsafe L2 block")]
    FailedToFetchUnsafeL2Block,
    /// The build task failed.
    #[error(transparent)]
    BuildTaskFailed(#[from] BuildTaskError),
    /// The seal task failed.
    #[error(transparent)]
    SealTaskFailed(#[from] SealTaskError),
    /// The consolidation forkchoice update call to the engine api failed.
    #[error(transparent)]
    ForkchoiceUpdateFailed(#[from] SynchronizeTaskError),
}

impl From<BuildAndSealError> for ConsolidateTaskError {
    fn from(err: BuildAndSealError) -> Self {
        match err {
            BuildAndSealError::Build(e) => Self::BuildTaskFailed(e),
            BuildAndSealError::Seal(e) => Self::SealTaskFailed(e),
        }
    }
}

impl EngineTaskError for ConsolidateTaskError {
    fn severity(&self) -> EngineTaskErrorSeverity {
        match self {
            Self::MissingUnsafeL2Block(_) => EngineTaskErrorSeverity::Reset,
            Self::FailedToFetchUnsafeL2Block => EngineTaskErrorSeverity::Temporary,
            Self::BuildTaskFailed(inner) => inner.severity(),
            Self::SealTaskFailed(inner) => inner.severity(),
            Self::ForkchoiceUpdateFailed(inner) => inner.severity(),
        }
    }
}
