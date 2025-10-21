use std::marker::PhantomData;

use crate::{
    db::{
        AccountTrieHistory, HashedAccountHistory, HashedStorageHistory, HashedStorageKey,
        MaybeDeleted, StorageTrieHistory, StorageTrieKey, VersionedValue,
    },
    OpProofsHashedCursor, OpProofsStorageError, OpProofsStorageResult, OpProofsTrieCursor,
};
use alloy_primitives::{B256, U256};
use reth_db::{
    cursor::{DbCursorRO, DbDupCursorRO},
    table::{DupSort, Table},
    transaction::DbTx,
    Database, DatabaseEnv,
};
use reth_primitives_traits::Account;
use reth_trie::{BranchNodeCompact, Nibbles, StoredNibbles};

/// Generic alias for dup cursor for T
pub(crate) type Dup<'tx, T> = <<DatabaseEnv as Database>::TX as DbTx>::DupCursor<T>;

/// Iterates versioned dup-sorted rows and returns the latest value (<= `max_block_number`),
/// skipping tombstones.
#[derive(Debug, Clone)]
pub struct BlockNumberVersionedCursor<T: Table + DupSort, Cursor> {
    _table: PhantomData<T>,
    cursor: Cursor,
    max_block_number: u64,
}

impl<V, T, Cursor> BlockNumberVersionedCursor<T, Cursor>
where
    T: Table<Value = VersionedValue<V>> + DupSort<SubKey = u64>,
    Cursor: DbCursorRO<T> + DbDupCursorRO<T>,
{
    /// Initializes new `BlockNumberVersionedCursor`
    pub const fn new(cursor: Cursor, max_block_number: u64) -> Self {
        Self { _table: PhantomData, cursor, max_block_number }
    }

    /// Resolve the latest version for `key` with `block_number` <= `max_block_number`.
    /// Strategy:
    /// - `seek_by_key_subkey(key, max)` gives first dup >= max.
    ///   - if exactly == max → it's our latest
    ///   - if > max → `prev_dup()` is latest < max (or None)
    /// - if no dup >= max:
    ///   - if key exists → `last_dup()` is latest < max
    ///   - else → None
    fn latest_version_for_key(
        &mut self,
        key: T::Key,
    ) -> OpProofsStorageResult<Option<(T::Key, T::Value)>> {
        // First dup with subkey >= max_block_number
        let seek_res = self
            .cursor
            .seek_by_key_subkey(key.clone(), self.max_block_number)
            .map_err(|e| OpProofsStorageError::Other(e.into()))?;

        if let Some(vv) = seek_res {
            if vv.block_number > self.max_block_number {
                // step back to the last dup < max
                return self.cursor.prev_dup().map_err(|e| OpProofsStorageError::Other(e.into()));
            }
            // already at the dup = max
            return Ok(Some((key, vv)))
        }

        // No dup >= max ⇒ either key absent or all dups < max. Check if key exists:
        if self
            .cursor
            .seek_exact(key.clone())
            .map_err(|e| OpProofsStorageError::Other(e.into()))?
            .is_none()
        {
            return Ok(None);
        }

        // Key exists ⇒ take last dup (< max).
        if let Some(vv) =
            self.cursor.last_dup().map_err(|e| OpProofsStorageError::Other(e.into()))?
        {
            return Ok(Some((key, vv)))
        }
        Ok(None)
    }

    /// Returns a non-deleted latest version for exactly `key`, if any.
    fn seek_exact(&mut self, key: T::Key) -> OpProofsStorageResult<Option<(T::Key, V)>> {
        if let Some((latest_key, latest_value)) = self.latest_version_for_key(key)? &&
            let MaybeDeleted(Some(v)) = latest_value.value
        {
            return Ok(Some((latest_key, v)));
        }
        Ok(None)
    }

    /// Walk forward from `first_key` (inclusive) until we find a *live* latest-≤-max value.
    /// `first_key` must already be a *real key* in the table.
    fn next_live_from(
        &mut self,
        mut first_key: T::Key,
    ) -> OpProofsStorageResult<Option<(T::Key, V)>> {
        loop {
            // Compute latest version ≤ max for this key
            if let Some((k, v)) = self.seek_exact(first_key.clone())? {
                return Ok(Some((k, v)));
            }

            // Move to next distinct key, or EOF
            let Some((next_key, _)) =
                self.cursor.next_no_dup().map_err(|e| OpProofsStorageError::Other(e.into()))?
            else {
                return Ok(None);
            };

            first_key = next_key;
        }
    }

    /// Seek to the first non-deleted latest version at or after `start_key`.
    /// Logic:
    /// - Try exact key first (above). If alive, return it.
    /// - Otherwise hop to next distinct key and repeat until we find a live version or hit EOF.
    fn seek(&mut self, start_key: T::Key) -> OpProofsStorageResult<Option<(T::Key, V)>> {
        // Position MDBX at first key >= start_key
        if let Some((first_key, _)) =
            self.cursor.seek(start_key).map_err(|e| OpProofsStorageError::Other(e.into()))?
        {
            return self.next_live_from(first_key)
        }
        Ok(None)
    }

    /// Advance to the next distinct key from the current MDBX position
    /// and return its non-deleted latest version, if any.
    /// Next distinct key; if not positioned, start from `T::Key::default()`.
    fn next(&mut self) -> OpProofsStorageResult<Option<(T::Key, V)>>
    where
        T::Key: Default,
    {
        // If not positioned, start from the beginning (default key).
        if self.cursor.current().map_err(|e| OpProofsStorageError::Other(e.into()))?.is_none() {
            let Some((first_key, _)) = self
                .cursor
                .seek(T::Key::default())
                .map_err(|e| OpProofsStorageError::Other(e.into()))?
            else {
                return Ok(None);
            };
            return self.next_live_from(first_key);
        }

        // Otherwise advance to next distinct key and resume the walk.
        let Some((next_key, _)) =
            self.cursor.next_no_dup().map_err(|e| OpProofsStorageError::Other(e.into()))?
        else {
            return Ok(None);
        };
        self.next_live_from(next_key)
    }
}

/// MDBX implementation of `OpProofsTrieCursor`.
#[derive(Debug)]
pub struct MdbxTrieCursor<T: Table + DupSort, Cursor> {
    inner: BlockNumberVersionedCursor<T, Cursor>,
    hashed_address: Option<B256>,
}

impl<
        V,
        T: Table<Value = VersionedValue<V>> + DupSort<SubKey = u64>,
        Cursor: DbCursorRO<T> + DbDupCursorRO<T>,
    > MdbxTrieCursor<T, Cursor>
{
    /// Initializes new `MdbxTrieCursor`
    pub const fn new(cursor: Cursor, max_block_number: u64, hashed_address: Option<B256>) -> Self {
        Self { inner: BlockNumberVersionedCursor::new(cursor, max_block_number), hashed_address }
    }
}

impl<Cursor> OpProofsTrieCursor for MdbxTrieCursor<AccountTrieHistory, Cursor>
where
    Cursor: DbCursorRO<AccountTrieHistory> + DbDupCursorRO<AccountTrieHistory> + Send + Sync,
{
    fn seek_exact(
        &mut self,
        path: Nibbles,
    ) -> OpProofsStorageResult<Option<(Nibbles, BranchNodeCompact)>> {
        self.inner
            .seek_exact(StoredNibbles(path))
            .map(|opt| opt.map(|(StoredNibbles(n), node)| (n, node)))
    }

    fn seek(
        &mut self,
        path: Nibbles,
    ) -> OpProofsStorageResult<Option<(Nibbles, BranchNodeCompact)>> {
        self.inner
            .seek(StoredNibbles(path))
            .map(|opt| opt.map(|(StoredNibbles(n), node)| (n, node)))
    }

    fn next(&mut self) -> OpProofsStorageResult<Option<(Nibbles, BranchNodeCompact)>> {
        self.inner.next().map(|opt| opt.map(|(StoredNibbles(n), node)| (n, node)))
    }

    fn current(&mut self) -> OpProofsStorageResult<Option<Nibbles>> {
        self.inner
            .cursor
            .current()
            .map_err(|e| OpProofsStorageError::Other(e.into()))
            .map(|opt| opt.map(|(StoredNibbles(n), _)| n))
    }
}

impl<Cursor> OpProofsTrieCursor for MdbxTrieCursor<StorageTrieHistory, Cursor>
where
    Cursor: DbCursorRO<StorageTrieHistory> + DbDupCursorRO<StorageTrieHistory> + Send + Sync,
{
    fn seek_exact(
        &mut self,
        path: Nibbles,
    ) -> OpProofsStorageResult<Option<(Nibbles, BranchNodeCompact)>> {
        if let Some(address) = self.hashed_address {
            let key = StorageTrieKey::new(address, StoredNibbles(path));
            return self.inner.seek_exact(key).map(|opt| {
                opt.and_then(|(k, node)| (k.hashed_address == address).then_some((k.path.0, node)))
            })
        }
        Ok(None)
    }

    fn seek(
        &mut self,
        path: Nibbles,
    ) -> OpProofsStorageResult<Option<(Nibbles, BranchNodeCompact)>> {
        if let Some(address) = self.hashed_address {
            let key = StorageTrieKey::new(address, StoredNibbles(path));
            return self.inner.seek(key).map(|opt| opt.map(|(k, node)| (k.path.0, node)))
        }
        Ok(None)
    }

    fn next(&mut self) -> OpProofsStorageResult<Option<(Nibbles, BranchNodeCompact)>> {
        if let Some(address) = self.hashed_address {
            return self.inner.next().map(|opt| {
                opt.and_then(|(k, node)| (k.hashed_address == address).then_some((k.path.0, node)))
            })
        }
        Ok(None)
    }

    fn current(&mut self) -> OpProofsStorageResult<Option<Nibbles>> {
        self.inner
            .cursor
            .current()
            .map_err(|e| OpProofsStorageError::Other(e.into()))
            .map(|opt| opt.map(|(k, _)| k.path.0))
    }
}

/// MDBX implementation of `OpProofsHashedCursor` for storage state.
#[derive(Debug)]
pub struct MdbxStorageCursor<Cursor> {
    inner: BlockNumberVersionedCursor<HashedStorageHistory, Cursor>,
    hashed_address: B256,
}

impl<Cursor> MdbxStorageCursor<Cursor>
where
    Cursor: DbCursorRO<HashedStorageHistory> + DbDupCursorRO<HashedStorageHistory> + Send + Sync,
{
    ///  Initializes new [`MdbxStorageCursor`]
    pub const fn new(cursor: Cursor, block_number: u64, hashed_address: B256) -> Self {
        Self { inner: BlockNumberVersionedCursor::new(cursor, block_number), hashed_address }
    }
}

impl<Cursor> OpProofsHashedCursor for MdbxStorageCursor<Cursor>
where
    Cursor: DbCursorRO<HashedStorageHistory> + DbDupCursorRO<HashedStorageHistory> + Send + Sync,
{
    type Value = U256;

    fn seek(&mut self, key: B256) -> OpProofsStorageResult<Option<(B256, Self::Value)>> {
        let storage_key = HashedStorageKey::new(self.hashed_address, key);
        self.inner.seek(storage_key).map(|opt| opt.map(|(k, v)| (k.hashed_storage_key, v.0)))
    }

    fn next(&mut self) -> OpProofsStorageResult<Option<(B256, Self::Value)>> {
        self.inner.next().map(|opt| opt.map(|(k, v)| (k.hashed_storage_key, v.0)))
    }
}

/// MDBX implementation of `OpProofsHashedCursor` for account state.
#[derive(Debug)]
pub struct MdbxAccountCursor<Cursor> {
    inner: BlockNumberVersionedCursor<HashedAccountHistory, Cursor>,
}

impl<Cursor> MdbxAccountCursor<Cursor>
where
    Cursor: DbCursorRO<HashedAccountHistory> + DbDupCursorRO<HashedAccountHistory> + Send + Sync,
{
    /// Initializes new `MdbxAccountCursor`
    pub const fn new(cursor: Cursor, block_number: u64) -> Self {
        Self { inner: BlockNumberVersionedCursor::new(cursor, block_number) }
    }
}

impl<Cursor> OpProofsHashedCursor for MdbxAccountCursor<Cursor>
where
    Cursor: DbCursorRO<HashedAccountHistory> + DbDupCursorRO<HashedAccountHistory> + Send + Sync,
{
    type Value = Account;

    fn seek(&mut self, key: B256) -> OpProofsStorageResult<Option<(B256, Self::Value)>> {
        self.inner.seek(key)
    }

    fn next(&mut self) -> OpProofsStorageResult<Option<(B256, Self::Value)>> {
        self.inner.next()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::db::{models, StorageValue};
    use reth_db::{
        mdbx::{init_db_for, DatabaseArguments},
        DatabaseEnv,
    };
    use reth_db_api::{
        cursor::DbDupCursorRW,
        transaction::{DbTx, DbTxMut},
        Database,
    };
    use reth_trie::{BranchNodeCompact, Nibbles, StoredNibbles};
    use tempfile::TempDir;

    fn setup_db() -> DatabaseEnv {
        let tmp = TempDir::new().expect("create tmpdir");
        init_db_for::<_, models::Tables>(tmp, DatabaseArguments::default()).expect("init db")
    }

    fn stored(path: Nibbles) -> StoredNibbles {
        StoredNibbles(path)
    }

    fn node() -> BranchNodeCompact {
        BranchNodeCompact::default()
    }

    fn append_account_trie(
        wtx: &<DatabaseEnv as Database>::TXMut,
        key: StoredNibbles,
        block: u64,
        val: Option<BranchNodeCompact>,
    ) {
        let mut c = wtx.cursor_dup_write::<AccountTrieHistory>().expect("dup write cursor");
        let vv = VersionedValue { block_number: block, value: MaybeDeleted(val) };
        c.append_dup(key, vv).expect("append dup");
    }

    fn append_storage_trie(
        wtx: &<DatabaseEnv as Database>::TXMut,
        address: B256,
        path: Nibbles,
        block: u64,
        val: Option<BranchNodeCompact>,
    ) {
        let mut c = wtx.cursor_dup_write::<StorageTrieHistory>().expect("dup write cursor");
        let key = StorageTrieKey::new(address, StoredNibbles(path));
        let vv = VersionedValue { block_number: block, value: MaybeDeleted(val) };
        c.append_dup(key, vv).expect("append dup");
    }

    fn append_hashed_storage(
        wtx: &<DatabaseEnv as Database>::TXMut,
        addr: B256,
        slot: B256,
        block: u64,
        val: Option<U256>,
    ) {
        let mut c = wtx.cursor_dup_write::<HashedStorageHistory>().expect("dup write");
        let key = HashedStorageKey::new(addr, slot);
        let vv = VersionedValue { block_number: block, value: MaybeDeleted(val.map(StorageValue)) };
        c.append_dup(key, vv).expect("append dup");
    }

    fn append_hashed_account(
        wtx: &<DatabaseEnv as Database>::TXMut,
        key: B256,
        block: u64,
        val: Option<Account>,
    ) {
        let mut c = wtx.cursor_dup_write::<HashedAccountHistory>().expect("dup write");
        let vv = VersionedValue { block_number: block, value: MaybeDeleted(val) };
        c.append_dup(key, vv).expect("append dup");
    }

    // Open a dup-RO cursor and wrap it in a BlockNumberVersionedCursor with a given bound.
    fn version_cursor(
        tx: &<DatabaseEnv as Database>::TX,
        max_block: u64,
    ) -> BlockNumberVersionedCursor<AccountTrieHistory, Dup<'_, AccountTrieHistory>> {
        let cur = tx.cursor_dup_read::<AccountTrieHistory>().expect("dup ro cursor");
        BlockNumberVersionedCursor::new(cur, max_block)
    }

    fn account_trie_cursor(
        tx: &'_ <DatabaseEnv as Database>::TX,
        max_block: u64,
    ) -> MdbxTrieCursor<AccountTrieHistory, Dup<'_, AccountTrieHistory>> {
        let c = tx.cursor_dup_read::<AccountTrieHistory>().expect("dup ro cursor");
        // For account trie the address is not used; pass None.
        MdbxTrieCursor::new(c, max_block, None)
    }

    // Helper: build a Storage trie cursor bound to an address
    fn storage_trie_cursor(
        tx: &'_ <DatabaseEnv as Database>::TX,
        max_block: u64,
        address: B256,
    ) -> MdbxTrieCursor<StorageTrieHistory, Dup<'_, StorageTrieHistory>> {
        let c = tx.cursor_dup_read::<StorageTrieHistory>().expect("dup ro cursor");
        MdbxTrieCursor::new(c, max_block, Some(address))
    }

    fn storage_cursor(
        tx: &'_ <DatabaseEnv as Database>::TX,
        max_block: u64,
        address: B256,
    ) -> MdbxStorageCursor<Dup<'_, HashedStorageHistory>> {
        let c = tx.cursor_dup_read::<HashedStorageHistory>().expect("dup ro cursor");
        MdbxStorageCursor::new(c, max_block, address)
    }

    fn account_cursor(
        tx: &'_ <DatabaseEnv as Database>::TX,
        max_block: u64,
    ) -> MdbxAccountCursor<Dup<'_, HashedAccountHistory>> {
        let c = tx.cursor_dup_read::<HashedAccountHistory>().expect("dup ro cursor");
        MdbxAccountCursor::new(c, max_block)
    }

    // Assert helper: ensure the chosen VersionedValue has the expected block and deletion flag.
    fn assert_block(
        got: Option<(StoredNibbles, VersionedValue<BranchNodeCompact>)>,
        expected_block: u64,
        expect_deleted: bool,
    ) {
        let (_, vv) = got.expect("expected Some(..)");
        assert_eq!(vv.block_number, expected_block, "wrong block chosen");
        let is_deleted = matches!(vv.value, MaybeDeleted(None));
        assert_eq!(is_deleted, expect_deleted, "tombstone mismatch");
    }

    /// No entry for key → None.
    #[test]
    fn latest_version_for_key_none_when_key_absent() {
        let db = setup_db();
        let tx = db.tx().expect("ro tx");
        let mut cursor = version_cursor(&tx, 100);

        let out = cursor
            .latest_version_for_key(stored(Nibbles::default()))
            .expect("should not return error");
        assert!(out.is_none(), "absent key must return None");
    }

    /// Exact match at max (live) → pick it.
    #[test]
    fn latest_version_for_key_picks_value_at_max_if_present() {
        let db = setup_db();
        let k = stored(Nibbles::from_nibbles([0x0A]));
        {
            let wtx = db.tx_mut().expect("rw tx");
            append_account_trie(&wtx, k.clone(), 10, Some(node()));
            append_account_trie(&wtx, k.clone(), 50, Some(node())); // == max
            wtx.commit().expect("commit");
        }

        let tx = db.tx().expect("ro tx");
        let mut core = version_cursor(&tx, 50);

        let out = core.latest_version_for_key(k).expect("ok");
        assert_block(out, 50, false);
    }

    /// When `seek_by_key_subkey` points to the subkey > max - fallback to the prev.
    #[test]
    fn latest_version_for_key_picks_latest_below_max_when_next_is_above() {
        let db = setup_db();
        let k = stored(Nibbles::from_nibbles([0x0A]));
        {
            let wtx = db.tx_mut().expect("rw tx");
            append_account_trie(&wtx, k.clone(), 10, Some(node()));
            append_account_trie(&wtx, k.clone(), 30, Some(node())); // expected
            append_account_trie(&wtx, k.clone(), 70, Some(node())); // > max
            wtx.commit().expect("commit");
        }

        let tx = db.tx().expect("ro tx");
        let mut core = version_cursor(&tx, 50);

        let out = core.latest_version_for_key(k).expect("ok");
        assert_block(out, 30, false);
    }

    /// No ≥ max but key exists → use last < max.
    #[test]
    fn latest_version_for_key_picks_last_below_max_when_none_at_or_above() {
        let db = setup_db();
        let k = stored(Nibbles::from_nibbles([0x0A]));
        {
            let wtx = db.tx_mut().expect("rw tx");
            append_account_trie(&wtx, k.clone(), 10, Some(node()));
            append_account_trie(&wtx, k.clone(), 40, Some(node())); // expected (max=100)
            wtx.commit().expect("commit");
        }

        let tx = db.tx().expect("ro tx");
        let mut core = version_cursor(&tx, 100);

        let out = core.latest_version_for_key(k).expect("ok");
        assert_block(out, 40, false);
    }

    /// All entries are > max → None.
    #[test]
    fn latest_version_for_key_none_when_everything_is_above_max() {
        let db = setup_db();
        let k1 = stored(Nibbles::from_nibbles([0x0A]));
        let k2 = stored(Nibbles::from_nibbles([0x0B]));

        {
            let wtx = db.tx_mut().expect("rw tx");
            append_account_trie(&wtx, k1.clone(), 60, Some(node()));
            append_account_trie(&wtx, k1.clone(), 70, Some(node()));
            append_account_trie(&wtx, k2, 40, Some(node()));
            wtx.commit().expect("commit");
        }

        let tx = db.tx().expect("ro tx");
        let mut core = version_cursor(&tx, 50);

        let out = core.latest_version_for_key(k1).expect("ok");
        assert!(out.is_none(), "no dup ≤ max ⇒ None");
    }

    /// Single dup < max → pick it.
    #[test]
    fn latest_version_for_key_picks_single_below_max() {
        let db = setup_db();
        let k = stored(Nibbles::from_nibbles([0x0A]));
        {
            let wtx = db.tx_mut().expect("rw tx");
            append_account_trie(&wtx, k.clone(), 25, Some(node())); // < max
            wtx.commit().expect("commit");
        }

        let tx = db.tx().expect("ro tx");
        let mut core = version_cursor(&tx, 50);

        let out = core.latest_version_for_key(k).expect("ok");
        assert_block(out, 25, false);
    }

    /// Single dup == max → pick it.
    #[test]
    fn latest_version_for_key_picks_single_at_max() {
        let db = setup_db();
        let k = stored(Nibbles::from_nibbles([0x0A]));
        {
            let wtx = db.tx_mut().expect("rw tx");
            append_account_trie(&wtx, k.clone(), 50, Some(node())); // == max
            wtx.commit().expect("commit");
        }

        let tx = db.tx().expect("ro tx");
        let mut core = version_cursor(&tx, 50);

        let out = core.latest_version_for_key(k).expect("ok");
        assert_block(out, 50, false);
    }

    /// Latest ≤ max is a tombstone → return it (this API doesn't filter).
    #[test]
    fn latest_version_for_key_returns_tombstone_if_latest_is_deleted() {
        let db = setup_db();
        let k = stored(Nibbles::from_nibbles([0x0A]));
        {
            let wtx = db.tx_mut().expect("rw tx");
            append_account_trie(&wtx, k.clone(), 10, Some(node()));
            append_account_trie(&wtx, k.clone(), 90, None); // latest ≤ max, but deleted
            wtx.commit().expect("commit");
        }

        let tx = db.tx().expect("ro tx");
        let mut core = version_cursor(&tx, 100);

        let out = core.latest_version_for_key(k).expect("ok");
        assert_block(out, 90, true);
    }

    /// Should skip tombstones and return None when the latest ≤ max is deleted.
    #[test]
    fn seek_exact_skips_tombstone_returns_none() {
        let db = setup_db();
        let k = stored(Nibbles::from_nibbles([0x0A]));
        {
            let wtx = db.tx_mut().expect("rw tx");
            append_account_trie(&wtx, k.clone(), 10, Some(node()));
            append_account_trie(&wtx, k.clone(), 90, None); // latest ≤ max is tombstoned
            wtx.commit().expect("commit");
        }

        let tx = db.tx().expect("ro tx");
        let mut core = version_cursor(&tx, 100);

        let out = core.seek_exact(k).expect("ok");
        assert!(out.is_none(), "seek_exact must filter out deleted latest value");
    }

    /// Empty table → None.
    #[test]
    fn seek_empty_returns_none() {
        let db = setup_db();
        let tx = db.tx().expect("ro tx");
        let mut cur = version_cursor(&tx, 100);

        let out = cur.seek(stored(Nibbles::from_nibbles([0x0A]))).expect("ok");
        assert!(out.is_none());
    }

    /// Start at an existing key whose latest ≤ max is live → returns that key.
    #[test]
    fn seek_at_live_key_returns_it() {
        let db = setup_db();
        let k = stored(Nibbles::from_nibbles([0x0A]));
        {
            let wtx = db.tx_mut().expect("rw tx");
            append_account_trie(&wtx, k.clone(), 10, Some(node()));
            append_account_trie(&wtx, k.clone(), 20, Some(node())); // latest ≤ max
            wtx.commit().expect("commit");
        }
        let tx = db.tx().expect("ro tx");
        let mut cur = version_cursor(&tx, 50);

        let out = cur.seek(k.clone()).expect("ok").expect("some");
        assert_eq!(out.0, k);
    }

    /// Start at an existing key whose latest ≤ max is tombstoned → skip to next key with live
    /// value.
    #[test]
    fn seek_skips_tombstoned_key_to_next_live_key() {
        let db = setup_db();
        let k1 = stored(Nibbles::from_nibbles([0x0A]));
        let k2 = stored(Nibbles::from_nibbles([0x0B]));

        {
            let wtx = db.tx_mut().expect("rw tx");
            // Key 0x10 latest ≤ max is deleted
            append_account_trie(&wtx, k1.clone(), 10, Some(node()));
            append_account_trie(&wtx, k1.clone(), 20, None); // tombstone at latest ≤ max
                                                             // Next key has live
            append_account_trie(&wtx, k2.clone(), 5, Some(node()));
            wtx.commit().expect("commit");
        }
        let tx = db.tx().expect("ro tx");
        let mut cur = version_cursor(&tx, 50);

        let out = cur.seek(k1).expect("ok").expect("some");
        assert_eq!(out.0, k2);
    }

    /// Start between keys → returns the next key’s live latest ≤ max.
    #[test]
    fn seek_between_keys_returns_next_key() {
        let db = setup_db();
        let k1 = stored(Nibbles::from_nibbles([0x0A]));
        let k2 = stored(Nibbles::from_nibbles([0x0C]));
        let k3 = stored(Nibbles::from_nibbles([0x0B]));

        {
            let wtx = db.tx_mut().expect("rw tx");
            append_account_trie(&wtx, k1, 10, Some(node()));
            append_account_trie(&wtx, k2.clone(), 10, Some(node()));
            wtx.commit().expect("commit");
        }
        let tx = db.tx().expect("ro tx");
        let mut cur = version_cursor(&tx, 100);

        // Start at 0x15 (between 0x10 and 0x20)

        let out = cur.seek(k3).expect("ok").expect("some");
        assert_eq!(out.0, k2);
    }

    /// Start after the last key → None.
    #[test]
    fn seek_after_last_returns_none() {
        let db = setup_db();
        let k1 = stored(Nibbles::from_nibbles([0x0A]));
        let k2 = stored(Nibbles::from_nibbles([0x0B]));
        let k3 = stored(Nibbles::from_nibbles([0x0C]));

        {
            let wtx = db.tx_mut().expect("rw tx");
            append_account_trie(&wtx, k1, 10, Some(node()));
            append_account_trie(&wtx, k2, 10, Some(node()));
            wtx.commit().expect("commit");
        }
        let tx = db.tx().expect("ro tx");
        let mut cur = version_cursor(&tx, 100);

        let out = cur.seek(k3).expect("ok");
        assert!(out.is_none());
    }

    /// If the first key at-or-after has only versions > max, it is effectively not visible → skip
    /// to next.
    #[test]
    fn seek_skips_keys_with_only_versions_above_max() {
        let db = setup_db();
        let k1 = stored(Nibbles::from_nibbles([0x0A]));
        let k2 = stored(Nibbles::from_nibbles([0x0B]));

        {
            let wtx = db.tx_mut().expect("rw tx");
            append_account_trie(&wtx, k1.clone(), 60, Some(node()));
            append_account_trie(&wtx, k2.clone(), 40, Some(node()));
            wtx.commit().expect("commit");
        }
        let tx = db.tx().expect("ro tx");
        let mut cur = version_cursor(&tx, 50);

        let out = cur.seek(k1).expect("ok").expect("some");
        assert_eq!(out.0, k2);
    }

    /// Start at a key with mixed versions; latest ≤ max is tombstone → skip to next key with live.
    #[test]
    fn seek_mixed_versions_tombstone_latest_skips_to_next_key() {
        let db = setup_db();
        let k1 = stored(Nibbles::from_nibbles([0x0A]));
        let k2 = stored(Nibbles::from_nibbles([0x0B]));

        {
            let wtx = db.tx_mut().expect("rw tx");
            append_account_trie(&wtx, k1.clone(), 10, Some(node()));
            append_account_trie(&wtx, k1.clone(), 30, None);
            append_account_trie(&wtx, k2.clone(), 5, Some(node()));
            wtx.commit().expect("commit");
        }
        let tx = db.tx().expect("ro tx");
        let mut cur = version_cursor(&tx, 30);

        let out = cur.seek(k1).expect("ok").expect("some");
        assert_eq!(out.0, k2);
    }

    /// When not positioned should start from default key and return the first live key.
    #[test]
    fn next_unpositioned_starts_from_default_returns_first_live() {
        let db = setup_db();
        let k1 = stored(Nibbles::from_nibbles([0x0A]));
        let k2 = stored(Nibbles::from_nibbles([0x0B]));

        {
            let wtx = db.tx_mut().expect("rw tx");
            append_account_trie(&wtx, k1.clone(), 10, Some(node())); // first live
            append_account_trie(&wtx, k2, 10, Some(node()));
            wtx.commit().expect("commit");
        }

        let tx = db.tx().expect("ro tx");
        // Unpositioned cursor
        let mut cur = version_cursor(&tx, 100);

        let out = cur.next().expect("ok").expect("some");
        assert_eq!(out.0, k1);
    }

    /// After positioning on a live key via `seek()`, `next()` should advance to the next live key.
    #[test]
    fn next_advances_from_current_live_to_next_live() {
        let db = setup_db();
        let k1 = stored(Nibbles::from_nibbles([0x0A]));
        let k2 = stored(Nibbles::from_nibbles([0x0B]));

        {
            let wtx = db.tx_mut().expect("rw tx");
            append_account_trie(&wtx, k1.clone(), 10, Some(node())); // live
            append_account_trie(&wtx, k2.clone(), 10, Some(node())); // next live
            wtx.commit().expect("commit");
        }

        let tx = db.tx().expect("ro tx");
        let mut cur = version_cursor(&tx, 100);

        // Position at k1
        let _ = cur.seek(k1).expect("ok").expect("some");
        // Next should yield k2
        let out = cur.next().expect("ok").expect("some");
        assert_eq!(out.0, k2);
    }

    /// If the next key's latest ≤ max is tombstone, `next()` should skip to the next live key.
    #[test]
    fn next_skips_tombstoned_key_to_next_live() {
        let db = setup_db();
        let k1 = stored(Nibbles::from_nibbles([0x0A]));
        let k2 = stored(Nibbles::from_nibbles([0x0B])); // will be tombstoned at latest ≤ max
        let k3 = stored(Nibbles::from_nibbles([0x0C])); // next live

        {
            let wtx = db.tx_mut().expect("rw tx");
            // k1 live
            append_account_trie(&wtx, k1.clone(), 10, Some(node()));
            // k2: latest ≤ max is tombstone
            append_account_trie(&wtx, k2.clone(), 10, Some(node()));
            append_account_trie(&wtx, k2, 20, None);
            // k3 live
            append_account_trie(&wtx, k3.clone(), 10, Some(node()));
            wtx.commit().expect("commit");
        }

        let tx = db.tx().expect("ro tx");
        let mut cur = version_cursor(&tx, 50);

        // Position at k1
        let _ = cur.seek(k1).expect("ok").expect("some");
        // next should skip k2 (tombstoned latest) and return k3
        let out = cur.next().expect("ok").expect("some");
        assert_eq!(out.0, k3);
    }

    /// If positioned on the last live key, `next()` should return None (EOF).
    #[test]
    fn next_returns_none_at_eof() {
        let db = setup_db();
        let k1 = stored(Nibbles::from_nibbles([0x0A]));
        let k2 = stored(Nibbles::from_nibbles([0x0B])); // last key

        {
            let wtx = db.tx_mut().expect("rw tx");
            append_account_trie(&wtx, k1, 10, Some(node()));
            append_account_trie(&wtx, k2.clone(), 10, Some(node())); // last live
            wtx.commit().expect("commit");
        }

        let tx = db.tx().expect("ro tx");
        let mut cur = version_cursor(&tx, 100);

        // Position at the last key k2
        let _ = cur.seek(k2).expect("ok").expect("some");
        // `next()` should hit EOF
        let out = cur.next().expect("ok");
        assert!(out.is_none());
    }

    /// If the first key has only versions > max, `next()` should skip it and return the next live
    /// key.
    #[test]
    fn next_skips_keys_with_only_versions_above_max() {
        let db = setup_db();
        let k1 = stored(Nibbles::from_nibbles([0x0A])); // only > max
        let k2 = stored(Nibbles::from_nibbles([0x0B])); // ≤ max live

        {
            let wtx = db.tx_mut().expect("rw tx");
            // k1 only above max (max=50)
            append_account_trie(&wtx, k1, 60, Some(node()));
            // k2 within max
            append_account_trie(&wtx, k2.clone(), 40, Some(node()));
            wtx.commit().expect("commit");
        }

        let tx = db.tx().expect("ro tx");
        // Unpositioned; `next()` will start from default and walk
        let mut cur = version_cursor(&tx, 50);

        let out = cur.next().expect("ok").expect("some");
        assert_eq!(out.0, k2);
    }

    /// Empty table: `next()` should return None.
    #[test]
    fn next_on_empty_returns_none() {
        let db = setup_db();
        let tx = db.tx().expect("ro tx");
        let mut cur = version_cursor(&tx, 100);

        let out = cur.next().expect("ok");
        assert!(out.is_none());
    }

    // ----------------- Account trie cursor thin-wrapper checks -----------------

    #[test]
    fn account_seek_exact_live_maps_key_and_value() {
        let db = setup_db();
        let k = Nibbles::from_nibbles([0x0A]);

        {
            let wtx = db.tx_mut().expect("rw tx");
            append_account_trie(&wtx, StoredNibbles(k), 10, Some(node()));
            wtx.commit().expect("commit");
        }

        let tx = db.tx().expect("ro tx");

        // Build wrapper
        let mut cur = account_trie_cursor(&tx, 100);

        // Wrapper should return (Nibbles, BranchNodeCompact)
        let out = OpProofsTrieCursor::seek_exact(&mut cur, k).expect("ok").expect("some");
        assert_eq!(out.0, k);
    }

    #[test]
    fn account_seek_exact_filters_tombstone() {
        let db = setup_db();
        let k = Nibbles::from_nibbles([0x0B]);

        {
            let wtx = db.tx_mut().expect("rw tx");
            append_account_trie(&wtx, StoredNibbles(k), 5, Some(node()));
            append_account_trie(&wtx, StoredNibbles(k), 9, None); // latest ≤ max tombstone
            wtx.commit().expect("commit");
        }

        let tx = db.tx().expect("ro tx");
        let mut cur = account_trie_cursor(&tx, 10);

        let out = OpProofsTrieCursor::seek_exact(&mut cur, k).expect("ok");
        assert!(out.is_none(), "account seek_exact must filter tombstone");
    }

    #[test]
    fn account_seek_and_next_and_current_roundtrip() {
        let db = setup_db();
        let k1 = Nibbles::from_nibbles([0x01]);
        let k2 = Nibbles::from_nibbles([0x02]);

        {
            let wtx = db.tx_mut().expect("rw tx");
            append_account_trie(&wtx, StoredNibbles(k1), 10, Some(node()));
            append_account_trie(&wtx, StoredNibbles(k2), 10, Some(node()));
            wtx.commit().expect("commit");
        }

        let tx = db.tx().expect("ro tx");
        let mut cur = account_trie_cursor(&tx, 100);

        // seek at k1
        let out1 = OpProofsTrieCursor::seek(&mut cur, k1).expect("ok").expect("some");
        assert_eq!(out1.0, k1);

        // current should be k1
        let cur_k = OpProofsTrieCursor::current(&mut cur).expect("ok").expect("some");
        assert_eq!(cur_k, k1);

        // next should move to k2
        let out2 = OpProofsTrieCursor::next(&mut cur).expect("ok").expect("some");
        assert_eq!(out2.0, k2);
    }

    // ----------------- Storage trie cursor thin-wrapper checks -----------------

    #[test]
    fn storage_seek_exact_respects_address_filter() {
        let db = setup_db();

        let addr_a = B256::from([0xAA; 32]);
        let addr_b = B256::from([0xBB; 32]);

        let path = Nibbles::from_nibbles([0x0D]);

        {
            let wtx = db.tx_mut().expect("rw tx");
            // insert only under B
            append_storage_trie(&wtx, addr_b, path, 10, Some(node()));
            wtx.commit().expect("commit");
        }

        let tx = db.tx().expect("ro tx");

        // Cursor bound to A must not see B’s data
        let mut cur_a = storage_trie_cursor(&tx, 100, addr_a);
        let out_a = OpProofsTrieCursor::seek_exact(&mut cur_a, path).expect("ok");
        assert!(out_a.is_none(), "no data for addr A");

        // Cursor bound to B should see it
        let mut cur_b = storage_trie_cursor(&tx, 100, addr_b);
        let out_b = OpProofsTrieCursor::seek_exact(&mut cur_b, path).expect("ok").expect("some");
        assert_eq!(out_b.0, path);
    }

    #[test]
    fn storage_seek_returns_first_key_for_bound_address() {
        let db = setup_db();

        let addr_a = B256::from([0x11; 32]);
        let addr_b = B256::from([0x22; 32]);

        let p1 = Nibbles::from_nibbles([0x01]);
        let p2 = Nibbles::from_nibbles([0x02]);

        {
            let wtx = db.tx_mut().expect("rw tx");
            // For A: only p2
            append_storage_trie(&wtx, addr_a, p2, 10, Some(node()));
            // For B: p1
            append_storage_trie(&wtx, addr_b, p1, 10, Some(node()));
            wtx.commit().expect("commit");
        }

        let tx = db.tx().expect("ro tx");
        let mut cur_a = storage_trie_cursor(&tx, 100, addr_a);

        // seek at p1: for A there is no p1; the next key >= p1 under A is p2
        let out = OpProofsTrieCursor::seek(&mut cur_a, p1).expect("ok").expect("some");
        assert_eq!(out.0, p2);
    }

    #[test]
    fn storage_next_stops_at_address_boundary() {
        let db = setup_db();

        let addr_a = B256::from([0x33; 32]);
        let addr_b = B256::from([0x44; 32]);

        let p1 = Nibbles::from_nibbles([0x05]); // under A
        let p2 = Nibbles::from_nibbles([0x06]); // under B (next key overall)

        {
            let wtx = db.tx_mut().expect("rw tx");
            append_storage_trie(&wtx, addr_a, p1, 10, Some(node()));
            append_storage_trie(&wtx, addr_b, p2, 10, Some(node()));
            wtx.commit().expect("commit");
        }

        let tx = db.tx().expect("ro tx");
        let mut cur_a = storage_trie_cursor(&tx, 100, addr_a);

        // position at p1 (A)
        let _ = OpProofsTrieCursor::seek_exact(&mut cur_a, p1).expect("ok").expect("some");

        // next should reach boundary; impl filters different address and returns None
        let out = OpProofsTrieCursor::next(&mut cur_a).expect("ok");
        assert!(out.is_none(), "next() should stop when next key is a different address");
    }

    #[test]
    fn storage_current_maps_key() {
        let db = setup_db();

        let addr = B256::from([0x55; 32]);
        let p = Nibbles::from_nibbles([0x09]);

        {
            let wtx = db.tx_mut().expect("rw tx");
            append_storage_trie(&wtx, addr, p, 10, Some(node()));
            wtx.commit().expect("commit");
        }

        let tx = db.tx().expect("ro tx");
        let mut cur = storage_trie_cursor(&tx, 100, addr);

        let _ = OpProofsTrieCursor::seek_exact(&mut cur, p).expect("ok").expect("some");

        let now = OpProofsTrieCursor::current(&mut cur).expect("ok").expect("some");
        assert_eq!(now, p);
    }

    #[test]
    fn hashed_storage_seek_maps_slot_and_value() {
        let db = setup_db();
        let addr = B256::from([0xAA; 32]);
        let slot = B256::from([0x10; 32]);

        {
            let wtx = db.tx_mut().expect("rw");
            append_hashed_storage(&wtx, addr, slot, 10, Some(U256::from(7)));
            wtx.commit().expect("commit");
        }

        let tx = db.tx().expect("ro");
        let mut cur = storage_cursor(&tx, 100, addr);

        let (got_slot, got_val) =
            OpProofsHashedCursor::seek(&mut cur, slot).expect("ok").expect("some");
        assert_eq!(got_slot, slot);
        assert_eq!(got_val, U256::from(7));
    }

    #[test]
    fn hashed_storage_seek_filters_tombstone() {
        let db = setup_db();
        let addr = B256::from([0xAB; 32]);
        let slot = B256::from([0x11; 32]);

        {
            let wtx = db.tx_mut().expect("rw");
            append_hashed_storage(&wtx, addr, slot, 5, Some(U256::from(1)));
            append_hashed_storage(&wtx, addr, slot, 9, None); // latest ≤ max is tombstone
            wtx.commit().expect("commit");
        }

        let tx = db.tx().expect("ro");
        let mut cur = storage_cursor(&tx, 10, addr);

        let out = OpProofsHashedCursor::seek(&mut cur, slot).expect("ok");
        assert!(out.is_none(), "wrapper must filter tombstoned latest");
    }

    #[test]
    fn hashed_storage_seek_and_next_roundtrip() {
        let db = setup_db();
        let addr = B256::from([0xAC; 32]);
        let s1 = B256::from([0x01; 32]);
        let s2 = B256::from([0x02; 32]);

        {
            let wtx = db.tx_mut().expect("rw");
            append_hashed_storage(&wtx, addr, s1, 10, Some(U256::from(11)));
            append_hashed_storage(&wtx, addr, s2, 10, Some(U256::from(22)));
            wtx.commit().expect("commit");
        }

        let tx = db.tx().expect("ro");
        let mut cur = storage_cursor(&tx, 100, addr);

        let (k1, v1) = OpProofsHashedCursor::seek(&mut cur, s1).expect("ok").expect("some");
        assert_eq!((k1, v1), (s1, U256::from(11)));

        let (k2, v2) = OpProofsHashedCursor::next(&mut cur).expect("ok").expect("some");
        assert_eq!((k2, v2), (s2, U256::from(22)));
    }

    #[test]
    fn hashed_account_seek_maps_key_and_value() {
        let db = setup_db();
        let key = B256::from([0x20; 32]);

        {
            let wtx = db.tx_mut().expect("rw");
            append_hashed_account(&wtx, key, 10, Some(Account::default()));
            wtx.commit().expect("commit");
        }

        let tx = db.tx().expect("ro");
        let mut cur = account_cursor(&tx, 100);

        let (got_key, _acc) = OpProofsHashedCursor::seek(&mut cur, key).expect("ok").expect("some");
        assert_eq!(got_key, key);
    }

    #[test]
    fn hashed_account_seek_filters_tombstone() {
        let db = setup_db();
        let key = B256::from([0x21; 32]);

        {
            let wtx = db.tx_mut().expect("rw");
            append_hashed_account(&wtx, key, 5, Some(Account::default()));
            append_hashed_account(&wtx, key, 9, None); // latest ≤ max is tombstone
            wtx.commit().expect("commit");
        }

        let tx = db.tx().expect("ro");
        let mut cur = account_cursor(&tx, 10);

        let out = OpProofsHashedCursor::seek(&mut cur, key).expect("ok");
        assert!(out.is_none(), "wrapper must filter tombstoned latest");
    }

    #[test]
    fn hashed_account_seek_and_next_roundtrip() {
        let db = setup_db();
        let k1 = B256::from([0x01; 32]);
        let k2 = B256::from([0x02; 32]);

        {
            let wtx = db.tx_mut().expect("rw");
            append_hashed_account(&wtx, k1, 10, Some(Account::default()));
            append_hashed_account(&wtx, k2, 10, Some(Account::default()));
            wtx.commit().expect("commit");
        }

        let tx = db.tx().expect("ro");
        let mut cur = account_cursor(&tx, 100);

        let (got1, _) = OpProofsHashedCursor::seek(&mut cur, k1).expect("ok").expect("some");
        assert_eq!(got1, k1);

        let (got2, _) = OpProofsHashedCursor::next(&mut cur).expect("ok").expect("some");
        assert_eq!(got2, k2);
    }
}
