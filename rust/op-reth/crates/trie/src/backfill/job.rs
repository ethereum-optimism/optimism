//! [`BackfillJob`] implementation.

use super::{changesets::compute_block_backfill_diff, error::BackfillError};
use crate::{
    BlockStateDiff, OpProofsBackfillProvider, OpProofsBackfillStore,
    OpProofsHashedAccountCursorFactory, OpProofsProviderRO, OpProofsSnapshotInitProvider,
    OpProofsSnapshotProviderRO, OpProofsTrieCursorFactory, SnapshotHashedCursorFactory,
    SnapshotInitJob, SnapshotInitStatus, SnapshotTrieCursorFactory, proof::DatabaseStateRoot,
};
use alloy_eips::{BlockNumHash, NumHash, eip1898::BlockWithParent};
use alloy_primitives::BlockNumber;
use reth_primitives_traits::AlloyBlockHeader;
use reth_provider::{
    BlockHashReader, BlockNumReader, ChangeSetReader, DBProvider, HeaderProvider, ProviderError,
    StageCheckpointReader, StorageChangeSetReader, StorageSettingsCache,
};
use reth_trie::{HashedPostState, StateRoot, hashed_cursor::HashedPostStateCursorFactory};
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

/// Default number of blocks written per MDBX transaction.
///
/// Conservative — a crash mid-batch loses at most this many blocks of work, and the in-memory
/// per-block diff buffer stays small. Operators can opt up via [`BackfillJob::with_batch_size`].
pub const DEFAULT_BACKFILL_BATCH_SIZE: usize = 10;

/// Backfill job for proofs storage.
#[derive(Debug)]
pub struct BackfillJob<P, S: OpProofsBackfillStore + Send> {
    provider: P,
    storage: S,
    /// Number of blocks written per MDBX transaction. Amortizes commit cost across blocks at the
    /// price of restart granularity. See [`DEFAULT_BACKFILL_BATCH_SIZE`].
    batch_size: usize,
}

impl<P, S: OpProofsBackfillStore + Send> BackfillJob<P, S> {
    /// Create a new backfill job using [`DEFAULT_BACKFILL_BATCH_SIZE`].
    pub const fn new(provider: P, storage: S) -> Self {
        Self { provider, storage, batch_size: DEFAULT_BACKFILL_BATCH_SIZE }
    }

    /// Override the batch size (number of blocks per MDBX transaction).
    ///
    /// `batch_size` is clamped to `1` if zero — backfill needs to make per-block progress.
    pub fn with_batch_size(mut self, batch_size: usize) -> Self {
        self.batch_size = batch_size.max(1);
        self
    }
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
    /// `[target_earliest_block, latest]`. Blocks are written in batches of
    /// [`Self::with_batch_size`] (default [`DEFAULT_BACKFILL_BATCH_SIZE`]) — each batch holds
    /// one MDBX transaction open across K blocks and commits once, amortizing fsync cost.
    /// A crash mid-batch loses at most K blocks of progress; on restart, resume from the
    /// current `earliest`.
    ///
    /// Returns immediately if `target_earliest_block >= current earliest`.
    pub fn run(&self, target_earliest_block: u64) -> Result<(), BackfillError> {
        let current_earliest = self.storage.provider_ro()?.get_earliest_block()?;

        if target_earliest_block >= current_earliest.number {
            return Ok(());
        }
        self.drive_batched_loop(current_earliest, target_earliest_block, "standard", |bp, n| {
            self.backfill_block(bp, n)
        })
    }

    /// Shared batched per-block driver, used by [`Self::run`] and [`Self::run_with_snapshot`].
    ///
    /// Holds one RW backfill provider open across `batch_size` blocks, dispatches per-block work
    /// to the supplied closure (which uses the same `bp` for reads + writes, so MDBX same-tx
    /// visibility makes in-flight writes from earlier blocks of the batch visible), and commits
    /// once per batch. Mirrors the previous per-block `drive_loop` shape but with the bp opened
    /// at the batch boundary instead of per call.
    fn drive_batched_loop<F>(
        &self,
        current_earliest: NumHash,
        target_earliest_block: u64,
        kind: &'static str,
        mut process_block: F,
    ) -> Result<(), BackfillError>
    where
        F: FnMut(&S::BackfillProvider<'_>, BlockNumber) -> Result<PhaseTimings, BackfillError>,
    {
        let total = current_earliest.number - target_earliest_block;
        let start = Instant::now();
        let mut phase_totals = PhaseTimings::default();
        info!(
            target: "trie::backfill::job",
            from = current_earliest.number,
            to = target_earliest_block,
            total,
            batch_size = self.batch_size,
            kind,
            "Starting proofs backfill"
        );

        let mut next_block = current_earliest.number;
        while next_block > target_earliest_block {
            let batch_end =
                next_block.saturating_sub(self.batch_size as u64).max(target_earliest_block);
            let batch_low = batch_end + 1;
            let batch_high = next_block;

            let bp = self.storage.backfill_provider()?;

            for block_number in (batch_low..=batch_high).rev() {
                let timings = process_block(&bp, block_number)?;
                phase_totals.add(timings);

                let done = current_earliest.number - block_number + 1;
                let is_final = block_number == target_earliest_block + 1;
                if done.is_multiple_of(LOG_EVERY) || is_final {
                    self.log_progress(start, done, total, &phase_totals);
                }
            }

            let (_, commit_duration) = timed(|| bp.commit())?;
            // Amortize the batch's commit time across the blocks in it so per-block averages
            // remain comparable across batch sizes.
            phase_totals.commit += commit_duration;
            next_block = batch_end;
        }

        let final_avg = phase_totals.averages(total);
        info!(
            target: "trie::backfill::job",
            blocks = total,
            elapsed = ?start.elapsed(),
            avg_compute = ?final_avg.compute,
            avg_prepend = ?final_avg.prepend,
            avg_validate = ?final_avg.validate,
            avg_commit = ?final_avg.commit,
            kind,
            "Proofs backfill complete"
        );

        Ok(())
    }

    /// Per-block work for the standard backfill path. Called inside [`Self::drive_batched_loop`]
    /// with the batch's shared open RW provider; reads in [`Self::compute_diff_via`] go through
    /// it so same-tx writes from earlier iterations are visible.
    fn backfill_block(
        &self,
        bp: &S::BackfillProvider<'_>,
        block_number: BlockNumber,
    ) -> Result<PhaseTimings, BackfillError> {
        let block_ref = self.resolve_block_ref(block_number)?;
        let (diff, compute) = self.compute_diff_via(bp, block_number)?;
        let (_, prepend) = timed(|| bp.prepend_block(block_ref, diff))?;
        let validate = self.validate_state_root(bp, block_number)?;
        Ok(PhaseTimings { compute, prepend, validate, commit: Duration::ZERO })
    }

    fn log_progress(&self, start: Instant, done: u64, total: u64, phase_totals: &PhaseTimings) {
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
            avg_commit = ?avg.commit,
            "progress: {progress_pct:.2}% ({blocks_per_sec:.1} blk/s, ETA {eta_secs:.0}s)"
        );
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

    /// Compute the per-block backfill diff (trie node + leaf before-values) and time the call.
    ///
    /// The batched [`Self::run`] passes the open RW backfill provider so cursors see writes
    /// made earlier in the same MDBX transaction (the prepended blocks of this batch).
    /// MDBX same-tx visibility means each `compute_diff_via(&bp, N)` sees the in-flight
    /// `prepend_block` writes for blocks > N.
    fn compute_diff_via<RO>(
        &self,
        proofs_ro: &RO,
        block_number: BlockNumber,
    ) -> Result<(BlockStateDiff, Duration), BackfillError>
    where
        RO: OpProofsProviderRO,
    {
        timed(|| {
            // History-aware cursors at `max_block_number = block_number`.
            let trie_factory = OpProofsTrieCursorFactory::new(proofs_ro, block_number);
            let hashed_factory = OpProofsHashedAccountCursorFactory::new(proofs_ro, block_number);
            let (trie_updates, post_state) = compute_block_backfill_diff(
                &self.provider,
                trie_factory,
                hashed_factory,
                block_number,
            )?;
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
        + Send
        + Sync,
    S: OpProofsBackfillStore + Clone + Send,
{
    /// Backfill using a `Ready` snapshot to accelerate per-block reads.
    ///
    /// Mirrors [`Self::run`] but reads trie state at each iteration's
    /// `block_number` from the snapshot tables instead of the V2 merge-walk,
    /// and advances the snapshot anchor atomically with each `prepend_block`
    /// (so the snapshot stays in sync with the moving `earliest`).
    ///
    /// **Snapshot preconditions** (handled internally by `ensure_snapshot_ready`):
    /// - If no snapshot exists, runs [`SnapshotInitJob`] at the current `earliest` and then
    ///   proceeds.
    /// - If a `Ready` snapshot exists at the current `earliest`, proceeds.
    /// - Otherwise (snapshot at a different anchor, or partial build in progress at a different
    ///   anchor), errors out and asks the caller to drop or finish the snapshot.
    pub fn run_with_snapshot(&self, target_earliest_block: u64) -> Result<(), BackfillError> {
        let current_earliest = self.storage.provider_ro()?.get_earliest_block()?;
        if target_earliest_block >= current_earliest.number {
            return Ok(());
        }
        self.ensure_snapshot_ready(current_earliest)?;
        self.drive_batched_loop(
            current_earliest,
            target_earliest_block,
            "snapshot-accelerated",
            |bp, n| self.backfill_block_with_snapshot(bp, n),
        )
    }

    /// Ensure a `Ready` snapshot exists at `current_earliest`.
    ///
    /// - `Completed` at matching anchor → ok, return.
    /// - `Completed` at a different anchor → [`BackfillError::SnapshotAnchorMismatch`].
    /// - `NotStarted` / `InProgress` → delegate to [`SnapshotInitJob`] (which handles fresh build
    ///   and crash-resume; errors on drift).
    fn ensure_snapshot_ready(&self, current_earliest: NumHash) -> Result<(), BackfillError> {
        let target = BlockNumHash::new(current_earliest.number, current_earliest.hash);
        let init_anchor =
            self.storage.snapshot_initialization_provider()?.snapshot_init_anchor()?;

        if let (SnapshotInitStatus::Completed, Some(existing)) =
            (init_anchor.status, init_anchor.block)
        {
            if existing == target {
                return Ok(());
            }
            return Err(BackfillError::SnapshotAnchorMismatch {
                expected: target,
                found: existing,
            });
        }

        info!(
            target: "trie::backfill::job",
            anchor = ?target,
            status = ?init_anchor.status,
            "Bootstrapping snapshot before backfill"
        );

        SnapshotInitJob::new(&self.provider, self.storage.clone()).run(current_earliest.number)?;
        Ok(())
    }

    /// Per-block work for the snapshot-accelerated path. Reads via the open RW provider's
    /// snapshot cursors (which see in-flight `update_snapshot` writes via MDBX same-tx
    /// visibility), then advances snapshot anchor + proofs window in the same tx, then
    /// validates against reth's header at `E-1`.
    fn backfill_block_with_snapshot(
        &self,
        bp: &S::BackfillProvider<'_>,
        block_number: BlockNumber,
    ) -> Result<PhaseTimings, BackfillError> {
        let block_ref = self.resolve_block_ref(block_number)?;
        let (diff, compute) = self.compute_diff_with_snapshot_via(bp, block_number)?;

        // After this iteration the proofs window's earliest moves from E to E-1, so the
        // snapshot anchor advances to the parent block.
        let new_anchor = BlockNumHash::new(block_number - 1, block_ref.parent);

        // Advance snapshot anchor + proofs window in the same tx.
        let (_, prepend) = timed(|| -> Result<(), BackfillError> {
            bp.update_snapshot(new_anchor, &diff)?;
            bp.prepend_block(block_ref, diff)?;
            Ok(())
        })?;

        let validate = self.validate_state_root_with_snapshot(bp, block_number)?;
        Ok(PhaseTimings { compute, prepend, validate, commit: Duration::ZERO })
    }

    /// Compute the per-block backfill diff using snapshot trie + leaf cursors.
    ///
    /// The batched [`Self::run_batched_with_snapshot`] passes the open RW backfill provider
    /// (which implements [`OpProofsSnapshotProviderRO`] via the trait hierarchy), so cursors
    /// see the in-flight `update_snapshot` writes from earlier blocks in the same MDBX
    /// transaction.
    fn compute_diff_with_snapshot_via<SP>(
        &self,
        sp: &SP,
        block_number: BlockNumber,
    ) -> Result<(BlockStateDiff, Duration), BackfillError>
    where
        SP: OpProofsSnapshotProviderRO,
    {
        // `block_number` is unused on the snapshot path: the snapshot reflects state at its
        // anchor, which the caller guaranteed equals `block_number` via `ensure_snapshot_ready`
        // (or via the prior iteration's `update_snapshot` advancing the anchor).
        let _ = block_number;
        timed(|| {
            let trie_factory = SnapshotTrieCursorFactory::new(sp);
            let hashed_factory = SnapshotHashedCursorFactory::new(sp);
            let (sorted_trie_updates, sorted_post_state) = compute_block_backfill_diff(
                &self.provider,
                trie_factory,
                hashed_factory,
                block_number,
            )?;
            Ok(BlockStateDiff { sorted_trie_updates, sorted_post_state })
        })
    }

    /// Validate the just-prepended state at `block_number - 1` using snapshot
    /// cursors (the snapshot has been updated to anchor at `E-1` within the
    /// same tx, so its reads reflect the new state).
    fn validate_state_root_with_snapshot<BP>(
        &self,
        bp: &BP,
        block_number: BlockNumber,
    ) -> Result<Duration, BackfillError>
    where
        BP: OpProofsSnapshotProviderRO,
    {
        let (_, elapsed) = timed(|| -> Result<(), BackfillError> {
            let expected_root = self
                .provider
                .header_by_number(block_number - 1)?
                .ok_or_else(|| ProviderError::HeaderNotFound((block_number - 1).into()))?
                .state_root();

            let state_sorted = HashedPostState::default().into_sorted();
            let computed_root = StateRoot::new(
                SnapshotTrieCursorFactory::new(bp),
                HashedPostStateCursorFactory::new(
                    SnapshotHashedCursorFactory::new(bp),
                    &state_sorted,
                ),
            )
            .root()?;

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
