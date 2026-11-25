//! Contains error types for the [`crate::ConsolidateTask`].

use crate::{
    BuildTaskError, EngineTaskError, FinalityViolation, ReplaceDeniedError, SealTaskError,
    SynchronizeTaskError,
    task_queue::tasks::{BuildAndImportError, task::EngineTaskErrorSeverity},
};
use alloy_eips::BlockNumHash;
use thiserror::Error;

/// An error that occurs when running the [`crate::ConsolidateTask`].
#[derive(Debug, Error)]
pub enum ConsolidateTaskError {
    /// A delegated safe head would move the safe labels below finality.
    #[error("Delegated safe head breaks finality: {0}")]
    BreaksFinality(#[from] FinalityViolation),
    /// Failed to fetch the execution layer's block at the height of a delegated safe head below the
    /// safe head.
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
    /// The derived block is denied, or the denied blocks are not known yet.
    #[error(transparent)]
    Denied(#[from] ReplaceDeniedError),
    /// The derived attributes do not build on the local-safe head, so they are stale.
    #[error("Derived attributes on {parent:?} do not build on the local-safe head {local_safe:?}")]
    StaleAttributes {
        /// The parent the attributes build on.
        parent: BlockNumHash,
        /// The local-safe head.
        local_safe: BlockNumHash,
    },
}

impl From<BuildAndImportError> for ConsolidateTaskError {
    fn from(err: BuildAndImportError) -> Self {
        match err {
            BuildAndImportError::Build(e) => Self::BuildTaskFailed(e.into()),
            BuildAndImportError::Seal(e) => Self::SealTaskFailed(e),
            BuildAndImportError::Denied(e) => Self::Denied(e),
        }
    }
}

impl EngineTaskError for ConsolidateTaskError {
    fn severity(&self) -> EngineTaskErrorSeverity {
        match self {
            Self::BreaksFinality(_) | Self::StaleAttributes { .. } => EngineTaskErrorSeverity::Skip,
            Self::MissingUnsafeL2Block(_) => EngineTaskErrorSeverity::Reset,
            Self::FailedToFetchDelegatedSafeBlock(_) | Self::FailedToFetchUnsafeL2Block => {
                EngineTaskErrorSeverity::Temporary
            }
            Self::BuildTaskFailed(inner) => inner.severity(),
            Self::SealTaskFailed(inner) => inner.severity(),
            Self::ForkchoiceUpdateFailed(inner) => inner.severity(),
            Self::Denied(inner) => inner.severity(),
        }
    }
}
