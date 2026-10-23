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
    db: Option<Arc<dyn SafeDb>>,
}

impl SafeHeadRecorder {
    /// Creates a recorder writing to `db`.
    pub(crate) const fn new(db: Option<Arc<dyn SafeDb>>) -> Self {
        Self { db }
    }

    /// Records that `safe_head` became safe as of `derived_from`, the L1 block its attributes
    /// were derived from. Fails only once the write has failed on every attempt.
    pub(crate) async fn record(
        &self,
        safe_head: L2BlockInfo,
        derived_from: BlockNumHash,
    ) -> Result<(), SafeDbError> {
        let Some(db) = &self.db else {
            return Ok(());
        };
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
        let Some(db) = &self.db else {
            return Ok(());
        };
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
mod tests;
