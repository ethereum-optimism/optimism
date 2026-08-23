//! Contains error types for the [`crate::ConsolidateTask`].

use crate::{
    BuildTaskError, EngineTaskError, SealTaskError, SynchronizeTaskError,
    task_queue::tasks::{BuildAndSealError, task::EngineTaskErrorSeverity},
};
use thiserror::Error;

/// An error that occurs when running the [`crate::ConsolidateTask`].
#[derive(Debug, Error)]
pub enum ConsolidateTaskError {
    /// The unsafe L2 block to consolidate against is not in the execution layer, and the
    /// engine's initial EL sync has already finished.
    ///
    /// The unsafe head named a block the execution layer cannot serve, so the two have diverged
    /// — the engine may have restarted, or the heads are inconsistent. op-node answers this with
    /// a reset that re-discovers the execution layer's actual heads
    /// (`op-node/rollup/attributes/attributes.go:216-221`, a `rollup.ResetEvent` re-running
    /// `FindL2Heads`); after the walkback realigns the unsafe head, the consolidation gate
    /// (`attributes.go:185-193`) flips to force-building the attributes instead of fetching them,
    /// and the chain advances again.
    #[error("Unsafe L2 block is missing {0}")]
    MissingUnsafeL2Block(u64),
    /// The unsafe L2 block to consolidate against is not in the execution layer, while the
    /// engine's initial EL sync is still in flight.
    ///
    /// The execution layer has accepted a future sync target but has not filled this height in
    /// yet. Resetting now would issue a forkchoice update that re-targets the EL away from its
    /// sync target and prevents the sync from ever completing, so the miss is a paced stall
    /// instead — op-node's `IsEngineInitialELSyncing` arm of the same fetch miss
    /// (`op-node/rollup/attributes/attributes.go:205-215`, an `EngineTemporaryErrorEvent`).
    #[error("Unsafe L2 block {0} is not available while EL sync is in flight")]
    AwaitingELSyncUnsafeL2Block(u64),
    /// Failed to fetch the unsafe L2 block.
    ///
    /// A transport or RPC failure that does not say the block is absent — op-node's generic arm
    /// of the consolidation fetch, a temporary error
    /// (`op-node/rollup/attributes/attributes.go:222-227`).
    #[error("Failed to fetch the unsafe L2 block")]
    FailedToFetchUnsafeL2Block,
    /// The attributes' parent sits at the local-safe height but is not the local-safe head:
    /// reorg inconsistency between the queued attributes and the head they were derived onto.
    ///
    /// The mirror of op-node's queued-attributes conflict check, and its answer — a reset, so
    /// derivation rebuilds attributes on the head the engine actually has
    /// (`op-node/rollup/attributes/attributes.go:172-182`, a `ResetEvent`).
    #[error("Consolidation attributes parent conflicts with the local-safe head")]
    ParentConflictsWithLocalSafe,
    /// The deny list could not be read while deciding whether the block may be adopted.
    ///
    /// Fails closed: without a deny-list answer the block can be neither promoted nor reorged, so
    /// consolidation stalls and retries — the posture of op-node's consolidation deny check
    /// (`op-node/rollup/attributes/attributes.go:241-247`).
    #[error("The deny list could not be read while consolidating")]
    DenyListUnavailable,
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
            Self::MissingUnsafeL2Block(_) | Self::ParentConflictsWithLocalSafe => {
                EngineTaskErrorSeverity::Reset
            }
            Self::AwaitingELSyncUnsafeL2Block(_) |
            Self::FailedToFetchUnsafeL2Block |
            Self::DenyListUnavailable => EngineTaskErrorSeverity::Temporary,
            Self::BuildTaskFailed(inner) => inner.severity(),
            Self::SealTaskFailed(inner) => inner.severity(),
            Self::ForkchoiceUpdateFailed(inner) => inner.severity(),
        }
    }
}
