//! Whether the execution layer makes progress while a consolidation waits for it to sync.

use serde_json::{Map, Value};
use std::{mem, time::Duration};
use tokio::time::Instant;

/// The default bound on how long a consolidation waiting for the execution layer to sync tolerates
/// no reported progress before the engine resets.
///
/// Neither execution layer reports progress while it downloads headers: op-geth answers
/// `eth_syncing` with `false` until its beacon skeleton links to the local chain, and op-reth's
/// headers stage moves its checkpoint only once the whole range is downloaded. That download can
/// take hours on a large chain, and a reset during it brings back the reset loop the wait exists to
/// avoid, while a longer bound only delays the reset of an execution layer that can never fetch the
/// missing block.
pub const DEFAULT_EL_SYNC_STALL_TIMEOUT: Duration = Duration::from_secs(24 * 60 * 60);

/// How often a waiting consolidation checks on the execution layer: it reads the sync progress and
/// resends the unchanged forkchoice to learn whether the execution layer now has the unsafe head.
/// The retries in between only look the derived block up, which is cheap for an execution layer
/// that is busy syncing.
pub(super) const CHECK_INTERVAL: Duration = Duration::from_secs(30);

/// How often a waiting consolidation reports the execution layer's progress, or its lack of it.
pub(super) const PROGRESS_REPORT_INTERVAL: Duration = Duration::from_secs(5 * 60);

/// What the execution layer reports about its sync on one check of a waiting consolidation.
#[derive(Debug, Clone, PartialEq, Eq)]
pub(super) enum ElSyncReading {
    /// `eth_syncing` reports no sync in progress, and this is the latest block number.
    NotSyncing {
        /// The number of the execution layer's latest block.
        latest_number: u64,
    },
    /// `eth_syncing` reports a sync in progress, with these progress fields.
    Syncing(Map<String, Value>),
}

impl ElSyncReading {
    /// Whether the execution layer reports no sync while its latest block is `parent_number`: it
    /// has the parent of the next block but fetches nothing toward the unsafe head.
    pub(super) const fn is_idle_at(&self, parent_number: u64) -> bool {
        matches!(self, Self::NotSyncing { latest_number } if *latest_number == parent_number)
    }

    /// The progress in an `eth_syncing` status object: every field but the block the sync started
    /// from, which never changes, and the block it syncs toward, which can move while the sync
    /// itself makes no progress.
    pub(super) fn from_sync_status(mut status: Map<String, Value>) -> Self {
        status.remove("startingBlock");
        status.remove("highestBlock");
        Self::Syncing(status)
    }
}

/// Whether a consolidation keeps waiting for the execution layer to sync.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(super) enum ElSyncWaitOutcome {
    /// The execution layer reported progress.
    Progressed,
    /// The execution layer has reported no progress for the stall timeout.
    Stalled,
    /// The wait starts.
    Started,
    /// The execution layer reported no progress, or could not be read, within the stall timeout.
    Unchanged,
}

/// What one check of a waiting consolidation found.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(super) struct ElSyncWaitStep {
    /// Whether the wait goes on, starts, or has stalled.
    pub(super) outcome: ElSyncWaitOutcome,
    /// Whether to report the execution layer's progress, or its lack of it.
    pub(super) reports_progress: bool,
    /// How long the execution layer may still report no progress before the engine resets.
    pub(super) time_until_reset: Duration,
}

/// Tracks the execution layer's sync across the retries of a waiting consolidation.
#[derive(Debug)]
pub(super) struct ElSyncProgress {
    /// When a check's forkchoice probe was last answered.
    checked_at: Option<Instant>,
    /// When the wait started or the execution layer last reported progress.
    last_progress_at: Option<Instant>,
    /// The first reading that could be read, or the last one that differed from it.
    last_reading: Option<ElSyncReading>,
    /// When the execution layer's progress was last reported.
    progress_reported_at: Option<Instant>,
    /// How long the execution layer may report no progress.
    stall_timeout: Duration,
}

impl ElSyncProgress {
    /// Tracks a wait bounded by `stall_timeout`.
    pub(super) const fn new(stall_timeout: Duration) -> Self {
        Self {
            checked_at: None,
            last_progress_at: None,
            last_reading: None,
            progress_reported_at: None,
            stall_timeout,
        }
    }

    /// Whether a retry at `now` checks on the execution layer: when the wait starts, every
    /// [`CHECK_INTERVAL`] after the last answered check, and on the retry after a check whose
    /// forkchoice probe failed.
    pub(super) fn is_check_due(&self, now: Instant) -> bool {
        self.checked_at.is_none_or(|checked_at| now.duration_since(checked_at) >= CHECK_INTERVAL)
    }

    /// Records the execution layer's sync `reading` at `now`, `None` when it could not be read,
    /// and returns what this check found.
    ///
    /// Any change of a reading counts as progress, including a lower latest block: an execution
    /// layer whose datadir is wiped while it is awaited starts over from genesis. An unreadable
    /// reading is no evidence either way, so it neither counts as progress nor replaces the last
    /// reading, and the first reading that can be read is the baseline later ones are compared
    /// with. Progress, or its lack, is reported when the wait starts, when a sync starts or
    /// finishes, and otherwise at most every [`PROGRESS_REPORT_INTERVAL`].
    pub(super) fn record(
        &mut self,
        reading: Option<ElSyncReading>,
        now: Instant,
    ) -> ElSyncWaitStep {
        let Some(last_progress_at) = self.last_progress_at else {
            self.last_progress_at = Some(now);
            self.last_reading = reading;
            self.progress_reported_at = Some(now);
            return ElSyncWaitStep {
                outcome: ElSyncWaitOutcome::Started,
                reports_progress: true,
                time_until_reset: self.stall_timeout,
            };
        };

        let outcome = match (&self.last_reading, reading) {
            (None, Some(first_readable)) => {
                self.last_reading = Some(first_readable);
                self.unchanged_or_stalled(last_progress_at, now)
            }
            (Some(last), Some(reading)) if *last != reading => {
                let sync_kind_changed = mem::discriminant(last) != mem::discriminant(&reading);
                self.last_reading = Some(reading);
                self.last_progress_at = Some(now);
                if sync_kind_changed {
                    self.progress_reported_at = None;
                }
                ElSyncWaitOutcome::Progressed
            }
            _ => self.unchanged_or_stalled(last_progress_at, now),
        };

        let reports_progress = outcome != ElSyncWaitOutcome::Stalled &&
            self.progress_reported_at.is_none_or(|reported_at| {
                now.duration_since(reported_at) >= PROGRESS_REPORT_INTERVAL
            });
        if reports_progress {
            self.progress_reported_at = Some(now);
        }
        let since_progress = now.duration_since(self.last_progress_at.unwrap_or(now));
        ElSyncWaitStep {
            outcome,
            reports_progress,
            time_until_reset: self.stall_timeout.saturating_sub(since_progress),
        }
    }

    /// Whether a wait has started, which an answered check's reading does.
    #[cfg(test)]
    pub(super) const fn has_started(&self) -> bool {
        self.last_progress_at.is_some()
    }

    /// Records that the execution layer answered a check's forkchoice probe at `now`. A probe that
    /// failed is not recorded, so the next retry checks again.
    pub(super) const fn check_answered(&mut self, now: Instant) {
        self.checked_at = Some(now);
    }

    /// Whether the execution layer has reported no progress for the stall timeout at `now`.
    fn unchanged_or_stalled(&self, last_progress_at: Instant, now: Instant) -> ElSyncWaitOutcome {
        if now.duration_since(last_progress_at) >= self.stall_timeout {
            ElSyncWaitOutcome::Stalled
        } else {
            ElSyncWaitOutcome::Unchanged
        }
    }
}

#[cfg(test)]
mod tests {
    use super::{ElSyncWaitOutcome::*, *};

    const STALL_SECS: u64 = 3600;
    const CHECK_SECS: u64 = CHECK_INTERVAL.as_secs();
    const REPORT_SECS: u64 = PROGRESS_REPORT_INTERVAL.as_secs();

    /// A reading of an execution layer that is not syncing a range, at `latest_number`.
    const fn at(latest_number: u64) -> Option<ElSyncReading> {
        Some(ElSyncReading::NotSyncing { latest_number })
    }

    /// A reading of an execution layer syncing toward `highest_block`, whose `stage` checkpoint is
    /// at `stage_block`.
    fn staged(stage_block: u64, highest_block: u64) -> Option<ElSyncReading> {
        let status = serde_json::json!({
            "currentBlock": "0x0",
            "highestBlock": format!("{highest_block:#x}"),
            "stages": [{ "name": "Headers", "block": format!("{stage_block:#x}") }],
            "startingBlock": "0x0",
        });
        let serde_json::Value::Object(status) = status else { unreachable!() };
        Some(ElSyncReading::from_sync_status(status))
    }

    /// Records each check, `(seconds since the previous one, reading)`, and returns the last one.
    fn last_step(checks: &[(u64, Option<ElSyncReading>)]) -> ElSyncWaitStep {
        let mut progress = ElSyncProgress::new(Duration::from_secs(STALL_SECS));
        let mut now = Instant::now();
        let mut last = None;
        for (elapsed_secs, reading) in checks {
            now += Duration::from_secs(*elapsed_secs);
            last = Some(progress.record(reading.clone(), now));
        }
        last.expect("at least one check")
    }

    #[rstest::rstest]
    #[case::first_reading_starts_the_wait(&[(0, at(100))], Started)]
    #[case::first_unreadable_reading_starts_the_wait(&[(0, None)], Started)]
    #[case::unchanged_just_below_bound_waits(&[(0, at(100)), (STALL_SECS - 1, at(100))], Unchanged)]
    #[case::unchanged_for_bound_stalls(&[(0, at(100)), (STALL_SECS, at(100))], Stalled)]
    #[case::advancing_latest_block_progresses(&[(0, at(100)), (STALL_SECS - 1, at(101))], Progressed)]
    #[case::stalls_after_progress(&[(0, at(100)), (STALL_SECS - 1, at(101)), (STALL_SECS, at(101))], Stalled)]
    #[case::wiped_el_restarting_counts_as_progress(&[(0, at(100)), (STALL_SECS - 1, at(0))], Progressed)]
    #[case::sync_starting_counts_as_progress(&[(0, at(100)), (STALL_SECS - 1, staged(10, 500))], Progressed)]
    #[case::unchanged_sync_for_bound_stalls(&[(0, staged(10, 500)), (STALL_SECS, staged(10, 500))], Stalled)]
    #[case::advancing_stage_progresses(&[(0, staged(10, 500)), (STALL_SECS - 1, staged(20, 500))], Progressed)]
    #[case::moving_sync_target_alone_stalls(&[(0, staged(10, 500)), (STALL_SECS, staged(10, 600))], Stalled)]
    #[case::unreadable_reading_is_not_progress(&[(0, at(100)), (STALL_SECS - 1, None), (1, at(100))], Stalled)]
    #[case::unreadable_readings_stall(&[(0, None), (STALL_SECS, None)], Stalled)]
    #[case::first_readable_reading_is_not_progress(&[(0, None), (1, at(100))], Unchanged)]
    #[case::first_readable_reading_keeps_the_bound(&[(0, None), (1, at(100)), (STALL_SECS - 1, at(100))], Stalled)]
    fn sync_progress_bounds_the_wait(
        #[case] checks: &[(u64, Option<ElSyncReading>)],
        #[case] expected: ElSyncWaitOutcome,
    ) {
        assert_eq!(last_step(checks).outcome, expected);
    }

    /// Whether a retry `elapsed_secs` after the wait starts checks on the execution layer, when
    /// the first check's forkchoice probe was answered or failed.
    #[rstest::rstest]
    #[case::wait_start_checks(None, 0, true)]
    #[case::retry_before_the_interval_does_not_check(Some(true), CHECK_SECS - 1, false)]
    #[case::retry_after_the_interval_checks(Some(true), CHECK_SECS, true)]
    #[case::failed_probe_checks_on_the_next_retry(Some(false), 2, true)]
    fn checks_are_due_on_their_cadence(
        #[case] first_check_answered: Option<bool>,
        #[case] elapsed_secs: u64,
        #[case] expected: bool,
    ) {
        let mut progress = ElSyncProgress::new(Duration::from_secs(STALL_SECS));
        let started_at = Instant::now();
        if let Some(answered) = first_check_answered {
            progress.record(at(100), started_at);
            if answered {
                progress.check_answered(started_at);
            }
        }
        assert_eq!(progress.is_check_due(started_at + Duration::from_secs(elapsed_secs)), expected);
    }

    #[rstest::rstest]
    #[case::wait_start_reports(&[(0, at(100))], true)]
    #[case::progress_before_the_interval_is_quiet(&[(0, at(100)), (REPORT_SECS - 1, at(101))], false)]
    #[case::progress_after_the_interval_reports(&[(0, at(100)), (REPORT_SECS, at(101))], true)]
    #[case::no_progress_before_the_interval_is_quiet(&[(0, at(100)), (REPORT_SECS - 1, at(100))], false)]
    #[case::no_progress_after_the_interval_reports(&[(0, at(100)), (REPORT_SECS, at(100))], true)]
    #[case::unreadable_after_the_interval_reports(&[(0, at(100)), (REPORT_SECS, None)], true)]
    #[case::sync_starting_reports(&[(0, at(100)), (1, staged(10, 500))], true)]
    #[case::sync_finishing_reports(&[(0, staged(10, 500)), (1, at(500))], true)]
    #[case::first_readable_reading_is_quiet(&[(0, None), (1, at(100))], false)]
    #[case::stall_is_not_reported_as_progress(&[(0, at(100)), (STALL_SECS, at(100))], false)]
    fn progress_is_reported_on_its_cadence(
        #[case] checks: &[(u64, Option<ElSyncReading>)],
        #[case] expected: bool,
    ) {
        assert_eq!(last_step(checks).reports_progress, expected);
    }

    #[rstest::rstest]
    #[case::wait_start_has_the_whole_bound(&[(0, at(100))], STALL_SECS)]
    #[case::no_progress_uses_up_the_bound(&[(0, at(100)), (100, at(100))], STALL_SECS - 100)]
    #[case::progress_restores_the_bound(&[(0, at(100)), (100, at(101))], STALL_SECS)]
    #[case::first_readable_reading_does_not_restore_the_bound(&[(0, None), (100, at(100))], STALL_SECS - 100)]
    fn time_until_reset_counts_from_the_last_progress(
        #[case] checks: &[(u64, Option<ElSyncReading>)],
        #[case] expected_secs: u64,
    ) {
        assert_eq!(last_step(checks).time_until_reset, Duration::from_secs(expected_secs));
    }
}
