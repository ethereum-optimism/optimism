//! Per-block backfill diff computation. See [`compute_block_backfill_diff`].
//!
//! Equivalent to `reth_trie_db::changesets::compute_block_trie_changesets_inner`,
//! but reads the trie from the **op-reth proofs storage** at `max_block_number = N`
//! instead of reth's current-state tables — making per-block cost scale with
//! the block's diff size, not the tail size of changesets between N and the DB tip.

use crate::{OpProofsProviderRO, backfill::error::BackfillError, proof::DatabaseStateRoot};
use alloy_primitives::BlockNumber;
use reth_provider::{
    BlockNumReader, ChangeSetReader, DBProvider, ProviderError, StorageChangeSetReader,
    StorageSettingsCache,
};
use reth_trie::{StateRoot, TrieInput};
use reth_trie_common::{HashedPostStateSorted, updates::TrieUpdatesSorted};
use reth_trie_db::from_reverts_auto;

/// Compute the backfill diff for `block_number`:
/// - `HashedPostStateSorted` — per-block leaf revert (state before block N ran), reused as
///   `BlockStateDiff::sorted_post_state`.
/// - `TrieUpdatesSorted` — trie-node before-values for paths block N touched, written into the four
///   `V2*TrieChangeSets` tables by `prepend_block`.
///
/// `proofs_provider` must reflect proofs-storage state at the start of this
/// iteration (`earliest == block_number`); callers open a fresh RO provider
/// per iteration so it sees writes from the previous `prepend_block`.
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
    // cursor at max=N. The returned `TrieUpdates` describes trie@N-1 relative
    // to the cursor's view at max=N for the paths block N touched — which is
    // the changeset we want (`Some` for modified/destroyed, `None` for newly
    // created branches).
    let input = TrieInput {
        nodes: Default::default(),
        state: individual_state_revert.clone().into(),
        prefix_sets: individual_state_revert.construct_prefix_sets(),
    };
    let (_, trie_updates) =
        StateRoot::overlay_root_from_nodes_with_updates(proofs_provider, block_number, input)
            .map_err(ProviderError::other)?;
    Ok(trie_updates.into_sorted())
}
