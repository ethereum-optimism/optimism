//! [`OpProofsBackfillProvider`] implementation for [`MdbxProofsProviderV2`].

use super::{MdbxProofsProviderV2, NUM_OF_INDICES_IN_SHARD, write::HistoryCollector};
use crate::{
    BlockStateDiff, OpProofsStorageError, OpProofsStorageResult,
    api::{OpProofsBackfillProvider, WriteCounts},
    db::models::{
        AccountTrieShardedKey, BlockNumberHashedAddress, HashedAccountBeforeTx,
        HashedAccountShardedKey, HashedStorageShardedKey, StorageTrieShardedKey,
        TrieChangeSetsEntry, V2AccountTrieChangeSets, V2AccountsTrieHistory,
        V2HashedAccountChangeSets, V2HashedAccountsHistory, V2HashedStorageChangeSets,
        V2HashedStoragesHistory, V2StorageTrieChangeSets, V2StoragesTrieHistory,
    },
};
use alloy_eips::eip1898::BlockWithParent;
use alloy_primitives::{B256, BlockNumber};
use reth_db::{
    BlockNumberList,
    cursor::{DbCursorRO, DbCursorRW},
    models::sharded_key::ShardedKey,
    table::Table,
    transaction::{DbTx, DbTxMut},
};
use reth_primitives_traits::StorageEntry;
use reth_trie::{
    HashedPostStateSorted, StoredNibbles, StoredNibblesSubKey, updates::TrieUpdatesSorted,
};
use std::{collections::BTreeMap, fmt::Debug};
use tracing::debug;

/// Insert `block_number` at the front of the first history-bitmap shard for a logical key.
///
/// Backfill prepends blocks in descending order, so `block_number` is always strictly
/// less than every value already stored for this key. The existing
/// [`super::write::append_history_indices_batched`] function only touches the last/sentinel
/// shard; this function instead seeks the **first** shard and prepends there.
///
/// If the first shard would exceed [`NUM_OF_INDICES_IN_SHARD`] entries after the insert,
/// it is split: the new earlier portion gets a fresh shard key keyed by its maximum
/// block number, and the remainder stays under the original shard key.
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
            // Build the merged sequence: [block_number, ...existing...].
            // block_number < all existing values (prepend invariant), so the result is sorted.
            let mut all_values: Vec<u64> =
                std::iter::once(block_number).chain(existing.iter()).collect();

            if all_values.len() <= NUM_OF_INDICES_IN_SHARD {
                // Fits — update shard in-place (its max value, i.e. key, is unchanged).
                let new_list = BlockNumberList::new_pre_sorted(all_values);
                cursor.upsert(old_key, &new_list)?;
            } else {
                // Overflow — split into two shards:
                //   first_chunk: [block_number, ..., a_{N-1}]  →  new key = a_{N-1}
                //   rest:        [a_N, ..., a_K]               →  keep old_key (max unchanged)
                let rest: Vec<u64> = all_values.split_off(NUM_OF_INDICES_IN_SHARD);
                let first_chunk_max = *all_values.last().expect("non-empty");
                let new_first_key = make_shard_key(first_chunk_max);
                let first_list = BlockNumberList::new_pre_sorted(all_values);
                let rest_list = BlockNumberList::new_pre_sorted(rest);
                // Keep the existing shard key for the upper portion.
                cursor.upsert(old_key, &rest_list)?;
                // Insert the new lower shard.
                cursor.upsert(new_first_key, &first_list)?;
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

    fn commit(self) -> OpProofsStorageResult<()> {
        self.tx.commit()?;
        Ok(())
    }
}
