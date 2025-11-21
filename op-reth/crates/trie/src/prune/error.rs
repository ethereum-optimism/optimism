use crate::OpProofsStorageError;
use reth_provider::ProviderError;
use std::{
    fmt,
    fmt::{Display, Formatter},
    time::Duration,
};
use strum::Display;
use thiserror::Error;

/// Result of [`OpProofStoragePruner::run`](crate::OpProofStoragePruner::run) execution.
pub type OpProofStoragePrunerResult = Result<PrunerOutput, PrunerError>;

/// Successful prune summary.
#[derive(Debug, Clone, Default, Eq, PartialEq)]
pub struct PrunerOutput {
    /// Total elapsed wall time for this run (fetch + apply).
    pub duration: Duration,
    /// Earliest block at the start of the run.
    pub start_block: u64,
    /// New earliest block at the end of the run.
    pub end_block: u64,
    /// Total number of entries removed across tables.
    pub total_entries_pruned: u64,
}

impl Display for PrunerOutput {
    fn fmt(&self, f: &mut Formatter<'_>) -> fmt::Result {
        let blocks = self.end_block.saturating_sub(self.start_block);
        write!(
            f,
            "Pruned {}→{} ({} blocks), entries={}, elapsed={:.3}s",
            self.start_block,
            self.end_block,
            blocks,
            self.total_entries_pruned,
            self.duration.as_secs_f64(),
        )
    }
}

/// Error returned by the pruner.
#[derive(Debug, Error, Display)]
pub enum PrunerError {
    /// Wrapped error from the underlying `OpProofStorage` layer.
    Storage(#[from] OpProofsStorageError),

    /// Wrapped error from the reth db provider.
    Provider(#[from] ProviderError),

    /// Block not found in the underlying reth storage provider.
    BlockNotFound(u64),

    /// The pruner timed out before finishing the prune
    TimedOut(Duration),
}
