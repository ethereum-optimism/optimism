//! History-aware cursor over the [`V2StoragesTrie`] v2 `DupSort` table.

use alloy_primitives::B256;
use reth_db::{
    DatabaseError,
    cursor::{DbCursorRO, DbDupCursorRO},
};
use reth_trie::{
    BranchNodeCompact, Nibbles, StoredNibbles, StoredNibblesSubKey,
    trie_cursor::{TrieCursor, TrieStorageCursor},
};

use super::{MergeState, find_next_live, resolve_historical};
use crate::db::models::{
    BlockNumberHashedAddress, StorageTrieShardedKey, V2StorageTrieChangeSets, V2StoragesTrie,
    V2StoragesTrieHistory,
};

/// History-aware cursor over the [`V2StoragesTrie`] v2 `DupSort` table.
///
/// Uses the same dual-cursor merge strategy as [`super::V2AccountTrieCursor`] but
/// scoped to a single `hashed_address`. Both the current-state `DupSort`
/// entries and the history-bitmap entries are walked in parallel to discover
/// keys that may have been deleted after `max_block_number`.
#[derive(Debug)]
pub struct V2StorageTrieCursor<C, HC, CC> {
    /// Current state cursor (`DupSort`).
    cursor: C,
    /// History bitmap cursor for resolving individual keys.
    history_cursor: HC,
    /// History bitmap cursor for merge-walking deleted keys.
    history_walk_cursor: HC,
    /// Changeset cursor (`DupSort`).
    changeset_cursor: CC,
    /// Target hashed address.
    hashed_address: B256,
    /// Target block number.
    max_block_number: u64,
    /// Shared merge-walk state.
    state: MergeState<StoredNibbles, BranchNodeCompact>,
    /// Fast path: when `true`, skip all history/changeset lookups.
    is_latest: bool,
}

impl<C, HC, CC> V2StorageTrieCursor<C, HC, CC> {
    /// Create a new [`V2StorageTrieCursor`].
    pub const fn new(
        cursor: C,
        history_cursor: HC,
        history_walk_cursor: HC,
        changeset_cursor: CC,
        hashed_address: B256,
        max_block_number: u64,
        is_latest: bool,
    ) -> Self {
        Self {
            cursor,
            history_cursor,
            history_walk_cursor,
            changeset_cursor,
            hashed_address,
            max_block_number,
            state: MergeState::new(),
            is_latest,
        }
    }
}

impl<C, HC, CC> V2StorageTrieCursor<C, HC, CC>
where
    C: DbCursorRO<V2StoragesTrie> + DbDupCursorRO<V2StoragesTrie>,
    HC: DbCursorRO<V2StoragesTrieHistory>,
    CC: DbCursorRO<V2StorageTrieChangeSets> + DbDupCursorRO<V2StorageTrieChangeSets>,
{
    /// Resolve a key using the walk cursor for the `FromCurrentState` case.
    ///
    /// May disrupt the walk cursor position — only call from `seek_exact`.
    fn resolve_node_standalone(
        &mut self,
        path: Nibbles,
    ) -> Result<Option<BranchNodeCompact>, DatabaseError> {
        let nibbles_cmp = StoredNibbles(path);
        let addr = self.hashed_address;
        let max_block_number = self.max_block_number;
        let hc = &mut self.history_cursor;
        let cc = &mut self.changeset_cursor;
        let cur = &mut self.cursor;
        resolve_historical::<V2StoragesTrieHistory, _, _>(
            hc,
            max_block_number,
            |bn| StorageTrieShardedKey::new(addr, nibbles_cmp.clone(), bn),
            |k| k.hashed_address == addr && k.key == nibbles_cmp,
            |block| {
                Ok(cc
                    .seek_by_key_subkey(
                        BlockNumberHashedAddress((block, addr)),
                        StoredNibblesSubKey(path),
                    )?
                    .filter(|e| e.nibbles == StoredNibblesSubKey(path))
                    .and_then(|e| e.node))
            },
            || {
                Ok(cur
                    .seek_by_key_subkey(addr, StoredNibblesSubKey(path))?
                    .filter(|e| e.nibbles == StoredNibblesSubKey(path))
                    .map(|e| e.node))
            },
        )
    }

    /// Advance the history walk cursor past all shards of `key` (for this
    /// address) and return the next distinct nibbles key, if any.
    ///
    /// Takes the cursor and address directly so this can be called both from
    /// `seek_exact` and from the `advance_hist` closure inside
    /// [`Self::find_next_live`] (via the borrow-split `hwc`).
    fn advance_history_past(
        hwc: &mut HC,
        addr: B256,
        key: &StoredNibbles,
    ) -> Result<Option<StoredNibbles>, DatabaseError> {
        let seek = StorageTrieShardedKey::new(addr, key.clone(), u64::MAX);
        let entry = hwc.seek(seek)?.filter(|(k, _)| k.hashed_address == addr);
        match entry {
            Some((k, _)) if k.key == *key => {
                Ok(hwc.next()?.filter(|(k, _)| k.hashed_address == addr).map(|(k, _)| k.key))
            }
            Some((k, _)) => Ok(Some(k.key)),
            None => Ok(None),
        }
    }

    /// Merge-walk both the current-state `DupSort` cursor and the history-bitmap
    /// cursor, yielding the next path whose node is live at `max_block_number`.
    fn find_next_live(&mut self) -> Result<Option<(Nibbles, BranchNodeCompact)>, DatabaseError> {
        let cursor = &mut self.cursor;
        let hwc = &mut self.history_walk_cursor;
        let hc = &mut self.history_cursor;
        let cc = &mut self.changeset_cursor;
        let addr = self.hashed_address;
        let max = self.max_block_number;
        find_next_live(
            &mut self.state,
            || Ok(cursor.next_dup()?.map(|(_, v)| (StoredNibbles(v.nibbles.0), v.node))),
            |k| Self::advance_history_past(hwc, addr, k),
            |k, cs| {
                let path = k.0;
                let nibbles_cmp = k.clone();
                resolve_historical::<V2StoragesTrieHistory, _, _>(
                    hc,
                    max,
                    |bn| StorageTrieShardedKey::new(addr, nibbles_cmp.clone(), bn),
                    |shk| shk.hashed_address == addr && shk.key == nibbles_cmp,
                    |block| {
                        Ok(cc
                            .seek_by_key_subkey(
                                BlockNumberHashedAddress((block, addr)),
                                StoredNibblesSubKey(path),
                            )?
                            .filter(|e| e.nibbles == StoredNibblesSubKey(path))
                            .and_then(|e| e.node))
                    },
                    || Ok(cs),
                )
            },
        )
        .map(|opt| opt.map(|(k, v)| (k.0, v)))
    }
}

impl<C, HC, CC> TrieCursor for V2StorageTrieCursor<C, HC, CC>
where
    C: DbCursorRO<V2StoragesTrie> + DbDupCursorRO<V2StoragesTrie> + Send,
    HC: DbCursorRO<V2StoragesTrieHistory> + Send,
    CC: DbCursorRO<V2StorageTrieChangeSets> + DbDupCursorRO<V2StorageTrieChangeSets> + Send,
{
    fn seek_exact(
        &mut self,
        key: Nibbles,
    ) -> Result<Option<(Nibbles, BranchNodeCompact)>, DatabaseError> {
        self.state.seeked = true;

        if self.is_latest {
            // Fast path: direct DupSort lookup.
            let entry = self
                .cursor
                .seek_by_key_subkey(self.hashed_address, StoredNibblesSubKey(key))?
                .filter(|e| e.nibbles == StoredNibblesSubKey(key));
            if entry.is_some() {
                self.state.last_key = Some(StoredNibbles(key));
            }
            return Ok(entry.map(|e| (key, e.node)));
        }

        let node = self.resolve_node_standalone(key)?;

        // Re-sync walk state so a subsequent next() starts after `key`.
        let cs_at_key =
            self.cursor.seek_by_key_subkey(self.hashed_address, StoredNibblesSubKey(key))?;
        self.state.cs_next = match cs_at_key {
            Some(e) if e.nibbles == StoredNibblesSubKey(key) => {
                self.cursor.next_dup()?.map(|(_, v)| (StoredNibbles(v.nibbles.0), v.node))
            }
            Some(e) => Some((StoredNibbles(e.nibbles.0), e.node)),
            None => None,
        };
        let path = StoredNibbles(key);
        self.state.hist_next_key =
            Self::advance_history_past(&mut self.history_walk_cursor, self.hashed_address, &path)?;

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
            // Fast path: direct DupSort walk.
            let entry =
                self.cursor.seek_by_key_subkey(self.hashed_address, StoredNibblesSubKey(key))?;
            if let Some(ref e) = entry {
                self.state.last_key = Some(StoredNibbles(e.nibbles.0));
            }
            return Ok(entry.map(|e| (e.nibbles.0, e.node)));
        }

        // Initialize both merge cursors at the target key.
        self.state.cs_next = self
            .cursor
            .seek_by_key_subkey(self.hashed_address, StoredNibblesSubKey(key))?
            .map(|e| (StoredNibbles(e.nibbles.0), e.node));
        let hist_seek = StorageTrieShardedKey::new(self.hashed_address, StoredNibbles(key), 0);
        self.state.hist_next_key = self
            .history_walk_cursor
            .seek(hist_seek)?
            .filter(|(k, _)| k.hashed_address == self.hashed_address)
            .map(|(k, _)| k.key);
        self.find_next_live()
    }

    fn next(&mut self) -> Result<Option<(Nibbles, BranchNodeCompact)>, DatabaseError> {
        if !self.state.seeked {
            return self.seek(Nibbles::default());
        }

        if self.is_latest {
            let entry = self.cursor.next_dup()?.map(|(_, v)| v);
            if let Some(ref e) = entry {
                self.state.last_key = Some(StoredNibbles(e.nibbles.0));
            }
            return Ok(entry.map(|e| (e.nibbles.0, e.node)));
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

impl<C, HC, CC> TrieStorageCursor for V2StorageTrieCursor<C, HC, CC>
where
    C: DbCursorRO<V2StoragesTrie> + DbDupCursorRO<V2StoragesTrie> + Send,
    HC: DbCursorRO<V2StoragesTrieHistory> + Send,
    CC: DbCursorRO<V2StorageTrieChangeSets> + DbDupCursorRO<V2StorageTrieChangeSets> + Send,
{
    fn set_hashed_address(&mut self, hashed_address: B256) {
        self.hashed_address = hashed_address;
        self.state.reset();
    }
}
