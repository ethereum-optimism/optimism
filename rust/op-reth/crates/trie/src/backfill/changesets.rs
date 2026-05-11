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
//! The trie-changeset computation mirrors the algorithm in
//! `reth_trie_db::changesets::compute_block_trie_changesets_inner` but reads
//! the trie from the **op-reth proofs storage** at `max_block_number = N`
//! instead of from reth's current-state tables. That swap is what makes the
//! per-block cost scale with `k_N` (this one block's diff) rather than
//! `K_tail` (every changeset entry between N and the DB tip).
//!
//! Three passes against `OpProofsTrieCursorFactory` / `OpProofsHashedAccountCursorFactory`
//! at `max = block_number`:
//!
//! 1. **Reconstruct trie@N-1.** `overlay_root_from_nodes_with_updates` with the
//!    per-block leaf revert as the *state* overlay, no node overlay, and
//!    prefix sets covering block N's affected paths. The proofs cursor serves
//!    state@N; the overlay walks it back to state@N-1 only for the paths block
//!    N changed. Output: `cumulative_trie_updates_prev` — the branch nodes as
//!    they existed at the end of N-1, expressed as a delta from the proofs
//!    cursor's view at max=N.
//!
//! 2. **Discover which branch paths block N touched.** The same call with no
//!    overlays — just prefix sets + the proofs cursor at max=N. The returned
//!    `TrieUpdates` is the set of (path, node@N) pairs for branch nodes that
//!    the Merkle walk had to compute. We don't care about the node *values*
//!    in this output; we only use the path list to drive the diff in step 3.
//!
//! 3. **Look up before-values.** For each path from step 2, seek into the
//!    trie@N-1 view: `InMemoryTrieCursorFactory` layering the step-1 delta
//!    over `OpProofsTrieCursorFactory` at max=N. The result is the (path,
//!    before-value) tuples that `prepend_block` writes to the changeset
//!    tables.

use crate::{
    OpProofsProviderRO, backfill::error::BackfillError,
    cursor_factory::OpProofsTrieCursorFactory, proof::DatabaseStateRoot,
};
use alloy_primitives::BlockNumber;
use reth_provider::{
    BlockNumReader, ChangeSetReader, DBProvider, ProviderError, StorageChangeSetReader,
    StorageSettingsCache,
};
use reth_trie::{
    StateRoot, TrieInput, changesets::compute_trie_changesets,
    trie_cursor::InMemoryTrieCursorFactory,
};
use reth_trie_common::{HashedPostState, HashedPostStateSorted, updates::TrieUpdatesSorted};
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
    let prefix_sets = individual_state_revert.construct_prefix_sets();
    let state_overlay: HashedPostState = individual_state_revert.clone().into();

    // Pass 1 — reconstruct trie@N-1.
    // Apply the per-block leaf revert on top of the proofs cursor at max=N;
    // walk only the paths block N touched. Output is the branch-node deltas
    // needed to land at state@N-1.
    let input_prev = TrieInput {
        nodes: Default::default(),
        state: state_overlay,
        prefix_sets: prefix_sets.clone(),
    };
    let (_, trie_at_block_minus_one) = StateRoot::overlay_root_from_nodes_with_updates(
        proofs_provider.clone(),
        block_number,
        input_prev,
    )
    .map_err(ProviderError::other)?;
    let trie_at_block_minus_one = trie_at_block_minus_one.into_sorted();

    // Pass 2 — discover the branch paths block N touched.
    // Same prefix sets, no overlays — the walk against the proofs cursor at
    // max=N returns the (path, node@N) pairs for branches the Merkle
    // computation reached. We use the path list, not the node values.
    let input = TrieInput { nodes: Default::default(), state: Default::default(), prefix_sets };
    let (_, trie_updates_at_block) = StateRoot::overlay_root_from_nodes_with_updates(
        proofs_provider.clone(),
        block_number,
        input,
    )
    .map_err(ProviderError::other)?;
    let trie_updates_at_block = trie_updates_at_block.into_sorted();

    // Pass 3 — look up each path's value at N-1.
    // Layer pass-1's delta over the proofs cursor at max=N to get a cursor
    // that returns trie state at N-1. Seek each path from pass 2 and record
    // (path, before-value).
    let proofs_factory = OpProofsTrieCursorFactory::new(proofs_provider, block_number);
    let overlay_factory = InMemoryTrieCursorFactory::new(proofs_factory, &trie_at_block_minus_one);
    compute_trie_changesets(&overlay_factory, &trie_updates_at_block)
        .map_err(ProviderError::other)
        .map_err(BackfillError::Provider)
}
