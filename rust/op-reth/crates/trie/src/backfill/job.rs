//! [`BackfillJob`] implementation.

use super::{context::BackfillContext, error::BackfillError};
use crate::{
    BlockStateDiff, OpProofsBackfillProvider, OpProofsProviderRO,
    OpProofsStore, proof::DatabaseStateRoot,
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

        let mut context = BackfillContext::initialize(&self.provider, current_earliest.number)?;
        for block_number in (target_earliest_block + 1..=current_earliest.number).rev() {
            debug_assert_eq!(
                context.current_block(),
                block_number,
                "backfill iteration desynced from rolling context"
            );
            self.backfill_block(block_number, &mut context)?;
        }

        Ok(())
    }

    /// Backfill a single block `E`: write its historical records and advance `earliest` to `E-1`.
    fn backfill_block(
        &self,
        block_number: BlockNumber,
        context: &mut BackfillContext,
    ) -> Result<(), BackfillError> {
        let block_hash = self
            .provider
            .block_hash(block_number)?
            .ok_or_else(|| ProviderError::HeaderNotFound(block_number.into()))?;
        let parent_hash = self
            .provider
            .block_hash(block_number - 1)?
            .ok_or_else(|| ProviderError::HeaderNotFound((block_number - 1).into()))?;

        // Trie node before-values + per-block leaf revert; the rolling context
        // amortizes the cumulative-revert scan across all backfill steps.
        let (trie_updates, post_state) = context.step(&self.provider)?;

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
