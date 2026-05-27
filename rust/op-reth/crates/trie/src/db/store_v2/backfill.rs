//! [`OpProofsBackfillProvider`] implementation for [`MdbxProofsProviderV2`].

use super::{MdbxProofsProviderV2, NUM_OF_INDICES_IN_SHARD, write::HistoryCollector};
use crate::{
    BlockStateDiff, OpProofsStorageError, OpProofsStorageResult,
    api::{OpProofsBackfillProvider, WriteCounts},
    db::{
        SnapshotMeta, SnapshotMetaKey, SnapshotStatus,
        models::{
            AccountTrieShardedKey, BlockNumberHashedAddress, HashedAccountBeforeTx,
            HashedAccountShardedKey, HashedStorageShardedKey, StorageTrieShardedKey,
            TrieChangeSetsEntry, V2AccountTrieChangeSets, V2AccountsTrieHistory,
            V2AccountsTrieSnapshot, V2HashedAccountChangeSets, V2HashedAccountsHistory,
            V2HashedStorageChangeSets, V2HashedStoragesHistory, V2SnapshotMeta,
            V2StorageTrieChangeSets, V2StoragesTrieHistory, V2StoragesTrieSnapshot,
        },
    },
};
use alloy_eips::{BlockNumHash, eip1898::BlockWithParent};
use alloy_primitives::{B256, BlockNumber};
use reth_db::{
    BlockNumberList,
    cursor::{DbCursorRO, DbCursorRW, DbDupCursorRO},
    models::sharded_key::ShardedKey,
    table::Table,
    transaction::{DbTx, DbTxMut},
};
use reth_primitives_traits::StorageEntry;
use reth_trie::{
    HashedPostStateSorted, StorageTrieEntry, StoredNibbles, StoredNibblesSubKey,
    updates::TrieUpdatesSorted,
};
use std::{collections::BTreeMap, fmt::Debug};
use tracing::debug;

/// Insert `block_number` at the front of the first history-bitmap shard for a logical key.
///
/// Backfill prepends blocks in descending order, so `block_number` is always strictly
/// less than every value already stored for this key. The existing
/// `append_history_indices_batched` (in `super::write`) only touches the last/sentinel
/// shard; this function instead seeks the **first** shard and prepends there.
///
/// When the first shard is already at [`NUM_OF_INDICES_IN_SHARD`] entries, a fresh
/// singleton shard is created at key `(_, block_number)`. Subsequent prepends fill that
/// singleton in place until it too reaches capacity, at which point the same branch
/// fires again. This amortises to ~1 upsert per prepend plus one shard-creation
/// every `NUM_OF_INDICES_IN_SHARD` prepends — replacing an earlier scheme that
/// rewrote the full shard on every overflow.
fn prepend_history_index_for_key<T>(
    cursor: &mut (impl DbCursorRO<T> + DbCursorRW<T>),
    block_number: BlockNumber,
    first_shard_key: T::Key,
    make_shard_key: impl Fn(BlockNumber) -> T::Key,
    sentinel_key: T::Key,
    same_logical_key: impl Fn(&T::Key) -> bool,
) -> OpProofsStorageResult<()>
where
    T: Table<Value = BlockNumberList>,
    T::Key: Clone,
{
    match cursor.seek(first_shard_key)? {
        Some((old_key, existing)) if same_logical_key(&old_key) => {
            // `block_number` is strictly less than every value in `existing` by
            // the prepend invariant (we walk blocks in descending order).
            let existing_len = existing.iter().count();
            if existing_len < NUM_OF_INDICES_IN_SHARD {
                // Fits — prepend in place. The shard's max (= its key) is
                // unchanged since `block_number` is smaller than all entries.
                let all_values: Vec<u64> =
                    std::iter::once(block_number).chain(existing.iter()).collect();
                let new_list = BlockNumberList::new_pre_sorted(all_values);
                cursor.upsert(old_key, &new_list)?;
            } else {
                // The current front shard is full. Start a fresh singleton
                // shard at `block_number` and leave the full one alone.
                // Subsequent prepends fill the new singleton in place until
                // it too hits the cap, at which point this branch fires
                // again. Amortises to ~1 upsert per prepend plus one
                // shard-creation every `NUM_OF_INDICES_IN_SHARD` prepends.
                debug_assert!(
                    existing_len <= NUM_OF_INDICES_IN_SHARD,
                    "history shard exceeded NUM_OF_INDICES_IN_SHARD: {existing_len}"
                );
                let new_list = BlockNumberList::new_pre_sorted([block_number]);
                cursor.upsert(make_shard_key(block_number), &new_list)?;
            }
        }
        _ => {
            // No existing shard for this key — create the sentinel shard.
            let new_list = BlockNumberList::new_pre_sorted([block_number]);
            cursor.upsert(sentinel_key, &new_list)?;
        }
    }
    Ok(())
}

impl<TX: DbTxMut + DbTx + Send + Sync + Debug + 'static> MdbxProofsProviderV2<TX> {
    /// Upsert the singleton row in [`V2SnapshotMeta`].
    ///
    /// Shared helper used by both
    /// [`OpProofsSnapshotInitProvider`](crate::api::OpProofsSnapshotInitProvider)
    /// (init lifecycle transitions) and
    /// [`OpProofsBackfillProvider::update_snapshot`] (per-iteration anchor
    /// updates).
    pub(super) fn write_snapshot_meta(&self, meta: SnapshotMeta) -> OpProofsStorageResult<()> {
        let mut cur = self.tx.cursor_write::<V2SnapshotMeta>()?;
        cur.upsert(SnapshotMetaKey::Singleton, &meta)?;
        Ok(())
    }

    /// Returns `true` if any changeset entry already exists for `block_number`.
    ///
    /// Uses `V2HashedAccountChangeSets` as the sentinel table: nearly every block
    /// touches at least one account. For the rare empty block the write loop is a
    /// no-op regardless, so a false-negative here is harmless.
    fn changeset_exists_for_block(&self, block_number: BlockNumber) -> OpProofsStorageResult<bool> {
        let mut cs = self.tx.cursor_read::<V2HashedAccountChangeSets>()?;
        Ok(cs.seek(block_number)?.is_some_and(|(bn, _)| bn == block_number))
    }

    /// Write changeset entries for `block_number` directly from `diff` (already before-values)
    /// without reading or modifying the current-state tables.
    fn prepend_block_changesets(
        &self,
        block_number: BlockNumber,
        diff: BlockStateDiff,
        collector: &mut HistoryCollector,
    ) -> OpProofsStorageResult<WriteCounts> {
        let BlockStateDiff { sorted_trie_updates, sorted_post_state } = diff;
        Ok(WriteCounts {
            account_trie_updates_written_total: self.write_account_trie_cs(
                block_number,
                &sorted_trie_updates,
                collector,
            )?,
            storage_trie_updates_written_total: self.write_storage_trie_cs(
                block_number,
                &sorted_trie_updates,
                collector,
            )?,
            hashed_accounts_written_total: self.write_hashed_accounts_cs(
                block_number,
                &sorted_post_state,
                collector,
            )?,
            hashed_storages_written_total: self.write_hashed_storages_cs(
                block_number,
                &sorted_post_state,
                collector,
            )?,
        })
    }

    fn write_account_trie_cs(
        &self,
        block_number: BlockNumber,
        updates: &TrieUpdatesSorted,
        collector: &mut HistoryCollector,
    ) -> OpProofsStorageResult<u64> {
        let mut cs = self.tx.cursor_dup_write::<V2AccountTrieChangeSets>()?;
        let mut count = 0u64;
        for (nibbles, maybe_node) in updates.account_nodes_ref() {
            let stored = StoredNibbles(*nibbles);
            cs.upsert(
                block_number,
                &TrieChangeSetsEntry {
                    nibbles: StoredNibblesSubKey(*nibbles),
                    node: maybe_node.clone(),
                },
            )?;
            collector.account_trie.entry(stored).or_default().push(block_number);
            count += 1;
        }
        Ok(count)
    }

    fn write_storage_trie_cs(
        &self,
        block_number: BlockNumber,
        updates: &TrieUpdatesSorted,
        collector: &mut HistoryCollector,
    ) -> OpProofsStorageResult<u64> {
        let mut cs = self.tx.cursor_dup_write::<V2StorageTrieChangeSets>()?;
        let mut count = 0u64;
        for (hashed_address, nodes) in updates.storage_tries_ref() {
            let cs_key = BlockNumberHashedAddress((block_number, *hashed_address));
            for (nibbles, maybe_node) in nodes.storage_nodes_ref() {
                cs.upsert(
                    cs_key,
                    &TrieChangeSetsEntry {
                        nibbles: StoredNibblesSubKey(*nibbles),
                        node: maybe_node.clone(),
                    },
                )?;
                collector
                    .storage_trie
                    .entry((*hashed_address, StoredNibbles(*nibbles)))
                    .or_default()
                    .push(block_number);
                count += 1;
            }
        }
        Ok(count)
    }

    fn write_hashed_accounts_cs(
        &self,
        block_number: BlockNumber,
        post_state: &HashedPostStateSorted,
        collector: &mut HistoryCollector,
    ) -> OpProofsStorageResult<u64> {
        let mut cs = self.tx.cursor_dup_write::<V2HashedAccountChangeSets>()?;
        let mut count = 0u64;
        for &(hashed_address, maybe_account) in &post_state.accounts {
            cs.upsert(block_number, &HashedAccountBeforeTx::new(hashed_address, maybe_account))?;
            collector.hashed_accounts.entry(hashed_address).or_default().push(block_number);
            count += 1;
        }
        Ok(count)
    }

    fn write_hashed_storages_cs(
        &self,
        block_number: BlockNumber,
        post_state: &HashedPostStateSorted,
        collector: &mut HistoryCollector,
    ) -> OpProofsStorageResult<u64> {
        let mut cs = self.tx.cursor_dup_write::<V2HashedStorageChangeSets>()?;
        let mut count = 0u64;
        for (hashed_address, storage) in &post_state.storages {
            let cs_key = BlockNumberHashedAddress((block_number, *hashed_address));
            for &(slot, value) in &storage.storage_slots {
                cs.upsert(cs_key, &StorageEntry { key: slot, value })?;
                collector
                    .hashed_storages
                    .entry((*hashed_address, slot))
                    .or_default()
                    .push(block_number);
                count += 1;
            }
        }
        Ok(count)
    }

    /// Flush history-bitmap entries collected during a prepend operation.
    ///
    /// Unlike [`Self::flush_collected_history`] (which appends to the sentinel/last shard),
    /// this inserts into the **first** shard because the new block number is smaller than
    /// all existing entries.
    fn prepend_collected_history(&self, collector: HistoryCollector) -> OpProofsStorageResult<()> {
        self.prepend_account_trie_history(collector.account_trie)?;
        self.prepend_storage_trie_history(collector.storage_trie)?;
        self.prepend_hashed_account_history(collector.hashed_accounts)?;
        self.prepend_hashed_storage_history(collector.hashed_storages)?;
        Ok(())
    }

    fn prepend_account_trie_history(
        &self,
        entries: BTreeMap<StoredNibbles, Vec<BlockNumber>>,
    ) -> OpProofsStorageResult<()> {
        let mut cursor = self.tx.cursor_write::<V2AccountsTrieHistory>()?;
        for (nibbles, blocks) in entries {
            for block_number in blocks {
                prepend_history_index_for_key(
                    &mut cursor,
                    block_number,
                    AccountTrieShardedKey::new(nibbles.clone(), 0),
                    |h| AccountTrieShardedKey::new(nibbles.clone(), h),
                    AccountTrieShardedKey::new(nibbles.clone(), u64::MAX),
                    |k| k.key == nibbles,
                )?;
            }
        }
        Ok(())
    }

    fn prepend_storage_trie_history(
        &self,
        entries: BTreeMap<(B256, StoredNibbles), Vec<BlockNumber>>,
    ) -> OpProofsStorageResult<()> {
        let mut cursor = self.tx.cursor_write::<V2StoragesTrieHistory>()?;
        for ((addr, nibbles), blocks) in entries {
            for block_number in blocks {
                prepend_history_index_for_key(
                    &mut cursor,
                    block_number,
                    StorageTrieShardedKey::new(addr, nibbles.clone(), 0),
                    |h| StorageTrieShardedKey::new(addr, nibbles.clone(), h),
                    StorageTrieShardedKey::new(addr, nibbles.clone(), u64::MAX),
                    |k| k.hashed_address == addr && k.key == nibbles,
                )?;
            }
        }
        Ok(())
    }

    fn prepend_hashed_account_history(
        &self,
        entries: BTreeMap<B256, Vec<BlockNumber>>,
    ) -> OpProofsStorageResult<()> {
        let mut cursor = self.tx.cursor_write::<V2HashedAccountsHistory>()?;
        for (addr, blocks) in entries {
            for block_number in blocks {
                prepend_history_index_for_key(
                    &mut cursor,
                    block_number,
                    HashedAccountShardedKey::new(addr, 0),
                    |h| HashedAccountShardedKey::new(addr, h),
                    HashedAccountShardedKey::new(addr, u64::MAX),
                    |k| k.0.key == addr,
                )?;
            }
        }
        Ok(())
    }

    fn prepend_hashed_storage_history(
        &self,
        entries: BTreeMap<(B256, B256), Vec<BlockNumber>>,
    ) -> OpProofsStorageResult<()> {
        let mut cursor = self.tx.cursor_write::<V2HashedStoragesHistory>()?;
        for ((addr, slot), blocks) in entries {
            for block_number in blocks {
                prepend_history_index_for_key(
                    &mut cursor,
                    block_number,
                    HashedStorageShardedKey {
                        hashed_address: addr,
                        sharded_key: ShardedKey::new(slot, 0),
                    },
                    |h| HashedStorageShardedKey {
                        hashed_address: addr,
                        sharded_key: ShardedKey::new(slot, h),
                    },
                    HashedStorageShardedKey {
                        hashed_address: addr,
                        sharded_key: ShardedKey::new(slot, u64::MAX),
                    },
                    |k| k.hashed_address == addr && k.sharded_key.key == slot,
                )?;
            }
        }
        Ok(())
    }
}

impl<TX: DbTxMut + DbTx + Send + Sync + Debug + 'static> OpProofsBackfillProvider
    for MdbxProofsProviderV2<TX>
{
    fn prepend_block(
        &self,
        block_ref: BlockWithParent,
        diff: BlockStateDiff,
    ) -> OpProofsStorageResult<WriteCounts> {
        let block_number = block_ref.block.number;
        let proof_window = self.get_proof_window_inner()?;
        if block_ref.block.hash != proof_window.earliest.hash {
            return Err(OpProofsStorageError::PrependOutOfOrder {
                block_number,
                block_hash: block_ref.block.hash,
                earliest_block_number: proof_window.earliest.number,
                earliest_block_hash: proof_window.earliest.hash,
            });
        }

        if self.changeset_exists_for_block(block_number)? {
            debug!(target: "op-reth::trie::backfill", block_number, "changeset already exists, skipping prepend");
            return Ok(WriteCounts::default());
        }

        let mut collector = HistoryCollector::default();
        let counts = self.prepend_block_changesets(block_number, diff, &mut collector)?;
        self.prepend_collected_history(collector)?;
        self.set_earliest_block_number_inner(block_number - 1, block_ref.parent)?;
        Ok(counts)
    }

    fn clear_snapshot(&self) -> OpProofsStorageResult<()> {
        self.tx.clear::<V2AccountsTrieSnapshot>()?;
        self.tx.clear::<V2StoragesTrieSnapshot>()?;
        self.tx.clear::<V2SnapshotMeta>()?;
        Ok(())
    }

    fn update_snapshot(
        &self,
        new_anchor: BlockNumHash,
        trie_updates: &TrieUpdatesSorted,
    ) -> OpProofsStorageResult<u64> {
        // Refuse to mutate a Building snapshot: its rows are still being
        // populated, so applying a diff against it would corrupt the result.
        let SnapshotMeta { status, .. } = self.read_snapshot_meta()?;
        if status != SnapshotStatus::Ready {
            return Err(OpProofsStorageError::SnapshotUpdateNotReady { status });
        }

        let mut count = 0u64;

        // Account trie diff.
        let mut acc = self.tx.cursor_write::<V2AccountsTrieSnapshot>()?;
        for (nibbles, maybe_node) in trie_updates.account_nodes_ref() {
            let key = StoredNibbles(*nibbles);
            match maybe_node {
                Some(node) => acc.upsert(key, node)?,
                None => {
                    if acc.seek_exact(key)?.is_some() {
                        acc.delete_current()?;
                    }
                }
            }
            count += 1;
        }

        // Storage trie diff.
        let mut stor = self.tx.cursor_dup_write::<V2StoragesTrieSnapshot>()?;
        for (hashed_address, nodes) in trie_updates.storage_tries_ref() {
            for (nibbles, maybe_node) in nodes.storage_nodes_ref() {
                let subkey = StoredNibblesSubKey(*nibbles);
                let existing = stor
                    .seek_by_key_subkey(*hashed_address, subkey.clone())?
                    .filter(|e| e.nibbles == subkey)
                    .is_some();
                if existing {
                    stor.delete_current()?;
                }
                if let Some(node) = maybe_node {
                    stor.upsert(
                        *hashed_address,
                        &StorageTrieEntry { nibbles: subkey, node: node.clone() },
                    )?;
                }
                count += 1;
            }
        }

        // Advance the anchor atomically with the diff (status stays Ready).
        self.write_snapshot_meta(SnapshotMeta::new(new_anchor, SnapshotStatus::Ready))?;

        Ok(count)
    }

    fn commit(self) -> OpProofsStorageResult<()> {
        self.tx.commit()?;
        Ok(())
    }
}
