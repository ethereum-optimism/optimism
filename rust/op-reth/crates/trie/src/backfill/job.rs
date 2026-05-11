//! [`BackfillJob`] implementation.

use super::{changesets::compute_block_backfill_diff, error::BackfillError};
use crate::{
    BlockStateDiff, OpProofsBackfillProvider, OpProofsProviderRO, OpProofsStore,
    proof::DatabaseStateRoot,
};
use alloy_eips::{BlockNumHash, eip1898::BlockWithParent};
use alloy_primitives::BlockNumber;
use derive_more::Constructor;
use reth_primitives_traits::AlloyBlockHeader;
use reth_provider::{
    BlockHashReader, BlockNumReader, ChangeSetReader, DBProvider, HeaderProvider, ProviderError,
    StageCheckpointReader, StorageChangeSetReader, StorageSettingsCache,
};
use reth_trie::StateRoot;
use reth_trie_common::HashedPostState;
use std::time::Instant;
use tracing::info;

/// How often to emit a progress line during a long backfill, measured in
/// blocks committed.
const LOG_EVERY: u64 = 1_000;

/// Backfill job for proofs storage.
#[derive(Debug, Constructor)]
pub struct BackfillJob<P, S: OpProofsStore + Send> {
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
    S: OpProofsStore + Send,
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
        info!(
            target: "reth::op-proofs::backfill",
            from = current_earliest.number,
            to = target_earliest_block,
            total,
            "Starting proofs backfill"
        );

        for block_number in (target_earliest_block + 1..=current_earliest.number).rev() {
            self.backfill_block(block_number)?;

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
                info!(
                    target: "reth::op-proofs::backfill",
                    done,
                    total,
                    "progress: {progress_pct:.2}% ({blocks_per_sec:.1} blk/s, ETA {eta_secs:.0}s)"
                );
            }
        }

        info!(
            target: "reth::op-proofs::backfill",
            blocks = total,
            elapsed = ?start.elapsed(),
            "Proofs backfill complete"
        );

        Ok(())
    }

    /// Backfill a single block `E`: write its historical records and advance `earliest` to `E-1`.
    fn backfill_block(&self, block_number: BlockNumber) -> Result<(), BackfillError> {
        let block_hash = self
            .provider
            .block_hash(block_number)?
            .ok_or_else(|| ProviderError::HeaderNotFound(block_number.into()))?;
        let parent_hash = self
            .provider
            .block_hash(block_number - 1)?
            .ok_or_else(|| ProviderError::HeaderNotFound((block_number - 1).into()))?;

        // Open a fresh RO proofs provider for this iteration: it sees writes
        // committed by the previous `prepend_block`, so its cursor at max=N
        // already reflects state@N. Dropped before opening the RW backfill
        // provider below to avoid holding two transactions on the same env.
        let trie_updates;
        let post_state;
        {
            let proofs_ro = self.storage.provider_ro()?;
            (trie_updates, post_state) =
                compute_block_backfill_diff(&self.provider, &proofs_ro, block_number)?;
        }

        let block_ref = BlockWithParent {
            block: BlockNumHash::new(block_number, block_hash),
            parent: parent_hash,
        };

        let bp = self.storage.backfill_provider()?;
        bp.prepend_block(
            block_ref,
            BlockStateDiff { sorted_trie_updates: trie_updates, sorted_post_state: post_state },
        )?;

        // Validate the written before-values by computing a full state root at block_number - 1
        // using the backfill provider (which now includes the prepended data in its transaction).
        // `&bp` implements `OpProofsProviderRO`, so it reads its own uncommitted writes.
        let expected_root = self
            .provider
            .header_by_number(block_number - 1)?
            .ok_or_else(|| ProviderError::HeaderNotFound((block_number - 1).into()))?
            .state_root();
        let computed_root =
            StateRoot::overlay_root(&bp, block_number - 1, HashedPostState::default())?;
        if computed_root != expected_root {
            return Err(BackfillError::StateRootMismatch {
                block_number,
                computed: computed_root,
                expected: expected_root,
            });
        }

        bp.commit()?;

        Ok(())
    }
}
