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
    /// The unsafe L2 block is missing because the execution layer is still syncing toward the
    /// unsafe head.
    #[error("Unsafe L2 block {0} is missing while the execution layer syncs")]
    MissingUnsafeL2BlockDuringElSync(u64),
    /// Failed to fetch the unsafe L2 block.
    #[error("Failed to fetch the unsafe L2 block")]
    FailedToFetchUnsafeL2Block,
    /// The unchanged forkchoice update asking whether the execution layer still syncs toward the
    /// unsafe head, for a missing unsafe L2 block, failed.
    #[error("Forkchoice update for the missing unsafe L2 block failed: {0}")]
    MissingUnsafeL2BlockProbeFailed(#[source] SynchronizeTaskError),
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
            // Only an execution layer that answers but still syncs makes the task wait on a fixed
            // interval. A failed RPC is retried on the growing failure backoff, so an outage is
            // not polled at the wait's cadence; the tasks that outrank this one would hit the same
            // failing execution layer anyway.
            Self::FailedToFetchDelegatedSafeBlock(_) | Self::FailedToFetchUnsafeL2Block => {
                EngineTaskErrorSeverity::Temporary
            }
            Self::MissingUnsafeL2BlockDuringElSync(_) => EngineTaskErrorSeverity::Wait,
            Self::BuildTaskFailed(inner) => inner.severity(),
            Self::SealTaskFailed(inner) => inner.severity(),
            Self::ForkchoiceUpdateFailed(inner) | Self::MissingUnsafeL2BlockProbeFailed(inner) => {
                inner.severity()
            }
        }
    }
}
