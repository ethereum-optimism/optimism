//! Records the L2 safe head reached as of each L1 block into the safe head database.

use alloy_eips::BlockNumHash;
use backon::{ExponentialBuilder, Retryable};
use kona_protocol::L2BlockInfo;
use kona_safedb::{SafeDb, SafeDbError};
use std::{sync::Arc, time::Duration};

/// Retries of a failed safe head database write before the failure is treated as fatal.
const WRITE_RETRIES: usize = 4;

/// Delay before the first retry of a failed write. Doubles on every further attempt.
const FIRST_RETRY_DELAY: Duration = Duration::from_millis(50);

/// Why recorded history is being truncated.
#[derive(Debug, Clone, Copy)]
pub(crate) enum TruncationReason {
    /// The execution layer finished syncing and derivation starts from its safe head.
    ElSyncCompleted,
    /// The engine reset its forkchoice and derivation restarts from the reset head.
    EngineReset,
}

/// Writes derivation progress into a [`SafeDb`].
///
/// An entry claims that L1 data made an L2 block safe, so only a head produced by attributes
/// this node derived is recorded, keyed by the L1 block those attributes were derived from. A
/// head the engine adopts on a reset or at EL-sync completion was not derived here; it
/// truncates the history to the point derivation resumes from instead of extending it.
#[derive(Debug)]
pub(crate) struct SafeHeadRecorder {
    db: Arc<dyn SafeDb>,
}

impl SafeHeadRecorder {
    /// Creates a recorder writing to `db`.
    pub(crate) const fn new(db: Arc<dyn SafeDb>) -> Self {
        Self { db }
    }

    /// Records that `safe_head` became safe as of `derived_from`, the L1 block its attributes
    /// were derived from. Fails only once the write has failed on every attempt.
    pub(crate) async fn record(
        &self,
        safe_head: L2BlockInfo,
        derived_from: BlockNumHash,
    ) -> Result<(), SafeDbError> {
        if !self.db.enabled() {
            return Ok(());
        }
        let db = self.db.clone();
        Self::write_with_retries("safe head update", || {
            db.safe_head_updated(safe_head, derived_from)
        })
        .await?;
        debug!(
            target: "derivation",
            l1_block = derived_from.number,
            safe_head = safe_head.block_info.number,
            "Recorded safe head"
        );
        Ok(())
    }

    /// Truncates recorded history so that `resume_head`, the block derivation restarts from, is
    /// the tip again. Entries above it are recorded again as derivation passes them.
    ///
    /// When the recorded tip is below `resume_head`, derivation never revisits the blocks
    /// between them, so they can never be recorded. A later query would not report that gap; it
    /// would answer with an older safe head. The whole history is cleared instead, so queries
    /// report no history rather than a wrong answer.
    pub(crate) async fn truncate(
        &self,
        resume_head: L2BlockInfo,
        reason: TruncationReason,
    ) -> Result<(), SafeDbError> {
        if !self.db.enabled() {
            return Ok(());
        }
        let db = self.db.clone();
        match db.last_entry() {
            Ok(tip) if tip.safe_head.number < resume_head.block_info.number => {
                warn!(
                    target: "derivation",
                    ?reason,
                    tip = tip.safe_head.number,
                    resume_head = resume_head.block_info.number,
                    "Safe head db tip is below the derivation resume point; clearing the history rather than leaving a gap"
                );
                return Self::write_with_retries("safe head db clear", || db.clear()).await;
            }
            Ok(_) | Err(SafeDbError::NotFound) => {}
            Err(e) => warn!(target: "derivation", %e, "Failed to read the safe head db tip"),
        }

        Self::write_with_retries("safe head reset", || db.safe_head_reset(resume_head)).await?;
        info!(
            target: "derivation",
            ?reason,
            resume_head = resume_head.block_info.number,
            "Truncated safe head db to the derivation resume point"
        );
        Ok(())
    }

    /// Runs a database write, retrying before giving up.
    ///
    /// A missing entry is not reported by later queries: [`SafeDb::safe_head_at_l1`] answers with
    /// the highest entry at or below the requested L1 block, so a skipped write is served as an
    /// older safe head. Retries absorb a transient failure; after that the error is returned so
    /// the node stops instead of leaving the gap behind.
    async fn write_with_retries(
        operation: &'static str,
        write: impl Fn() -> Result<(), SafeDbError>,
    ) -> Result<(), SafeDbError> {
        (|| async { write() })
            .retry(
                ExponentialBuilder::default()
                    .with_min_delay(FIRST_RETRY_DELAY)
                    .with_max_times(WRITE_RETRIES),
            )
            .notify(|e: &SafeDbError, delay: Duration| {
                warn!(target: "derivation", %e, ?delay, operation, "Safe head db write failed; retrying")
            })
            .await
            .inspect_err(|e| {
                error!(target: "derivation", %e, operation, "Safe head db write failed on every attempt")
            })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::B256;
    use kona_protocol::BlockInfo;
    use kona_safedb::SafeHeadRecord;
    use rstest::rstest;
    use std::sync::{
        Mutex,
        atomic::{AtomicU32, Ordering},
    };

    const DERIVED_FROM_L1: u64 = 100;
    /// Differs from [`DERIVED_FROM_L1`] so a passing test cannot be recording the safe head's
    /// own L1 origin by mistake.
    const SAFE_HEAD_L1_ORIGIN: u64 = 42;
    const RESUME_HEAD: u64 = 20;

    fn derived_from() -> BlockNumHash {
        BlockNumHash { hash: B256::repeat_byte(1), number: DERIVED_FROM_L1 }
    }

    fn l2_block(number: u64) -> L2BlockInfo {
        L2BlockInfo {
            block_info: BlockInfo { hash: B256::repeat_byte(2), number, ..Default::default() },
            l1_origin: BlockNumHash { hash: B256::ZERO, number: SAFE_HEAD_L1_ORIGIN },
            seq_num: 0,
        }
    }

    #[derive(Debug, Clone, Copy, PartialEq, Eq)]
    enum Write {
        Cleared,
        Reset { resume_head: u64 },
        Updated { safe_head: u64, l1: u64 },
    }

    /// Records every write; fails the first `failures` writes with [`SafeDbError::Closed`].
    #[derive(Debug)]
    struct FakeSafeDb {
        attempts: AtomicU32,
        enabled: bool,
        failures: AtomicU32,
        tip: Option<u64>,
        writes: Mutex<Vec<Write>>,
    }

    impl Default for FakeSafeDb {
        fn default() -> Self {
            Self {
                attempts: AtomicU32::new(0),
                enabled: true,
                failures: AtomicU32::new(0),
                tip: None,
                writes: Mutex::new(Vec::new()),
            }
        }
    }

    impl FakeSafeDb {
        fn disabled() -> Self {
            Self { enabled: false, ..Default::default() }
        }

        fn failing(failures: u32) -> Self {
            Self { failures: AtomicU32::new(failures), ..Default::default() }
        }

        fn with_tip(tip: Option<u64>) -> Self {
            Self { tip, ..Default::default() }
        }

        fn writes(&self) -> Vec<Write> {
            self.writes.lock().unwrap().clone()
        }

        fn attempts(&self) -> u32 {
            self.attempts.load(Ordering::SeqCst)
        }

        fn record(&self, write: Write) -> Result<(), SafeDbError> {
            self.attempts.fetch_add(1, Ordering::SeqCst);
            let failed = self
                .failures
                .fetch_update(Ordering::SeqCst, Ordering::SeqCst, |remaining| {
                    remaining.checked_sub(1)
                })
                .is_ok();
            if failed {
                return Err(SafeDbError::Closed);
            }
            self.writes.lock().unwrap().push(write);
            Ok(())
        }
    }

    impl SafeDb for FakeSafeDb {
        fn enabled(&self) -> bool {
            self.enabled
        }
        fn safe_head_updated(
            &self,
            safe_head: L2BlockInfo,
            l1_head: BlockNumHash,
        ) -> Result<(), SafeDbError> {
            self.record(Write::Updated {
                safe_head: safe_head.block_info.number,
                l1: l1_head.number,
            })
        }
        fn safe_head_reset(&self, safe_head: L2BlockInfo) -> Result<(), SafeDbError> {
            self.record(Write::Reset { resume_head: safe_head.block_info.number })
        }
        fn safe_head_at_l1(&self, _: u64) -> Result<SafeHeadRecord, SafeDbError> {
            unimplemented!("the recorder only writes")
        }
        fn first_entry(&self) -> Result<SafeHeadRecord, SafeDbError> {
            unimplemented!("the recorder only writes")
        }
        fn last_entry(&self) -> Result<SafeHeadRecord, SafeDbError> {
            let tip = self.tip.ok_or(SafeDbError::NotFound)?;
            let id = BlockNumHash { hash: B256::ZERO, number: tip };
            Ok(SafeHeadRecord { l1: id, safe_head: id })
        }
        fn l1_at_safe_head(&self, _: u64) -> Result<SafeHeadRecord, SafeDbError> {
            unimplemented!("the recorder only writes")
        }
        fn clear(&self) -> Result<(), SafeDbError> {
            self.record(Write::Cleared)
        }
        fn close(&self) -> Result<(), SafeDbError> {
            Ok(())
        }
    }

    fn recorder(db: &Arc<FakeSafeDb>) -> SafeHeadRecorder {
        SafeHeadRecorder::new(db.clone())
    }

    #[tokio::test]
    async fn a_record_is_keyed_by_the_l1_block_the_head_was_derived_from() {
        let db = Arc::new(FakeSafeDb::default());

        recorder(&db).record(l2_block(RESUME_HEAD + 1), derived_from()).await.unwrap();

        assert_eq!(
            db.writes(),
            vec![Write::Updated { safe_head: RESUME_HEAD + 1, l1: DERIVED_FROM_L1 }],
            "the key must be the derivation source, not the safe head's own L1 origin"
        );
    }

    #[rstest]
    #[case::empty_history(None, Write::Reset { resume_head: RESUME_HEAD })]
    #[case::tip_at_resume_head(Some(RESUME_HEAD), Write::Reset { resume_head: RESUME_HEAD })]
    #[case::tip_above_resume_head(Some(RESUME_HEAD + 10), Write::Reset { resume_head: RESUME_HEAD })]
    #[case::tip_below_resume_head(Some(RESUME_HEAD - 5), Write::Cleared)]
    #[tokio::test]
    async fn a_truncation_resets_to_the_resume_head_unless_the_tip_is_below_it(
        #[case] tip: Option<u64>,
        #[case] expected: Write,
    ) {
        let db = Arc::new(FakeSafeDb::with_tip(tip));

        recorder(&db).truncate(l2_block(RESUME_HEAD), TruncationReason::EngineReset).await.unwrap();

        assert_eq!(db.writes(), vec![expected]);
    }

    #[rstest]
    #[case::first_attempt(0, true, 1)]
    #[case::last_retry(WRITE_RETRIES as u32, true, WRITE_RETRIES as u32 + 1)]
    #[case::every_attempt_fails(u32::MAX, false, WRITE_RETRIES as u32 + 1)]
    #[tokio::test(start_paused = true)]
    async fn a_failed_write_is_retried_a_bounded_number_of_times(
        #[case] failures: u32,
        #[case] expect_recorded: bool,
        #[case] expected_attempts: u32,
    ) {
        let db = Arc::new(FakeSafeDb::failing(failures));

        let result = recorder(&db).record(l2_block(RESUME_HEAD + 1), derived_from()).await;

        assert_eq!(result.is_ok(), expect_recorded);
        assert_eq!(db.attempts(), expected_attempts);
        assert_eq!(db.writes().len(), usize::from(expect_recorded));
    }

    #[tokio::test]
    async fn a_disabled_database_is_never_written() {
        let db = Arc::new(FakeSafeDb::disabled());
        let recorder = recorder(&db);

        recorder.record(l2_block(RESUME_HEAD + 1), derived_from()).await.unwrap();
        recorder.truncate(l2_block(RESUME_HEAD), TruncationReason::ElSyncCompleted).await.unwrap();

        assert!(db.writes().is_empty());
        assert_eq!(db.attempts(), 0);
    }
}
