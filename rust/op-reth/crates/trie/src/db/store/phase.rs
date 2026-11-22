//! Per-phase timing and unique-key counts for writes and prunes.
//!
//! Both are exported as histograms labelled by operation, phase and table, so the cost of each
//! phase can be compared per block and per unique key. Phases are timed inside the uncommitted
//! write transaction, so they do not sum to the persistence `write`/`prune` durations: those
//! also include proof window updates, and the commit is timed separately. They are recorded
//! before the commit too, so a batch whose commit then fails has still been recorded.

use std::time::{Duration, Instant};

/// Table label for the account trie tables.
pub(super) const ACCOUNT_TRIE: &str = "account_trie";
/// Table label for the storage trie tables.
pub(super) const STORAGE_TRIE: &str = "storage_trie";
/// Table label for the hashed account tables.
pub(super) const HASHED_ACCOUNTS: &str = "hashed_accounts";
/// Table label for the hashed storage tables.
pub(super) const HASHED_STORAGES: &str = "hashed_storages";

/// What a stretch of work cost the thread that did it.
#[derive(Debug, Default, Clone, Copy)]
pub(super) struct Cost {
    /// Time spent.
    pub(super) duration: Duration,
    /// Major page faults: pages the thread had to wait to read from disk.
    pub(super) major_faults: u64,
}

/// Unique keys touched on one table in one phase, and what that cost.
#[derive(Debug, Default, Clone, Copy)]
pub(super) struct TableStats {
    /// Unique keys touched.
    pub(super) keys: usize,
    /// Time spent and disk reads.
    pub(super) cost: Cost,
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
        self.tables().iter().map(|(_, stats)| stats.cost.duration).sum()
    }

    /// Total major page faults across the four tables.
    pub(super) fn major_faults(&self) -> u64 {
        self.tables().iter().map(|(_, stats)| stats.cost.major_faults).sum()
    }

    /// Records each table's duration and major page faults for `phase` of `op`.
    pub(super) fn record_cost(&self, op: &'static str, phase: &'static str) {
        for (table, stats) in self.tables() {
            let labels = [("operation", op), ("phase", phase), ("table", table)];
            record(
                "optimism_trie.storage.phase_duration_seconds",
                "Time one phase of a batch write or prune spent on one table",
                &labels,
                stats.cost.duration.as_secs_f64(),
            );
            record(
                "optimism_trie.storage.phase_major_faults",
                "Major page faults of the persistence thread in one phase of a batch write or \
                 prune, on one table",
                &labels,
                stats.cost.major_faults as f64,
            );
        }
    }

    /// Records how many unique keys `op` touched in each table's history bitmaps.
    pub(super) fn record_unique_keys(&self, op: &'static str) {
        for (table, stats) in self.tables() {
            let labels = [("operation", op), ("table", table)];
            record(
                "optimism_trie.storage.history_unique_keys",
                "Unique keys whose history bitmaps a batch write or prune touched in one table",
                &labels,
                stats.keys as f64,
            );
        }
    }
}

/// Measures the [`Cost`] of the work the calling thread does between [`Self::start`] and
/// [`Self::stop`].
pub(super) struct Stopwatch {
    start: Instant,
    major_faults: u64,
}

impl Stopwatch {
    pub(super) fn start() -> Self {
        Self { major_faults: thread_major_faults(), start: Instant::now() }
    }

    pub(super) fn stop(self) -> Cost {
        Cost {
            duration: self.start.elapsed(),
            major_faults: thread_major_faults().saturating_sub(self.major_faults),
        }
    }
}

/// Runs `f` and returns its result and what it cost the calling thread.
pub(super) fn timed<T>(f: impl FnOnce() -> T) -> (T, Cost) {
    let stopwatch = Stopwatch::start();
    let result = f();
    (result, stopwatch.stop())
}

/// Major page faults so far on the calling thread, or `0` where the count is unavailable.
///
/// Persistence reads the memory-mapped database, so each major fault is a page read from disk
/// that the thread waited on.
///
/// The count is only read on Linux, and only with the `metrics` feature. Without that feature
/// nothing exports it, and the per-batch debug logs report 0 major faults.
#[cfg_attr(not(all(target_os = "linux", feature = "metrics")), allow(clippy::missing_const_for_fn))]
fn thread_major_faults() -> u64 {
    #[cfg(all(target_os = "linux", feature = "metrics"))]
    {
        // Field 12 of `/proc/thread-self/stat`. The comm field before it may contain spaces, so
        // count fields from the closing parenthesis, which ends it.
        std::fs::read_to_string("/proc/thread-self/stat")
            .ok()
            .and_then(|stat| stat.rsplit_once(')')?.1.split_whitespace().nth(9)?.parse().ok())
            .unwrap_or(0)
    }
    #[cfg(not(all(target_os = "linux", feature = "metrics")))]
    0
}

/// Records what a batch write spent writing current state and changesets, across all tables.
pub(super) fn record_write_state(cost: Cost) {
    record(
        "optimism_trie.storage.write_state_duration_seconds",
        "Time a batch write spent writing current state and changesets",
        &[],
        cost.duration.as_secs_f64(),
    );
    record(
        "optimism_trie.storage.write_state_major_faults",
        "Major page faults of the persistence thread while a batch write wrote current state \
         and changesets",
        &[],
        cost.major_faults as f64,
    );
}

/// Records `value` in the histogram `name` with `labels`. The histogram is exported with `help`
/// as its help text.
#[cfg_attr(not(feature = "metrics"), allow(clippy::missing_const_for_fn))]
fn record(
    name: &'static str,
    help: &'static str,
    labels: &[(&'static str, &'static str)],
    value: f64,
) {
    #[cfg(feature = "metrics")]
    {
        metrics::describe_histogram!(name, help);
        metrics::histogram!(name, labels).record(value);
    }
    #[cfg(not(feature = "metrics"))]
    let _ = (name, help, labels, value);
}

#[cfg(all(test, feature = "metrics"))]
mod tests {
    use super::*;
    use metrics_util::debugging::DebuggingRecorder;
    use std::collections::BTreeSet;

    // A histogram is exported with help text only if it was described. One recorded without a
    // description shows up with no word on what it measures.
    #[test]
    fn every_histogram_is_recorded_with_help_text() {
        let recorder = DebuggingRecorder::new();
        let snapshotter = recorder.snapshotter();
        metrics::with_local_recorder(&recorder, || {
            let stats = PhaseStats::default();
            stats.record_cost("prune", "history");
            stats.record_unique_keys("prune");
            record_write_state(Cost::default());
        });

        let mut described = BTreeSet::new();
        for (key, _, help, _) in snapshotter.snapshot().into_vec() {
            let name = key.key().name().to_string();
            assert!(help.is_some_and(|help| !help.is_empty()), "{name} has no help text");
            described.insert(name);
        }
        assert_eq!(
            described.into_iter().collect::<Vec<_>>(),
            [
                "optimism_trie.storage.history_unique_keys",
                "optimism_trie.storage.phase_duration_seconds",
                "optimism_trie.storage.phase_major_faults",
                "optimism_trie.storage.write_state_duration_seconds",
                "optimism_trie.storage.write_state_major_faults",
            ]
        );
    }
}
