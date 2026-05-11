//! Per-block backfill diff computation.
//!
//! For block N, [`compute_block_backfill_diff`] returns:
//! - `HashedPostStateSorted` — per-block leaf revert (account & storage values
//!   before block N ran). Read directly from reth's `AccountChangeSets` /
//!   `StorageChangeSets` and reused as `BlockStateDiff::sorted_post_state`.
//! - `TrieUpdatesSorted` — trie-node before-values for paths block N touched.
//!   Written into the four changeset tables by `prepend_block`.
//!
//! # Algorithm
//!
//! Conceptually equivalent to
//! `reth_trie_db::changesets::compute_block_trie_changesets_inner`, but reads
//! the trie from the **op-reth proofs storage** at `max_block_number = N`
//! instead of from reth's current-state tables. That swap is what makes the
//! per-block cost scale with `k_N` (this block's diff) rather than `K_tail`
//! (every changeset entry between N and the DB tip).
//!
//! A single call to `overlay_root_from_nodes_with_updates` does the work:
//! - **Cursor**: `OpProofsHashedAccountCursorFactory` / `OpProofsTrieCursorFactory`
//!   at `max=N` — serves state@N.
//! - **State overlay**: the per-block leaf revert (`individual_state_revert`).
//!   Walked together with the cursor, this yields state@N-1 for every leaf
//!   block N touched.
//! - **Prefix sets**: only the paths block N's leaf revert covers.
//!
//! The returned `TrieUpdates` is the difference between trie@N (cursor view)
//! and trie@N-1 (after applying the overlay) for those paths — which is
//! exactly the trie changeset we want:
//! - branch modified at N → `(path, Some(value_at_N-1))`
//! - branch destroyed at N (existed at N-1, gone at N) → `(path, Some(value_at_N-1))`
//! - branch created at N (existed at N, gone at N-1) → `(path, None)` via `removed_nodes`

use crate::{
    OpProofsProviderRO, backfill::error::BackfillError, proof::DatabaseStateRoot,
};
use alloy_primitives::BlockNumber;
use reth_provider::{
    BlockNumReader, ChangeSetReader, DBProvider, ProviderError, StorageChangeSetReader,
    StorageSettingsCache,
};
use reth_trie::{StateRoot, TrieInput};
use reth_trie_common::{HashedPostStateSorted, updates::TrieUpdatesSorted};
use reth_trie_db::from_reverts_auto;

/// Compute the backfill diff for `block_number`: the trie-node before-values
/// for the changeset table, and the per-block leaf revert reused as
/// `BlockStateDiff::sorted_post_state`.
///
/// `proofs_provider` must reflect the proofs-storage state *at the start of
/// this iteration* — i.e. `earliest == block_number`. Callers should open a
/// fresh RO provider per iteration so it sees writes committed by the
/// previous `prepend_block`.
pub(super) fn compute_block_backfill_diff<P, R>(
    reth_provider: &P,
    proofs_provider: R,
    block_number: BlockNumber,
) -> Result<(TrieUpdatesSorted, HashedPostStateSorted), BackfillError>
where
    P: ChangeSetReader
        + StorageChangeSetReader
        + BlockNumReader
        + DBProvider
        + StorageSettingsCache,
    R: OpProofsProviderRO + Clone,
{
    // Per-block leaf revert: doubles as `post_state` for `prepend_block` and
    // as the state overlay for the trie@N-1 reconstruction below.
    let individual_state_revert = from_reverts_auto(reth_provider, block_number..=block_number)?;
    let trie_changesets = compute_trie_changesets_against_proofs(
        proofs_provider,
        block_number,
        &individual_state_revert,
    )?;
    Ok((trie_changesets, individual_state_revert))
}

fn compute_trie_changesets_against_proofs<R>(
    proofs_provider: R,
    block_number: BlockNumber,
    individual_state_revert: &HashedPostStateSorted,
) -> Result<TrieUpdatesSorted, BackfillError>
where
    R: OpProofsProviderRO + Clone,
{
    // Apply block N's leaf revert as a state overlay on top of the proofs
    // cursor at max=N, then walk just the paths block N touched. The returned
    // `TrieUpdates` describes how trie@N-1 differs from the cursor's view at
    // max=N — which is exactly the changeset:
    //   - modified branch  → (path, Some(value_at_N-1))
    //   - destroyed at N   → (path, Some(value_at_N-1))
    //   - created at N     → (path, None)   (via `removed_nodes`)
    let input = TrieInput {
        nodes: Default::default(),
        state: individual_state_revert.clone().into(),
        prefix_sets: individual_state_revert.construct_prefix_sets(),
    };
    let (_, trie_updates) = StateRoot::overlay_root_from_nodes_with_updates(
        proofs_provider,
        block_number,
        input,
    )
    .map_err(ProviderError::other)?;
    Ok(trie_updates.into_sorted())
}
