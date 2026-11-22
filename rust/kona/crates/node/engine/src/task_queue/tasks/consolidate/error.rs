//! Contains error types for the [`crate::ConsolidateTask`].

use crate::{
    BuildTaskError, EngineTaskError, FinalityViolation, SealTaskError, SynchronizeTaskError,
    task_queue::tasks::{BuildAndImportError, task::EngineTaskErrorSeverity},
};
use thiserror::Error;

/// An error that occurs when running the [`crate::ConsolidateTask`].
#[derive(Debug, Error)]
pub enum ConsolidateTaskError {
    /// A delegated safe head would move the safe labels below finality.
    #[error("Delegated safe head breaks finality: {0}")]
    BreaksFinality(#[from] FinalityViolation),
    /// Failed to fetch the execution layer's block at the height of a delegated safe head.
    #[error("Failed to fetch the block at delegated safe head height {0}")]
    FailedToFetchDelegatedSafeBlock(u64),
    /// The unsafe L2 block is missing.
    #[error("Unsafe L2 block is missing {0}")]
    MissingUnsafeL2Block(u64),
    /// Failed to fetch the unsafe L2 block.
    #[error("Failed to fetch the unsafe L2 block")]
    FailedToFetchUnsafeL2Block,
    /// Starting the build of a block from the payload attributes failed.
    #[error(transparent)]
    BuildTaskFailed(#[from] BuildTaskError),
    /// Sealing or importing the block built from the payload attributes failed.
    #[error(transparent)]
    SealTaskFailed(#[from] SealTaskError),
    /// The consolidation forkchoice update call to the engine api failed.
    #[error(transparent)]
    ForkchoiceUpdateFailed(#[from] SynchronizeTaskError),
}

impl From<BuildAndImportError> for ConsolidateTaskError {
    fn from(err: BuildAndImportError) -> Self {
        match err {
            BuildAndImportError::Build(e) => Self::BuildTaskFailed(e.into()),
            BuildAndImportError::Seal(e) => Self::SealTaskFailed(e),
        }
    }
}

impl EngineTaskError for ConsolidateTaskError {
    fn severity(&self) -> EngineTaskErrorSeverity {
        match self {
            Self::BreaksFinality(_) => EngineTaskErrorSeverity::Skip,
            Self::MissingUnsafeL2Block(_) => EngineTaskErrorSeverity::Reset,
            Self::FailedToFetchDelegatedSafeBlock(_) | Self::FailedToFetchUnsafeL2Block => {
                EngineTaskErrorSeverity::Temporary
            }
            Self::BuildTaskFailed(inner) => inner.severity(),
            Self::SealTaskFailed(inner) => inner.severity(),
            Self::ForkchoiceUpdateFailed(inner) => inner.severity(),
        }
    }
}
