//! Migration to history layout version 1: re-key stale history shards.
//!
//! Reads resolve a historical value by seeking the first shard keyed above the target block
//! (`find_source`), and appends only ever extend the shard keyed `u64::MAX`. Both rely on every
//! logical key's shards satisfying:
//!
//! - every shard except the last is keyed by the largest block it holds, and
//! - the last shard is keyed `u64::MAX`.
//!
//! Unwinds before ethereum-optimism/optimism#23334 broke this: they trimmed the top of a shard but
//! kept its old key, and could delete the sentinel. A stale key shadows the later shards, so reads
//! in the band below it silently resolve to current state (#22501). Fixing the unwind does not
//! repair shards it already damaged, so this migration re-keys them. Only shard keys change: the
//! block lists, the changesets and current state are already correct.
//!
//! Limits of the repair:
//!
//! - Re-keying cannot recover entries that are already gone. After a deep unwind onto a different
//!   chain, a later rechunk can overwrite a stale shard outright; the history then looks well
//!   formed but is missing entries. Rebuild the store if that may have happened.
//! - History whose shards are out of block order is refused rather than re-keyed.
//! - The version only certifies the layout at the time of migration. A release without this
//!   migration ignores the version, and its unwind can damage the store again; `op-reth proofs
//!   check-history` scans the shards regardless of the version, and `--repair` re-keys them again.
//!
//! This module is temporary. Once every supported upgrade path has passed layout version 1, delete
//! it together with `op-reth proofs check-history`, and make `ensure_history_layout` refuse a store
//! below version 1 with a rebuild instruction, as `drop_legacy_v1_tables` does for v1 stores.

use super::{MdbxProofsProvider, MdbxProofsStorage, layout::HISTORY_LAYOUT_VERSION};
use crate::{
    OpProofsStorageError, OpProofsStorageResult,
    db::models::{
        AccountTrieShardedKey, HashedAccountShardedKey, HashedStorageShardedKey,
        StorageTrieShardedKey, V2AccountsTrieHistory, V2HashedAccountsHistory,
        V2HashedStoragesHistory, V2StoragesTrieHistory,
    },
};
use reth_db::{
    BlockNumberList, Database, DatabaseError,
    cursor::{DbCursorRO, DbCursorRW},
    models::sharded_key::ShardedKey,
    table::Table,
    transaction::{DbTx, DbTxMut},
};
use std::path::Path;
use tracing::info;

/// Number of history shards whose key violates the layout, per history table.
#[derive(Debug, Default, Clone, Copy, PartialEq, Eq)]
pub struct MisplacedShardCounts {
    /// Shards in `V2AccountsTrieHistory`.
    pub account_trie: usize,
    /// Shards in `V2StoragesTrieHistory`.
    pub storage_trie: usize,
    /// Shards in `V2HashedAccountsHistory`.
    pub hashed_accounts: usize,
    /// Shards in `V2HashedStoragesHistory`.
    pub hashed_storages: usize,
}

impl MisplacedShardCounts {
    /// Total number of misplaced shards across all history tables.
    pub const fn total(&self) -> usize {
        self.account_trie + self.storage_trie + self.hashed_accounts + self.hashed_storages
    }
}

/// What [`MdbxProofsStorage::check_history_layout_at`] found.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct HistoryLayoutReport {
    /// Layout version recorded in the store.
    pub version: u64,
    /// Shards whose key violates the layout, whatever the recorded version says.
    pub misplaced: MisplacedShardCounts,
}

/// What a [`HistoryLayoutReport`] means for the operator.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum HistoryLayoutVerdict {
    /// No misplaced shards.
    Clean,
    /// Misplaced shards in a store below the current version; starting this release on the store
    /// re-keys them.
    NeedsMigration,
    /// Misplaced shards in a store already at the current version: damaged after migrating, most
    /// likely by an older release.
    DamagedAfterMigration,
    /// The store records a layout newer than this release supports.
    NewerLayout,
}

impl HistoryLayoutReport {
    /// Classifies the report.
    pub const fn verdict(&self) -> HistoryLayoutVerdict {
        match self {
            report if report.version > HISTORY_LAYOUT_VERSION => HistoryLayoutVerdict::NewerLayout,
            report if report.misplaced.total() == 0 => HistoryLayoutVerdict::Clean,
            report if report.version < HISTORY_LAYOUT_VERSION => {
                HistoryLayoutVerdict::NeedsMigration
            }
            _ => HistoryLayoutVerdict::DamagedAfterMigration,
        }
    }
}

impl MdbxProofsStorage {
    /// Reports the recorded layout version and every misplaced shard of the store at `path`,
    /// without migrating it.
    ///
    /// Scans in a write transaction that is dropped uncommitted: a full scan of a production store
    /// can outlast MDBX's read-transaction timeout, which write transactions are not subject to.
    /// Opening still creates tables this release adds and drops empty legacy v1 tables.
    pub fn check_history_layout_at(path: &Path) -> OpProofsStorageResult<HistoryLayoutReport> {
        let storage = Self::open_existing(path)?;
        let provider = MdbxProofsProvider::new(storage.env.tx_mut()?);
        Ok(HistoryLayoutReport {
            version: provider.history_layout_version()?,
            misplaced: provider.count_misplaced_shards().map_err(|e| e.at(path))?,
        })
    }

    /// Re-keys the misplaced shards of the store at `path` whatever its recorded version, and
    /// records the current version. Repairs a store damaged after migrating, such as by an older
    /// release.
    pub fn repair_history_layout_at(path: &Path) -> OpProofsStorageResult<MisplacedShardCounts> {
        let storage = Self::open_existing(path)?;
        let found = MdbxProofsProvider::new(storage.env.tx()?).history_layout_version()?;
        if found > HISTORY_LAYOUT_VERSION {
            return Err(OpProofsStorageError::NewerHistoryLayout {
                found,
                supported: HISTORY_LAYOUT_VERSION,
                path: path.to_path_buf(),
            });
        }
        storage.migrate_history_layout(path)
    }

    /// Re-keys every misplaced shard and records [`HISTORY_LAYOUT_VERSION`], in one write
    /// transaction. Fails without changing anything if the history cannot be re-keyed.
    pub(super) fn migrate_history_layout(
        &self,
        path: &Path,
    ) -> OpProofsStorageResult<MisplacedShardCounts> {
        let provider = MdbxProofsProvider::new(self.env.tx_mut()?);
        info!(
            target: "trie::db",
            from = provider.history_layout_version()?,
            to = HISTORY_LAYOUT_VERSION,
            "Migrating proofs history layout; this scans every history table once"
        );
        let rekeyed = provider.rekey_misplaced_shards().map_err(|e| e.at(path))?;
        provider.set_history_layout_version(HISTORY_LAYOUT_VERSION)?;
        provider.tx.commit()?;

        info!(
            target: "trie::db",
            version = HISTORY_LAYOUT_VERSION,
            account_trie = rekeyed.account_trie,
            storage_trie = rekeyed.storage_trie,
            hashed_accounts = rekeyed.hashed_accounts,
            hashed_storages = rekeyed.hashed_storages,
            total = rekeyed.total(),
            "Migrated proofs history layout"
        );
        Ok(rekeyed)
    }

    /// Opens the store at `path`, refusing to create one: a mistyped path must not pass a check.
    fn open_existing(path: &Path) -> OpProofsStorageResult<Self> {
        if !path.join("mdbx.dat").is_file() {
            return Err(OpProofsStorageError::MissingProofsDatabase { path: path.to_path_buf() });
        }
        Self::open_env(path)
    }
}

/// Error from scanning or re-keying a history table, before the store's path is attached.
#[derive(Debug)]
enum ScanError {
    /// Damage that re-keying cannot repair.
    Unrepairable {
        table: &'static str,
        reason: &'static str,
    },
    Storage(OpProofsStorageError),
}

impl ScanError {
    fn at(self, path: &Path) -> OpProofsStorageError {
        match self {
            Self::Unrepairable { table, reason } => OpProofsStorageError::UnrepairableHistory {
                table,
                reason,
                path: path.to_path_buf(),
            },
            Self::Storage(err) => err,
        }
    }
}

impl From<DatabaseError> for ScanError {
    fn from(err: DatabaseError) -> Self {
        Self::Storage(err.into())
    }
}

const fn unrepairable<T: Table>(reason: &'static str) -> ScanError {
    ScanError::Unrepairable { table: T::NAME, reason }
}

/// A history shard to move from a misplaced key to the key the layout requires.
#[derive(Debug)]
struct ShardMove<K> {
    from: K,
    to: K,
}

impl<TX: DbTx> MdbxProofsProvider<TX> {
    fn count_misplaced_shards(&self) -> Result<MisplacedShardCounts, ScanError> {
        Ok(MisplacedShardCounts {
            account_trie: self.misplaced_account_trie_shards()?.len(),
            storage_trie: self.misplaced_storage_trie_shards()?.len(),
            hashed_accounts: self.misplaced_hashed_account_shards()?.len(),
            hashed_storages: self.misplaced_hashed_storage_shards()?.len(),
        })
    }

    fn misplaced_account_trie_shards(
        &self,
    ) -> Result<Vec<ShardMove<AccountTrieShardedKey>>, ScanError> {
        self.misplaced_shards::<V2AccountsTrieHistory>(
            |a, b| a.key == b.key,
            |key| key.highest_block_number,
            |key, highest| AccountTrieShardedKey::new(key.key.clone(), highest),
        )
    }

    fn misplaced_storage_trie_shards(
        &self,
    ) -> Result<Vec<ShardMove<StorageTrieShardedKey>>, ScanError> {
        self.misplaced_shards::<V2StoragesTrieHistory>(
            |a, b| a.hashed_address == b.hashed_address && a.key == b.key,
            |key| key.highest_block_number,
            |key, highest| StorageTrieShardedKey::new(key.hashed_address, key.key.clone(), highest),
        )
    }

    fn misplaced_hashed_account_shards(
        &self,
    ) -> Result<Vec<ShardMove<HashedAccountShardedKey>>, ScanError> {
        self.misplaced_shards::<V2HashedAccountsHistory>(
            |a, b| a.0.key == b.0.key,
            |key| key.0.highest_block_number,
            |key, highest| HashedAccountShardedKey::new(key.0.key, highest),
        )
    }

    fn misplaced_hashed_storage_shards(
        &self,
    ) -> Result<Vec<ShardMove<HashedStorageShardedKey>>, ScanError> {
        self.misplaced_shards::<V2HashedStoragesHistory>(
            |a, b| a.hashed_address == b.hashed_address && a.sharded_key.key == b.sharded_key.key,
            |key| key.sharded_key.highest_block_number,
            |key, highest| HashedStorageShardedKey {
                hashed_address: key.hashed_address,
                sharded_key: ShardedKey::new(key.sharded_key.key, highest),
            },
        )
    }

    /// Walks a history table in key order and returns a move for every shard whose key differs
    /// from the one the layout requires, in key order.
    ///
    /// Re-keying is only sound when a logical key's shards are already in block order, so an
    /// empty shard or a shard that does not start after the previous one's largest block aborts
    /// the scan instead of yielding moves.
    fn misplaced_shards<T>(
        &self,
        same_logical_key: impl Fn(&T::Key, &T::Key) -> bool,
        highest_block: impl Fn(&T::Key) -> u64,
        with_highest_block: impl Fn(&T::Key, u64) -> T::Key,
    ) -> Result<Vec<ShardMove<T::Key>>, ScanError>
    where
        T: Table<Value = BlockNumberList>,
    {
        info!(target: "trie::db", table = T::NAME, "Scanning history shard keys");
        let mut moves = Vec::new();
        let mut cursor = self.tx.cursor_read::<T>()?;
        let mut rows = cursor.walk(None)?.peekable();
        // Largest block of the previous shard, while it belongs to the same logical key.
        let mut previous_largest = None;
        while let Some(row) = rows.next() {
            let (key, list) = row?;
            let (Some(first), Some(largest)) = (list.iter().next(), list.iter().next_back()) else {
                return Err(unrepairable::<T>("a shard is empty"));
            };
            if previous_largest.is_some_and(|previous| first <= previous) {
                return Err(unrepairable::<T>("a key's shards overlap or are out of block order"));
            }

            let has_next_shard =
                matches!(rows.peek(), Some(Ok((next, _))) if same_logical_key(&key, next));
            let expected_highest = if has_next_shard { largest } else { u64::MAX };
            previous_largest = has_next_shard.then_some(largest);
            if highest_block(&key) != expected_highest {
                moves.push(ShardMove { to: with_highest_block(&key, expected_highest), from: key });
            }
        }
        Ok(moves)
    }
}

impl<TX: DbTxMut + DbTx> MdbxProofsProvider<TX> {
    fn rekey_misplaced_shards(&self) -> Result<MisplacedShardCounts, ScanError> {
        Ok(MisplacedShardCounts {
            account_trie: self
                .rekey_shards::<V2AccountsTrieHistory>(self.misplaced_account_trie_shards()?)?,
            storage_trie: self
                .rekey_shards::<V2StoragesTrieHistory>(self.misplaced_storage_trie_shards()?)?,
            hashed_accounts: self
                .rekey_shards::<V2HashedAccountsHistory>(self.misplaced_hashed_account_shards()?)?,
            hashed_storages: self
                .rekey_shards::<V2HashedStoragesHistory>(self.misplaced_hashed_storage_shards()?)?,
        })
    }

    /// Moves each shard from its current key to its expected key.
    ///
    /// `moves` must be in table key order. A shard's expected key can be another misplaced
    /// shard's current key (two unwinds can leave a shard under a stale key equal to the largest
    /// block of the shard after it); key order frees the stale key before that shard moves onto
    /// it.
    fn rekey_shards<T>(&self, moves: Vec<ShardMove<T::Key>>) -> Result<usize, ScanError>
    where
        T: Table<Value = BlockNumberList>,
    {
        let mut cursor = self.tx.cursor_write::<T>()?;
        let moved = moves.len();
        for ShardMove { from, to } in moves {
            let (_, list) = cursor
                .seek_exact(from)?
                .ok_or_else(|| unrepairable::<T>("a shard vanished during the migration"))?;
            cursor.delete_current()?;
            if cursor.seek_exact(to.clone())?.is_some() {
                return Err(unrepairable::<T>("a shard's target key already holds another shard"));
            }
            cursor.insert(to, &list)?;
        }
        Ok(moved)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        BlockStateDiff,
        api::{OpProofsInitProvider, OpProofsProviderRO, OpProofsProviderRw},
    };
    use alloy_eips::{BlockNumHash, NumHash, eip1898::BlockWithParent};
    use alloy_primitives::B256;
    use reth_primitives_traits::Account;
    use reth_trie::{HashedPostState, Nibbles, hashed_cursor::HashedCursor, updates::TrieUpdates};
    use std::ops::RangeInclusive;
    use tempfile::TempDir;

    const ADDR: B256 = B256::repeat_byte(0xA1);

    fn block_hash(number: u64) -> B256 {
        B256::left_padding_from(&number.to_be_bytes())
    }

    /// A current store in which account `ADDR` takes nonce `n` at every block `n` in `blocks`.
    fn setup(blocks: RangeInclusive<u64>) -> (TempDir, MdbxProofsStorage) {
        let dir = TempDir::new().expect("tmpdir");
        let storage = MdbxProofsStorage::new(dir.path()).expect("open storage");
        let provider = MdbxProofsProvider::new(storage.env.tx_mut().expect("rw tx"));
        provider.store_hashed_accounts(vec![(ADDR, Some(Account::default()))]).expect("account");
        provider.set_initial_state_anchor(BlockNumHash::new(0, block_hash(0))).expect("anchor");
        provider.commit_initial_state().expect("initial state");
        OpProofsInitProvider::commit(provider).expect("commit initial state");
        store_blocks(&storage, blocks);
        (dir, storage)
    }

    /// Opens the store at `dir` the way the node does, migrating it.
    fn open(dir: &TempDir) -> OpProofsStorageResult<MdbxProofsStorage> {
        MdbxProofsStorage::new(dir.path())
    }

    /// Opens the store at `dir` without checking its layout.
    fn open_raw(dir: &TempDir) -> MdbxProofsStorage {
        MdbxProofsStorage::open_env(dir.path()).expect("open env")
    }

    fn check(dir: &TempDir) -> HistoryLayoutReport {
        MdbxProofsStorage::check_history_layout_at(dir.path()).expect("check")
    }

    fn store_blocks(storage: &MdbxProofsStorage, blocks: RangeInclusive<u64>) {
        let provider = MdbxProofsProvider::new(storage.env.tx_mut().expect("rw tx"));
        let blocks = blocks
            .map(|number| {
                let mut post_state = HashedPostState::default();
                post_state
                    .accounts
                    .insert(ADDR, Some(Account { nonce: number, ..Default::default() }));
                let block = BlockWithParent::new(
                    block_hash(number - 1),
                    NumHash::new(number, block_hash(number)),
                );
                let diff = BlockStateDiff {
                    sorted_trie_updates: TrieUpdates::default().into_sorted(),
                    sorted_post_state: post_state.into_sorted(),
                };
                (block, diff)
            })
            .collect();
        provider.store_trie_updates_batch(blocks).expect("store blocks");
        OpProofsProviderRw::commit(provider).expect("commit blocks");
    }

    /// Records `version` as the store's layout version; 0 removes it, as an older release left it.
    fn set_version(storage: &MdbxProofsStorage, version: u64) {
        let tx = storage.env.tx_mut().expect("rw tx");
        if version == 0 {
            tx.delete::<crate::db::models::V2StoreMeta>(
                crate::db::models::StoreMetaKey::HistoryLayoutVersion,
                None,
            )
            .expect("delete version");
            tx.commit().expect("commit version");
        } else {
            let provider = MdbxProofsProvider::new(tx);
            provider.set_history_layout_version(version).expect("set version");
            provider.tx.commit().expect("commit version");
        }
    }

    fn version(storage: &MdbxProofsStorage) -> u64 {
        MdbxProofsProvider::new(storage.env.tx().expect("ro tx"))
            .history_layout_version()
            .expect("version")
    }

    /// Replaces `ADDR`'s history shards with `shards`, given as `(key, blocks)`.
    fn set_shards(storage: &MdbxProofsStorage, shards: &[(u64, RangeInclusive<u64>)]) {
        let tx = storage.env.tx_mut().expect("rw tx");
        {
            let mut cursor = tx.cursor_write::<V2HashedAccountsHistory>().expect("cursor");
            while cursor
                .seek(HashedAccountShardedKey::new(ADDR, 0))
                .expect("seek")
                .is_some_and(|(key, _)| key.0.key == ADDR)
            {
                cursor.delete_current().expect("delete shard");
            }
            for (highest, blocks) in shards {
                let list = BlockNumberList::new_pre_sorted(blocks.clone());
                cursor.upsert(HashedAccountShardedKey::new(ADDR, *highest), &list).expect("put");
            }
        }
        tx.commit().expect("commit shards");
    }

    /// `ADDR`'s history shards as `(key, first block, last block)`.
    fn shards(storage: &MdbxProofsStorage) -> Vec<(u64, u64, u64)> {
        let tx = storage.env.tx().expect("ro tx");
        let mut cursor = tx.cursor_read::<V2HashedAccountsHistory>().expect("cursor");
        cursor
            .walk(Some(HashedAccountShardedKey::new(ADDR, 0)))
            .expect("walk")
            .map(|row| row.expect("row"))
            .take_while(|(key, _)| key.0.key == ADDR)
            .map(|(key, list)| {
                let first = list.iter().next().expect("nonempty");
                let last = list.iter().next_back().expect("nonempty");
                (key.0.highest_block_number, first, last)
            })
            .collect()
    }

    /// Blocks in `blocks` at which the historical read does not return nonce = block number.
    fn wrong_reads(storage: &MdbxProofsStorage, blocks: RangeInclusive<u64>) -> Vec<u64> {
        let provider = MdbxProofsProvider::new(storage.env.tx().expect("ro tx"));
        blocks
            .filter(|&number| {
                let (_, account) = provider
                    .account_hashed_cursor(number)
                    .expect("cursor")
                    .seek(ADDR)
                    .expect("seek")
                    .expect("account exists");
                account.nonce != number
            })
            .collect()
    }

    fn hashed_accounts(count: usize) -> MisplacedShardCounts {
        MisplacedShardCounts { hashed_accounts: count, ..Default::default() }
    }

    fn put_rows<T>(storage: &MdbxProofsStorage, rows: &[(T::Key, Vec<u64>)])
    where
        T: Table<Value = BlockNumberList>,
    {
        let tx = storage.env.tx_mut().expect("rw tx");
        for (key, blocks) in rows {
            tx.put::<T>(key.clone(), BlockNumberList::new_pre_sorted(blocks.iter().copied()))
                .expect("put row");
        }
        tx.commit().expect("commit rows");
    }

    fn rows<T>(storage: &MdbxProofsStorage) -> Vec<(T::Key, Vec<u64>)>
    where
        T: Table<Value = BlockNumberList>,
    {
        let tx = storage.env.tx().expect("ro tx");
        let mut cursor = tx.cursor_read::<T>().expect("cursor");
        cursor
            .walk(None)
            .expect("walk")
            .map(|row| row.expect("row"))
            .map(|(key, list)| (key, list.iter().collect()))
            .collect()
    }

    #[test]
    fn a_new_store_is_created_at_the_current_layout() {
        let dir = TempDir::new().expect("tmpdir");
        let storage = open(&dir).expect("open storage");
        assert_eq!(version(&storage), HISTORY_LAYOUT_VERSION);
    }

    #[test]
    fn migration_leaves_well_formed_history_unchanged() {
        let (dir, storage) = setup(1..=4_100);
        let before = shards(&storage);
        assert_eq!(
            before,
            vec![(2_000, 1, 2_000), (4_000, 2_001, 4_000), (u64::MAX, 4_001, 4_100)]
        );
        set_version(&storage, 0);

        assert_eq!(storage.migrate_history_layout(dir.path()).unwrap(), Default::default());
        assert_eq!(shards(&storage), before);
        assert_eq!(version(&storage), HISTORY_LAYOUT_VERSION);
    }

    // The layout left by unwinding to 1801 and refilling to 2100 before #23334.
    #[test]
    fn opening_an_old_store_restores_reads_shadowed_by_a_stale_shard_key() {
        let (dir, storage) = setup(1..=2_100);
        set_shards(&storage, &[(2_000, 1..=1_800), (u64::MAX, 1_801..=2_100)]);
        set_version(&storage, 0);
        assert_eq!(wrong_reads(&storage, 1..=2_100), (1_800..=1_999).collect::<Vec<_>>());
        drop(storage);

        let report = check(&dir);
        assert_eq!(report, HistoryLayoutReport { version: 0, misplaced: hashed_accounts(1) });
        assert_eq!(report.verdict(), HistoryLayoutVerdict::NeedsMigration);
        let storage = open_raw(&dir);
        assert_eq!(
            shards(&storage),
            vec![(2_000, 1, 1_800), (u64::MAX, 1_801, 2_100)],
            "check must not modify the store"
        );
        drop(storage);

        let storage = open(&dir).expect("migrate on open");
        assert_eq!(shards(&storage), vec![(1_800, 1, 1_800), (u64::MAX, 1_801, 2_100)]);
        assert_eq!(wrong_reads(&storage, 1..=2_100), Vec::<u64>::new());
        store_blocks(&storage, 2_101..=4_500);
        assert_eq!(wrong_reads(&storage, 1..=4_500), Vec::<u64>::new());
        drop(storage);

        assert_eq!(check(&dir).verdict(), HistoryLayoutVerdict::Clean);
    }

    // A release without this migration ignores the version, so its unwind can damage a store that
    // is already marked current. Opening does not migrate again; check reports it and repair fixes
    // it.
    #[test]
    fn damage_after_migrating_is_reported_and_repaired_on_request() {
        let (dir, storage) = setup(1..=2_100);
        set_shards(&storage, &[(2_000, 1..=1_800), (u64::MAX, 1_801..=2_100)]);
        drop(storage);

        let storage = open(&dir).expect("open current store");
        assert_eq!(shards(&storage), vec![(2_000, 1, 1_800), (u64::MAX, 1_801, 2_100)]);
        drop(storage);
        let report = check(&dir);
        assert_eq!(
            report,
            HistoryLayoutReport { version: HISTORY_LAYOUT_VERSION, misplaced: hashed_accounts(1) }
        );
        assert_eq!(report.verdict(), HistoryLayoutVerdict::DamagedAfterMigration);

        assert_eq!(
            MdbxProofsStorage::repair_history_layout_at(dir.path()).unwrap(),
            hashed_accounts(1)
        );
        assert_eq!(check(&dir).verdict(), HistoryLayoutVerdict::Clean);
        let storage = open(&dir).expect("open repaired store");
        assert_eq!(wrong_reads(&storage, 1..=2_100), Vec::<u64>::new());
    }

    #[test]
    fn a_store_from_a_newer_layout_is_refused() {
        let (dir, storage) = setup(1..=1);
        set_version(&storage, HISTORY_LAYOUT_VERSION + 1);
        drop(storage);

        let is_newer = |err: &OpProofsStorageError| {
            matches!(err, OpProofsStorageError::NewerHistoryLayout { found, supported, path }
                if *found == HISTORY_LAYOUT_VERSION + 1
                    && *supported == HISTORY_LAYOUT_VERSION
                    && path == dir.path())
        };
        let err = open(&dir).unwrap_err();
        assert!(is_newer(&err), "unexpected error: {err:?}");
        let err = MdbxProofsStorage::repair_history_layout_at(dir.path()).unwrap_err();
        assert!(is_newer(&err), "unexpected error: {err:?}");
        assert_eq!(check(&dir).verdict(), HistoryLayoutVerdict::NewerLayout);
    }

    #[test]
    fn check_and_repair_refuse_a_missing_store() {
        let dir = TempDir::new().expect("tmpdir");
        let missing = dir.path().join("historical-proofs");

        for err in [
            MdbxProofsStorage::check_history_layout_at(&missing).unwrap_err(),
            MdbxProofsStorage::repair_history_layout_at(&missing).unwrap_err(),
        ] {
            assert!(
                matches!(&err, OpProofsStorageError::MissingProofsDatabase { path } if *path == missing),
                "unexpected error: {err:?}"
            );
        }
        assert!(!missing.exists(), "a check must not create a store");
    }

    // The layout left by unwinding to 1801 before #23334, before any refill: the sentinel is gone.
    #[test]
    fn migration_restores_a_missing_sentinel() {
        let (dir, storage) = setup(1..=1_800);
        set_shards(&storage, &[(2_000, 1..=1_800)]);
        set_version(&storage, 0);

        assert_eq!(storage.migrate_history_layout(dir.path()).unwrap(), hashed_accounts(1));
        assert_eq!(shards(&storage), vec![(u64::MAX, 1, 1_800)]);

        // Without the sentinel these appends would start a new last shard behind the stale one.
        store_blocks(&storage, 1_801..=2_100);
        assert_eq!(shards(&storage), vec![(2_000, 1, 2_000), (u64::MAX, 2_001, 2_100)]);
        assert_eq!(wrong_reads(&storage, 1..=2_100), Vec::<u64>::new());
    }

    // Key order disagrees with block order (a deep unwind onto a different chain): re-keying by
    // position would move the oldest shard to the sentinel, so the store must not open.
    #[test]
    fn a_store_with_shards_out_of_block_order_is_refused() {
        let (dir, storage) = setup(1..=2_000);
        let layout = vec![(2_000, 1_801, 2_000), (3_000, 1, 1_800)];
        set_shards(&storage, &[(2_000, 1_801..=2_000), (3_000, 1..=1_800)]);
        set_version(&storage, 0);
        drop(storage);

        let is_unrepairable = |err: &OpProofsStorageError| {
            matches!(err, OpProofsStorageError::UnrepairableHistory { table, path, .. }
                if *table == V2HashedAccountsHistory::NAME && path == dir.path())
        };
        let err = open(&dir).unwrap_err();
        assert!(is_unrepairable(&err), "unexpected error: {err:?}");
        let err = MdbxProofsStorage::check_history_layout_at(dir.path()).unwrap_err();
        assert!(is_unrepairable(&err), "unexpected error: {err:?}");

        let storage = open_raw(&dir);
        assert_eq!(version(&storage), 0, "a refused migration must not record the version");
        assert_eq!(shards(&storage), layout, "a refused migration must not change the store");
    }

    // The migration is one transaction across all four tables: a refusal in the last table scanned
    // must undo the re-keys already made in the first.
    #[test]
    fn a_refused_migration_leaves_every_table_unchanged() {
        let (dir, storage) = setup(1..=1);
        let path = || -> reth_trie::StoredNibbles { Nibbles::from_nibbles([0xf, 0xe]).into() };
        put_rows::<V2AccountsTrieHistory>(
            &storage,
            &[(AccountTrieShardedKey::new(path(), 9), vec![3])],
        );
        let slot = |highest| HashedStorageShardedKey {
            hashed_address: ADDR,
            sharded_key: ShardedKey::new(B256::repeat_byte(1), highest),
        };
        put_rows::<V2HashedStoragesHistory>(&storage, &[(slot(5), vec![4, 5]), (slot(9), vec![2])]);
        let account_trie_before = rows::<V2AccountsTrieHistory>(&storage);
        set_version(&storage, 0);
        drop(storage);

        let err = open(&dir).unwrap_err();
        assert!(
            matches!(&err, OpProofsStorageError::UnrepairableHistory { table, .. }
                if *table == V2HashedStoragesHistory::NAME),
            "unexpected error: {err:?}"
        );
        let storage = open_raw(&dir);
        assert_eq!(rows::<V2AccountsTrieHistory>(&storage), account_trie_before);
        assert_eq!(version(&storage), 0);
    }

    // The last shard of a key is judged against the next row of a different key, so a sentinel
    // directly followed by a neighbouring slot's shards must be left alone.
    #[test]
    fn migration_respects_storage_slot_boundaries() {
        let (dir, storage) = setup(1..=1);
        let key = |address: u8, slot: u8, highest| HashedStorageShardedKey {
            hashed_address: B256::repeat_byte(address),
            sharded_key: ShardedKey::new(B256::repeat_byte(slot), highest),
        };
        put_rows::<V2HashedStoragesHistory>(
            &storage,
            &[
                (key(1, 1, 4), vec![1, 4]),
                (key(1, 1, u64::MAX), vec![5, 6]),
                (key(1, 2, 9), vec![3]),
                (key(2, 1, u64::MAX), vec![2]),
            ],
        );
        set_version(&storage, 0);

        let rekeyed = storage.migrate_history_layout(dir.path()).unwrap();
        assert_eq!(rekeyed, MisplacedShardCounts { hashed_storages: 1, ..Default::default() });
        assert_eq!(
            rows::<V2HashedStoragesHistory>(&storage),
            vec![
                (key(1, 1, 4), vec![1, 4]),
                (key(1, 1, u64::MAX), vec![5, 6]),
                (key(1, 2, u64::MAX), vec![3]),
                (key(2, 1, u64::MAX), vec![2]),
            ]
        );
    }

    // Trie paths `0` and `00` are distinct logical keys that sort next to each other.
    #[test]
    fn migration_respects_storage_trie_path_boundaries() {
        let (dir, storage) = setup(1..=1);
        let address = B256::repeat_byte(1);
        let key = |path: &[u8], highest| {
            StorageTrieShardedKey::new(address, Nibbles::from_nibbles(path).into(), highest)
        };
        put_rows::<V2StoragesTrieHistory>(
            &storage,
            &[
                (key(&[0], u64::MAX), vec![1, 2]),
                (key(&[0, 0], 7), vec![3]),
                (key(&[0, 1], 2), vec![2]),
                (key(&[0, 1], u64::MAX), vec![4]),
            ],
        );
        set_version(&storage, 0);

        let rekeyed = storage.migrate_history_layout(dir.path()).unwrap();
        assert_eq!(rekeyed, MisplacedShardCounts { storage_trie: 1, ..Default::default() });
        assert_eq!(
            rows::<V2StoragesTrieHistory>(&storage),
            vec![
                (key(&[0], u64::MAX), vec![1, 2]),
                (key(&[0, 0], u64::MAX), vec![3]),
                (key(&[0, 1], 2), vec![2]),
                (key(&[0, 1], u64::MAX), vec![4]),
            ]
        );
    }

    // Two unwinds can leave a shard under a stale key that is exactly the expected key of the
    // shard after it; moving shards in key order frees that key first.
    #[test]
    fn migration_moves_shards_whose_targets_collide_with_stale_keys() {
        let (dir, storage) = setup(1..=2_100);
        set_shards(
            &storage,
            &[(2_000, 1..=1_200), (3_200, 1_201..=2_000), (u64::MAX, 2_001..=2_100)],
        );
        set_version(&storage, 0);
        assert!(!wrong_reads(&storage, 1..=2_100).is_empty());

        assert_eq!(storage.migrate_history_layout(dir.path()).unwrap(), hashed_accounts(2));
        assert_eq!(
            shards(&storage),
            vec![(1_200, 1, 1_200), (2_000, 1_201, 2_000), (u64::MAX, 2_001, 2_100)]
        );
        assert_eq!(wrong_reads(&storage, 1..=2_100), Vec::<u64>::new());
    }

    #[test]
    fn verdict_classifies_every_report() {
        let report = |version, misplaced| HistoryLayoutReport {
            version,
            misplaced: hashed_accounts(misplaced),
        };
        let current = HISTORY_LAYOUT_VERSION;
        for (version, misplaced, verdict) in [
            (0, 0, HistoryLayoutVerdict::Clean),
            (0, 3, HistoryLayoutVerdict::NeedsMigration),
            (current, 0, HistoryLayoutVerdict::Clean),
            (current, 3, HistoryLayoutVerdict::DamagedAfterMigration),
            (current + 1, 0, HistoryLayoutVerdict::NewerLayout),
            (current + 1, 3, HistoryLayoutVerdict::NewerLayout),
        ] {
            assert_eq!(
                report(version, misplaced).verdict(),
                verdict,
                "version {version}, {misplaced} misplaced"
            );
        }
    }
}
