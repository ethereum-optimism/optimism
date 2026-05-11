//! Rolling backfill context for incremental cumulative-revert caching.
//!
//! `reth_trie_db::changesets::compute_block_trie_changesets` performs an
//! `O(K_tail)` scan of every changeset entry from `block + 1` to the DB tip on
//! each call. For descending backfill that scan grows by exactly one block per
//! iteration, so we keep the cumulative state revert in memory and extend it by
//! one block per [`BackfillContext::step`] call. The trie re-roots
//! (`overlay_root_from_nodes_with_updates`) still run per block — caching their
//! output is left for a future iteration.

use crate::backfill::error::BackfillError;
use alloy_primitives::BlockNumber;
use reth_provider::{
    BlockNumReader, ChangeSetReader, DBProvider, ProviderError, StorageChangeSetReader,
    StorageSettingsCache,
};
use reth_trie::{
    StateRoot, TrieInputSorted, changesets::compute_trie_changesets,
    trie_cursor::InMemoryTrieCursorFactory,
};
use reth_trie_common::{HashedPostStateSorted, updates::TrieUpdatesSorted};
use reth_trie_db::{
    DatabaseHashedCursorFactory, DatabaseStateRoot, DatabaseTrieCursorFactory, TrieTableAdapter,
    from_reverts_auto, with_adapter,
};
use std::sync::Arc;

/// Rolling context that incrementally maintains the cumulative state revert
/// across descending backfill iterations.
pub(super) struct BackfillContext {
    /// Cumulative state revert covering `(current_block + 1)..tip`.
    ///
    /// Applied as an overlay over DB-tip state, this yields the leaf state at the
    /// end of `current_block`. Equivalent to reth's `cumulative_state_revert` in
    /// `compute_block_trie_changesets_inner`.
    cumulative_state_revert: HashedPostStateSorted,
    /// Block we're about to process on the next [`Self::step`] call.
    current_block: BlockNumber,
}

impl BackfillContext {
    /// Bootstrap by scanning every changeset from `start_block + 1` to DB tip.
    ///
    /// This is the one-time `O(K_tail)` cost; subsequent [`Self::step`] calls
    /// extend the revert by a single block's worth of entries.
    pub(super) fn initialize<P>(
        provider: &P,
        start_block: BlockNumber,
    ) -> Result<Self, BackfillError>
    where
        P: ChangeSetReader
            + StorageChangeSetReader
            + BlockNumReader
            + DBProvider
            + StorageSettingsCache,
    {
        let cumulative_state_revert = from_reverts_auto(provider, (start_block + 1)..)?;
        Ok(Self { cumulative_state_revert, current_block: start_block })
    }

    /// The block this context will process on the next [`Self::step`] call.
    pub(super) const fn current_block(&self) -> BlockNumber {
        self.current_block
    }

    /// Compute trie changesets (before-values) and per-block leaf reverts for
    /// [`Self::current_block`], then advance state for the next descending block.
    pub(super) fn step<P>(
        &mut self,
        provider: &P,
    ) -> Result<(TrieUpdatesSorted, HashedPostStateSorted), BackfillError>
    where
        P: ChangeSetReader
            + StorageChangeSetReader
            + BlockNumReader
            + DBProvider
            + StorageSettingsCache,
    {
        let block_number = self.current_block;

        // Per-block leaf revert (also doubles as the post-state for prepend_block).
        let individual_state_revert = from_reverts_auto(provider, block_number..=block_number)?;

        // cumulative_state_revert_prev = state at end of (block_number - 1).
        let mut cumulative_state_revert_prev = self.cumulative_state_revert.clone();
        cumulative_state_revert_prev.extend_ref_and_sort(&individual_state_revert);

        let changesets = with_adapter!(provider, |A| compute_changesets_with_reverts::<_, A>(
            provider,
            self.cumulative_state_revert.clone(),
            cumulative_state_revert_prev,
            &individual_state_revert,
        ))?;

        // Advance: extend cumulative revert by this block, then decrement.
        self.cumulative_state_revert.extend_ref_and_sort(&individual_state_revert);
        self.current_block = block_number.saturating_sub(1);

        Ok((changesets, individual_state_revert))
    }
}

type DbStateRoot<'a, TX, A> =
    StateRoot<DatabaseTrieCursorFactory<&'a TX, A>, DatabaseHashedCursorFactory<&'a TX>>;

/// Adapted from `reth_trie_db::changesets::compute_block_trie_changesets_inner`.
///
/// Caller supplies pre-built reverts so the changeset table scans are amortized
/// across all descending backfill steps. Otherwise mirrors steps #6 (trie at
/// `block - 1`), #9 (trie updates at `block`), and #11 (diff to extract
/// before-values) of reth's algorithm verbatim.
fn compute_changesets_with_reverts<P, A>(
    provider: &P,
    cumulative_state_revert: HashedPostStateSorted,
    cumulative_state_revert_prev: HashedPostStateSorted,
    individual_state_revert: &HashedPostStateSorted,
) -> Result<TrieUpdatesSorted, ProviderError>
where
    P: DBProvider + StorageSettingsCache,
    A: TrieTableAdapter,
{
    // #6 — trie state at end of (block - 1).
    let prefix_sets_prev = cumulative_state_revert_prev.construct_prefix_sets();
    let input_prev = TrieInputSorted::new(
        Arc::default(),
        Arc::new(cumulative_state_revert_prev),
        prefix_sets_prev,
    );
    let cumulative_trie_updates_prev =
        DbStateRoot::<_, A>::overlay_root_from_nodes_with_updates(provider.tx_ref(), input_prev)
            .map_err(ProviderError::other)?
            .1
            .into_sorted();

    // #9 — trie updates at end of block (deltas applied during block).
    let prefix_sets = individual_state_revert.construct_prefix_sets();
    let input = TrieInputSorted::new(
        Arc::new(cumulative_trie_updates_prev.clone()),
        Arc::new(cumulative_state_revert),
        prefix_sets,
    );
    let trie_updates =
        DbStateRoot::<_, A>::overlay_root_from_nodes_with_updates(provider.tx_ref(), input)
            .map_err(ProviderError::other)?
            .1
            .into_sorted();

    // #11 — look up before-values via the `block - 1` overlay.
    let db_cursor_factory = DatabaseTrieCursorFactory::<_, A>::new(provider.tx_ref());
    let overlay_factory =
        InMemoryTrieCursorFactory::new(db_cursor_factory, &cumulative_trie_updates_prev);
    compute_trie_changesets(&overlay_factory, &trie_updates).map_err(ProviderError::other)
}
