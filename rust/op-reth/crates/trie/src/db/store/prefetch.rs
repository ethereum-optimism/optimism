//! Page-cache warming for [`MdbxProofsStorage`].
//!
//! Persistence reads its pages one at a time on a single thread. The functions here read the same
//! pages beforehand, from several read-only transactions at once, and drop the results. The pages
//! are then in the OS page cache when the persistence transaction reaches them.

use super::MdbxProofsStorage;
use crate::{
    BlockStateDiff, OpProofsStorageResult,
    db::models::{
        AccountTrieShardedKey, BlockNumberHashedAddress, HashedAccountShardedKey,
        HashedStorageShardedKey, StorageTrieShardedKey, V2AccountTrieChangeSets, V2AccountsTrie,
        V2AccountsTrieHistory, V2HashedAccountChangeSets, V2HashedAccounts,
        V2HashedAccountsHistory, V2HashedStorageChangeSets, V2HashedStorages,
        V2HashedStoragesHistory, V2StorageTrieChangeSets, V2StoragesTrie, V2StoragesTrieHistory,
    },
};
use alloy_primitives::{B256, BlockNumber};
use reth_db::{
    Database, DatabaseError,
    cursor::{DbCursorRO, DbDupCursorRO},
    models::sharded_key::ShardedKey,
    table::Table,
    transaction::DbTx,
};
use reth_trie::{Nibbles, StoredNibbles, StoredNibblesSubKey};
use std::{ops::RangeInclusive, thread};

impl MdbxProofsStorage {
    /// Reads the pages persisting `diff` reads: for each key it writes, the current-state row
    /// and the last history shard, which the write appends to. Each of `threads` readers takes
    /// a contiguous share of each table's keys. Returns the number of keys.
    pub(super) fn warm_write(
        &self,
        diff: &BlockStateDiff,
        threads: usize,
    ) -> OpProofsStorageResult<usize> {
        let readers = threads.max(1);
        in_parallel(readers, |reader| self.warm_write_share(diff, reader, readers))
    }

    /// Reads `reader`'s share of the keys of `diff`.
    fn warm_write_share(
        &self,
        diff: &BlockStateDiff,
        reader: usize,
        readers: usize,
    ) -> OpProofsStorageResult<usize> {
        let tx = self.env.tx()?;
        let (trie, state) = (&diff.sorted_trie_updates, &diff.sorted_post_state);
        let mut keys = 0;

        let mut rows = tx.cursor_read::<V2AccountsTrie>()?;
        let mut history = tx.cursor_read::<V2AccountsTrieHistory>()?;
        for (path, _) in share(trie.account_nodes_ref().iter(), reader, readers) {
            let path = StoredNibbles(*path);
            rows.seek_exact(path.clone())?;
            history.seek_exact(AccountTrieShardedKey::new(path, u64::MAX))?;
            keys += 1;
        }

        let mut rows = tx.cursor_dup_read::<V2StoragesTrie>()?;
        let mut history = tx.cursor_read::<V2StoragesTrieHistory>()?;
        for (address, path) in share(storage_nodes(diff), reader, readers) {
            rows.seek_by_key_subkey(address, StoredNibblesSubKey(path))?;
            history.seek_exact(StorageTrieShardedKey::new(
                address,
                StoredNibbles(path),
                u64::MAX,
            ))?;
            keys += 1;
        }

        let mut rows = tx.cursor_read::<V2HashedAccounts>()?;
        let mut history = tx.cursor_read::<V2HashedAccountsHistory>()?;
        for (address, _) in share(state.accounts.iter(), reader, readers) {
            rows.seek_exact(*address)?;
            history.seek_exact(HashedAccountShardedKey::new(*address, u64::MAX))?;
            keys += 1;
        }

        let mut rows = tx.cursor_dup_read::<V2HashedStorages>()?;
        let mut history = tx.cursor_read::<V2HashedStoragesHistory>()?;
        for (address, slot) in share(storage_slots(diff), reader, readers) {
            rows.seek_by_key_subkey(address, slot)?;
            history.seek_exact(HashedStorageShardedKey {
                hashed_address: address,
                sharded_key: ShardedKey::new(slot, u64::MAX),
            })?;
            keys += 1;
        }

        Ok(keys)
    }

    /// Reads the pages pruning `blocks` reads: their changesets, and the first history shard of
    /// each key they changed, which the prune edits. Up to `threads` readers each take a
    /// contiguous share of `blocks`. Returns the number of changeset entries.
    pub(super) fn warm_prune(
        &self,
        blocks: RangeInclusive<BlockNumber>,
        threads: usize,
    ) -> OpProofsStorageResult<usize> {
        let (first, last) = blocks.into_inner();
        if first > last {
            return Ok(0);
        }
        let count = (last - first).saturating_add(1) as usize;
        let readers = threads.max(1).min(count);
        in_parallel(readers, |reader| {
            let (start, end) = (count * reader / readers, count * (reader + 1) / readers);
            self.warm_prune_blocks(first + start as u64..=first + (end - 1) as u64)
        })
    }

    /// Reads the changesets of `blocks` and, for each entry, the part of its key's history the
    /// prune reads.
    fn warm_prune_blocks(
        &self,
        blocks: RangeInclusive<BlockNumber>,
    ) -> OpProofsStorageResult<usize> {
        let tx = self.env.tx()?;
        // The storage changesets of every address in `blocks`.
        let storages = BlockNumberHashedAddress((*blocks.start(), B256::ZERO))..=
            BlockNumberHashedAddress((*blocks.end(), B256::repeat_byte(0xff)));
        let mut keys = 0;

        let mut history = tx.cursor_read::<V2AccountsTrieHistory>()?;
        for change in tx.cursor_dup_read::<V2AccountTrieChangeSets>()?.walk_range(blocks.clone())? {
            let path = StoredNibbles(change?.1.nibbles.0);
            read_first_shard(&mut history, AccountTrieShardedKey::new(path, 0))?;
            keys += 1;
        }

        let mut history = tx.cursor_read::<V2StoragesTrieHistory>()?;
        for change in
            tx.cursor_dup_read::<V2StorageTrieChangeSets>()?.walk_range(storages.clone())?
        {
            let (key, change) = change?;
            let path = StoredNibbles(change.nibbles.0);
            read_first_shard(&mut history, StorageTrieShardedKey::new(key.0.1, path, 0))?;
            keys += 1;
        }

        let mut history = tx.cursor_read::<V2HashedAccountsHistory>()?;
        for change in tx.cursor_dup_read::<V2HashedAccountChangeSets>()?.walk_range(blocks)? {
            let address = change?.1.hashed_address;
            read_first_shard(&mut history, HashedAccountShardedKey::new(address, 0))?;
            keys += 1;
        }

        let mut history = tx.cursor_read::<V2HashedStoragesHistory>()?;
        for change in tx.cursor_dup_read::<V2HashedStorageChangeSets>()?.walk_range(storages)? {
            let (key, change) = change?;
            let slot = HashedStorageShardedKey {
                hashed_address: key.0.1,
                sharded_key: ShardedKey::new(change.key, 0),
            };
            read_first_shard(&mut history, slot)?;
            keys += 1;
        }

        Ok(keys)
    }
}

/// The storage trie nodes `diff` writes, each with its hashed address.
fn storage_nodes(diff: &BlockStateDiff) -> impl Iterator<Item = (B256, Nibbles)> + Clone {
    diff.sorted_trie_updates.storage_tries_ref().iter().flat_map(|(address, nodes)| {
        nodes.storage_nodes_ref().iter().map(move |(path, _)| (*address, *path))
    })
}

/// The hashed storage slots `diff` writes, each with its hashed address.
fn storage_slots(diff: &BlockStateDiff) -> impl Iterator<Item = (B256, B256)> + Clone {
    diff.sorted_post_state.storages.iter().flat_map(|(address, storage)| {
        storage.storage_slots_ref().iter().map(move |(slot, _)| (*address, *slot))
    })
}

/// Reads what a prune reads of one key's history: its first shard and the entry after it.
fn read_first_shard<T: Table>(
    cursor: &mut impl DbCursorRO<T>,
    first_shard_key: T::Key,
) -> Result<(), DatabaseError> {
    if cursor.seek(first_shard_key)?.is_some() {
        cursor.next()?;
    }
    Ok(())
}

/// Returns `reader`'s share of `keys`: one of `readers` contiguous runs of near-equal length, so
/// that neighbouring keys, which often share a page, fall to the same reader.
fn share<I: Iterator + Clone>(
    keys: I,
    reader: usize,
    readers: usize,
) -> impl Iterator<Item = I::Item> {
    let count = keys.clone().count();
    let start = count * reader / readers;
    keys.skip(start).take(count * (reader + 1) / readers - start)
}

/// Runs `read` for every reader in `0..readers` at the same time, reader 0 on the calling
/// thread, and returns the sum of what they read. Panics if a reader panics or cannot be
/// spawned.
fn in_parallel(
    readers: usize,
    read: impl Fn(usize) -> OpProofsStorageResult<usize> + Sync,
) -> OpProofsStorageResult<usize> {
    let read = &read;
    thread::scope(|scope| {
        let spawned: Vec<_> = (1..readers)
            .map(|reader| {
                thread::Builder::new()
                    .name("proofs-prefetch".into())
                    .spawn_scoped(scope, move || read(reader))
                    .expect("failed to spawn proofs-prefetch reader")
            })
            .collect();
        let mut keys = read(0)?;
        for reader in spawned {
            keys += reader.join().expect("warming reader panicked")?;
        }
        Ok(keys)
    })
}

#[cfg(test)]
mod tests {
    use super::{
        super::{NUM_OF_INDICES_IN_SHARD, write::HistoryCollector},
        *,
    };
    use crate::{
        OpProofsStore,
        api::{OpProofsInitProvider, OpProofsProviderRw, WriteCounts},
    };
    use alloy_eips::{BlockNumHash, NumHash, eip1898::BlockWithParent};
    use alloy_primitives::U256;
    use reth_primitives_traits::Account;
    use reth_trie::{BranchNodeCompact, HashedPostState, HashedStorage, updates::TrieUpdates};
    use std::collections::BTreeSet;
    use tempfile::TempDir;

    fn block(number: u64) -> BlockWithParent {
        let hash = |number: u64| B256::from(U256::from(number));
        BlockWithParent::new(hash(number - 1), NumHash::new(number, hash(number)))
    }

    /// Creates a store holding `diffs` as blocks `1..` on top of an empty block 0.
    fn store_with_blocks(
        diffs: impl IntoIterator<Item = BlockStateDiff>,
    ) -> (TempDir, MdbxProofsStorage) {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).unwrap();
        let init = store.initialization_provider().unwrap();
        init.set_initial_state_anchor(BlockNumHash::new(0, block(1).parent)).unwrap();
        init.commit_initial_state().unwrap();
        OpProofsInitProvider::commit(init).unwrap();

        let blocks = (1..).zip(diffs).map(|(number, diff)| (block(number), diff)).collect();
        let provider = store.provider_rw().unwrap();
        provider.store_trie_updates_batch(blocks).unwrap();
        OpProofsProviderRw::commit(provider).unwrap();
        (dir, store)
    }

    /// A diff with two keys per table for each of `addresses`: one set and one removed.
    fn diff(addresses: &[B256]) -> BlockStateDiff {
        let mut trie = TrieUpdates::default();
        let mut state = HashedPostState::default();
        for (i, address) in addresses.iter().enumerate() {
            trie.account_nodes.insert(Nibbles::from_nibbles([i as u8, 1]), Default::default());
            trie.removed_nodes.insert(Nibbles::from_nibbles([i as u8, 2]));
            let nodes = trie.storage_tries.entry(*address).or_default();
            nodes.storage_nodes.insert(Nibbles::from_nibbles([3]), BranchNodeCompact::default());
            nodes.removed_nodes.insert(Nibbles::from_nibbles([4]));

            state.accounts.insert(*address, Some(Account::default()));
            state.accounts.insert(B256::with_last_byte(i as u8), None);
            let mut storage = HashedStorage::default();
            storage.storage.insert(B256::repeat_byte(5), U256::from(1));
            storage.storage.insert(B256::repeat_byte(6), U256::ZERO);
            state.storages.insert(*address, storage);
        }
        BlockStateDiff {
            sorted_trie_updates: trie.into_sorted(),
            sorted_post_state: state.into_sorted(),
        }
    }

    /// A diff that changes only the account at `address`.
    fn account_diff(address: B256, nonce: u64) -> BlockStateDiff {
        let mut state = HashedPostState::default();
        state.accounts.insert(address, Some(Account { nonce, ..Default::default() }));
        BlockStateDiff { sorted_post_state: state.into_sorted(), ..Default::default() }
    }

    // Warming reads as many keys as the write stores and as many changeset entries as the prune
    // removes, however many readers share the work.
    #[test]
    fn warming_reads_what_the_write_and_the_prune_touch() {
        let (a, b, c) = (B256::repeat_byte(0xa), B256::repeat_byte(0xb), B256::repeat_byte(0xc));
        let (_dir, store) = store_with_blocks([diff(&[a]), diff(&[b, a]), diff(&[c])]);
        let next = diff(&[c, a, b]);
        let total = |counts: WriteCounts| {
            (counts.account_trie_updates_written_total +
                counts.storage_trie_updates_written_total +
                counts.hashed_accounts_written_total +
                counts.hashed_storages_written_total) as usize
        };

        // One reader, a number that does not divide the keys evenly, and more readers than keys.
        let warmed_write = [1, 5, 64].map(|readers| store.warm_write(&next, readers).unwrap());
        let warmed_prune = [1, 2, 64].map(|readers| store.warm_prune(1..=2, readers).unwrap());

        let provider = store.provider_rw().unwrap();
        let mut history = HistoryCollector::default();
        let written = total(provider.store_block_updates(4, next.clone(), &mut history).unwrap());
        let pruned = total(provider.prune_earliest_state(block(2)).unwrap());
        assert_eq!((written, pruned), (24, 24));
        assert_eq!(warmed_write, [written; 3]);
        assert_eq!(warmed_prune, [pruned; 3]);

        // The storage keys warming reads are the keys the write records history for.
        let nodes: BTreeSet<_> =
            storage_nodes(&next).map(|(address, path)| (address, StoredNibbles(path))).collect();
        assert_eq!(nodes, history.storage_trie.into_keys().collect::<BTreeSet<_>>());
        let slots: BTreeSet<_> = storage_slots(&next).collect();
        assert_eq!(slots, history.hashed_storages.into_keys().collect::<BTreeSet<_>>());
    }

    // With a key's history in two shards, a write appends to the last one and a prune edits the
    // first one, so those are the shards warming seeks.
    #[test]
    fn warming_reads_the_history_shard_each_pass_edits() {
        let address = B256::repeat_byte(0xa);
        let blocks = NUM_OF_INDICES_IN_SHARD as u64 + 1;
        let (_dir, store) = store_with_blocks((1..=blocks).map(|n| account_diff(address, n)));

        assert_eq!(store.warm_write(&account_diff(address, 0), 4).unwrap(), 1);
        // Each of the eight blocks changed the account once.
        assert_eq!(store.warm_prune(1..=8, 4).unwrap(), 8);

        let tx = store.env.tx().unwrap();
        let mut history = tx.cursor_read::<V2HashedAccountsHistory>().unwrap();
        // Reading the first shard leaves the cursor on the entry after it: the last shard.
        read_first_shard(&mut history, HashedAccountShardedKey::new(address, 0)).unwrap();
        let (_, last) = history.current().unwrap().expect("last shard");
        let (_, first) = history.prev().unwrap().expect("first shard");
        assert_eq!((first.iter().next(), last.iter().next_back()), (Some(1), Some(blocks)));
        assert!(first.iter().next_back() < last.iter().next());
    }

    #[test]
    fn warming_tolerates_missing_rows_and_empty_ranges() {
        let (_dir, store) = store_with_blocks([]);

        assert_eq!(store.warm_write(&diff(&[B256::repeat_byte(0xa)]), 3).unwrap(), 8);
        assert_eq!(store.warm_write(&BlockStateDiff::default(), 3).unwrap(), 0);
        assert_eq!(store.warm_prune(5..=9, 3).unwrap(), 0);
        #[allow(clippy::reversed_empty_ranges)]
        let empty = 9..=5;
        assert_eq!(store.warm_prune(empty, 3).unwrap(), 0);
    }

    #[test]
    fn shares_cover_every_key_once() {
        // No keys, fewer keys than readers, an uneven split, and one reader.
        for (keys, readers) in [(0, 3), (2, 3), (10, 3), (10, 1)] {
            let shared: Vec<_> =
                (0..readers).flat_map(|reader| share(0..keys, reader, readers)).collect();
            assert_eq!(shared, (0..keys).collect::<Vec<_>>());
        }
    }

    #[test]
    fn in_parallel_runs_every_reader_and_reports_errors() {
        assert_eq!(in_parallel(4, Ok).unwrap(), 6);
        let failed = in_parallel(4, |reader| match reader {
            2 => Err(DatabaseError::Other("read failed".into()).into()),
            _ => Ok(1),
        });
        assert!(failed.is_err());
    }
}
