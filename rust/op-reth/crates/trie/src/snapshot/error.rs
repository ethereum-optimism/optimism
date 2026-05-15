//! Error type for snapshot-init operations.

use crate::{OpProofsStorageError, api::SnapshotInitStatus};
use alloy_primitives::B256;
use reth_db::DatabaseError;
use reth_execution_errors::StateRootError;
use reth_provider::ProviderError;

/// Error type for [`super::SnapshotInitJob`].
#[derive(Debug, thiserror::Error)]
pub enum SnapshotError {
    /// Error bubbled up from proofs storage operations.
    #[error(transparent)]
    Storage(#[from] OpProofsStorageError),
    /// Error from reth provider operations.
    #[error(transparent)]
    Provider(#[from] ProviderError),
    /// State root computation failed.
    #[error(transparent)]
    StateRoot(#[from] StateRootError),
    /// Computed state root does not match the expected root from the header.
    #[error(
        "State root mismatch at block {block_number}: computed {computed:?}, expected {expected:?}"
    )]
    StateRootMismatch {
        /// Block whose state root was validated.
        block_number: u64,
        /// Computed root from the snapshot tables + live hashed leaves.
        computed: B256,
        /// Expected root from reth's block header.
        expected: B256,
    },
    /// Snapshot init target block is outside the proofs window.
    #[error(
        "snapshot init target block {target_block} is outside proof window [{earliest}, {latest}]"
    )]
    SnapshotInitTargetOutsideWindow {
        /// The block requested as the snapshot anchor.
        target_block: u64,
        /// Current earliest persisted block.
        earliest: u64,
        /// Current latest persisted block.
        latest: u64,
    },
    /// Resume requested but the existing `Building` anchor doesn't match.
    #[error("snapshot resume drift detected at anchor block {anchor_block}: {reason}")]
    SnapshotResumeDriftDetected {
        /// Anchor block of the partial snapshot already on disk.
        anchor_block: u64,
        /// Human-readable reason describing the drift.
        reason: &'static str,
    },
    /// A snapshot already exists at a different anchor; the caller must drop
    /// it before requesting a new build.
    #[error(
        "snapshot already exists at block {existing_block} (status {existing_status:?}); drop it before rebuilding"
    )]
    SnapshotAlreadyExists {
        /// Anchor block of the existing snapshot.
        existing_block: u64,
        /// Lifecycle status of the existing snapshot.
        existing_status: SnapshotInitStatus,
    },
}

impl From<DatabaseError> for SnapshotError {
    fn from(err: DatabaseError) -> Self {
        Self::Storage(OpProofsStorageError::from(err))
    }
}
