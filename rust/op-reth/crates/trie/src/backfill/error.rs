//! Error type for backfill operations.

use crate::{OpProofsStorageError, snapshot::SnapshotError};
use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use reth_execution_errors::StateRootError;
use reth_provider::ProviderError;

/// Error type for backfill operations.
#[derive(Debug, thiserror::Error)]
pub enum BackfillError {
    /// Error bubbled up from proofs storage operations.
    #[error(transparent)]
    Storage(#[from] OpProofsStorageError),
    /// Error from reth provider operations.
    #[error(transparent)]
    Provider(#[from] ProviderError),
    /// State root computation failed.
    #[error(transparent)]
    StateRoot(#[from] StateRootError),
    /// Error from snapshot-init / state operations during snapshot-accelerated backfill.
    #[error(transparent)]
    Snapshot(#[from] SnapshotError),
    /// Computed state root does not match the expected root from the header.
    #[error(
        "State root mismatch at block {block_number}: computed {computed:?}, expected {expected:?}"
    )]
    StateRootMismatch {
        /// Block number being validated (the block whose before-state is being checked).
        block_number: u64,
        /// Computed root from the proofs storage overlay.
        computed: B256,
        /// Expected root from reth's block header.
        expected: B256,
    },
    /// Snapshot-accelerated backfill requires the snapshot anchor to equal
    /// the proofs window's current `earliest`.
    #[error(
        "snapshot anchor {found:?} does not match proofs `earliest` {expected:?}; \
         drop the snapshot and re-init at {expected:?} or run backfill without --use-snapshot"
    )]
    SnapshotAnchorMismatch {
        /// Current `earliest` block of the proofs window.
        expected: BlockNumHash,
        /// Anchor block of the existing snapshot.
        found: BlockNumHash,
    },
}
