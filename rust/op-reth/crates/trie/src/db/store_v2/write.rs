//! Read-write helpers for [`MdbxProofsProviderV2`].

use super::MdbxProofsProviderV2;
use crate::{
    BlockStateDiff, OpProofsStorageResult,
    api::WriteCounts,
    db::{
        BlockNumberHash, ProofWindowKey, V2ProofWindow,
        models::{
            AccountTrieShardedKey, BlockNumberHashedAddress, HashedAccountBeforeTx,
            HashedAccountShardedKey, HashedStorageShardedKey, StorageTrieShardedKey,
            TrieChangeSetsEntry, V2AccountTrieChangeSets, V2AccountsTrie, V2AccountsTrieHistory,
            V2HashedAccountChangeSets, V2HashedAccounts, V2HashedAccountsHistory,
            V2HashedStorageChangeSets, V2HashedStorages, V2HashedStoragesHistory,
            V2StorageTrieChangeSets, V2StoragesTrie, V2StoragesTrieHistory,
        },
    },
};
use alloy_primitives::{B256, BlockNumber, U256};
use reth_db::{
    BlockNumberList, DatabaseError,
    cursor::{DbCursorRO, DbCursorRW, DbDupCursorRO, DbDupCursorRW},
    models::sharded_key::ShardedKey,
    table::Table,
    transaction::{DbTx, DbTxMut},
};
use reth_primitives_traits::{Account, StorageEntry};
use reth_trie::{
    BranchNodeCompact, HashedPostStateSorted, StorageTrieEntry, StoredNibbles, StoredNibblesSubKey,
    updates::TrieUpdatesSorted,
};
use std::collections::{BTreeMap, BTreeSet};

use super::NUM_OF_INDICES_IN_SHARD;

/// Collector for batched history bitmap appends.
///
/// Instead of performing one `seek_exact` + decode + push + re-encode +
/// `upsert` per entry (the old inline approach), we collect `(key, block)`
/// pairs and flush them at the end of a batch.  For keys that appear in
/// multiple blocks within the batch this turns N round-trips into 1.
/// The `BTreeMap` also gives sorted iteration, so cursor seeks during
/// flush are sequential (cache-friendly).
#[derive(Default)]
pub(super) struct HistoryCollector {
    pub(super) account_trie: BTreeMap<StoredNibbles, Vec<BlockNumber>>,
    pub(super) storage_trie: BTreeMap<(B256, StoredNibbles), Vec<BlockNumber>>,
    pub(super) hashed_accounts: BTreeMap<B256, Vec<BlockNumber>>,
    pub(super) hashed_storages: BTreeMap<(B256, B256), Vec<BlockNumber>>,
}

/// Pre-opened write cursors for the 8 tables touched by
/// [`MdbxProofsProviderV2::store_block_updates`].
struct WriteCursors<TX: DbTxMut + DbTx> {
    account_trie_state: <TX as DbTxMut>::CursorMut<V2AccountsTrie>,
    account_trie_cs: <TX as DbTxMut>::DupCursorMut<V2AccountTrieChangeSets>,
    storage_trie_state: <TX as DbTxMut>::DupCursorMut<V2StoragesTrie>,
    storage_trie_cs: <TX as DbTxMut>::DupCursorMut<V2StorageTrieChangeSets>,
    hashed_accounts_state: <TX as DbTxMut>::CursorMut<V2HashedAccounts>,
    hashed_accounts_cs: <TX as DbTxMut>::DupCursorMut<V2HashedAccountChangeSets>,
    hashed_storages_state: <TX as DbTxMut>::DupCursorMut<V2HashedStorages>,
    hashed_storages_cs: <TX as DbTxMut>::DupCursorMut<V2HashedStorageChangeSets>,
}

impl<TX: DbTxMut + DbTx> WriteCursors<TX> {
    fn new(tx: &TX) -> OpProofsStorageResult<Self> {
        Ok(Self {
            account_trie_state: tx.cursor_write::<V2AccountsTrie>()?,
            account_trie_cs: tx.cursor_dup_write::<V2AccountTrieChangeSets>()?,
            storage_trie_state: tx.cursor_dup_write::<V2StoragesTrie>()?,
            storage_trie_cs: tx.cursor_dup_write::<V2StorageTrieChangeSets>()?,
            hashed_accounts_state: tx.cursor_write::<V2HashedAccounts>()?,
            hashed_accounts_cs: tx.cursor_dup_write::<V2HashedAccountChangeSets>()?,
            hashed_storages_state: tx.cursor_dup_write::<V2HashedStorages>()?,
            hashed_storages_cs: tx.cursor_dup_write::<V2HashedStorageChangeSets>()?,
        })
    }
}

/// Append multiple block numbers to a sharded history bitmap in a single
/// seek+decode+push-all+upsert round-trip.
///
/// This is the batched equivalent of a single-block
/// `append_history_index_with_cursor` call.
///
/// # Assumption: `block_numbers` always belong to the last shard
///
/// This function **only seeks the last shard** (keyed by `sharded_key_factory(u64::MAX)`).
/// It relies on the invariant that all values in `block_numbers` are strictly greater than
/// every block number already stored for this key across *all* shards. Because shards are
/// ordered and the last shard (sentinel key `u64::MAX`) holds the global maximum for the key,
/// verifying `first_new > last_shard.last()` is sufficient: a block that follows the
/// last-shard tail necessarily follows every earlier shard too.
///
/// Callers must guarantee that block numbers are provided in strictly increasing order across
/// successive calls (i.e. history is append-only).
fn append_history_indices_batched<T>(
    cursor: &mut (impl DbCursorRO<T> + DbCursorRW<T>),
    block_numbers: &[BlockNumber],
    sharded_key_factory: impl Fn(BlockNumber) -> T::Key,
) -> OpProofsStorageResult<()>
where
    T: Table<Value = BlockNumberList>,
    T::Key: Clone,
{
    if block_numbers.is_empty() {
        return Ok(());
    }

    let last_key = sharded_key_factory(u64::MAX);

    let mut last_shard = cursor
        .seek_exact(last_key.clone())?
        .map(|(_, list)| list)
        .unwrap_or_else(BlockNumberList::empty);

    if let Some(first_new) = block_numbers.first().copied() {
        debug_assert!(
            block_numbers.windows(2).all(|w| w[0] <= w[1]),
            "append_history_indices_batched expects non-decreasing block_numbers"
        );
        // Verifies the last-shard assumption: `first_new > existing_last` guarantees
        // that block_numbers extend beyond all previously stored entries (since the last
        // shard contains the global maximum), so they correctly belong in the last shard.
        if let Some(existing_last) = last_shard.iter().next_back() {
            debug_assert!(
                first_new > existing_last,
                "block_numbers must extend the last shard's tail \
                 (first_new={first_new} <= existing_last={existing_last}); \
                 all new block numbers must be strictly greater than any previously \
                 stored block number for this key"
            );
        }
    }

    for &bn in block_numbers {
        last_shard
            .push(bn)
            .map_err(|e| DatabaseError::Other(format!("IntegerList push error: {e}")))?;
    }

    // Fast path: fits in one shard
    if last_shard.len() <= NUM_OF_INDICES_IN_SHARD as u64 {
        cursor.upsert(last_key, &last_shard)?;
        return Ok(());
    }

    // Slow path: rechunk
    if cursor.seek_exact(last_key)?.is_some() {
        cursor.delete_current()?;
    }

    let all_values: Vec<u64> = last_shard.iter().collect();
    let total_chunks = all_values.chunks(NUM_OF_INDICES_IN_SHARD).len();

    for (i, chunk) in all_values.chunks(NUM_OF_INDICES_IN_SHARD).enumerate() {
        let shard = BlockNumberList::new_pre_sorted(chunk.iter().copied());
        let key = if i < total_chunks - 1 {
            sharded_key_factory(*chunk.last().expect("non-empty chunk"))
        } else {
            sharded_key_factory(u64::MAX)
        };
        cursor.upsert(key, &shard)?;
    }

    Ok(())
}

/// Append one storage-trie node into the changeset and record it in the history collector.
fn append_storage_trie_entry(
    cs_cursor: &mut impl DbDupCursorRW<V2StorageTrieChangeSets>,
    cs_key: BlockNumberHashedAddress,
    nibbles: StoredNibblesSubKey,
    node: Option<BranchNodeCompact>,
    collector: &mut HistoryCollector,
) -> OpProofsStorageResult<()> {
    let (block_number, hashed_address) = cs_key.0;
    cs_cursor.append_dup(cs_key, TrieChangeSetsEntry { nibbles: nibbles.clone(), node })?;
    collector
        .storage_trie
        .entry((hashed_address, StoredNibbles(nibbles.0)))
        .or_default()
        .push(block_number);
    Ok(())
}

/// Snapshot all existing storage-trie nodes for `hashed_address` into the changeset and
/// collector, then delete them from the state table.
///
/// Returns the set of nibble subkeys that were snapshotted so the caller can avoid
/// double-recording changeset entries for those nodes in the subsequent per-node loop.
///
/// Used for the `is_deleted` (full-wipe) path in [`MdbxProofsProviderV2::write_storage_trie`].
fn snapshot_and_wipe_storage_trie(
    state_cursor: &mut (
             impl DbCursorRO<V2StoragesTrie>
             + DbDupCursorRO<V2StoragesTrie>
             + DbDupCursorRW<V2StoragesTrie>
         ),
    cs_cursor: &mut impl DbDupCursorRW<V2StorageTrieChangeSets>,
    cs_key: BlockNumberHashedAddress,
    collector: &mut HistoryCollector,
) -> OpProofsStorageResult<BTreeSet<StoredNibblesSubKey>> {
    let hashed_address = cs_key.0.1;
    let mut wiped_nibbles = BTreeSet::new();
    if let Some((_key, first_entry)) = state_cursor.seek_exact(hashed_address)? {
        wiped_nibbles.insert(first_entry.nibbles.clone());
        append_storage_trie_entry(
            cs_cursor,
            cs_key,
            first_entry.nibbles,
            Some(first_entry.node),
            collector,
        )?;
        while let Some((_, entry)) = state_cursor.next_dup()? {
            wiped_nibbles.insert(entry.nibbles.clone());
            append_storage_trie_entry(
                cs_cursor,
                cs_key,
                entry.nibbles,
                Some(entry.node),
                collector,
            )?;
        }
        if state_cursor.seek_exact(hashed_address)?.is_some() {
            state_cursor.delete_current_duplicates()?;
        }
    }
    Ok(wiped_nibbles)
}

/// Write a single storage-trie node update: snapshot the old node into the changeset,
/// record in the history collector, then apply the new value (upsert or delete).
///
/// Used for the per-node path in [`MdbxProofsProviderV2::write_storage_trie`].
fn write_storage_trie_node(
    state_cursor: &mut (impl DbDupCursorRO<V2StoragesTrie> + DbCursorRW<V2StoragesTrie>),
    cs_cursor: &mut impl DbDupCursorRW<V2StorageTrieChangeSets>,
    cs_key: BlockNumberHashedAddress,
    subkey: StoredNibblesSubKey,
    maybe_node: &Option<BranchNodeCompact>,
    collector: &mut HistoryCollector,
) -> OpProofsStorageResult<()> {
    let hashed_address = cs_key.0.1;
    let old_entry = state_cursor
        .seek_by_key_subkey(hashed_address, subkey.clone())?
        .filter(|e| e.nibbles == subkey);
    let had_old = old_entry.is_some();
    let old_node = old_entry.map(|e| e.node);

    append_storage_trie_entry(cs_cursor, cs_key, subkey.clone(), old_node, collector)?;

    if had_old {
        state_cursor.delete_current()?;
    }
    if let Some(node) = maybe_node {
        state_cursor
            .upsert(hashed_address, &StorageTrieEntry { nibbles: subkey, node: node.clone() })?;
    }
    Ok(())
}

/// Append one hashed-storage slot into the changeset and record it in the history collector.
fn append_hashed_storage_entry(
    cs_cursor: &mut impl DbDupCursorRW<V2HashedStorageChangeSets>,
    cs_key: BlockNumberHashedAddress,
    entry: StorageEntry,
    collector: &mut HistoryCollector,
) -> OpProofsStorageResult<()> {
    let (block_number, hashed_address) = cs_key.0;
    cs_cursor.append_dup(cs_key, entry)?;
    collector.hashed_storages.entry((hashed_address, entry.key)).or_default().push(block_number);
    Ok(())
}

/// Snapshot all existing hashed-storage slots for `hashed_address` into the changeset and
/// collector, delete them from the state table, and return the set of slot keys that were wiped.
///
/// Used for the `is_wiped` (full-wipe) path in [`MdbxProofsProviderV2::write_hashed_storages`].
fn snapshot_and_wipe_hashed_storage(
    state_cursor: &mut (
             impl DbCursorRO<V2HashedStorages>
             + DbDupCursorRO<V2HashedStorages>
             + DbDupCursorRW<V2HashedStorages>
         ),
    cs_cursor: &mut impl DbDupCursorRW<V2HashedStorageChangeSets>,
    cs_key: BlockNumberHashedAddress,
    collector: &mut HistoryCollector,
) -> OpProofsStorageResult<alloy_primitives::map::B256Set> {
    let hashed_address = cs_key.0.1;
    let mut wiped_slots = alloy_primitives::map::B256Set::default();
    if let Some(entry) = state_cursor.seek_by_key_subkey(hashed_address, B256::ZERO)? {
        append_hashed_storage_entry(cs_cursor, cs_key, entry, collector)?;
        wiped_slots.insert(entry.key);
        while let Some(entry) = state_cursor.next_dup_val()? {
            append_hashed_storage_entry(cs_cursor, cs_key, entry, collector)?;
            wiped_slots.insert(entry.key);
        }
        if state_cursor.seek_exact(hashed_address)?.is_some() {
            state_cursor.delete_current_duplicates()?;
        }
    }
    Ok(wiped_slots)
}

/// Write a single hashed-storage slot update: snapshot the old value into the changeset,
/// record in the history collector, then apply the new value (upsert or delete).
///
/// Used for the per-slot path in [`MdbxProofsProviderV2::write_hashed_storages`].
fn write_hashed_storage_slot(
    state_cursor: &mut (impl DbDupCursorRO<V2HashedStorages> + DbCursorRW<V2HashedStorages>),
    cs_cursor: &mut impl DbDupCursorRW<V2HashedStorageChangeSets>,
    cs_key: BlockNumberHashedAddress,
    storage_key: B256,
    value: U256,
    collector: &mut HistoryCollector,
) -> OpProofsStorageResult<()> {
    let hashed_address = cs_key.0.1;
    let old_entry = state_cursor
        .seek_by_key_subkey(hashed_address, storage_key)?
        .filter(|e| e.key == storage_key);
    let had_old = old_entry.is_some();
    let old_value = old_entry.map(|e| e.value).unwrap_or(U256::ZERO);

    append_hashed_storage_entry(
        cs_cursor,
        cs_key,
        StorageEntry { key: storage_key, value: old_value },
        collector,
    )?;

    if had_old {
        state_cursor.delete_current()?;
    }
    if value != U256::ZERO {
        state_cursor.upsert(hashed_address, &StorageEntry { key: storage_key, value })?;
    }
    Ok(())
}

impl<TX: DbTxMut + DbTx> MdbxProofsProviderV2<TX> {
    pub(super) fn set_earliest_block_number_inner(
        &self,
        block_number: u64,
        hash: B256,
    ) -> OpProofsStorageResult<()> {
        let mut cursor = self.tx.cursor_write::<V2ProofWindow>()?;
        cursor.upsert(ProofWindowKey::EarliestBlock, &BlockNumberHash::new(block_number, hash))?;
        Ok(())
    }

    pub(super) fn set_latest_block_number_inner(
        &self,
        block_number: u64,
        hash: B256,
    ) -> OpProofsStorageResult<()> {
        let mut cursor = self.tx.cursor_write::<V2ProofWindow>()?;
        cursor.upsert(ProofWindowKey::LatestBlock, &BlockNumberHash::new(block_number, hash))?;
        Ok(())
    }

    /// Prune-specific history removal: for a given logical key, seek its first
    /// history shard and walk forward, removing all block numbers that fall
    /// within `range`.  Requires only **one seek per unique key** (instead
    /// of one seek per block) and uses a simple range check instead of a
    /// set-membership lookup.
    pub(super) fn prune_history_range_for_key<T>(
        cursor: &mut (impl DbCursorRO<T> + DbCursorRW<T>),
        range: &std::ops::RangeInclusive<u64>,
        first_shard_key: T::Key,
        same_logical_key: impl Fn(&T::Key) -> bool,
    ) -> OpProofsStorageResult<()>
    where
        T: Table<Value = BlockNumberList>,
        T::Key: Clone,
    {
        let mut entry = cursor.seek(first_shard_key)?;
        while let Some((key, list)) = entry &&
            same_logical_key(&key) &&
            list.iter().next().is_some_and(|first| first <= *range.end())
        {
            let original_len: usize = list
                .len()
                .try_into()
                .map_err(|e| DatabaseError::Other(format!("shard length overflow: {e}")))?;
            let filtered: Vec<u64> = list.iter().filter(|&bn| !range.contains(&bn)).collect();

            if filtered.is_empty() {
                // Entire shard pruned — delete and advance.
                cursor.delete_current()?;
                entry = cursor.current()?;
            } else if filtered.len() < original_len {
                // Partial prune — update shard and advance.
                let new_list = BlockNumberList::new_pre_sorted(filtered);
                cursor.upsert(key, &new_list)?;
                entry = cursor.next()?;
            } else {
                // No blocks in this shard were in range; advance to the next shard.
                entry = cursor.next()?;
            }
        }

        Ok(())
    }

    /// Remove block numbers from all 4 history bitmap tables by reading the
    /// changeset tables to find exactly which keys were affected.
    ///
    /// each changeset table is walked **once**, entries are
    /// deduplicated by key into a `BTreeMap<key, BTreeSet<block_number>>`,
    /// and then each unique key's bitmap shard(s) are edited in a single
    /// batch operation through a **reused cursor**.
    /// Single forward scan over account-trie changesets for `range`.
    ///
    /// In one pass: restores the pre-range state for every affected path,
    /// collects the set of affected paths for history bitmap removal, and
    /// deletes the changeset entries.
    ///
    /// Correctness: the first occurrence of each path in a forward scan is the
    /// *smallest* block number in the range, whose old-value is exactly the
    /// state before the entire range — the value we need to restore.
    /// Scan `V2AccountTrieChangeSets` over `range`, collect `path → old_node`
    /// restorations (first occurrence wins), and delete all scanned changeset entries.
    fn scan_and_delete_account_trie_cs(
        &self,
        range: &std::ops::RangeInclusive<u64>,
    ) -> OpProofsStorageResult<BTreeMap<StoredNibbles, Option<BranchNodeCompact>>> {
        let mut restorations = BTreeMap::new();
        let mut cs = self.tx.cursor_dup_write::<V2AccountTrieChangeSets>()?;
        let mut entry = cs.seek(*range.start())?;
        while let Some((block_num, val)) = entry {
            if !range.contains(&block_num) {
                break;
            }
            restorations.entry(StoredNibbles(val.nibbles.0)).or_insert(val.node);
            while let Some((_, val)) = cs.next_dup()? {
                restorations.entry(StoredNibbles(val.nibbles.0)).or_insert(val.node);
            }
            cs.delete_current_duplicates()?;
            entry = cs.current()?;
        }
        Ok(restorations)
    }

    /// Scan `V2HashedAccountChangeSets` over `range`, collect `address → old_account`
    /// restorations (first occurrence wins), and delete all scanned changeset entries.
    fn scan_and_delete_hashed_account_cs(
        &self,
        range: &std::ops::RangeInclusive<u64>,
    ) -> OpProofsStorageResult<BTreeMap<B256, Option<Account>>> {
        let mut restorations = BTreeMap::new();
        let mut cs = self.tx.cursor_dup_write::<V2HashedAccountChangeSets>()?;
        let mut entry = cs.seek(*range.start())?;
        while let Some((block_num, val)) = entry {
            if !range.contains(&block_num) {
                break;
            }
            restorations.entry(val.hashed_address).or_insert(val.info);
            while let Some((_, val)) = cs.next_dup()? {
                restorations.entry(val.hashed_address).or_insert(val.info);
            }
            cs.delete_current_duplicates()?;
            entry = cs.current()?;
        }
        Ok(restorations)
    }

    pub(super) fn unwind_and_collect_account_trie(
        &self,
        range: &std::ops::RangeInclusive<u64>,
    ) -> OpProofsStorageResult<BTreeSet<StoredNibbles>> {
        let restorations = self.scan_and_delete_account_trie_cs(range)?;
        let mut state = self.tx.cursor_write::<V2AccountsTrie>()?;
        for (path, old_node) in &restorations {
            match old_node {
                Some(node) => state.upsert(path.clone(), node)?,
                None => {
                    if state.seek_exact(path.clone())?.is_some() {
                        state.delete_current()?;
                    }
                }
            }
        }
        Ok(restorations.into_keys().collect())
    }

    /// Scan `V2StorageTrieChangeSets` over `range`, collect `(address, nibbles) → old_node`
    /// restorations (first occurrence wins), and delete all scanned changeset entries.
    fn scan_and_delete_storage_trie_cs(
        &self,
        range: &std::ops::RangeInclusive<u64>,
    ) -> OpProofsStorageResult<BTreeMap<(B256, StoredNibblesSubKey), Option<BranchNodeCompact>>>
    {
        let mut restorations = BTreeMap::new();
        let mut cs = self.tx.cursor_dup_write::<V2StorageTrieChangeSets>()?;
        let start = BlockNumberHashedAddress((*range.start(), B256::ZERO));
        let end = BlockNumberHashedAddress((*range.end(), B256::repeat_byte(0xff)));
        let mut entry = cs.seek(start)?;
        while let Some((key, val)) = entry {
            if key > end || key < start {
                break;
            }
            restorations.entry((key.0.1, val.nibbles.clone())).or_insert(val.node);
            while let Some((k, val)) = cs.next_dup()? {
                restorations.entry((k.0.1, val.nibbles.clone())).or_insert(val.node);
            }
            cs.delete_current_duplicates()?;
            entry = cs.current()?;
        }
        Ok(restorations)
    }

    /// Scan `V2HashedStorageChangeSets` over `range`, collect `(address, slot) → old_value`
    /// restorations (first occurrence wins), and delete all scanned changeset entries.
    fn scan_and_delete_hashed_storage_cs(
        &self,
        range: &std::ops::RangeInclusive<u64>,
    ) -> OpProofsStorageResult<BTreeMap<(B256, B256), U256>> {
        let mut restorations = BTreeMap::new();
        let mut cs = self.tx.cursor_dup_write::<V2HashedStorageChangeSets>()?;
        let start = BlockNumberHashedAddress((*range.start(), B256::ZERO));
        let end = BlockNumberHashedAddress((*range.end(), B256::repeat_byte(0xff)));
        let mut entry = cs.seek(start)?;
        while let Some((key, val)) = entry {
            if key > end || key < start {
                break;
            }
            restorations.entry((key.0.1, val.key)).or_insert(val.value);
            while let Some((k, val)) = cs.next_dup()? {
                restorations.entry((k.0.1, val.key)).or_insert(val.value);
            }
            cs.delete_current_duplicates()?;
            entry = cs.current()?;
        }
        Ok(restorations)
    }

    /// Single forward scan over storage-trie changesets for `range`.
    ///
    /// See [`Self::unwind_and_collect_account_trie`] for the correctness argument.
    pub(super) fn unwind_and_collect_storage_trie(
        &self,
        range: &std::ops::RangeInclusive<u64>,
    ) -> OpProofsStorageResult<BTreeSet<(B256, StoredNibbles)>> {
        let restorations = self.scan_and_delete_storage_trie_cs(range)?;
        let mut state = self.tx.cursor_dup_write::<V2StoragesTrie>()?;
        for ((addr, subkey), old_node) in &restorations {
            if state
                .seek_by_key_subkey(*addr, subkey.clone())?
                .filter(|e| e.nibbles == *subkey)
                .is_some()
            {
                state.delete_current()?;
            }
            if let Some(node) = old_node {
                state.upsert(
                    *addr,
                    &StorageTrieEntry { nibbles: subkey.clone(), node: node.clone() },
                )?;
            }
        }
        Ok(restorations.into_keys().map(|(addr, subkey)| (addr, StoredNibbles(subkey.0))).collect())
    }

    /// Single forward scan over hashed-account changesets for `range`.
    ///
    /// See [`Self::unwind_and_collect_account_trie`] for the correctness argument.
    pub(super) fn unwind_and_collect_hashed_accounts(
        &self,
        range: &std::ops::RangeInclusive<u64>,
    ) -> OpProofsStorageResult<BTreeSet<B256>> {
        let restorations = self.scan_and_delete_hashed_account_cs(range)?;
        let mut state = self.tx.cursor_write::<V2HashedAccounts>()?;
        for (addr, old_account) in &restorations {
            match old_account {
                Some(account) => state.upsert(*addr, account)?,
                None => {
                    if state.seek_exact(*addr)?.is_some() {
                        state.delete_current()?;
                    }
                }
            }
        }
        Ok(restorations.into_keys().collect())
    }

    /// Single forward scan over hashed-storage changesets for `range`.
    ///
    /// See [`Self::unwind_and_collect_account_trie`] for the correctness argument.
    pub(super) fn unwind_and_collect_hashed_storages(
        &self,
        range: &std::ops::RangeInclusive<u64>,
    ) -> OpProofsStorageResult<BTreeSet<(B256, B256)>> {
        let restorations = self.scan_and_delete_hashed_storage_cs(range)?;
        let mut state = self.tx.cursor_dup_write::<V2HashedStorages>()?;
        for ((addr, slot), old_value) in &restorations {
            if state.seek_by_key_subkey(*addr, *slot)?.filter(|e| e.key == *slot).is_some() {
                state.delete_current()?;
            }
            if *old_value != U256::ZERO {
                state.upsert(*addr, &StorageEntry { key: *slot, value: *old_value })?;
            }
        }
        Ok(restorations.into_keys().collect())
    }

    /// Core write logic for a single block.
    ///
    /// Delegates each data domain to a focused helper, then assembles counts.
    pub(super) fn store_block_updates(
        &self,
        block_number: BlockNumber,
        block_state_diff: BlockStateDiff,
        collector: &mut HistoryCollector,
    ) -> OpProofsStorageResult<WriteCounts> {
        let mut cursors = WriteCursors::new(&self.tx)?;
        let BlockStateDiff { sorted_trie_updates, sorted_post_state } = block_state_diff;
        Ok(WriteCounts {
            account_trie_updates_written_total: Self::write_account_trie(
                block_number,
                &sorted_trie_updates,
                &mut cursors,
                collector,
            )?,
            storage_trie_updates_written_total: Self::write_storage_trie(
                block_number,
                &sorted_trie_updates,
                &mut cursors,
                collector,
            )?,
            hashed_accounts_written_total: Self::write_hashed_accounts(
                block_number,
                &sorted_post_state,
                &mut cursors,
                collector,
            )?,
            hashed_storages_written_total: Self::write_hashed_storages(
                block_number,
                &sorted_post_state,
                &mut cursors,
                collector,
            )?,
        })
    }

    /// Write account trie branch-node updates for one block.
    ///
    /// For each changed path: save the old node to the changeset, record the
    /// block number in the history bitmap collector, then apply the new value
    /// (upsert or delete) to the current-state table.
    fn write_account_trie(
        block_number: BlockNumber,
        updates: &TrieUpdatesSorted,
        cursors: &mut WriteCursors<TX>,
        collector: &mut HistoryCollector,
    ) -> OpProofsStorageResult<u64> {
        let state_cursor = &mut cursors.account_trie_state;
        let cs_cursor = &mut cursors.account_trie_cs;
        let mut count = 0u64;

        for (nibbles, maybe_node) in updates.account_nodes_ref() {
            let stored = StoredNibbles(*nibbles);

            let old_entry = state_cursor.seek_exact(stored.clone())?;
            let old_node = old_entry.as_ref().map(|(_, node)| node.clone());
            let had_old = old_entry.is_some();

            cs_cursor.append_dup(
                block_number,
                TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(*nibbles), node: old_node },
            )?;
            collector.account_trie.entry(stored.clone()).or_default().push(block_number);

            match maybe_node {
                Some(node) => state_cursor.upsert(stored, node)?,
                None => {
                    if had_old {
                        state_cursor.delete_current()?;
                    }
                }
            }
            count += 1;
        }
        Ok(count)
    }

    /// Write storage trie branch-node updates for one block.
    ///
    /// Handles the `is_deleted` wipe path (snapshot all existing nodes into
    /// the changeset before clearing) as well as per-node updates.
    fn write_storage_trie(
        block_number: BlockNumber,
        updates: &TrieUpdatesSorted,
        cursors: &mut WriteCursors<TX>,
        collector: &mut HistoryCollector,
    ) -> OpProofsStorageResult<u64> {
        let state_cursor = &mut cursors.storage_trie_state;
        let cs_cursor = &mut cursors.storage_trie_cs;
        let mut count = 0u64;

        for (hashed_address, nodes) in updates.storage_tries_ref() {
            let cs_key = BlockNumberHashedAddress((block_number, *hashed_address));

            if nodes.is_deleted {
                let wiped_nibbles =
                    snapshot_and_wipe_storage_trie(state_cursor, cs_cursor, cs_key, collector)?;
                count += 1;

                // After a wipe the state table is empty for this address, so there is no
                // old node to seek or delete.  Nodes whose nibbles were already snapshotted
                // above must not be recorded again (that would create conflicting duplicate
                // changeset entries).  Brand-new nibbles get a `None` old-value entry so
                // that unwind knows to delete them.
                for (nibbles, maybe_node) in nodes.storage_nodes_ref() {
                    let subkey = StoredNibblesSubKey(*nibbles);
                    if !wiped_nibbles.contains(&subkey) {
                        append_storage_trie_entry(
                            cs_cursor,
                            cs_key,
                            subkey.clone(),
                            None,
                            collector,
                        )?;
                    }
                    if let Some(node) = maybe_node {
                        state_cursor.upsert(
                            *hashed_address,
                            &StorageTrieEntry { nibbles: subkey, node: node.clone() },
                        )?;
                    }
                    count += 1;
                }
            } else {
                for (nibbles, maybe_node) in nodes.storage_nodes_ref() {
                    write_storage_trie_node(
                        state_cursor,
                        cs_cursor,
                        cs_key,
                        StoredNibblesSubKey(*nibbles),
                        maybe_node,
                        collector,
                    )?;
                    count += 1;
                }
            }
        }
        Ok(count)
    }

    /// Write hashed-account updates for one block.
    ///
    /// For each changed account: save the old account to the changeset, record
    /// the block number in the history bitmap collector, then apply the new
    /// value (upsert or delete) to the current-state table.
    fn write_hashed_accounts(
        block_number: BlockNumber,
        post_state: &HashedPostStateSorted,
        cursors: &mut WriteCursors<TX>,
        collector: &mut HistoryCollector,
    ) -> OpProofsStorageResult<u64> {
        let state_cursor = &mut cursors.hashed_accounts_state;
        let cs_cursor = &mut cursors.hashed_accounts_cs;
        let mut count = 0u64;

        for (hashed_address, maybe_account) in &post_state.accounts {
            let old_entry = state_cursor.seek_exact(*hashed_address)?;
            let old_account = old_entry.as_ref().map(|(_, acc)| *acc);
            let had_old = old_entry.is_some();

            cs_cursor.append_dup(
                block_number,
                HashedAccountBeforeTx::new(*hashed_address, old_account),
            )?;
            collector.hashed_accounts.entry(*hashed_address).or_default().push(block_number);

            match maybe_account {
                Some(account) => state_cursor.upsert(*hashed_address, account)?,
                None => {
                    if had_old {
                        state_cursor.delete_current()?;
                    }
                }
            }
            count += 1;
        }
        Ok(count)
    }

    /// Write hashed-storage updates for one block.
    ///
    /// Handles the `is_wiped` path (snapshot all existing slots into the
    /// changeset before clearing) as well as per-slot updates.
    fn write_hashed_storages(
        block_number: BlockNumber,
        post_state: &HashedPostStateSorted,
        cursors: &mut WriteCursors<TX>,
        collector: &mut HistoryCollector,
    ) -> OpProofsStorageResult<u64> {
        let state_cursor = &mut cursors.hashed_storages_state;
        let cs_cursor = &mut cursors.hashed_storages_cs;
        let mut count = 0u64;

        for (hashed_address, storage) in &post_state.storages {
            let cs_key = BlockNumberHashedAddress((block_number, *hashed_address));

            if storage.is_wiped() {
                // Snapshot + wipe existing slots; track what was wiped so the
                // new-slots loop below doesn't double-append changeset entries.
                let wiped_slots =
                    snapshot_and_wipe_hashed_storage(state_cursor, cs_cursor, cs_key, collector)?;

                // Write new slots. Slots not seen during the wipe get a zero
                // old-value entry in the changeset.
                for (storage_key, value) in storage.storage_slots_ref() {
                    if !wiped_slots.contains(storage_key) {
                        append_hashed_storage_entry(
                            cs_cursor,
                            cs_key,
                            StorageEntry { key: *storage_key, value: U256::ZERO },
                            collector,
                        )?;
                    }
                    if *value != U256::ZERO {
                        state_cursor.upsert(
                            *hashed_address,
                            &StorageEntry { key: *storage_key, value: *value },
                        )?;
                    }
                    count += 1;
                }
            } else {
                for (storage_key, value) in storage.storage_slots_ref() {
                    write_hashed_storage_slot(
                        state_cursor,
                        cs_cursor,
                        cs_key,
                        *storage_key,
                        *value,
                        collector,
                    )?;
                    count += 1;
                }
            }
        }
        Ok(count)
    }

    /// Flush all collected history bitmap entries to the database.
    ///
    /// For each unique key, performs a single `seek_exact` + decode +
    /// push-all-block-numbers + re-encode + `upsert` instead of doing
    /// that per-entry.  The `BTreeMap` iteration order ensures cursor
    /// seeks are sequential within each table.
    pub(super) fn flush_collected_history(
        &self,
        collector: HistoryCollector,
    ) -> OpProofsStorageResult<()> {
        macro_rules! flush {
            ($table:ty, $entries:expr, $key_fn:expr) => {
                if !$entries.is_empty() {
                    let mut cursor = self.tx.cursor_write::<$table>()?;
                    for (key, blocks) in $entries {
                        append_history_indices_batched::<$table>(
                            &mut cursor,
                            &blocks,
                            |highest| $key_fn(key.clone(), highest),
                        )?;
                    }
                }
            };
        }

        flush!(V2AccountsTrieHistory, collector.account_trie, |path, highest| {
            AccountTrieShardedKey::new(path, highest)
        });
        flush!(V2StoragesTrieHistory, collector.storage_trie, |(addr, path), highest| {
            StorageTrieShardedKey::new(addr, path, highest)
        });
        flush!(V2HashedAccountsHistory, collector.hashed_accounts, |addr, highest| {
            HashedAccountShardedKey::new(addr, highest)
        });
        flush!(V2HashedStoragesHistory, collector.hashed_storages, |(addr, slot), highest| {
            HashedStorageShardedKey {
                hashed_address: addr,
                sharded_key: ShardedKey::new(slot, highest),
            }
        });

        Ok(())
    }

    /// Unwind all 4 data types in `range`: restore state, collect affected keys,
    /// delete changesets, then remove the affected block numbers from history bitmaps.
    pub(super) fn unwind_changesets_and_history(
        &self,
        range: &std::ops::RangeInclusive<u64>,
    ) -> OpProofsStorageResult<()> {
        let acct_trie_keys = self.unwind_and_collect_account_trie(range)?;
        let stor_trie_keys = self.unwind_and_collect_storage_trie(range)?;
        let acct_keys = self.unwind_and_collect_hashed_accounts(range)?;
        let stor_keys = self.unwind_and_collect_hashed_storages(range)?;

        self.prune_all_history(range, &acct_trie_keys, &stor_trie_keys, &acct_keys, &stor_keys)
    }

    /// Prune changesets for all 4 data types in `range`, then remove the
    /// affected block numbers from history bitmaps.
    pub(super) fn prune_changesets_and_history(
        &self,
        range: &std::ops::RangeInclusive<u64>,
    ) -> OpProofsStorageResult<WriteCounts> {
        let mut counts = WriteCounts::default();

        let acct_trie_keys = self.prune_account_trie_changesets(range, &mut counts)?;
        let stor_trie_keys = self.prune_storage_trie_changesets(range, &mut counts)?;
        let acct_keys = self.prune_hashed_account_changesets(range, &mut counts)?;
        let stor_keys = self.prune_hashed_storage_changesets(range, &mut counts)?;

        self.prune_all_history(range, &acct_trie_keys, &stor_trie_keys, &acct_keys, &stor_keys)?;

        Ok(counts)
    }

    /// Remove block numbers in `range` from all 4 history bitmap tables for the
    /// given sets of affected keys.
    fn prune_all_history(
        &self,
        range: &std::ops::RangeInclusive<u64>,
        acct_trie_keys: &BTreeSet<StoredNibbles>,
        stor_trie_keys: &BTreeSet<(B256, StoredNibbles)>,
        acct_keys: &BTreeSet<B256>,
        stor_keys: &BTreeSet<(B256, B256)>,
    ) -> OpProofsStorageResult<()> {
        self.prune_account_trie_history(range, acct_trie_keys)?;
        self.prune_storage_trie_history(range, stor_trie_keys)?;
        self.prune_hashed_account_history(range, acct_keys)?;
        self.prune_hashed_storage_history(range, stor_keys)?;
        Ok(())
    }

    /// Phase A: delete all account-trie changeset entries in `range`, returning the
    /// set of nibble paths that were affected (used in Phase B for history pruning).
    pub(super) fn prune_account_trie_changesets(
        &self,
        range: &std::ops::RangeInclusive<u64>,
        counts: &mut WriteCounts,
    ) -> OpProofsStorageResult<BTreeSet<StoredNibbles>> {
        let mut keys: BTreeSet<StoredNibbles> = BTreeSet::new();
        let mut cursor = self.tx.cursor_dup_write::<V2AccountTrieChangeSets>()?;
        let mut entry = cursor.seek(*range.start())?;
        while let Some((block_num, first_val)) = entry {
            if block_num > *range.end() {
                break;
            }
            counts.account_trie_updates_written_total += 1;
            keys.insert(StoredNibbles(first_val.nibbles.0));
            while let Some((_, val)) = cursor.next_dup()? {
                counts.account_trie_updates_written_total += 1;
                keys.insert(StoredNibbles(val.nibbles.0));
            }
            cursor.delete_current_duplicates()?;
            entry = cursor.current()?;
        }
        Ok(keys)
    }

    /// Phase A: delete all storage-trie changeset entries in `range`, returning the
    /// set of `(hashed_address, nibbles)` pairs that were affected.
    pub(super) fn prune_storage_trie_changesets(
        &self,
        range: &std::ops::RangeInclusive<u64>,
        counts: &mut WriteCounts,
    ) -> OpProofsStorageResult<BTreeSet<(B256, StoredNibbles)>> {
        let mut keys: BTreeSet<(B256, StoredNibbles)> = BTreeSet::new();
        let mut cursor = self.tx.cursor_dup_write::<V2StorageTrieChangeSets>()?;
        let start = BlockNumberHashedAddress((*range.start(), B256::ZERO));
        let end = BlockNumberHashedAddress((*range.end(), B256::repeat_byte(0xff)));
        let mut entry = cursor.seek(start)?;
        while let Some((key, first_val)) = entry {
            if key > end {
                break;
            }
            counts.storage_trie_updates_written_total += 1;
            keys.insert((key.0.1, StoredNibbles(first_val.nibbles.0)));
            while let Some((k, val)) = cursor.next_dup()? {
                counts.storage_trie_updates_written_total += 1;
                keys.insert((k.0.1, StoredNibbles(val.nibbles.0)));
            }
            cursor.delete_current_duplicates()?;
            entry = cursor.current()?;
        }
        Ok(keys)
    }

    /// Phase A: delete all hashed-account changeset entries in `range`, returning the
    /// set of hashed addresses that were affected.
    pub(super) fn prune_hashed_account_changesets(
        &self,
        range: &std::ops::RangeInclusive<u64>,
        counts: &mut WriteCounts,
    ) -> OpProofsStorageResult<BTreeSet<B256>> {
        let mut keys: BTreeSet<B256> = BTreeSet::new();
        let mut cursor = self.tx.cursor_dup_write::<V2HashedAccountChangeSets>()?;
        let mut entry = cursor.seek(*range.start())?;
        while let Some((block_num, first_val)) = entry {
            if block_num > *range.end() {
                break;
            }
            counts.hashed_accounts_written_total += 1;
            keys.insert(first_val.hashed_address);
            while let Some((_, val)) = cursor.next_dup()? {
                counts.hashed_accounts_written_total += 1;
                keys.insert(val.hashed_address);
            }
            cursor.delete_current_duplicates()?;
            entry = cursor.current()?;
        }
        Ok(keys)
    }

    /// Phase A: delete all hashed-storage changeset entries in `range`, returning the
    /// set of `(hashed_address, storage_key)` pairs that were affected.
    pub(super) fn prune_hashed_storage_changesets(
        &self,
        range: &std::ops::RangeInclusive<u64>,
        counts: &mut WriteCounts,
    ) -> OpProofsStorageResult<BTreeSet<(B256, B256)>> {
        let mut keys: BTreeSet<(B256, B256)> = BTreeSet::new();
        let mut cursor = self.tx.cursor_dup_write::<V2HashedStorageChangeSets>()?;
        let start = BlockNumberHashedAddress((*range.start(), B256::ZERO));
        let end = BlockNumberHashedAddress((*range.end(), B256::repeat_byte(0xff)));
        let mut entry = cursor.seek(start)?;
        while let Some((key, first_val)) = entry {
            if key > end {
                break;
            }
            counts.hashed_storages_written_total += 1;
            keys.insert((key.0.1, first_val.key));
            while let Some((k, val)) = cursor.next_dup()? {
                counts.hashed_storages_written_total += 1;
                keys.insert((k.0.1, val.key));
            }
            cursor.delete_current_duplicates()?;
            entry = cursor.current()?;
        }
        Ok(keys)
    }

    /// Phase B: remove `range` block numbers from account-trie history bitmaps for the
    /// given nibble paths.
    pub(super) fn prune_account_trie_history(
        &self,
        range: &std::ops::RangeInclusive<u64>,
        keys: &BTreeSet<StoredNibbles>,
    ) -> OpProofsStorageResult<()> {
        let mut cursor = self.tx.cursor_write::<V2AccountsTrieHistory>()?;
        for nibbles in keys {
            Self::prune_history_range_for_key(
                &mut cursor,
                range,
                AccountTrieShardedKey::new(nibbles.clone(), 0),
                |k| k.key == *nibbles,
            )?;
        }
        Ok(())
    }

    /// Phase B: remove `range` block numbers from storage-trie history bitmaps for the
    /// given `(hashed_address, nibbles)` pairs.
    pub(super) fn prune_storage_trie_history(
        &self,
        range: &std::ops::RangeInclusive<u64>,
        keys: &BTreeSet<(B256, StoredNibbles)>,
    ) -> OpProofsStorageResult<()> {
        let mut cursor = self.tx.cursor_write::<V2StoragesTrieHistory>()?;
        for (hashed_address, nibbles) in keys {
            Self::prune_history_range_for_key(
                &mut cursor,
                range,
                StorageTrieShardedKey::new(*hashed_address, nibbles.clone(), 0),
                |k| k.hashed_address == *hashed_address && k.key == *nibbles,
            )?;
        }
        Ok(())
    }

    /// Phase B: remove `range` block numbers from hashed-account history bitmaps for the
    /// given hashed addresses.
    pub(super) fn prune_hashed_account_history(
        &self,
        range: &std::ops::RangeInclusive<u64>,
        keys: &BTreeSet<B256>,
    ) -> OpProofsStorageResult<()> {
        let mut cursor = self.tx.cursor_write::<V2HashedAccountsHistory>()?;
        for addr in keys {
            Self::prune_history_range_for_key(
                &mut cursor,
                range,
                HashedAccountShardedKey::new(*addr, 0),
                |k| k.0.key == *addr,
            )?;
        }
        Ok(())
    }

    /// Phase B: remove `range` block numbers from hashed-storage history bitmaps for the
    /// given `(hashed_address, storage_key)` pairs.
    pub(super) fn prune_hashed_storage_history(
        &self,
        range: &std::ops::RangeInclusive<u64>,
        keys: &BTreeSet<(B256, B256)>,
    ) -> OpProofsStorageResult<()> {
        let mut cursor = self.tx.cursor_write::<V2HashedStoragesHistory>()?;
        for (hashed_address, storage_key) in keys {
            Self::prune_history_range_for_key(
                &mut cursor,
                range,
                HashedStorageShardedKey {
                    hashed_address: *hashed_address,
                    sharded_key: ShardedKey::new(*storage_key, 0),
                },
                |k| k.hashed_address == *hashed_address && k.sharded_key.key == *storage_key,
            )?;
        }
        Ok(())
    }
}
