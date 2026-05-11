//! Backfill job for extending proofs storage window backward.
//!
//! # Backfill plan (v2)
//!
//! Goal: extend proofs window from `[earliest, latest]` to
//! `[target_earliest, latest]` where `target_earliest < earliest`.
//!
//! ## Core boundary semantics
//!
//! - `earliest` is a **base-state boundary**, not "oldest block with its own changeset rows".
//! - To move boundary from `E` to `E-1`, we must materialize block `E` historical records
//!   (changesets + history bitmap entries), then set `earliest = E-1`.
//! - This mirrors prune behavior in reverse.
//!
//! ## Per-step algorithm (descending)
//!
//! For each `E` from current earliest down to `target_earliest + 1`:
//! 1. Build historical trie/state views from reth DB (no block execution):
//!    - `before_E`: state at end of `E-1` (start of block `E`)
//!    - `after_E`: state at end of `E` (start of block `E+1`)
//! 2. Derive block `E` changes:
//!    - leaf (hashed account/storage) before-values from reth `AccountChangeSets` /
//!      `StorageChangeSets` tables
//!    - trie node before-values via [`ChangesetCache::get_or_compute`]
//! 3. Write block `E` records into proofs history tables.
//! 4. Atomically move earliest marker to `E-1` and commit.
//!
//! ## Data sources
//!
//! - Leaf before-values come from reth changesets (`account_block_changeset` /
//!   `storage_changeset`).
//! - Trie node before-values come from [`ChangesetCache`], which uses
//!   `compute_block_trie_changesets` as a DB fallback.
//! - A shared [`ChangesetCache`] is kept across the whole backfill run so that blocks that are warm
//!   in cache are not recomputed.
//!
//! ## Write invariants
//!
//! - Do not rewind proofs current-state tables; they remain at `latest`.
//! - Backfill writes are prepend-style history inserts for older blocks.
//! - Each step must be idempotent/restart-safe: after crash, resume from current `earliest`.
//!
//! ## Validation per step
//!
//! - Window remains contiguous after commit.
//! - `earliest` decreases by exactly one per successful step.
//! - Historical reads at new boundary succeed, while reads below boundary fail as expected.

mod changesets;
mod error;
mod job;

#[cfg(test)]
mod tests;

pub use error::BackfillError;
pub use job::BackfillJob;
