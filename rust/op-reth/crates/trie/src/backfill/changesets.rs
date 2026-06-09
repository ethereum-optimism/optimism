//! Per-block backfill diff computation. See [`compute_block_backfill_diff`].
//!
//! Equivalent to `reth_trie_db::changesets::compute_block_trie_changesets_inner`,
//! but reads the trie via caller-supplied cursor factories — typically a
//! history-aware factory at `max_block_number = N` over the **op-reth proofs
//! storage** (making per-block cost scale with the block's diff size, not the
//! tail size of changesets between N and the DB tip), or a snapshot-backed
//! factory when a [`SnapshotStatus::Ready`](crate::db::SnapshotStatus::Ready)
//! snapshot is available at the right anchor.

use crate::backfill::error::BackfillError;
use alloy_primitives::BlockNumber;
use reth_provider::{
    BlockNumReader, ChangeSetReader, DBProvider, StorageChangeSetReader, StorageSettingsCache,
};
use reth_trie::{
    StateRoot,
    hashed_cursor::{HashedCursorFactory, HashedPostStateCursorFactory},
    trie_cursor::TrieCursorFactory,
};
use reth_trie_common::{HashedPostStateSorted, updates::TrieUpdatesSorted};
use reth_trie_db::from_reverts_auto;

/// Compute the backfill diff for `block_number`:
/// - `HashedPostStateSorted` — per-block leaf revert (state before block N ran), reused as
///   `BlockStateDiff::sorted_post_state`.
/// - `TrieUpdatesSorted` — trie-node before-values for paths block N touched, written into the four
///   `V2*TrieChangeSets` tables by `prepend_block`.
///
/// The trie + hashed cursor factories must read trie state **at the start of
/// this iteration** (`earliest == block_number`). For the history-aware path
/// this means cursors built with `max_block_number = block_number`; for the
/// snapshot path the snapshot's anchor must equal `block_number`.
pub(super) fn compute_block_backfill_diff<P, T, H>(
    reth_provider: &P,
    trie_cursor_factory: T,
    hashed_cursor_factory: H,
    block_number: BlockNumber,
) -> Result<(TrieUpdatesSorted, HashedPostStateSorted), BackfillError>
where
    P: ChangeSetReader
        + StorageChangeSetReader
        + BlockNumReader
        + DBProvider
        + StorageSettingsCache,
    T: TrieCursorFactory + Clone,
    H: HashedCursorFactory + Clone,
{
    // Per-block leaf revert: doubles as `post_state` for `prepend_block` and
    // as the state overlay for the trie@N-1 reconstruction below.
    let individual_state_revert = from_reverts_auto(reth_provider, block_number..=block_number)?;
    let trie_changesets = compute_trie_changesets_against_proofs(
        trie_cursor_factory,
        hashed_cursor_factory,
        &individual_state_revert,
    )?;
    Ok((trie_changesets, individual_state_revert))
}

fn compute_trie_changesets_against_proofs<T, H>(
    trie_cursor_factory: T,
    hashed_cursor_factory: H,
    individual_state_revert: &HashedPostStateSorted,
) -> Result<TrieUpdatesSorted, BackfillError>
where
    T: TrieCursorFactory + Clone,
    H: HashedCursorFactory + Clone,
{
    // Apply block N's leaf revert as a state overlay on top of the supplied
    // trie cursor at max=N. The returned `TrieUpdates` describes trie@N-1
    // relative to the cursor's view at max=N for the paths block N touched —
    // which is the changeset we want (`Some` for modified/destroyed, `None`
    // for newly created branches).
    let prefix_sets = individual_state_revert.construct_prefix_sets().freeze();
    let (_, trie_updates) = StateRoot::new(
        trie_cursor_factory,
        HashedPostStateCursorFactory::new(hashed_cursor_factory, individual_state_revert),
    )
    .with_prefix_sets(prefix_sets)
    .root_with_updates()?;
    Ok(trie_updates.into_sorted())
}
