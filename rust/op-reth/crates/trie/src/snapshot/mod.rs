//! One-time trie-state snapshot at a caller-supplied target block.
//!
//! Copies the trie state at the target block into the parallel
//! `V2*TrieSnapshot` tables and marks the meta row `Ready` once the snapshot's
//! computed state root matches reth's header.
//!
//! The driver is [`SnapshotInitJob`]; failures surface as [`SnapshotError`].
//!
//! ## Restart / resume
//!
//! Each chunked rw-tx commits independently; after a crash the meta stays at
//! [`SnapshotStatus::Building`] with the original anchor. A re-run inspects
//! [`OpProofsSnapshotInitProvider::snapshot_init_anchor`], discovers the
//! resume keys from the partially-populated destination tables, and continues
//! from there. Resume is only safe when the target block matches the existing
//! anchor — otherwise the init aborts with
//! [`SnapshotError::SnapshotResumeDriftDetected`].
//!
//! [`SnapshotStatus::Building`]: crate::db::SnapshotStatus::Building
//! [`OpProofsSnapshotInitProvider::snapshot_init_anchor`]: crate::OpProofsSnapshotInitProvider::snapshot_init_anchor

mod error;
mod job;

pub use error::SnapshotError;
pub use job::{SnapshotInitJob, SnapshotInitOutcome};

#[cfg(test)]
mod tests;
