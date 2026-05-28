//! [`BackfillJob`] implementation.

use super::{changesets::compute_block_backfill_diff, error::BackfillError};
use crate::{
    BlockStateDiff, OpProofsBackfillProvider, OpProofsBackfillStore, OpProofsProviderRO,
    proof::DatabaseStateRoot,
};
use alloy_eips::{NumHash, eip1898::BlockWithParent};
use alloy_primitives::BlockNumber;
use derive_more::Constructor;
use reth_primitives_traits::AlloyBlockHeader;
use reth_provider::{
    BlockHashReader, BlockNumReader, ChangeSetReader, DBProvider, HeaderProvider, ProviderError,
    StageCheckpointReader, StorageChangeSetReader, StorageSettingsCache,
};
use reth_trie::StateRoot;
use reth_trie_common::HashedPostState;
use std::time::{Duration, Instant};
use tracing::info;

/// How often to emit a progress line during a long backfill, measured in
/// blocks committed.
const LOG_EVERY: u64 = 1_000;

/// Run a fallible closure and return its value alongside the wall-clock
/// duration on success. Errors are propagated; the duration is not returned
/// when the closure fails.
#[inline]
fn timed<F, R, E>(f: F) -> Result<(R, Duration), E>
where
    F: FnOnce() -> Result<R, E>,
{
    let start = Instant::now();
    let r = f()?;
    Ok((r, start.elapsed()))
}

/// Cumulative time spent in each phase of [`BackfillJob::backfill_block`].
/// Reported alongside the progress line so operators can see which phase
/// dominates a slow backfill.
#[derive(Debug, Default, Clone, Copy)]
struct PhaseTimings {
    compute: Duration,
    prepend: Duration,
    validate: Duration,
    commit: Duration,
}

impl PhaseTimings {
    fn add(&mut self, other: Self) {
        self.compute += other.compute;
        self.prepend += other.prepend;
        self.validate += other.validate;
        self.commit += other.commit;
    }

    /// Per-block average. `done` must be > 0.
    fn averages(&self, done: u64) -> Self {
        let n = done as u32;
        Self {
            compute: self.compute / n,
            prepend: self.prepend / n,
            validate: self.validate / n,
            commit: self.commit / n,
        }
    }
}

/// Backfill job for proofs storage.
#[derive(Debug, Constructor)]
pub struct BackfillJob<P, S: OpProofsBackfillStore + Send> {
    provider: P,
    storage: S,
}

impl<P, S> BackfillJob<P, S>
where
    P: DBProvider
        + StageCheckpointReader
        + ChangeSetReader
        + StorageChangeSetReader
        + BlockNumReader
        + BlockHashReader
        + HeaderProvider
        + StorageSettingsCache
        + Send,
    S: OpProofsBackfillStore + Send,
{
    /// Backfill proofs data down to `target_earliest_block`.
    ///
    /// Extends the stored proof window from `[earliest, latest]` backward to
    /// `[target_earliest_block, latest]`. Each block is committed atomically so
    /// the job is restart-safe: on crash, resume from the current `earliest`.
    ///
    /// Returns immediately if `target_earliest_block >= current earliest`.
    pub fn run(&self, target_earliest_block: u64) -> Result<(), BackfillError> {
        let current_earliest = self.storage.provider_ro()?.get_earliest_block()?;

        if target_earliest_block >= current_earliest.number {
            return Ok(());
        }

        let total = current_earliest.number - target_earliest_block;
        let start = Instant::now();
        let mut phase_totals = PhaseTimings::default();
        info!(
            target: "trie::backfill::job",
            from = current_earliest.number,
            to = target_earliest_block,
            total,
            "Starting proofs backfill"
        );

        for block_number in (target_earliest_block + 1..=current_earliest.number).rev() {
            phase_totals.add(self.backfill_block(block_number)?);

            let done = current_earliest.number - block_number + 1;
            let is_final = block_number == target_earliest_block + 1;
            if done.is_multiple_of(LOG_EVERY) || is_final {
                let elapsed_secs = start.elapsed().as_secs_f64();
                let blocks_per_sec =
                    if elapsed_secs.is_normal() { done as f64 / elapsed_secs } else { 0.0 };
                let eta_secs = if blocks_per_sec.is_normal() && blocks_per_sec > 0.0 {
                    (total - done) as f64 / blocks_per_sec
                } else {
                    0.0
                };
                let progress_pct = (done as f64 / total as f64) * 100.0;
                let avg = phase_totals.averages(done);
                info!(
                    target: "trie::backfill::job",
                    done,
                    total,
                    avg_compute = ?avg.compute,
                    avg_prepend = ?avg.prepend,
                    avg_validate = ?avg.validate,
                    "progress: {progress_pct:.2}% ({blocks_per_sec:.1} blk/s, ETA {eta_secs:.0}s)"
                );
            }
        }

        let final_avg = phase_totals.averages(total);
        info!(
            target: "trie::backfill::job",
            blocks = total,
            elapsed = ?start.elapsed(),
            avg_compute = ?final_avg.compute,
            avg_prepend = ?final_avg.prepend,
            avg_validate = ?final_avg.validate,
            "Proofs backfill complete"
        );

        Ok(())
    }

    /// Backfill a single block `E`: write its historical records and advance `earliest` to `E-1`.
    ///
    /// Returns the wall-clock time spent in each phase, accumulated by
    /// [`Self::run`] into the running averages it reports.
    fn backfill_block(&self, block_number: BlockNumber) -> Result<PhaseTimings, BackfillError> {
        let block_ref = self.resolve_block_ref(block_number)?;
        let (diff, compute) = self.compute_diff(block_number)?;

        let bp = self.storage.backfill_provider()?;
        let (_, prepend) = timed(|| bp.prepend_block(block_ref, diff))?;
        let validate = self.validate_state_root(&bp, block_number)?;
        let (_, commit) = timed(|| bp.commit())?;

        Ok(PhaseTimings { compute, prepend, validate, commit })
    }

    /// Resolve the `(block, parent)` hashes for `block_number` from reth.
    fn resolve_block_ref(
        &self,
        block_number: BlockNumber,
    ) -> Result<BlockWithParent, BackfillError> {
        let block_hash = self
            .provider
            .block_hash(block_number)?
            .ok_or_else(|| ProviderError::HeaderNotFound(block_number.into()))?;
        let parent_hash = self
            .provider
            .block_hash(block_number - 1)?
            .ok_or_else(|| ProviderError::HeaderNotFound((block_number - 1).into()))?;
        Ok(BlockWithParent { block: NumHash::new(block_number, block_hash), parent: parent_hash })
    }

    /// Compute the per-block backfill diff (trie node + leaf before-values)
    /// and time the call.
    ///
    /// Opens a fresh RO proofs provider for this iteration: it sees writes
    /// committed by the previous `prepend_block`, so its cursor at max=N
    /// already reflects state@N. The RO tx is dropped before the caller
    /// opens the rw `backfill_provider` to avoid holding two transactions
    /// against the same env.
    fn compute_diff(
        &self,
        block_number: BlockNumber,
    ) -> Result<(BlockStateDiff, Duration), BackfillError> {
        timed(|| {
            let proofs_ro = self.storage.provider_ro()?;
            let (trie_updates, post_state) =
                compute_block_backfill_diff(&self.provider, proofs_ro, block_number)?;
            Ok(BlockStateDiff { sorted_trie_updates: trie_updates, sorted_post_state: post_state })
        })
    }

    /// Validate the just-prepended diff by computing a full state root at
    /// `block_number - 1` against the open backfill provider (which sees its
    /// own uncommitted writes) and comparing to the reth header.
    fn validate_state_root<BP>(
        &self,
        bp: &BP,
        block_number: BlockNumber,
    ) -> Result<Duration, BackfillError>
    where
        BP: OpProofsProviderRO,
    {
        let (_, elapsed) = timed(|| -> Result<(), BackfillError> {
            let expected_root = self
                .provider
                .header_by_number(block_number - 1)?
                .ok_or_else(|| ProviderError::HeaderNotFound((block_number - 1).into()))?
                .state_root();
            let computed_root =
                StateRoot::overlay_root(bp, block_number - 1, HashedPostState::default())?;
            if computed_root != expected_root {
                return Err(BackfillError::StateRootMismatch {
                    block_number,
                    computed: computed_root,
                    expected: expected_root,
                });
            }
            Ok(())
        })?;
        Ok(elapsed)
    }
}
