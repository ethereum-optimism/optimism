//! History-aware cursor over the [`V2HashedStorages`] v2 `DupSort` table.

use alloy_primitives::{B256, U256};
use reth_db::{
    DatabaseError,
    cursor::{DbCursorRO, DbDupCursorRO},
    models::sharded_key::ShardedKey,
};
use reth_trie::hashed_cursor::{HashedCursor, HashedStorageCursor};

use super::{MergeState, find_next_live, resolve_historical};
use crate::db::models::{
    BlockNumberHashedAddress, HashedStorageShardedKey, V2HashedStorageChangeSets, V2HashedStorages,
    V2HashedStoragesHistory,
};

/// History-aware cursor over the [`V2HashedStorages`] v2 `DupSort` table.
///
/// Uses the same dual-cursor merge strategy as [`super::V2AccountCursor`] but
/// scoped to a single `hashed_address`. Both the current-state `DupSort`
/// entries and the history-bitmap entries are walked in parallel to discover
/// storage slots that may have been deleted after `max_block_number`.
#[derive(Debug)]
pub struct V2StorageCursor<C, HC, CC> {
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
    /// Target block number for historical reads.
    max_block_number: u64,
    /// Shared merge-walk state.
    state: MergeState<B256, U256>,
    /// Fast path: when `true`, skip all history/changeset lookups.
    is_latest: bool,
}

impl<C, HC, CC> V2StorageCursor<C, HC, CC> {
    /// Create a new [`V2StorageCursor`].
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

impl<C, HC, CC> V2StorageCursor<C, HC, CC>
where
    C: DbCursorRO<V2HashedStorages> + DbDupCursorRO<V2HashedStorages>,
    HC: DbCursorRO<V2HashedStoragesHistory>,
    CC: DbCursorRO<V2HashedStorageChangeSets> + DbDupCursorRO<V2HashedStorageChangeSets>,
{
    /// Merge-walk both the current-state `DupSort` cursor and the history-bitmap
    /// cursor, yielding the next storage slot whose value is live at
    /// `max_block_number`.
    fn find_next_live(&mut self) -> Result<Option<(B256, U256)>, DatabaseError> {
        let cursor = &mut self.cursor;
        let hwc = &mut self.history_walk_cursor;
        let hc = &mut self.history_cursor;
        let cc = &mut self.changeset_cursor;
        let addr = self.hashed_address;
        let max = self.max_block_number;
        find_next_live(
            &mut self.state,
            || cursor.next_dup_val().map(|opt| opt.map(|e| (e.key, e.value))),
            |k| {
                let seek = HashedStorageShardedKey {
                    hashed_address: addr,
                    sharded_key: ShardedKey::new(*k, u64::MAX),
                };
                let entry = hwc.seek(seek)?.filter(|(shk, _)| shk.hashed_address == addr);
                match entry {
                    Some((shk, _)) if shk.sharded_key.key == *k => Ok(hwc
                        .next()?
                        .filter(|(shk, _)| shk.hashed_address == addr)
                        .map(|(shk, _)| shk.sharded_key.key)),
                    Some((shk, _)) => Ok(Some(shk.sharded_key.key)),
                    None => Ok(None),
                }
            },
            |k, cs| {
                resolve_historical::<V2HashedStoragesHistory, _, _>(
                    hc,
                    max,
                    |bn| HashedStorageShardedKey {
                        hashed_address: addr,
                        sharded_key: ShardedKey::new(*k, bn),
                    },
                    |shk| shk.hashed_address == addr && shk.sharded_key.key == *k,
                    |block| {
                        let entry = cc
                            .seek_by_key_subkey(BlockNumberHashedAddress((block, addr)), *k)?
                            .filter(|e| e.key == *k);
                        match entry {
                            Some(e) if e.value.is_zero() => Ok(None),
                            Some(e) => Ok(Some(e.value)),
                            None => Ok(None),
                        }
                    },
                    || Ok(cs.filter(|v| !v.is_zero())),
                )
            },
        )
    }
}

impl<C, HC, CC> HashedCursor for V2StorageCursor<C, HC, CC>
where
    C: DbCursorRO<V2HashedStorages> + DbDupCursorRO<V2HashedStorages> + Send,
    HC: DbCursorRO<V2HashedStoragesHistory> + Send,
    CC: DbCursorRO<V2HashedStorageChangeSets> + DbDupCursorRO<V2HashedStorageChangeSets> + Send,
{
    type Value = U256;

    fn seek(&mut self, subkey: B256) -> Result<Option<(B256, Self::Value)>, DatabaseError> {
        self.state.seeked = true;

        if self.is_latest {
            // Fast path: current state is authoritative.
            // Loop to skip zero-valued entries (tombstones).
            let mut entry = self.cursor.seek_by_key_subkey(self.hashed_address, subkey)?;
            while let Some(ref e) = entry {
                if !e.value.is_zero() {
                    return Ok(Some((e.key, e.value)));
                }
                entry = self.cursor.next_dup_val()?;
            }
            return Ok(None);
        }

        // Initialize both merge cursors at the target key.
        self.state.cs_next =
            self.cursor.seek_by_key_subkey(self.hashed_address, subkey)?.map(|e| (e.key, e.value));
        let hist_seek = HashedStorageShardedKey {
            hashed_address: self.hashed_address,
            sharded_key: ShardedKey::new(subkey, 0),
        };
        self.state.hist_next_key = self
            .history_walk_cursor
            .seek(hist_seek)?
            .filter(|(k, _)| k.hashed_address == self.hashed_address)
            .map(|(k, _)| k.sharded_key.key);
        self.find_next_live()
    }

    fn next(&mut self) -> Result<Option<(B256, Self::Value)>, DatabaseError> {
        if !self.state.seeked {
            return self.seek(B256::ZERO);
        }

        if self.is_latest {
            // Loop to skip zero-valued entries (tombstones).
            while let Some(e) = self.cursor.next_dup_val()? {
                if !e.value.is_zero() {
                    return Ok(Some((e.key, e.value)));
                }
            }
            return Ok(None);
        }

        self.find_next_live()
    }

    fn reset(&mut self) {
        self.state.reset();
    }
}

impl<C, HC, CC> HashedStorageCursor for V2StorageCursor<C, HC, CC>
where
    C: DbCursorRO<V2HashedStorages> + DbDupCursorRO<V2HashedStorages> + Send,
    HC: DbCursorRO<V2HashedStoragesHistory> + Send,
    CC: DbCursorRO<V2HashedStorageChangeSets> + DbDupCursorRO<V2HashedStorageChangeSets> + Send,
{
    fn is_storage_empty(&mut self) -> Result<bool, DatabaseError> {
        Ok(self.seek(B256::ZERO)?.is_none())
    }

    fn set_hashed_address(&mut self, hashed_address: B256) {
        self.hashed_address = hashed_address;
        self.state.reset();
    }
}
