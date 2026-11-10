//! Per-phase timing and unique-key counts for writes and prunes.
//!
//! Both are exported as histograms labelled by operation, phase and table, so the cost of each
//! phase can be compared per block and per unique key. Phases are timed inside the uncommitted
//! write transaction, so they do not sum to the persistence `write`/`prune` durations: those
//! also include proof window updates, and the commit is timed separately.

use std::time::{Duration, Instant};

/// Table label for the account trie tables.
pub(super) const ACCOUNT_TRIE: &str = "account_trie";
/// Table label for the storage trie tables.
pub(super) const STORAGE_TRIE: &str = "storage_trie";
/// Table label for the hashed account tables.
pub(super) const HASHED_ACCOUNTS: &str = "hashed_accounts";
/// Table label for the hashed storage tables.
pub(super) const HASHED_STORAGES: &str = "hashed_storages";

/// Unique keys touched and time spent on one table in one phase.
#[derive(Debug, Default, Clone, Copy)]
pub(super) struct TableStats {
    /// Unique keys touched.
    pub(super) keys: usize,
    /// Time spent.
    pub(super) duration: Duration,
}

/// [`TableStats`] for each of the four data types in one phase.
#[derive(Debug, Default, Clone, Copy)]
pub(super) struct PhaseStats {
    pub(super) account_trie: TableStats,
    pub(super) storage_trie: TableStats,
    pub(super) hashed_accounts: TableStats,
    pub(super) hashed_storages: TableStats,
}

impl PhaseStats {
    const fn tables(&self) -> [(&'static str, TableStats); 4] {
        [
            (ACCOUNT_TRIE, self.account_trie),
            (STORAGE_TRIE, self.storage_trie),
            (HASHED_ACCOUNTS, self.hashed_accounts),
            (HASHED_STORAGES, self.hashed_storages),
        ]
    }

    /// Total time spent across the four tables.
    pub(super) fn total(&self) -> Duration {
        self.tables().iter().map(|(_, stats)| stats.duration).sum()
    }

    /// Records each table's duration for `phase` of `op`.
    pub(super) fn record_durations(&self, op: &'static str, phase: &'static str) {
        for (table, stats) in self.tables() {
            record_duration(op, phase, table, stats.duration);
        }
    }

    /// Records each table's unique key count for `op`.
    pub(super) fn record_unique_keys(&self, op: &'static str) {
        for (table, stats) in self.tables() {
            record_unique_keys(op, table, stats.keys);
        }
    }
}

/// Runs `f` and returns its result together with the time it took.
pub(super) fn timed<T>(f: impl FnOnce() -> T) -> (T, Duration) {
    let start = Instant::now();
    let result = f();
    (result, start.elapsed())
}

/// Records how long `phase` of `op` took on `table`.
#[cfg_attr(not(feature = "metrics"), allow(clippy::missing_const_for_fn))]
fn record_duration(op: &'static str, phase: &'static str, table: &'static str, duration: Duration) {
    #[cfg(feature = "metrics")]
    metrics::histogram!(
        "optimism_trie.storage.phase_duration_seconds",
        "operation" => op,
        "phase" => phase,
        "table" => table,
    )
    .record(duration.as_secs_f64());
    #[cfg(not(feature = "metrics"))]
    let _ = (op, phase, table, duration);
}

/// Records how long a batch write spent writing current state and changesets, across all
/// tables.
#[cfg_attr(not(feature = "metrics"), allow(clippy::missing_const_for_fn))]
pub(super) fn record_write_state_duration(duration: Duration) {
    #[cfg(feature = "metrics")]
    metrics::histogram!("optimism_trie.storage.write_state_duration_seconds")
        .record(duration.as_secs_f64());
    #[cfg(not(feature = "metrics"))]
    let _ = duration;
}

/// Records how many unique keys `op` touched in `table`'s history bitmaps.
#[cfg_attr(not(feature = "metrics"), allow(clippy::missing_const_for_fn))]
fn record_unique_keys(op: &'static str, table: &'static str, keys: usize) {
    #[cfg(feature = "metrics")]
    metrics::histogram!(
        "optimism_trie.storage.history_unique_keys",
        "operation" => op,
        "table" => table,
    )
    .record(keys as f64);
    #[cfg(not(feature = "metrics"))]
    let _ = (op, table, keys);
}
