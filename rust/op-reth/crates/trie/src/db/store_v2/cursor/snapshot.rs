//! Plain (non-history-aware) cursors over the snapshot tables.
//!
//! Unlike the V2 history-aware cursors (see [`super::account_trie`] and
//! [`super::storage_trie`], [`super::account`], [`super::storage`]), these read directly from
//! snapshot tables without any merge walk: the snapshot tables already reflect trie state at the
//! snapshot's anchor block, so a single current-state read is authoritative.
//!
//! Used by the backfill job when a [`SnapshotStatus::Ready`] snapshot is
//! available — see `crate::backfill` for the rationale.
//!
//! [`SnapshotStatus::Ready`]: crate::db::models::SnapshotStatus::Ready

use alloy_primitives::{B256, U256};
use reth_db::{
    DatabaseError,
    cursor::{DbCursorRO, DbDupCursorRO},
};
use reth_primitives_traits::Account;
use reth_trie::{
    BranchNodeCompact, Nibbles, StoredNibbles, StoredNibblesSubKey,
    hashed_cursor::{HashedCursor, HashedStorageCursor},
    trie_cursor::{TrieCursor, TrieStorageCursor},
};

use crate::db::models::{
    V2AccountsTrieSnapshot, V2HashedAccountsSnapshot, V2HashedStoragesSnapshot,
    V2StoragesTrieSnapshot,
};

/// Plain account-trie cursor over [`V2AccountsTrieSnapshot`].
#[derive(Debug)]
pub struct V2AccountTrieSnapshotCursor<C> {
    cursor: C,
    last_key: Option<StoredNibbles>,
    /// Whether `seek*` has positioned the underlying cursor at least once
    /// since construction / `reset`. Guards `next` against undefined mdbx
    /// behavior when called on an unpositioned cursor.
    seeked: bool,
}

impl<C> V2AccountTrieSnapshotCursor<C> {
    /// Create a new snapshot cursor wrapping `cursor`.
    pub const fn new(cursor: C) -> Self {
        Self { cursor, last_key: None, seeked: false }
    }
}

impl<C> TrieCursor for V2AccountTrieSnapshotCursor<C>
where
    C: DbCursorRO<V2AccountsTrieSnapshot> + Send,
{
    fn seek_exact(
        &mut self,
        key: Nibbles,
    ) -> Result<Option<(Nibbles, BranchNodeCompact)>, DatabaseError> {
        self.seeked = true;
        let entry = self.cursor.seek_exact(StoredNibbles(key))?;
        if let Some((ref k, _)) = entry {
            self.last_key = Some(k.clone());
        }
        Ok(entry.map(|(k, v)| (k.0, v)))
    }

    fn seek(
        &mut self,
        key: Nibbles,
    ) -> Result<Option<(Nibbles, BranchNodeCompact)>, DatabaseError> {
        self.seeked = true;
        let entry = self.cursor.seek(StoredNibbles(key))?;
        if let Some((ref k, _)) = entry {
            self.last_key = Some(k.clone());
        }
        Ok(entry.map(|(k, v)| (k.0, v)))
    }

    fn next(&mut self) -> Result<Option<(Nibbles, BranchNodeCompact)>, DatabaseError> {
        if !self.seeked {
            return self.seek(Nibbles::default());
        }
        let entry = self.cursor.next()?;
        if let Some((ref k, _)) = entry {
            self.last_key = Some(k.clone());
        }
        Ok(entry.map(|(k, v)| (k.0, v)))
    }

    fn current(&mut self) -> Result<Option<Nibbles>, DatabaseError> {
        Ok(self.last_key.as_ref().map(|k| k.0))
    }

    fn reset(&mut self) {
        self.last_key = None;
        self.seeked = false;
    }
}

/// Plain storage-trie cursor over [`V2StoragesTrieSnapshot`] (a `DupSort` table).
#[derive(Debug)]
pub struct V2StorageTrieSnapshotCursor<C> {
    cursor: C,
    hashed_address: B256,
    last_key: Option<StoredNibbles>,
    /// Whether `seek*` has positioned the underlying cursor at least once
    /// for the current `hashed_address`. Guards `next` against undefined
    /// mdbx behavior when called on an unpositioned cursor.
    seeked: bool,
}

impl<C> V2StorageTrieSnapshotCursor<C> {
    /// Create a new snapshot cursor wrapping `cursor`, scoped to `hashed_address`.
    pub const fn new(cursor: C, hashed_address: B256) -> Self {
        Self { cursor, hashed_address, last_key: None, seeked: false }
    }
}

impl<C> TrieCursor for V2StorageTrieSnapshotCursor<C>
where
    C: DbCursorRO<V2StoragesTrieSnapshot> + DbDupCursorRO<V2StoragesTrieSnapshot> + Send,
{
    fn seek_exact(
        &mut self,
        key: Nibbles,
    ) -> Result<Option<(Nibbles, BranchNodeCompact)>, DatabaseError> {
        self.seeked = true;
        let subkey = StoredNibblesSubKey(key);
        let entry = self
            .cursor
            .seek_by_key_subkey(self.hashed_address, subkey.clone())?
            .filter(|e| e.nibbles == subkey);
        if entry.is_some() {
            self.last_key = Some(StoredNibbles(key));
        }
        Ok(entry.map(|e| (key, e.node)))
    }

    fn seek(
        &mut self,
        key: Nibbles,
    ) -> Result<Option<(Nibbles, BranchNodeCompact)>, DatabaseError> {
        self.seeked = true;
        let entry =
            self.cursor.seek_by_key_subkey(self.hashed_address, StoredNibblesSubKey(key))?;
        if let Some(ref e) = entry {
            self.last_key = Some(StoredNibbles(e.nibbles.0));
        }
        Ok(entry.map(|e| (e.nibbles.0, e.node)))
    }

    fn next(&mut self) -> Result<Option<(Nibbles, BranchNodeCompact)>, DatabaseError> {
        if !self.seeked {
            return self.seek(Nibbles::default());
        }
        let entry = self.cursor.next_dup()?.map(|(_, v)| v);
        if let Some(ref e) = entry {
            self.last_key = Some(StoredNibbles(e.nibbles.0));
        }
        Ok(entry.map(|e| (e.nibbles.0, e.node)))
    }

    fn current(&mut self) -> Result<Option<Nibbles>, DatabaseError> {
        Ok(self.last_key.as_ref().map(|k| k.0))
    }

    fn reset(&mut self) {
        self.last_key = None;
        self.seeked = false;
    }
}

impl<C> TrieStorageCursor for V2StorageTrieSnapshotCursor<C>
where
    C: DbCursorRO<V2StoragesTrieSnapshot> + DbDupCursorRO<V2StoragesTrieSnapshot> + Send,
{
    fn set_hashed_address(&mut self, hashed_address: B256) {
        self.hashed_address = hashed_address;
        self.last_key = None;
        self.seeked = false;
    }
}

/// Plain hashed-account leaf cursor over [`V2HashedAccountsSnapshot`].
#[derive(Debug)]
pub struct V2HashedAccountSnapshotCursor<C> {
    cursor: C,
    /// Whether `seek*` has positioned the underlying cursor at least once.
    /// Guards `next` against undefined mdbx behavior on an unpositioned cursor.
    seeked: bool,
}

impl<C> V2HashedAccountSnapshotCursor<C> {
    /// Create a new hashed-account snapshot cursor wrapping `cursor`.
    pub const fn new(cursor: C) -> Self {
        Self { cursor, seeked: false }
    }
}

impl<C> HashedCursor for V2HashedAccountSnapshotCursor<C>
where
    C: DbCursorRO<V2HashedAccountsSnapshot> + Send,
{
    type Value = Account;

    fn seek(&mut self, key: B256) -> Result<Option<(B256, Self::Value)>, DatabaseError> {
        self.seeked = true;
        self.cursor.seek(key)
    }

    fn next(&mut self) -> Result<Option<(B256, Self::Value)>, DatabaseError> {
        if !self.seeked {
            return self.seek(B256::ZERO);
        }
        self.cursor.next()
    }

    fn reset(&mut self) {
        self.seeked = false;
    }
}

/// Plain hashed-storage leaf cursor over [`V2HashedStoragesSnapshot`] (a
/// `DupSort` table). Yields `(storage_key, U256)` pairs, skipping any
/// zero-valued entries defensively (the snapshot writer never inserts zeros,
/// but the cursor mirrors the live [`super::storage::V2StorageCursor`]
/// invariant).
#[derive(Debug)]
pub struct V2HashedStorageSnapshotCursor<C> {
    cursor: C,
    hashed_address: B256,
    seeked: bool,
}

impl<C> V2HashedStorageSnapshotCursor<C> {
    /// Create a new hashed-storage snapshot cursor scoped to `hashed_address`.
    pub const fn new(cursor: C, hashed_address: B256) -> Self {
        Self { cursor, hashed_address, seeked: false }
    }
}

impl<C> HashedCursor for V2HashedStorageSnapshotCursor<C>
where
    C: DbCursorRO<V2HashedStoragesSnapshot> + DbDupCursorRO<V2HashedStoragesSnapshot> + Send,
{
    type Value = U256;

    fn seek(&mut self, subkey: B256) -> Result<Option<(B256, Self::Value)>, DatabaseError> {
        self.seeked = true;
        let mut entry = self.cursor.seek_by_key_subkey(self.hashed_address, subkey)?;
        while let Some(ref e) = entry {
            if !e.value.is_zero() {
                return Ok(Some((e.key, e.value)));
            }
            entry = self.cursor.next_dup_val()?;
        }
        Ok(None)
    }

    fn next(&mut self) -> Result<Option<(B256, Self::Value)>, DatabaseError> {
        if !self.seeked {
            return self.seek(B256::ZERO);
        }
        while let Some(e) = self.cursor.next_dup_val()? {
            if !e.value.is_zero() {
                return Ok(Some((e.key, e.value)));
            }
        }
        Ok(None)
    }

    fn reset(&mut self) {
        self.seeked = false;
    }
}

impl<C> HashedStorageCursor for V2HashedStorageSnapshotCursor<C>
where
    C: DbCursorRO<V2HashedStoragesSnapshot> + DbDupCursorRO<V2HashedStoragesSnapshot> + Send,
{
    fn is_storage_empty(&mut self) -> Result<bool, DatabaseError> {
        Ok(self.seek(B256::ZERO)?.is_none())
    }

    fn set_hashed_address(&mut self, hashed_address: B256) {
        self.hashed_address = hashed_address;
        self.seeked = false;
    }
}
