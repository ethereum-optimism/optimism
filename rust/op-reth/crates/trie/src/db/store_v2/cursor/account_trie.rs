//! History-aware cursor over the [`V2AccountsTrie`] v2 table.

use reth_db::{
    DatabaseError,
    cursor::{DbCursorRO, DbDupCursorRO},
};
use reth_trie::{
    BranchNodeCompact, Nibbles, StoredNibbles, StoredNibblesSubKey, trie_cursor::TrieCursor,
};

use super::{MergeState, find_next_live, resolve_historical};
use crate::db::models::{
    AccountTrieShardedKey, V2AccountTrieChangeSets, V2AccountsTrie, V2AccountsTrieHistory,
};

/// History-aware cursor over the [`V2AccountsTrie`] v2 tables.
///
/// Uses a **dual-cursor merge** to discover all trie paths that existed at
/// `max_block_number`. This is necessary because a key deleted *after* the
/// target block no longer exists in the current-state table and would be
/// missed by a walk of current state alone. The merge walks both the
/// current-state cursor and the history-bitmap cursor in sorted order,
/// yielding the minimum key from each, resolving its value at the target
/// block, and skipping keys that did not exist at that block.
#[derive(Debug)]
pub struct V2AccountTrieCursor<C, HC, CC> {
    /// Current state walk cursor.
    cursor: C,
    /// History bitmap cursor for resolving individual keys.
    history_cursor: HC,
    /// History bitmap cursor for merge-walking deleted keys.
    history_walk_cursor: HC,
    /// Changeset cursor.
    changeset_cursor: CC,
    /// Target block number.
    max_block_number: u64,
    /// Shared merge-walk state.
    state: MergeState<StoredNibbles, BranchNodeCompact>,
    /// Fast path: when `true`, skip all history/changeset lookups.
    is_latest: bool,
}

impl<C, HC, CC> V2AccountTrieCursor<C, HC, CC> {
    /// Create a new [`V2AccountTrieCursor`].
    pub const fn new(
        cursor: C,
        history_cursor: HC,
        history_walk_cursor: HC,
        changeset_cursor: CC,
        max_block_number: u64,
        is_latest: bool,
    ) -> Self {
        Self {
            cursor,
            history_cursor,
            history_walk_cursor,
            changeset_cursor,
            max_block_number,
            state: MergeState::new(),
            is_latest,
        }
    }
}

impl<C, HC, CC> V2AccountTrieCursor<C, HC, CC>
where
    C: DbCursorRO<V2AccountsTrie>,
    HC: DbCursorRO<V2AccountsTrieHistory>,
    CC: DbCursorRO<V2AccountTrieChangeSets> + DbDupCursorRO<V2AccountTrieChangeSets>,
{
    /// Resolve a key using the walk cursor for the `FromCurrentState` case.
    ///
    /// May disrupt the walk cursor position — only call when the walk state
    /// will be re-synced immediately afterward (e.g. in `seek_exact`).
    fn resolve_node_standalone(
        &mut self,
        path: &StoredNibbles,
    ) -> Result<Option<BranchNodeCompact>, DatabaseError> {
        let target = path.clone();
        let max_block_number = self.max_block_number;
        let hc = &mut self.history_cursor;
        let cc = &mut self.changeset_cursor;
        let cur = &mut self.cursor;
        resolve_historical::<V2AccountsTrieHistory, _, _>(
            hc,
            max_block_number,
            |bn| AccountTrieShardedKey::new(target.clone(), bn),
            |k| k.key == target,
            |block| {
                Ok(cc
                    .seek_by_key_subkey(block, StoredNibblesSubKey(target.0))?
                    .filter(|e| e.nibbles == StoredNibblesSubKey(target.0))
                    .and_then(|e| e.node))
            },
            || Ok(cur.seek_exact(target.clone())?.map(|(_, node)| node)),
        )
    }

    /// Advance the history walk cursor past all shards of `key` and return
    /// the next distinct key, if any.
    ///
    /// Takes the cursor directly so this can be called both from `seek_exact`
    /// (via `&mut self.history_walk_cursor`) and from the `advance_hist` closure
    /// inside [`Self::find_next_live`] (via the borrow-split `hwc`).
    fn advance_history_past(
        hwc: &mut HC,
        key: &StoredNibbles,
    ) -> Result<Option<StoredNibbles>, DatabaseError> {
        let entry = hwc.seek(AccountTrieShardedKey::new(key.clone(), u64::MAX))?;
        match entry {
            Some((k, _)) if k.key == *key => Ok(hwc.next()?.map(|(k, _)| k.key)),
            Some((k, _)) => Ok(Some(k.key)),
            None => Ok(None),
        }
    }

    /// Merge-walk both the current-state cursor and the history-bitmap cursor,
    /// yielding the next key (in ascending order) whose value is live at
    /// `max_block_number`.
    fn find_next_live(&mut self) -> Result<Option<(Nibbles, BranchNodeCompact)>, DatabaseError> {
        let cursor = &mut self.cursor;
        let hwc = &mut self.history_walk_cursor;
        let hc = &mut self.history_cursor;
        let cc = &mut self.changeset_cursor;
        let max = self.max_block_number;
        find_next_live(
            &mut self.state,
            || cursor.next(),
            |k| Self::advance_history_past(hwc, k),
            |k, cs| {
                resolve_historical::<V2AccountsTrieHistory, _, _>(
                    hc,
                    max,
                    |bn| AccountTrieShardedKey::new(k.clone(), bn),
                    |shk| shk.key == *k,
                    |block| {
                        Ok(cc
                            .seek_by_key_subkey(block, StoredNibblesSubKey(k.0))?
                            .filter(|e| e.nibbles == StoredNibblesSubKey(k.0))
                            .and_then(|e| e.node))
                    },
                    || Ok(cs),
                )
            },
        )
        .map(|opt| opt.map(|(k, v)| (k.0, v)))
    }
}

impl<C, HC, CC> TrieCursor for V2AccountTrieCursor<C, HC, CC>
where
    C: DbCursorRO<V2AccountsTrie> + Send,
    HC: DbCursorRO<V2AccountsTrieHistory> + Send,
    CC: DbCursorRO<V2AccountTrieChangeSets> + DbDupCursorRO<V2AccountTrieChangeSets> + Send,
{
    fn seek_exact(
        &mut self,
        key: Nibbles,
    ) -> Result<Option<(Nibbles, BranchNodeCompact)>, DatabaseError> {
        self.state.seeked = true;

        if self.is_latest {
            // Fast path: direct current-state lookup.
            let result = self.cursor.seek_exact(StoredNibbles(key))?;
            if result.is_some() {
                self.state.last_key = Some(StoredNibbles(key));
            }
            return Ok(result.map(|(_, node)| (key, node)));
        }

        let path = StoredNibbles(key);
        let node = self.resolve_node_standalone(&path)?;

        // Re-sync the walk state so a subsequent next() starts after `path`.
        let cs_at_key = self.cursor.seek(path.clone())?;
        self.state.cs_next = match cs_at_key {
            Some((k, _)) if k == path => self.cursor.next()?,
            other => other,
        };
        self.state.hist_next_key =
            Self::advance_history_past(&mut self.history_walk_cursor, &path)?;

        if node.is_some() {
            self.state.last_key = Some(path);
        }
        Ok(node.map(|n| (key, n)))
    }

    fn seek(
        &mut self,
        key: Nibbles,
    ) -> Result<Option<(Nibbles, BranchNodeCompact)>, DatabaseError> {
        self.state.seeked = true;

        if self.is_latest {
            // Fast path: direct current-state walk.
            let result = self.cursor.seek(StoredNibbles(key))?;
            if let Some((ref k, _)) = result {
                self.state.last_key = Some(k.clone());
            }
            return Ok(result.map(|(k, node)| (k.0, node)));
        }

        // Initialize both merge cursors at the target key.
        self.state.cs_next = self.cursor.seek(StoredNibbles(key))?;
        self.state.hist_next_key = self
            .history_walk_cursor
            .seek(AccountTrieShardedKey::new(StoredNibbles(key), 0))?
            .map(|(k, _)| k.key);
        self.find_next_live()
    }

    fn next(&mut self) -> Result<Option<(Nibbles, BranchNodeCompact)>, DatabaseError> {
        if !self.state.seeked {
            return self.seek(Nibbles::default());
        }

        if self.is_latest {
            let result = self.cursor.next()?;
            if let Some((ref k, _)) = result {
                self.state.last_key = Some(k.clone());
            }
            return Ok(result.map(|(k, node)| (k.0, node)));
        }

        self.find_next_live()
    }

    fn current(&mut self) -> Result<Option<Nibbles>, DatabaseError> {
        Ok(self.state.last_key.as_ref().map(|k| k.0))
    }

    fn reset(&mut self) {
        self.state.reset();
    }
}
