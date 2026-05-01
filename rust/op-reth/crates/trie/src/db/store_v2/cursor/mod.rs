//! V2 cursor implementations for the v2 table schema.
//!
//! These cursors implement **history-aware reads** using the v2 3-table-per-data-type pattern:
//!
//! | Purpose | Accounts | Storages | Account Trie | Storage Trie |
//! |---------|----------|----------|-------------|-------------|
//! | Current state | `V2HashedAccounts` | `V2HashedStorages` | `V2AccountsTrie` | `V2StoragesTrie` |
//! | `ChangeSets` | `V2HashedAccountChangeSets` | `V2HashedStorageChangeSets` | `V2AccountTrieChangeSets` | `V2StorageTrieChangeSets` |
//! | History | `V2HashedAccountsHistory` | `V2HashedStoragesHistory` | `V2AccountsTrieHistory` | `V2StoragesTrieHistory` |
//!
//! # Historical Lookup Strategy
//!
//! Each cursor accepts a `max_block_number` parameter. For each key encountered:
//!
//! 1. **History bitmap lookup**: Seek `ShardedKey(key, max_block_number + 1)` in the history table.
//!    This lands on the first shard whose `highest_block_number > max_block_number`. The bitmap
//!    tells us which blocks modified this key.
//! 2. **Find the first modification *after* `max_block_number`**: Using `rank` + `select` on the
//!    bitmap. `rank(max_block_number)` counts entries ≤ the target block; `select(rank)` returns
//!    the first entry strictly greater.
//! 3. **Determine where the value lives**:
//!    - If a block `> max_block_number` modified this key → read the **changeset** at that block.
//!      The changeset stores the value *before* that block's execution, which is the value at the
//!      end of `max_block_number`.
//!    - If no block after `max_block_number` modified this key → the **current state** table
//!      already has the correct value.

mod account;
mod account_trie;
mod storage;
mod storage_trie;

pub use account::V2AccountCursor;
pub use account_trie::V2AccountTrieCursor;
pub use storage::V2StorageCursor;
pub use storage_trie::V2StorageTrieCursor;

use reth_db::{BlockNumberList, DatabaseError, cursor::DbCursorRO, table::Table};

/// Shared merge-walk state used by all four history-aware V2 cursors.
#[derive(Debug)]
pub(super) struct MergeState<K, V> {
    /// Pre-fetched next `(key, value)` pair from the current-state walk.
    pub(super) cs_next: Option<(K, V)>,
    /// Pre-fetched next unique key from the history walk.
    pub(super) hist_next_key: Option<K>,
    /// Last key yielded by `seek`/`next` (used by trie cursors for `current()`).
    pub(super) last_key: Option<K>,
    /// Whether `seek`/`seek_exact` has initialized the merge cursors.
    pub(super) seeked: bool,
}

impl<K, V> MergeState<K, V> {
    pub(super) const fn new() -> Self {
        Self { cs_next: None, hist_next_key: None, last_key: None, seeked: false }
    }

    pub(super) fn reset(&mut self) {
        self.cs_next = None;
        self.hist_next_key = None;
        self.last_key = None;
        self.seeked = false;
    }
}

/// Drives the merge-walk loop, returning the next live `(key, value)` pair.
///
/// - `advance_cs`   — advance the current-state cursor, returning the next `(K, V)` pair.
/// - `advance_hist` — advance the history cursor past all shards of `key`.
/// - `resolve`      — determine the actual value at `max_block_number` (calls
///   [`resolve_historical`] internally). Receives the pre-fetched current-state value as
///   `Option<V>`.
pub(super) fn find_next_live<K, V: Clone>(
    state: &mut MergeState<K, V>,
    mut advance_cs: impl FnMut() -> Result<Option<(K, V)>, DatabaseError>,
    mut advance_hist: impl FnMut(&K) -> Result<Option<K>, DatabaseError>,
    mut resolve: impl FnMut(&K, Option<V>) -> Result<Option<V>, DatabaseError>,
) -> Result<Option<(K, V)>, DatabaseError>
where
    K: Ord + Clone,
{
    loop {
        // Step 1: Pick the minimum key across both cursors.
        let (min_key, cs_value) = match (&state.cs_next, &state.hist_next_key) {
            (Some((cs_k, cs_v)), Some(h_k)) => {
                if cs_k <= h_k {
                    (cs_k.clone(), Some(cs_v.clone()))
                } else {
                    (h_k.clone(), None)
                }
            }
            (Some((cs_k, cs_v)), None) => (cs_k.clone(), Some(cs_v.clone())),
            (None, Some(h_k)) => (h_k.clone(), None),
            (None, None) => return Ok(None),
        };

        // Step 2: Advance whichever cursor(s) produced min_key.
        if state.cs_next.as_ref().is_some_and(|(k, _)| *k == min_key) {
            state.cs_next = advance_cs()?;
        }
        if state.hist_next_key.as_ref().is_some_and(|k| *k == min_key) {
            state.hist_next_key = advance_hist(&min_key)?;
        }

        // Step 3: Resolve the value at max_block_number.
        if let Some(v) = resolve(&min_key, cs_value)? {
            state.last_key = Some(min_key.clone());
            return Ok(Some((min_key, v)));
        }
    }
}

/// Enum to define where to read the value for a given key at a specific block.
#[derive(Debug, Eq, PartialEq)]
pub(crate) enum ResolvedSource {
    /// Read the "before" value from the changeset at this block.
    /// The changeset stores the value *before* this block's execution,
    /// which equals the value at the end of `max_block_number`.
    FromChangeset(u64),
    /// No modification after the target block → current state has the value.
    FromCurrentState,
}

/// Search history bitmaps to determine where to read the value for a key
/// at a given `max_block_number`.
///
/// `seek_key_fn` is called with `max_block_number + 1` to produce the seek
/// key. This lands on the first shard whose `highest_block_number >
/// max_block_number`, guaranteeing the shard contains at least one entry
/// after the target block and eliminating the need for a `cursor.next()`
/// fallback. Callers do not need to apply the `+ 1` shift themselves.
///
/// The algorithm:
/// 1. Seek the first history shard with `highest_block_number > max_block_number` (by calling
///    `seek_key_fn(max_block_number + 1)`).
/// 2. Within that shard, find the first block strictly `> max_block_number`.
/// 3. If found → `FromChangeset(block)`.
/// 4. Otherwise → `FromCurrentState`.
pub(crate) fn find_source<T, C>(
    cursor: &mut C,
    max_block_number: u64,
    seek_key_fn: impl Fn(u64) -> T::Key,
    key_filter: impl Fn(&T::Key) -> bool,
) -> Result<ResolvedSource, DatabaseError>
where
    T: Table<Value = BlockNumberList>,
    C: DbCursorRO<T>,
{
    // 1. Build the seek key with max_block_number + 1 embedded, then filter to ensure the shard
    //    belongs to the expected key.
    let seek_key = seek_key_fn(max_block_number.saturating_add(1));
    let shard = cursor.seek(seek_key)?.filter(|(k, _)| key_filter(k));
    let Some((_, chunk)) = shard else {
        return Ok(ResolvedSource::FromCurrentState);
    };

    // 2. rank(n) = count of entries ≤ n. select(rank) = first entry > n.
    let rank = chunk.rank(max_block_number);
    Ok(chunk
        .select(rank)
        .map(ResolvedSource::FromChangeset)
        .unwrap_or(ResolvedSource::FromCurrentState))
}

/// Resolve a historical key's value at `max_block_number` using the history
/// bitmap to decide which source to read from.
///
/// - `seek_key_fn` and `key_filter` are forwarded to [`find_source`].
/// - `read_changeset(block)` is called when the value must come from the changeset at `block`.
/// - `read_current_state()` is called when no modification exists after `max_block_number` and the
///   current-state table is authoritative.
pub(crate) fn resolve_historical<HT, HC, V>(
    history_cursor: &mut HC,
    max_block_number: u64,
    seek_key_fn: impl Fn(u64) -> HT::Key,
    key_filter: impl Fn(&HT::Key) -> bool,
    read_changeset: impl FnOnce(u64) -> Result<Option<V>, DatabaseError>,
    read_current_state: impl FnOnce() -> Result<Option<V>, DatabaseError>,
) -> Result<Option<V>, DatabaseError>
where
    HT: Table<Value = BlockNumberList>,
    HC: DbCursorRO<HT>,
{
    match find_source::<HT, HC>(history_cursor, max_block_number, seek_key_fn, key_filter)? {
        ResolvedSource::FromChangeset(block) => read_changeset(block),
        ResolvedSource::FromCurrentState => read_current_state(),
    }
}

#[cfg(test)]
mod tests;
