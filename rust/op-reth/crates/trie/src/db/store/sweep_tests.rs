//! Tests for the sweep-based history prune, [`MdbxProofsStorage::with_history_sweep`].

use super::*;
use crate::{
    api::{OpProofsBackfillProvider, OpProofsInitProvider, OpProofsProviderRO, OpProofsProviderRw},
    db::models::{
        HashedAccountShardedKey, V2AccountsTrieHistory, V2HashedAccountsHistory,
        V2HashedStoragesHistory, V2StoragesTrieHistory,
    },
};
use alloy_eips::{BlockNumHash, NumHash, eip1898::BlockWithParent};
use alloy_primitives::{B256, U256};
use reth_db::{
    BlockNumberList,
    cursor::{DbCursorRO, DbCursorRW},
    table::Table,
    transaction::DbTxMut,
};
use reth_primitives_traits::Account;
use reth_trie::{
    BranchNodeCompact, HashedPostState, Nibbles, hashed_cursor::HashedCursor,
    trie_cursor::TrieCursor, updates::TrieUpdates,
};
use std::fmt::Debug;
use tempfile::TempDir;

/// An account and a contract that change in every block.
const HOT: B256 = B256::repeat_byte(0xa1);
/// An account and a contract that change in every third block.
const WARM: B256 = B256::repeat_byte(0xb1);
/// An account that exists for three blocks out of every seven.
const SHORT_LIVED: B256 = B256::repeat_byte(0xd1);

fn hash(number: u64) -> B256 {
    B256::from(U256::from(number))
}

fn block(number: u64) -> BlockWithParent {
    BlockWithParent::new(hash(number - 1), NumHash::new(number, hash(number)))
}

fn account(nonce: u64) -> Option<Account> {
    Some(Account { nonce, ..Default::default() })
}

/// A store with an empty state at block 0, which prunes history by sweep if `sweep` is set.
fn store(sweep: bool) -> (TempDir, MdbxProofsStorage) {
    let dir = TempDir::new().unwrap();
    let store = MdbxProofsStorage::new(dir.path()).unwrap().with_history_sweep(sweep);
    let init = store.initialization_provider().unwrap();
    init.set_initial_state_anchor(BlockNumHash::new(0, hash(0))).unwrap();
    init.commit_initial_state().unwrap();
    OpProofsInitProvider::commit(init).unwrap();
    (dir, store)
}

/// A diff that sets the nonce of each of `accounts` to `nonce`.
fn nonce_diff(accounts: &[B256], nonce: u64) -> BlockStateDiff {
    let mut state = HashedPostState::default();
    for address in accounts {
        state.accounts.insert(*address, account(nonce));
    }
    BlockStateDiff { sorted_post_state: state.into_sorted(), ..Default::default() }
}

/// The changes of block `number`, in all four data types: keys that change in every block, keys
/// that change every few blocks, keys that only this block changes, and keys that are removed
/// again. Every value written is the block number.
fn diff(number: u64) -> BlockStateDiff {
    let value = U256::from(number);
    let node = BranchNodeCompact::new(0b1, 0, 0, vec![], Some(hash(number)));
    let once = B256::from(U256::from(0xc000 + number));
    let reused = (number % 16) as u8;
    let mut trie = TrieUpdates::default();
    let mut state = HashedPostState::default();

    state.accounts.insert(HOT, account(number));
    state.accounts.insert(once, account(number));
    if number % 7 == 1 {
        state.accounts.insert(SHORT_LIVED, account(number));
    } else if number % 7 == 4 {
        state.accounts.insert(SHORT_LIVED, None);
    }

    let slots = &mut state.storages.entry(HOT).or_default().storage;
    slots.insert(B256::repeat_byte(0x11), value);
    slots.insert(once, value);
    if number.is_multiple_of(5) {
        slots.insert(B256::repeat_byte(0x22), U256::ZERO);
    } else if number.is_multiple_of(2) {
        slots.insert(B256::repeat_byte(0x22), value);
    }

    trie.account_nodes.insert(Nibbles::from_nibbles([1]), node.clone());
    trie.account_nodes.insert(Nibbles::from_nibbles([3, reused]), node.clone());
    if number.is_multiple_of(6) {
        trie.account_nodes.insert(Nibbles::from_nibbles([4]), node.clone());
    } else if number % 6 == 3 {
        trie.removed_nodes.insert(Nibbles::from_nibbles([4]));
    }

    let nodes = trie.storage_tries.entry(HOT).or_default();
    nodes.storage_nodes.insert(Nibbles::from_nibbles([5]), node.clone());
    nodes.storage_nodes.insert(Nibbles::from_nibbles([6, reused]), node.clone());
    if number % 6 == 1 {
        nodes.storage_nodes.insert(Nibbles::from_nibbles([7]), node.clone());
    } else if number % 6 == 4 {
        nodes.removed_nodes.insert(Nibbles::from_nibbles([7]));
    }

    if number.is_multiple_of(3) {
        state.accounts.insert(WARM, account(number));
        state.storages.entry(WARM).or_default().storage.insert(B256::repeat_byte(0x33), value);
        let nodes = trie.storage_tries.entry(WARM).or_default();
        nodes.storage_nodes.insert(Nibbles::from_nibbles([8]), node);
    }

    BlockStateDiff {
        sorted_trie_updates: trie.into_sorted(),
        sorted_post_state: state.into_sorted(),
    }
}

/// Stores block `number` with `diff` and then prunes, in the same transaction as the node does,
/// so that at most `window` blocks stay in the proof window.
fn advance(store: &MdbxProofsStorage, number: u64, diff: BlockStateDiff, window: u64) {
    let provider = store.provider_rw().unwrap();
    provider.store_trie_updates_batch(vec![(block(number), diff)]).unwrap();
    let earliest = number.saturating_sub(window);
    if earliest > provider.get_earliest_block().unwrap().number {
        provider.prune_earliest_state(block(earliest)).unwrap();
    }
    OpProofsProviderRw::commit(provider).unwrap();
}

fn prune_to(store: &MdbxProofsStorage, number: u64) {
    let provider = store.provider_rw().unwrap();
    provider.prune_earliest_state(block(number)).unwrap();
    OpProofsProviderRw::commit(provider).unwrap();
}

/// The blocks the store serves.
fn proof_window(store: &MdbxProofsStorage) -> RangeInclusive<u64> {
    let window = store.provider_ro().unwrap().get_proof_window().unwrap();
    window.earliest.number..=window.latest.number
}

fn walk_hashed<C: HashedCursor>(mut cursor: C) -> Vec<(B256, C::Value)> {
    let mut rows = Vec::new();
    let mut entry = cursor.seek(B256::ZERO).unwrap();
    while let Some(row) = entry {
        rows.push(row);
        entry = cursor.next().unwrap();
    }
    rows
}

fn walk_trie(mut cursor: impl TrieCursor) -> Vec<(Nibbles, BranchNodeCompact)> {
    let mut rows = Vec::new();
    let mut entry = cursor.seek(Nibbles::default()).unwrap();
    while let Some(row) = entry {
        rows.push(row);
        entry = cursor.next().unwrap();
    }
    rows
}

/// Everything the store serves at `block`: the accounts, the account trie, and the storage and
/// storage trie of the two contracts.
fn state_at(store: &MdbxProofsStorage, block: u64) -> impl PartialEq + Debug {
    let provider = store.provider_ro().unwrap();
    (
        walk_hashed(provider.account_hashed_cursor(block).unwrap()),
        walk_trie(provider.account_trie_cursor(block).unwrap()),
        [HOT, WARM].map(|address| {
            (
                walk_hashed(provider.storage_hashed_cursor(address, block).unwrap()),
                walk_trie(provider.storage_trie_cursor(address, block).unwrap()),
            )
        }),
    )
}

/// Asserts that every block in the proof window reads the same from both stores.
fn assert_same_history(sweep: &MdbxProofsStorage, per_key: &MdbxProofsStorage) {
    let window = proof_window(per_key);
    assert_eq!(proof_window(sweep), window);
    for at in window.clone() {
        assert!(
            state_at(sweep, at) == state_at(per_key, at),
            "block {at} reads differently with the proof window at {window:?}"
        );
    }
}

/// The nonce the store serves for `address` at `block`.
fn nonce_at(store: &MdbxProofsStorage, address: B256, block: u64) -> Option<u64> {
    let provider = store.provider_ro().unwrap();
    let mut cursor = provider.account_hashed_cursor(block).unwrap();
    cursor.seek(address).unwrap().filter(|(found, _)| *found == address).map(|(_, a)| a.nonce)
}

/// The rows of history table `T`: each key with the block numbers it holds.
fn history<T: Table<Value = BlockNumberList>>(
    store: &MdbxProofsStorage,
) -> Vec<(T::Key, Vec<u64>)> {
    let tx = store.env.tx().unwrap();
    let mut cursor = tx.cursor_read::<T>().unwrap();
    let mut rows = Vec::new();
    let mut entry = cursor.first().unwrap();
    while let Some((key, list)) = entry {
        rows.push((key, list.iter().collect()));
        entry = cursor.next().unwrap();
    }
    rows
}

/// The oldest block number left in history table `T`.
fn oldest<T: Table<Value = BlockNumberList>>(store: &MdbxProofsStorage) -> Option<u64> {
    history::<T>(store).into_iter().filter_map(|(_, blocks)| blocks.first().copied()).min()
}

/// The account history rows, as the address, the shard's key and the block numbers.
fn account_history(store: &MdbxProofsStorage) -> Vec<(B256, u64, Vec<u64>)> {
    history::<V2HashedAccountsHistory>(store)
        .into_iter()
        .map(|(key, blocks)| (key.0.key, key.0.highest_block_number, blocks))
        .collect()
}

/// Writes account history rows: for each address, a shard under `key` holding `blocks`.
fn plant_account_history(store: &MdbxProofsStorage, rows: &[(B256, u64, &[u64])]) {
    let tx = store.env.tx_mut().unwrap();
    {
        let mut cursor = tx.cursor_write::<V2HashedAccountsHistory>().unwrap();
        for (address, key, blocks) in rows {
            let list = BlockNumberList::new_pre_sorted(blocks.iter().copied());
            cursor.upsert(HashedAccountShardedKey::new(*address, *key), &list).unwrap();
        }
    }
    tx.commit().unwrap();
}

/// Sweeps `rows` rows of the account history, removing the block numbers up to `earliest`.
fn sweep_accounts(store: &MdbxProofsStorage, earliest: u64, rows: usize) -> usize {
    let provider = store.provider_rw().unwrap();
    let swept = provider.sweep_history::<V2HashedAccountsHistory>(earliest, rows).unwrap();
    OpProofsProviderRw::commit(provider).unwrap();
    swept
}

// A sweep leaves block numbers behind that the per-key prune removes. Readers must not notice:
// every block in the proof window reads the same from both stores, after every prune.
#[test]
fn sweep_prune_serves_the_same_history_as_per_key_prune() {
    const WINDOW: u64 = 8;
    let (_dir, per_key) = store(false);
    let (_dir, sweep) = store(true);

    for number in 1..=64 {
        advance(&per_key, number, diff(number), WINDOW);
        advance(&sweep, number, diff(number), WINDOW);
        assert_same_history(&sweep, &per_key);
    }

    // Both stores could be wrong in the same way, so check against what was written too: every
    // value is the number of the block that wrote it.
    assert_eq!(proof_window(&sweep), 56..=64);
    assert_eq!(nonce_at(&sweep, HOT, 56), Some(56));
    assert_eq!(nonce_at(&sweep, WARM, 62), Some(60));
    assert_eq!(nonce_at(&sweep, SHORT_LIVED, 58), Some(57));
    assert_eq!(nonce_at(&sweep, SHORT_LIVED, 60), None);
}

// An unwind removes the newest block numbers of a key and keeps the rest, which under a sweep
// includes numbers below the proof window.
#[test]
fn unwind_after_sweep_prune_serves_the_same_history_as_per_key_prune() {
    const WINDOW: u64 = 8;
    let (_dir, per_key) = store(false);
    let (_dir, sweep) = store(true);
    for number in 1..=40 {
        advance(&per_key, number, diff(number), WINDOW);
        advance(&sweep, number, diff(number), WINDOW);
    }

    // Drop blocks 36 to 40, then go on with different blocks in their place.
    for store in [&per_key, &sweep] {
        let provider = store.provider_rw().unwrap();
        provider.unwind_history(block(36)).unwrap();
        OpProofsProviderRw::commit(provider).unwrap();
    }
    assert_same_history(&sweep, &per_key);
    assert_eq!(proof_window(&sweep), 32..=35);
    assert_eq!(nonce_at(&sweep, HOT, 35), Some(35));

    for number in 36..=48 {
        advance(&per_key, number, diff(number + 1), WINDOW);
        advance(&sweep, number, diff(number + 1), WINDOW);
        assert_same_history(&sweep, &per_key);
    }
    assert_eq!(proof_window(&sweep), 40..=48);
    assert_eq!(nonce_at(&sweep, HOT, 40), Some(41));
}

// The sweep has to get around the whole table about once per window, or block numbers below the
// window pile up in the rows it does not reach.
#[test]
fn sweep_prune_drops_stale_block_numbers_within_two_windows() {
    const WINDOW: u64 = 8;
    let (_dir, store) = store(true);
    for number in 1..=64 {
        advance(&store, number, diff(number), WINDOW);
    }

    // The window starts at block 56, so two windows before it is block 40.
    assert!(oldest::<V2HashedAccountsHistory>(&store) > Some(40));
    assert!(oldest::<V2HashedStoragesHistory>(&store) > Some(40));
    assert!(oldest::<V2AccountsTrieHistory>(&store) > Some(40));
    assert!(oldest::<V2StoragesTrieHistory>(&store) > Some(40));
}

// The point of the sweep: one prune touches a share of the history rows in proportion to the
// blocks it prunes, not every row those blocks changed.
#[test]
fn one_prune_sweeps_a_share_of_the_history_rows() {
    let (_dir, store) = store(true);
    let changed_in_block_1: Vec<B256> = (1..=10).map(B256::with_last_byte).collect();
    let provider = store.provider_rw().unwrap();
    let blocks = (1..=20)
        .map(|number| match number {
            1 => (block(1), nonce_diff(&changed_in_block_1, 1)),
            _ => (block(number), nonce_diff(&[B256::repeat_byte(0xff)], number)),
        })
        .collect();
    provider.store_trie_updates_batch(blocks).unwrap();
    OpProofsProviderRw::commit(provider).unwrap();

    prune_to(&store, 1);

    // 11 rows and 19 blocks left in the window: pruning one block sweeps one row, the first.
    let holding_block_1: Vec<B256> = account_history(&store)
        .into_iter()
        .filter(|(_, _, blocks)| blocks.contains(&1))
        .map(|(address, ..)| address)
        .collect();
    assert_eq!(holding_block_1, changed_in_block_1[1..]);
}

#[test]
fn sweep_resumes_after_the_last_row_it_visited_and_wraps_around() {
    let (_dir, store) = store(true);
    let [a, b, c] = [1, 2, 3].map(B256::with_last_byte);
    let last = u64::MAX;
    plant_account_history(&store, &[(a, last, &[1, 9]), (b, last, &[1, 9]), (c, last, &[1, 9])]);

    assert_eq!(sweep_accounts(&store, 5, 2), 2);
    assert_eq!(
        account_history(&store),
        [(a, u64::MAX, vec![9]), (b, u64::MAX, vec![9]), (c, u64::MAX, vec![1, 9])]
    );

    // Goes on at `c`, the last row of the table, and from there to `a`, the first. Neither holds
    // a block number above 9, so both rows go.
    assert_eq!(sweep_accounts(&store, 9, 2), 2);
    assert_eq!(account_history(&store), [(b, u64::MAX, vec![9])]);
}

#[test]
fn sweep_ends_when_the_table_runs_out_of_rows() {
    let (_dir, store) = store(true);
    assert_eq!(sweep_accounts(&store, 5, 3), 0);

    let [a, b] = [1, 2].map(B256::with_last_byte);
    plant_account_history(&store, &[(a, u64::MAX, &[1]), (b, u64::MAX, &[2])]);
    assert_eq!(sweep_accounts(&store, 5, 3), 2);
    assert!(account_history(&store).is_empty());
}

// The row a sweep stopped at can be gone by the next one: an unwind or a per-key prune deleted
// it. The sweep then goes on at the row that follows.
#[test]
fn sweep_goes_on_at_the_next_row_when_the_saved_one_is_gone() {
    let (_dir, store) = store(true);
    let [a, b, c] = [1, 2, 3].map(B256::with_last_byte);
    let last = u64::MAX;
    plant_account_history(&store, &[(a, last, &[1, 9]), (b, last, &[1, 9]), (c, last, &[1, 9])]);
    assert_eq!(sweep_accounts(&store, 5, 1), 1);

    let tx = store.env.tx_mut().unwrap();
    assert!(
        tx.delete::<V2HashedAccountsHistory>(HashedAccountShardedKey::new(b, last), None).unwrap()
    );
    tx.commit().unwrap();

    assert_eq!(sweep_accounts(&store, 5, 1), 1);
    assert_eq!(account_history(&store), [(a, last, vec![9]), (c, last, vec![9])]);
}

// A prune up to the latest block leaves no block in the window, and nothing in the history.
#[test]
fn pruning_up_to_the_latest_block_sweeps_every_row() {
    let (_dir, store) = store(true);
    let [a, b, c] = [1, 2, 3].map(B256::with_last_byte);
    let provider = store.provider_rw().unwrap();
    let blocks = (1..=5).map(|number| (block(number), nonce_diff(&[a, b, c], number))).collect();
    provider.store_trie_updates_batch(blocks).unwrap();
    OpProofsProviderRw::commit(provider).unwrap();

    prune_to(&store, 5);

    assert!(account_history(&store).is_empty());
    assert_eq!([a, b, c].map(|address| nonce_at(&store, address, 5)), [Some(5); 3]);
}

// One sweep can clean a key's first shard and a later one delete its last shard, while the first
// still holds a number: the key is then left with a first shard only. Its numbers are all below
// the window, so readers pass over it, and the next change to the key starts a new last shard.
#[test]
fn a_key_reads_right_when_two_sweeps_split_its_shards() {
    let (_dir, store) = store(true);
    let other = B256::repeat_byte(0xff);
    let provider = store.provider_rw().unwrap();
    let blocks = (1..=30)
        .map(|number| match number {
            1..=6 => (block(number), nonce_diff(&[HOT], number)),
            _ => (block(number), nonce_diff(&[other], number)),
        })
        .collect();
    provider.store_trie_updates_batch(blocks).unwrap();
    OpProofsProviderRw::commit(provider).unwrap();
    // Split the six block numbers of `HOT` over two shards, as 2,000 of them would be.
    plant_account_history(&store, &[(HOT, 3, &[1, 2, 3]), (HOT, u64::MAX, &[4, 5, 6])]);

    // Three rows and a window of over 20 blocks: each of these prunes sweeps one row.
    prune_to(&store, 2);
    prune_to(&store, 6);
    assert_eq!(account_history(&store), [(HOT, 3, vec![3]), (other, u64::MAX, (7..=30).collect())]);
    assert_eq!([6, 30].map(|at| nonce_at(&store, HOT, at)), [Some(6); 2]);

    let provider = store.provider_rw().unwrap();
    provider.store_trie_updates_batch(vec![(block(31), nonce_diff(&[HOT], 31))]).unwrap();
    OpProofsProviderRw::commit(provider).unwrap();
    assert_eq!(
        account_history(&store),
        [(HOT, 3, vec![3]), (HOT, u64::MAX, vec![31]), (other, u64::MAX, (7..=30).collect())]
    );
    assert_eq!([6, 30, 31].map(|at| nonce_at(&store, HOT, at)), [Some(6), Some(6), Some(31)]);
}

// The sweep can be turned off again. The per-key prune then runs over a history that still
// holds block numbers from before the window.
#[test]
fn per_key_prune_after_a_sweep_serves_the_same_history() {
    const WINDOW: u64 = 8;
    let (_dir, per_key) = store(false);
    let (_dir, swept) = store(true);
    for number in 1..=32 {
        advance(&per_key, number, diff(number), WINDOW);
        advance(&swept, number, diff(number), WINDOW);
    }

    let swept = swept.with_history_sweep(false);
    for number in 33..=64 {
        advance(&per_key, number, diff(number), WINDOW);
        advance(&swept, number, diff(number), WINDOW);
        assert_same_history(&swept, &per_key);
    }
}

// A key changed in more blocks than one shard holds has several rows. A prune removes whole
// shards from the front and trims the one the window starts in.
#[test]
fn sweep_prune_trims_keys_that_span_several_shards() {
    let (_dir, store) = store(true);
    let provider = store.provider_rw().unwrap();
    let blocks = (1..=2_100).map(|number| (block(number), nonce_diff(&[HOT], number))).collect();
    provider.store_trie_updates_batch(blocks).unwrap();
    OpProofsProviderRw::commit(provider).unwrap();
    // Each shard as its key, its first block number and its last.
    let shards = |store: &MdbxProofsStorage| -> Vec<(u64, u64, u64)> {
        account_history(store)
            .into_iter()
            .map(|(_, key, blocks)| (key, blocks[0], *blocks.last().unwrap()))
            .collect()
    };
    assert_eq!(shards(&store), [(2_000, 1, 2_000), (u64::MAX, 2_001, 2_100)]);

    prune_to(&store, 1_801);
    assert_eq!(shards(&store), [(2_000, 1_802, 2_000), (u64::MAX, 2_001, 2_100)]);
    for at in [1_801, 1_802, 2_000, 2_001, 2_100] {
        assert_eq!(nonce_at(&store, HOT, at), Some(at));
    }

    prune_to(&store, 2_050);
    assert_eq!(shards(&store), [(u64::MAX, 2_051, 2_100)]);
    for at in [2_050, 2_051, 2_100] {
        assert_eq!(nonce_at(&store, HOT, at), Some(at));
    }
}

// Backfill adds the block before the proof window and takes every block number in the history
// to be above it. Under a sweep the history still holds numbers from before the window, here
// blocks 1 and 2 of `c` when block 2 is added again. So a swept database is not backfilled.
#[test]
fn backfill_refuses_a_database_whose_history_was_swept() {
    let (_dir, store) = store(true);
    let [a, b, c] = [1, 2, 3].map(B256::with_last_byte);
    let provider = store.provider_rw().unwrap();
    let blocks = (1..=12)
        .map(|number| match number {
            1..=3 => (block(number), nonce_diff(&[a, b, c], number)),
            _ => (block(number), BlockStateDiff::default()),
        })
        .collect();
    provider.store_trie_updates_batch(blocks).unwrap();
    OpProofsProviderRw::commit(provider).unwrap();

    // Each prune sweeps one of the three rows: `a`, then `b`. `c` keeps blocks 1 and 2.
    prune_to(&store, 1);
    prune_to(&store, 2);
    assert_eq!(
        account_history(&store),
        [(a, u64::MAX, vec![2, 3]), (b, u64::MAX, vec![3]), (c, u64::MAX, vec![1, 2, 3])]
    );

    // Block 2 set the three nonces to 2, from 1.
    let provider = store.backfill_provider().unwrap();
    let refused = provider.prepend_block(block(2), nonce_diff(&[a, b, c], 1));
    assert!(
        matches!(refused, Err(OpProofsStorageError::BackfillOverSweptHistory)),
        "expected a refusal, got {refused:?}"
    );
}
