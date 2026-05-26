//! History-aware cursor over the [`V2HashedAccounts`] v2 tables.

use alloy_primitives::B256;
use reth_db::{
    DatabaseError,
    cursor::{DbCursorRO, DbDupCursorRO},
};
use reth_primitives_traits::Account;
use reth_trie::hashed_cursor::HashedCursor;

use super::{MergeState, find_next_live, resolve_historical};
use crate::db::models::{
    HashedAccountShardedKey, V2HashedAccountChangeSets, V2HashedAccounts, V2HashedAccountsHistory,
};

/// History-aware cursor over the [`V2HashedAccounts`] v2 tables.
///
/// Uses a **dual-cursor merge** to discover all account keys that existed at
/// `max_block_number`. This is necessary because an account deleted *after*
/// the target block no longer exists in the current-state table and would be
/// missed by a walk of current state alone. The merge walks both the
/// current-state cursor and the history-bitmap cursor in sorted order,
/// yielding the minimum key from each, resolving its value at the target
/// block, and skipping keys that did not exist at that block.
#[derive(Debug)]
pub struct V2AccountCursor<C, HC, CC> {
    /// Current state walk cursor.
    cursor: C,
    /// History bitmap cursor for resolving individual keys.
    history_cursor: HC,
    /// History bitmap cursor for merge-walking deleted keys.
    history_walk_cursor: HC,
    /// Changeset cursor.
    changeset_cursor: CC,
    /// Target block number for historical reads.
    max_block_number: u64,
    /// Shared merge-walk state.
    state: MergeState<B256, Account>,
    /// Fast path: when `true`, skip all history/changeset lookups and
    /// read directly from the current-state table.
    is_latest: bool,
}

impl<C, HC, CC> V2AccountCursor<C, HC, CC> {
    /// Create a new [`V2AccountCursor`].
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

impl<C, HC, CC> V2AccountCursor<C, HC, CC>
where
    C: DbCursorRO<V2HashedAccounts>,
    HC: DbCursorRO<V2HashedAccountsHistory>,
    CC: DbCursorRO<V2HashedAccountChangeSets> + DbDupCursorRO<V2HashedAccountChangeSets>,
{
    /// Merge-walk both the current-state cursor and the history-bitmap cursor,
    /// yielding the next key (in ascending order) whose account is live at
    /// `max_block_number`.
    fn find_next_live(&mut self) -> Result<Option<(B256, Account)>, DatabaseError> {
        let cursor = &mut self.cursor;
        let hwc = &mut self.history_walk_cursor;
        let hc = &mut self.history_cursor;
        let cc = &mut self.changeset_cursor;
        let max = self.max_block_number;
        find_next_live(
            &mut self.state,
            || cursor.next(),
            |k| {
                let entry = hwc.seek(HashedAccountShardedKey::new(*k, u64::MAX))?;
                match entry {
                    Some((shk, _)) if shk.0.key == *k => Ok(hwc.next()?.map(|(shk, _)| shk.0.key)),
                    Some((shk, _)) => Ok(Some(shk.0.key)),
                    None => Ok(None),
                }
            },
            |k, cs| {
                resolve_historical::<V2HashedAccountsHistory, _, _>(
                    hc,
                    max,
                    |bn| HashedAccountShardedKey::new(*k, bn),
                    |shk| shk.0.key == *k,
                    |block| {
                        Ok(cc
                            .seek_by_key_subkey(block, *k)?
                            .filter(|e| e.hashed_address == *k)
                            .and_then(|e| e.info))
                    },
                    || Ok(cs),
                )
            },
        )
    }
}

impl<C, HC, CC> HashedCursor for V2AccountCursor<C, HC, CC>
where
    C: DbCursorRO<V2HashedAccounts> + Send,
    HC: DbCursorRO<V2HashedAccountsHistory> + Send,
    CC: DbCursorRO<V2HashedAccountChangeSets> + DbDupCursorRO<V2HashedAccountChangeSets> + Send,
{
    type Value = Account;

    fn seek(&mut self, key: B256) -> Result<Option<(B256, Self::Value)>, DatabaseError> {
        self.state.seeked = true;

        if self.is_latest {
            // Fast path: current state is authoritative, no history needed.
            return self.cursor.seek(key);
        }

        // Initialize both merge cursors at the target key.
        self.state.cs_next = self.cursor.seek(key)?;
        self.state.hist_next_key = self
            .history_walk_cursor
            .seek(HashedAccountShardedKey::new(key, 0))?
            .map(|(k, _)| k.0.key);
        self.find_next_live()
    }

    fn next(&mut self) -> Result<Option<(B256, Self::Value)>, DatabaseError> {
        if !self.state.seeked {
            return self.seek(B256::ZERO);
        }

        if self.is_latest {
            return self.cursor.next();
        }

        self.find_next_live()
    }

    fn reset(&mut self) {
        self.state.reset();
    }
}
