use crate::{api::WriteCounts, OpProofsStorageError};
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
    /// Time elapsed during the stat diff fetch phase(non-blocking).
    pub fetch_duration: Duration,
    /// Time elapsed during the prune phase.
    pub prune_duration: Duration,
    /// Earliest block at the start of the run.
    pub start_block: u64,
    /// New earliest block at the end of the run.
    pub end_block: u64,
    /// Number of entries updated/removed per table.
    pub write_counts: WriteCounts,
}

impl Display for PrunerOutput {
    fn fmt(&self, f: &mut Formatter<'_>) -> fmt::Result {
        let blocks = self.end_block.saturating_sub(self.start_block);
        let total_entries = self.write_counts.hashed_accounts_written_total +
            self.write_counts.hashed_storages_written_total +
            self.write_counts.account_trie_updates_written_total +
            self.write_counts.storage_trie_updates_written_total;
        write!(
            f,
            "Pruned {}→{} ({} blocks), entries={}, elapsed={:.3}s",
            self.start_block,
            self.end_block,
            blocks,
            total_entries,
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

#[cfg(test)]
mod tests {
    use super::PrunerOutput;
    use crate::api::WriteCounts;
    use std::time::Duration;

    #[test]
    fn test_pruner_output_display() {
        let pruner_output = PrunerOutput {
            duration: Duration::from_secs(10),
            fetch_duration: Duration::from_secs(5),
            prune_duration: Duration::from_secs(5),
            start_block: 1,
            end_block: 2,
            write_counts: WriteCounts::new(1, 2, 3, 4),
        };
        let formatted_pruner_output = format!("{}", pruner_output);

        assert_eq!(formatted_pruner_output, "Pruned 1→2 (1 blocks), entries=10, elapsed=10.000s");
    }
}
