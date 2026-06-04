//! Tests for the V2 MDBX proof storage.

use super::*;
use crate::db::models;
use alloy_eips::NumHash;
use reth_db::{
    BlockNumberList, Database, DatabaseEnv,
    cursor::{DbCursorRW, DbDupCursorRO},
    mdbx::{DatabaseArguments, init_db_for},
    models::sharded_key::ShardedKey,
    transaction::{DbTx, DbTxMut},
};
use reth_trie::{
    HashedStorage,
    updates::{StorageTrieUpdates, TrieUpdates},
};
use tempfile::TempDir;

use crate::{
    BlockStateDiff, OpProofsStorageError,
    api::{
        OpProofsBackfillProvider, OpProofsInitProvider, OpProofsProviderRO, OpProofsProviderRw,
        WriteCounts,
    },
    db::{
        ProofWindowKey, V2ProofWindow,
        models::{
            AccountTrieShardedKey, BlockNumberHashedAddress, HashedAccountBeforeTx,
            HashedAccountShardedKey, HashedStorageShardedKey, StorageTrieShardedKey,
            V2AccountTrieChangeSets, V2AccountsTrie, V2AccountsTrieHistory,
            V2HashedAccountChangeSets, V2HashedAccounts, V2HashedAccountsHistory,
            V2HashedStorageChangeSets, V2HashedStorages, V2HashedStoragesHistory,
            V2StorageTrieChangeSets, V2StoragesTrie, V2StoragesTrieHistory,
        },
    },
};
use alloy_eips::{BlockNumHash, eip1898::BlockWithParent};
use alloy_primitives::{B256, U256};
use reth_db::cursor::DbCursorRO;
use reth_primitives_traits::Account;
use reth_trie::{BranchNodeCompact, HashedPostState, Nibbles, StoredNibbles, StoredNibblesSubKey};

fn setup_db() -> DatabaseEnv {
    let tmp = TempDir::new().expect("create tmpdir");
    init_db_for::<_, models::Tables>(tmp, DatabaseArguments::default()).expect("init db")
}

fn make_block_ref(number: u64, hash: B256, parent: B256) -> BlockWithParent {
    BlockWithParent::new(parent, NumHash::new(number, hash))
}

fn sample_account() -> Account {
    Account { nonce: 1, balance: U256::from(100u64), ..Default::default() }
}

fn sample_node() -> BranchNodeCompact {
    BranchNodeCompact::new(0b1, 0, 0, vec![], Some(B256::repeat_byte(0xAB)))
}

// ========================== Init provider tests ==========================

#[test]
fn init_store_hashed_accounts_writes_to_current_state() {
    let db = setup_db();

    let addr = B256::from([0xAA; 32]);
    let account = sample_account();
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw tx"));
        provider.store_hashed_accounts(vec![(addr, Some(account))]).expect("write");
        OpProofsInitProvider::commit(provider).expect("commit");
    }

    let tx = db.tx().expect("ro tx");
    let mut cur = tx.cursor_read::<V2HashedAccounts>().expect("cursor");
    let (k, v) = cur.seek_exact(addr).expect("seek").expect("exists");
    assert_eq!(k, addr);
    assert_eq!(v.nonce, account.nonce);
    assert_eq!(v.balance, account.balance);
}

#[test]
fn init_store_hashed_storages_writes_to_current_state() {
    let db = setup_db();

    let addr = B256::from([0x11; 32]);
    let slot = B256::from([0x22; 32]);
    let val = U256::from(0x1234u64);
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw tx"));
        provider.store_hashed_storages(addr, vec![(slot, val)]).expect("write");
        OpProofsInitProvider::commit(provider).expect("commit");
    }

    let tx = db.tx().expect("ro tx");
    let mut cur = tx.cursor_dup_read::<V2HashedStorages>().expect("cursor");
    let entry = cur.seek_by_key_subkey(addr, slot).expect("seek").expect("exists");
    assert_eq!(entry.key, slot);
    assert_eq!(entry.value, val);
}

#[test]
fn init_store_account_branches_writes_to_current_state() {
    let db = setup_db();

    let path = Nibbles::from_nibbles_unchecked([0x12, 0x34]);
    let node = sample_node();
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw tx"));
        provider.store_account_branches(vec![(path, Some(node.clone()))]).expect("write");
        OpProofsInitProvider::commit(provider).expect("commit");
    }

    let tx = db.tx().expect("ro tx");
    let mut cur = tx.cursor_read::<V2AccountsTrie>().expect("cursor");
    let (k, v) = cur.seek_exact(StoredNibbles(path)).expect("seek").expect("exists");
    assert_eq!(k.0, path);
    assert_eq!(v, node);
}

#[test]
fn init_store_storage_branches_writes_to_current_state() {
    let db = setup_db();

    let addr = B256::from([0x55; 32]);
    let path = Nibbles::from_nibbles_unchecked([0x12, 0x34]);
    let node = sample_node();
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw tx"));
        provider.store_storage_branches(addr, vec![(path, Some(node.clone()))]).expect("write");
        OpProofsInitProvider::commit(provider).expect("commit");
    }

    let tx = db.tx().expect("ro tx");
    let mut cur = tx.cursor_dup_read::<V2StoragesTrie>().expect("cursor");
    let entry =
        cur.seek_by_key_subkey(addr, StoredNibblesSubKey(path)).expect("seek").expect("exists");
    assert_eq!(entry.nibbles.0, path);
    assert_eq!(entry.node, node);
}

// ========================== Store + unwind tests ==========================

#[test]
fn store_and_read_trie_updates_account() {
    let db = setup_db();

    let addr = B256::from([0xAA; 32]);
    let initial_account = Account { nonce: 1, ..Default::default() };
    let updated_account = Account { nonce: 2, ..Default::default() };

    // Initialize state
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.store_hashed_accounts(vec![(addr, Some(initial_account))]).expect("init");
        provider.set_initial_state_anchor(BlockNumHash::new(0, B256::ZERO)).expect("anchor");
        provider.commit_initial_state().expect("commit init");
        OpProofsInitProvider::commit(provider).expect("commit");
    }

    // Store block 1 update
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        let mut post_state = HashedPostState::default();
        post_state.accounts.insert(addr, Some(updated_account));

        let diff = BlockStateDiff {
            sorted_trie_updates: TrieUpdates::default().into_sorted(),
            sorted_post_state: post_state.into_sorted(),
        };

        let block_ref = make_block_ref(1, B256::repeat_byte(0x01), B256::ZERO);
        provider.store_trie_updates(block_ref, diff).expect("store");
        OpProofsProviderRw::commit(provider).expect("commit");
    }

    // Verify current state has the updated account
    {
        let tx = db.tx().expect("ro");
        let mut cur = tx.cursor_read::<V2HashedAccounts>().expect("cursor");
        let (_, acc) = cur.seek_exact(addr).expect("seek").expect("exists");
        assert_eq!(acc.nonce, 2, "current state should have updated nonce");
    }

    // Verify changeset has the old account
    {
        let tx = db.tx().expect("ro");
        let mut cur = tx.cursor_dup_read::<V2HashedAccountChangeSets>().expect("cursor");
        let entry = cur.seek_by_key_subkey(1u64, addr).expect("seek").expect("exists");
        assert_eq!(entry.hashed_address, addr);
        assert_eq!(entry.info.unwrap().nonce, 1, "changeset should have old nonce");
    }
}

#[test]
fn unwind_restores_old_state() {
    let db = setup_db();

    let addr = B256::from([0xAA; 32]);
    let account_v0 = Account { nonce: 0, ..Default::default() };
    let account_v1 = Account { nonce: 1, ..Default::default() };

    // Initialize with v0
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.store_hashed_accounts(vec![(addr, Some(account_v0))]).expect("init");
        provider.set_initial_state_anchor(BlockNumHash::new(0, B256::ZERO)).expect("anchor");
        provider.commit_initial_state().expect("commit init");
        OpProofsInitProvider::commit(provider).expect("commit");
    }

    // Store block 1
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        let mut post_state = HashedPostState::default();
        post_state.accounts.insert(addr, Some(account_v1));

        let diff = BlockStateDiff {
            sorted_trie_updates: TrieUpdates::default().into_sorted(),
            sorted_post_state: post_state.into_sorted(),
        };

        let block_ref = make_block_ref(1, B256::repeat_byte(0x01), B256::ZERO);
        provider.store_trie_updates(block_ref, diff).expect("store");
        OpProofsProviderRw::commit(provider).expect("commit");
    }

    // Verify v1 is current
    {
        let tx = db.tx().expect("ro");
        let mut cur = tx.cursor_read::<V2HashedAccounts>().expect("cursor");
        let (_, acc) = cur.seek_exact(addr).expect("seek").expect("exists");
        assert_eq!(acc.nonce, 1);
    }

    // Unwind block 1
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        let unwind_to = BlockWithParent::new(B256::ZERO, NumHash::new(1, B256::repeat_byte(0x01)));
        provider.unwind_history(unwind_to).expect("unwind");
        OpProofsProviderRw::commit(provider).expect("commit");
    }

    // Verify v0 is restored
    {
        let tx = db.tx().expect("ro");
        let mut cur = tx.cursor_read::<V2HashedAccounts>().expect("cursor");
        let (_, acc) = cur.seek_exact(addr).expect("seek").expect("exists");
        assert_eq!(acc.nonce, 0, "unwind should restore nonce to 0");
    }
}

#[test]
fn store_creates_history_bitmap() {
    let db = setup_db();

    let addr = B256::from([0xBB; 32]);

    // Initialize
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.store_hashed_accounts(vec![(addr, Some(Account::default()))]).expect("init");
        provider.set_initial_state_anchor(BlockNumHash::new(0, B256::ZERO)).expect("anchor");
        provider.commit_initial_state().expect("commit init");
        OpProofsInitProvider::commit(provider).expect("commit");
    }

    // Store 3 blocks
    let mut parent_hash = B256::ZERO;
    for block_num in 1..=3 {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        let hash = B256::repeat_byte(block_num as u8);
        let mut post_state = HashedPostState::default();
        post_state.accounts.insert(addr, Some(Account { nonce: block_num, ..Default::default() }));

        let diff = BlockStateDiff {
            sorted_trie_updates: TrieUpdates::default().into_sorted(),
            sorted_post_state: post_state.into_sorted(),
        };
        let block_ref = make_block_ref(block_num, hash, parent_hash);
        provider.store_trie_updates(block_ref, diff).expect("store");
        OpProofsProviderRw::commit(provider).expect("commit");
        parent_hash = hash;
    }

    // Verify history bitmap exists and contains blocks 1, 2, 3
    {
        let tx = db.tx().expect("ro");
        let mut cur = tx.cursor_read::<V2HashedAccountsHistory>().expect("cursor");
        let shard_key = HashedAccountShardedKey::new(addr, u64::MAX);
        let (_, bitmap) = cur.seek_exact(shard_key).expect("seek").expect("exists");
        let blocks: Vec<u64> = bitmap.iter().collect();
        assert_eq!(blocks, vec![1, 2, 3]);
    }
}

#[test]
fn prune_removes_changesets() {
    let db = setup_db();

    let addr = B256::from([0xCC; 32]);

    // Initialize
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.store_hashed_accounts(vec![(addr, Some(Account::default()))]).expect("init");
        provider.set_initial_state_anchor(BlockNumHash::new(0, B256::ZERO)).expect("anchor");
        provider.commit_initial_state().expect("commit init");
        OpProofsInitProvider::commit(provider).expect("commit");
    }

    // Store blocks 1-3
    let mut parent_hash = B256::ZERO;
    for block_num in 1u64..=3 {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        let hash = B256::repeat_byte(block_num as u8);
        let mut post_state = HashedPostState::default();
        post_state.accounts.insert(addr, Some(Account { nonce: block_num, ..Default::default() }));
        let diff = BlockStateDiff {
            sorted_trie_updates: TrieUpdates::default().into_sorted(),
            sorted_post_state: post_state.into_sorted(),
        };
        let block_ref = make_block_ref(block_num, hash, parent_hash);
        provider.store_trie_updates(block_ref, diff).expect("store");
        OpProofsProviderRw::commit(provider).expect("commit");
        parent_hash = hash;
    }

    // Prune blocks 1-2
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        let prune_ref = make_block_ref(2, B256::repeat_byte(0x02), B256::repeat_byte(0x01));
        provider.prune_earliest_state(prune_ref).expect("prune");
        OpProofsProviderRw::commit(provider).expect("commit");
    }

    // Verify changesets for blocks 1 and 2 are gone
    {
        let tx = db.tx().expect("ro");
        let mut cur = tx.cursor_read::<V2HashedAccountChangeSets>().expect("cursor");
        // Block 1 should be gone
        assert!(
            cur.seek_exact(1u64).expect("seek").is_none(),
            "block 1 changeset should be pruned"
        );
        // Block 2 should be gone
        assert!(
            cur.seek_exact(2u64).expect("seek").is_none(),
            "block 2 changeset should be pruned"
        );
        // Block 3 should still exist
        assert!(cur.seek_exact(3u64).expect("seek").is_some(), "block 3 changeset should remain");
    }

    // Current state should still be at block 3
    {
        let tx = db.tx().expect("ro");
        let mut cur = tx.cursor_read::<V2HashedAccounts>().expect("cursor");
        let (_, acc) = cur.seek_exact(addr).expect("seek").expect("exists");
        assert_eq!(acc.nonce, 3, "current state should be at block 3");
    }
}

// ========================== Helpers ==========================

/// Initialize database with accounts and set anchor at block 0.
fn init_state(db: &DatabaseEnv, accounts: Vec<(B256, Option<Account>)>) {
    let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
    if !accounts.is_empty() {
        provider.store_hashed_accounts(accounts).expect("init accounts");
    }
    provider.set_initial_state_anchor(BlockNumHash::new(0, B256::ZERO)).expect("anchor");
    provider.commit_initial_state().expect("commit init");
    OpProofsInitProvider::commit(provider).expect("commit");
}

/// Store a block diff.
fn store_block(db: &DatabaseEnv, block_ref: BlockWithParent, diff: BlockStateDiff) {
    let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
    provider.store_trie_updates(block_ref, diff).expect("store");
    OpProofsProviderRw::commit(provider).expect("commit");
}

/// Create a diff with one account change.
fn make_nonce_diff(addr: B256, nonce: u64) -> BlockStateDiff {
    let mut post_state = HashedPostState::default();
    post_state.accounts.insert(addr, Some(Account { nonce, ..Default::default() }));
    BlockStateDiff {
        sorted_trie_updates: TrieUpdates::default().into_sorted(),
        sorted_post_state: post_state.into_sorted(),
    }
}

/// Create a diff that touches one account *and* one storage slot — used by
/// shard-overflow tests where both history bitmaps must grow per block.
fn make_account_and_storage_diff(addr: B256, slot: B256, n: u64) -> BlockStateDiff {
    let mut post_state = HashedPostState::default();
    post_state.accounts.insert(addr, Some(Account { nonce: n, ..Default::default() }));
    let mut storage = HashedStorage::default();
    storage.storage.insert(slot, U256::from(n));
    post_state.storages.insert(addr, storage);
    BlockStateDiff {
        sorted_trie_updates: TrieUpdates::default().into_sorted(),
        sorted_post_state: post_state.into_sorted(),
    }
}

// ========================== Store trie updates tests ==========================

#[test]
fn store_trie_updates_out_of_order_rejects() {
    let db = setup_db();
    init_state(&db, vec![]);

    // Store block 1
    let b1 = make_block_ref(1, B256::repeat_byte(0x01), B256::ZERO);
    store_block(&db, b1, BlockStateDiff::default());

    // Try to store block 2 with wrong parent
    let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
    let bad_block = make_block_ref(2, B256::repeat_byte(0x02), B256::repeat_byte(0xFF));
    let res = provider.store_trie_updates(bad_block, BlockStateDiff::default());
    assert!(matches!(res, Err(OpProofsStorageError::OutOfOrder { .. })));
}

#[test]
fn store_trie_updates_comprehensive() {
    let db = setup_db();

    let addr1 = B256::from([0x11; 32]);
    let addr2 = B256::from([0x22; 32]);
    let slot1 = B256::from([0xA1; 32]);
    let path1 = Nibbles::from_nibbles_unchecked(vec![0, 1, 2, 3]);
    let path2 = Nibbles::from_nibbles_unchecked(vec![4, 5, 6, 7]);
    let removed_path = Nibbles::from_nibbles_unchecked(vec![7, 8, 9]);
    let storage_path1 = Nibbles::from_nibbles_unchecked(vec![1, 2, 3, 4]);

    let acc1_old = Account { nonce: 0, balance: U256::from(50), ..Default::default() };
    let acc1_new = Account { nonce: 1, balance: U256::from(100), ..Default::default() };
    let node1_old = BranchNodeCompact::new(0b1, 0, 0, vec![], Some(B256::repeat_byte(0x01)));
    let node1_new = BranchNodeCompact::default();
    let node2_new = BranchNodeCompact::default();
    let removed_node_old = BranchNodeCompact::new(0b1, 0, 0, vec![], Some(B256::repeat_byte(0x02)));
    let snode1_old = BranchNodeCompact::new(0b1, 0, 0, vec![], Some(B256::repeat_byte(0x03)));
    let snode1_new = BranchNodeCompact::default();
    let val1_old = U256::from(111u64);
    let val1_new = U256::from(1234u64);

    // Initialize state
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.store_hashed_accounts(vec![(addr1, Some(acc1_old))]).expect("init accounts");
        provider.store_hashed_storages(addr1, vec![(slot1, val1_old)]).expect("init storage");
        provider
            .store_account_branches(vec![
                (path1, Some(node1_old.clone())),
                (removed_path, Some(removed_node_old.clone())),
            ])
            .expect("init account trie");
        provider
            .store_storage_branches(addr1, vec![(storage_path1, Some(snode1_old))])
            .expect("init storage trie");
        provider.set_initial_state_anchor(BlockNumHash::new(0, B256::ZERO)).expect("anchor");
        provider.commit_initial_state().expect("commit init");
        OpProofsInitProvider::commit(provider).expect("commit");
    }

    // Build diff
    let mut trie_updates = TrieUpdates::default();
    trie_updates.account_nodes.insert(path1, node1_new.clone());
    trie_updates.account_nodes.insert(path2, node2_new.clone());
    trie_updates.removed_nodes.insert(removed_path);
    let mut stu1 = StorageTrieUpdates::default();
    stu1.storage_nodes.insert(storage_path1, snode1_new.clone());
    trie_updates.storage_tries.insert(addr1, stu1);

    let mut post_state = HashedPostState::default();
    post_state.accounts.insert(addr1, Some(acc1_new));
    post_state.accounts.insert(addr2, None); // deletion of non-existing
    let mut storage1 = HashedStorage::default();
    storage1.storage.insert(slot1, val1_new);
    post_state.storages.insert(addr1, storage1);

    let diff = BlockStateDiff {
        sorted_trie_updates: trie_updates.into_sorted(),
        sorted_post_state: post_state.into_sorted(),
    };

    let block = make_block_ref(42, B256::repeat_byte(0x42), B256::ZERO);
    store_block(&db, block, diff);

    // Verify current state
    let tx = db.tx().expect("ro");

    // Account: addr1 should have new account
    let mut acc_cur = tx.cursor_read::<V2HashedAccounts>().expect("cursor");
    let (_, acc) = acc_cur.seek_exact(addr1).expect("seek").expect("exists");
    assert_eq!(acc.nonce, acc1_new.nonce);
    assert!(acc_cur.seek_exact(addr2).expect("seek").is_none(), "addr2 was never created");

    // Storage: addr1/slot1 should have new value
    let mut stor_cur = tx.cursor_dup_read::<V2HashedStorages>().expect("cursor");
    let entry = stor_cur.seek_by_key_subkey(addr1, slot1).expect("seek").expect("exists");
    assert_eq!(entry.value, val1_new);

    // Account trie: path1 new, path2 new, removed_path gone
    let mut trie_cur = tx.cursor_read::<V2AccountsTrie>().expect("cursor");
    let (_, n) = trie_cur.seek_exact(StoredNibbles(path1)).expect("seek").expect("exists");
    assert_eq!(n, node1_new);
    let (_, n2) = trie_cur.seek_exact(StoredNibbles(path2)).expect("seek").expect("exists");
    assert_eq!(n2, node2_new);
    assert!(
        trie_cur.seek_exact(StoredNibbles(removed_path)).expect("seek").is_none(),
        "removed path should be gone"
    );

    // Storage trie: addr1/storage_path1 should have new node
    let mut strie_cur = tx.cursor_dup_read::<V2StoragesTrie>().expect("cursor");
    let e = strie_cur
        .seek_by_key_subkey(addr1, StoredNibblesSubKey(storage_path1))
        .expect("seek")
        .expect("exists");
    assert_eq!(e.node, snode1_new);

    // Verify account changeset has old values
    let mut cs_cur = tx.cursor_read::<V2HashedAccountChangeSets>().expect("cursor");
    let mut entries = Vec::new();
    let mut walker = cs_cur.walk(Some(42u64)).expect("walk");
    while let Some(Ok((bn, entry))) = walker.next() {
        if bn != 42 {
            break;
        }
        entries.push(entry);
    }
    assert!(entries.iter().any(|e| e.hashed_address == addr1 && e.info == Some(acc1_old)));

    // Verify account trie changeset has old values
    let mut tcs_cur = tx.cursor_read::<V2AccountTrieChangeSets>().expect("cursor");
    let mut tentries = Vec::new();
    let mut walker = tcs_cur.walk(Some(42u64)).expect("walk");
    while let Some(Ok((bn, entry))) = walker.next() {
        if bn != 42 {
            break;
        }
        tentries.push(entry);
    }
    assert!(tentries.iter().any(|e| e.nibbles.0 == path1 && e.node == Some(node1_old.clone())));
    assert!(
        tentries
            .iter()
            .any(|e| e.nibbles.0 == removed_path && e.node == Some(removed_node_old.clone()))
    );
    assert!(tentries.iter().any(|e| e.nibbles.0 == path2 && e.node.is_none()));

    // Verify V2ProofWindow latest
    let mut pw_cur = tx.cursor_read::<V2ProofWindow>().expect("cursor");
    let (_, val) = pw_cur.seek_exact(ProofWindowKey::LatestBlock).expect("seek").expect("exists");
    assert_eq!(val.number(), 42);
}

#[test]
fn store_trie_updates_empty_collections() {
    let db = setup_db();
    init_state(&db, vec![]);

    let block = make_block_ref(42, B256::repeat_byte(0x42), B256::ZERO);
    store_block(&db, block, BlockStateDiff::default());

    // All changeset tables should be empty
    let tx = db.tx().expect("ro");
    let mut cur1 = tx.cursor_read::<V2HashedAccountChangeSets>().expect("cursor");
    assert!(cur1.first().expect("first").is_none(), "Account changesets should be empty");
    let mut cur2 = tx.cursor_read::<V2AccountTrieChangeSets>().expect("cursor");
    assert!(cur2.first().expect("first").is_none(), "Account trie changesets should be empty");

    // V2ProofWindow should be updated
    let mut pw_cur = tx.cursor_read::<V2ProofWindow>().expect("cursor");
    let (_, val) = pw_cur.seek_exact(ProofWindowKey::LatestBlock).expect("seek").expect("exists");
    assert_eq!(val.number(), 42);
}

#[test]
fn store_trie_updates_multiple_blocks() {
    let db = setup_db();
    let addr = B256::from([0x21; 32]);
    init_state(&db, vec![(addr, Some(Account::default()))]);

    let b1 = make_block_ref(1, B256::repeat_byte(0x01), B256::ZERO);
    store_block(&db, b1, make_nonce_diff(addr, 10));

    let b2 = make_block_ref(2, B256::repeat_byte(0x02), B256::repeat_byte(0x01));
    store_block(&db, b2, make_nonce_diff(addr, 20));

    // Current state should have latest nonce
    let tx = db.tx().expect("ro");
    let mut cur = tx.cursor_read::<V2HashedAccounts>().expect("cursor");
    let (_, acc) = cur.seek_exact(addr).expect("seek").expect("exists");
    assert_eq!(acc.nonce, 20);

    // Changeset at block 1 should have old nonce (0)
    let mut cs = tx.cursor_read::<V2HashedAccountChangeSets>().expect("cursor");
    let (_, entry) = cs.seek_exact(1u64).expect("seek").expect("exists");
    assert_eq!(entry.info.unwrap().nonce, 0);

    // Changeset at block 2 should have old nonce (10)
    let (_, entry2) = cs.seek_exact(2u64).expect("seek").expect("exists");
    assert_eq!(entry2.info.unwrap().nonce, 10);
}

#[test]
fn store_trie_updates_deleted_account_trie() {
    let db = setup_db();

    let acc_path = Nibbles::from_nibbles_unchecked([0x0A, 0x0B, 0x0C]);
    let node = sample_node();

    // Initialize with a trie node
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.store_account_branches(vec![(acc_path, Some(node.clone()))]).expect("init");
        provider.set_initial_state_anchor(BlockNumHash::new(0, B256::ZERO)).expect("anchor");
        provider.commit_initial_state().expect("commit init");
        OpProofsInitProvider::commit(provider).expect("commit");
    }

    // Store block that removes the node
    let mut trie_updates = TrieUpdates::default();
    trie_updates.removed_nodes.insert(acc_path);
    let diff =
        BlockStateDiff { sorted_trie_updates: trie_updates.into_sorted(), ..Default::default() };
    let block = make_block_ref(7, B256::repeat_byte(0x07), B256::ZERO);
    store_block(&db, block, diff);

    // Current state should not have the node
    let tx = db.tx().expect("ro");
    let mut cur = tx.cursor_read::<V2AccountsTrie>().expect("cursor");
    assert!(
        cur.seek_exact(StoredNibbles(acc_path)).expect("seek").is_none(),
        "node should be removed from current state"
    );

    // Changeset should have the old node
    let mut cs = tx.cursor_read::<V2AccountTrieChangeSets>().expect("cursor");
    let (_, entry) = cs.seek_exact(7u64).expect("seek").expect("exists");
    assert_eq!(entry.node, Some(node), "changeset should have old node");
}

#[test]
fn store_trie_updates_deleted_storage_trie() {
    let db = setup_db();

    let addr = B256::from([0xAB; 32]);
    let st_path = Nibbles::from_nibbles_unchecked([0x01, 0x02, 0x03]);
    let node = sample_node();

    // Initialize with a storage trie node
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.store_storage_branches(addr, vec![(st_path, Some(node.clone()))]).expect("init");
        provider.set_initial_state_anchor(BlockNumHash::new(0, B256::ZERO)).expect("anchor");
        provider.commit_initial_state().expect("commit init");
        OpProofsInitProvider::commit(provider).expect("commit");
    }

    // Store block that removes the storage trie node
    let mut trie_updates = TrieUpdates::default();
    let mut st_updates = StorageTrieUpdates::default();
    st_updates.removed_nodes.insert(st_path);
    trie_updates.storage_tries.insert(addr, st_updates);
    let diff =
        BlockStateDiff { sorted_trie_updates: trie_updates.into_sorted(), ..Default::default() };
    let block = make_block_ref(8, B256::repeat_byte(0x08), B256::ZERO);
    store_block(&db, block, diff);

    // Current state should not have the node
    let tx = db.tx().expect("ro");
    let mut cur = tx.cursor_dup_read::<V2StoragesTrie>().expect("cursor");
    let result = cur
        .seek_by_key_subkey(addr, StoredNibblesSubKey(st_path))
        .expect("seek")
        .filter(|e| e.nibbles.0 == st_path);
    assert!(result.is_none(), "node should be removed from current state");

    // Changeset should have the old node
    let mut cs = tx.cursor_read::<V2StorageTrieChangeSets>().expect("cursor");
    let start = BlockNumberHashedAddress((8, B256::ZERO));
    let (_, entry) = cs.seek(start).expect("seek").expect("exists");
    assert_eq!(entry.node, Some(node), "changeset should have old node");
}

#[test]
fn store_trie_updates_wiped_storage_trie_nodes() {
    let db = setup_db();

    let addr_wiped = B256::from([0x10; 32]);
    let addr_live = B256::from([0xF0; 32]);
    let p1 = Nibbles::from_nibbles_unchecked([0x01, 0x02]);
    let p2 = Nibbles::from_nibbles_unchecked([0x0A, 0x0B, 0x0C]);
    let n1 = BranchNodeCompact::default();
    let n2 = BranchNodeCompact::default();

    // Seed storage trie nodes for addr_wiped
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider
            .store_storage_branches(addr_wiped, vec![(p1, Some(n1)), (p2, Some(n2))])
            .expect("seed");
        provider.set_initial_state_anchor(BlockNumHash::new(0, B256::ZERO)).expect("anchor");
        provider.commit_initial_state().expect("commit init");
        OpProofsInitProvider::commit(provider).expect("commit");
    }

    // Build diff that wipes addr_wiped's storage trie and adds a node for addr_live
    let mut trie_updates = TrieUpdates::default();
    let mut wiped_updates = StorageTrieUpdates::default();
    wiped_updates.set_deleted(true);
    trie_updates.storage_tries.insert(addr_wiped, wiped_updates);

    let live_path = Nibbles::from_nibbles_unchecked([0xEE, 0xFF]);
    let live_node = BranchNodeCompact::default();
    let mut live_updates = StorageTrieUpdates::default();
    live_updates.storage_nodes.insert(live_path, live_node.clone());
    trie_updates.storage_tries.insert(addr_live, live_updates);

    let diff =
        BlockStateDiff { sorted_trie_updates: trie_updates.into_sorted(), ..Default::default() };
    let block = make_block_ref(123, B256::repeat_byte(0x7B), B256::ZERO);
    store_block(&db, block, diff);

    // Verify: addr_wiped's storage trie nodes should be deleted from current state
    let tx = db.tx().expect("ro");
    let mut cur = tx.cursor_dup_read::<V2StoragesTrie>().expect("cursor");
    assert!(
        cur.seek_exact(addr_wiped).expect("seek").is_none(),
        "wiped address should have no storage trie nodes"
    );

    // addr_live should have its node
    let e = cur
        .seek_by_key_subkey(addr_live, StoredNibblesSubKey(live_path))
        .expect("seek")
        .expect("exists");
    assert_eq!(e.node, live_node);
}

#[test]
fn store_trie_updates_wiped_storage() {
    let db = setup_db();

    let addr = B256::from([0x55; 32]);
    let s1 = B256::from([0x01; 32]);
    let s2 = B256::from([0x02; 32]);
    let v1 = U256::from(111u64);
    let v2 = U256::from(222u64);

    // Seed prior storage
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.store_hashed_storages(addr, vec![(s1, v1), (s2, v2)]).expect("seed");
        provider.set_initial_state_anchor(BlockNumHash::new(0, B256::ZERO)).expect("anchor");
        provider.commit_initial_state().expect("commit init");
        OpProofsInitProvider::commit(provider).expect("commit");
    }

    // Build diff that wipes storage
    let mut post_state = HashedPostState::default();
    post_state.storages.insert(addr, HashedStorage::new(true));
    let diff = BlockStateDiff {
        sorted_trie_updates: TrieUpdates::default().into_sorted(),
        sorted_post_state: post_state.into_sorted(),
    };
    let block = make_block_ref(42, B256::repeat_byte(0x42), B256::ZERO);
    store_block(&db, block, diff);

    // Current state: slots should be deleted
    let tx = db.tx().expect("ro");
    let mut cur = tx.cursor_dup_read::<V2HashedStorages>().expect("cursor");
    assert!(
        cur.seek_exact(addr).expect("seek").is_none(),
        "wiped storage should have no entries in current state"
    );

    // Changeset should have old values
    let mut cs = tx.cursor_read::<V2HashedStorageChangeSets>().expect("cursor");
    let start = BlockNumberHashedAddress((42, addr));
    let mut old_values = Vec::new();
    let mut walker = cs.walk(Some(start)).expect("walk");
    while let Some(Ok((key, entry))) = walker.next() {
        if key.0.0 != 42 || key.0.1 != addr {
            break;
        }
        old_values.push((entry.key, entry.value));
    }
    assert!(old_values.iter().any(|(k, v)| *k == s1 && *v == v1));
    assert!(old_values.iter().any(|(k, v)| *k == s2 && *v == v2));
}

#[test]
fn store_trie_updates_wiped_and_non_wiped_mixed_order() {
    let db = setup_db();

    let addr_wiped = B256::from([0x01; 32]);
    let addr_live = B256::from([0xF0; 32]);
    let ws1 = B256::from([0xA1; 32]);
    let wv1 = U256::from(111u64);
    let ls1 = B256::from([0xB1; 32]);
    let lv1_old = U256::from(333u64);
    let lv1_new = U256::from(999u64);

    // Seed storage for both addresses
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.store_hashed_storages(addr_wiped, vec![(ws1, wv1)]).expect("seed wiped");
        provider.store_hashed_storages(addr_live, vec![(ls1, lv1_old)]).expect("seed live");
        provider.set_initial_state_anchor(BlockNumHash::new(0, B256::ZERO)).expect("anchor");
        provider.commit_initial_state().expect("commit init");
        OpProofsInitProvider::commit(provider).expect("commit");
    }

    // Build diff: wipe addr_wiped, update addr_live
    let mut post_state = HashedPostState::default();
    post_state.storages.insert(addr_wiped, HashedStorage::new(true));
    let mut live_storage = HashedStorage::default();
    live_storage.storage.insert(ls1, lv1_new);
    post_state.storages.insert(addr_live, live_storage);

    let diff = BlockStateDiff {
        sorted_trie_updates: TrieUpdates::default().into_sorted(),
        sorted_post_state: post_state.into_sorted(),
    };
    let block = make_block_ref(77, B256::repeat_byte(0x4D), B256::ZERO);
    store_block(&db, block, diff);

    // Verify: wiped address has no storage
    let tx = db.tx().expect("ro");
    let mut cur = tx.cursor_dup_read::<V2HashedStorages>().expect("cursor");
    assert!(
        cur.seek_exact(addr_wiped).expect("seek").is_none(),
        "wiped addr should have no storage"
    );

    // Live address has new value
    let entry = cur.seek_by_key_subkey(addr_live, ls1).expect("seek").expect("exists");
    assert_eq!(entry.value, lv1_new);
}

// ========================== Fetch tests ==========================

#[test]
fn fetch_trie_updates_basic() {
    let db = setup_db();

    let addr1 = B256::from([0x11; 32]);
    let addr2 = B256::from([0x22; 32]);
    let slot1 = B256::from([0xA1; 32]);
    let path1 = Nibbles::from_nibbles_unchecked(vec![0, 1, 2, 3]);
    let acc1_old = Account { nonce: 0, ..Default::default() };
    let acc1_new = Account { nonce: 1, balance: U256::from(100), ..Default::default() };
    let node1_old = sample_node();
    let node1_new = BranchNodeCompact::default();
    let val1_old = U256::from(111u64);
    let val1_new = U256::from(1234u64);

    // Initialize
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.store_hashed_accounts(vec![(addr1, Some(acc1_old))]).expect("init accounts");
        provider.store_hashed_storages(addr1, vec![(slot1, val1_old)]).expect("init storage");
        provider.store_account_branches(vec![(path1, Some(node1_old))]).expect("init trie");
        provider.set_initial_state_anchor(BlockNumHash::new(0, B256::ZERO)).expect("anchor");
        provider.commit_initial_state().expect("commit init");
        OpProofsInitProvider::commit(provider).expect("commit");
    }

    // Build diff and store
    let mut trie_updates = TrieUpdates::default();
    trie_updates.account_nodes.insert(path1, node1_new);

    let mut post_state = HashedPostState::default();
    post_state.accounts.insert(addr1, Some(acc1_new));
    post_state.accounts.insert(addr2, None); // deletion
    let mut storage1 = HashedStorage::default();
    storage1.storage.insert(slot1, val1_new);
    post_state.storages.insert(addr1, storage1);

    let diff = BlockStateDiff {
        sorted_trie_updates: trie_updates.into_sorted(),
        sorted_post_state: post_state.into_sorted(),
    };
    let block = make_block_ref(1, B256::repeat_byte(0x01), B256::ZERO);
    store_block(&db, block, diff);

    // Fetch block 1
    let provider = MdbxProofsProviderV2::new(db.tx().expect("ro"));
    let got = provider.fetch_trie_updates(1).expect("fetch");

    // Verify: accounts should have current values
    assert!(
        got.sorted_post_state.accounts.iter().any(|(a, v)| *a == addr1 && v == &Some(acc1_new))
    );

    // Verify: trie updates should have current node
    assert!(!got.sorted_trie_updates.account_nodes_ref().is_empty());

    // Verify: storages should have current value
    assert!(!got.sorted_post_state.storages.is_empty());
}

#[test]
fn fetch_trie_updates_empty_changeset() {
    let db = setup_db();
    init_state(&db, vec![]);

    let block = make_block_ref(1, B256::repeat_byte(0x01), B256::ZERO);
    store_block(&db, block, BlockStateDiff::default());

    let provider = MdbxProofsProviderV2::new(db.tx().expect("ro"));
    let got = provider.fetch_trie_updates(1).expect("fetch");
    assert!(got.sorted_trie_updates.account_nodes_ref().is_empty());
    assert!(got.sorted_trie_updates.storage_tries_ref().is_empty());
    assert!(got.sorted_post_state.accounts.is_empty());
    assert!(got.sorted_post_state.storages.is_empty());
}

// ========================== Proof window tests ==========================

#[test]
fn test_proof_window() {
    let db = setup_db();

    // Empty store: both endpoints are None.
    {
        let provider = MdbxProofsProviderV2::new(db.tx().expect("ro"));
        assert!(matches!(provider.get_earliest_block(), Err(OpProofsStorageError::NoBlocksFound)));
        assert!(matches!(provider.get_latest_block(), Err(OpProofsStorageError::NoBlocksFound)));
    }

    // Bootstrap via the init flow: commit_initial_state sets both earliest and latest.
    let anchor_hash = B256::repeat_byte(0x42);
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider
            .set_initial_state_anchor(BlockNumHash { number: 42, hash: anchor_hash })
            .expect("anchor");
        provider.commit_initial_state().expect("commit init");
        OpProofsInitProvider::commit(provider).expect("commit");
    }

    {
        let provider = MdbxProofsProviderV2::new(db.tx().expect("ro"));
        assert_eq!(provider.get_earliest_block().expect("get"), NumHash::new(42, anchor_hash));
        assert_eq!(provider.get_latest_block().expect("get"), NumHash::new(42, anchor_hash));
    }
}

// ========================== Prune tests ==========================

#[test]
fn test_prune_earliest_state_no_op() {
    let db = setup_db();
    init_state(&db, vec![]);

    // Attempt to prune with a block that is not newer than earliest
    let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
    let block_0 = make_block_ref(0, B256::repeat_byte(0x00), B256::ZERO);
    let result = provider.prune_earliest_state(block_0);
    assert!(
        matches!(result, Err(OpProofsStorageError::PruneBeyondEarliest { .. })),
        "expected PruneBeyondEarliest, got {result:?}"
    );
}

#[test]
fn test_prune_earliest_state_uninitialized_guard() {
    let db = setup_db();
    // Don't initialize — pruning uninitialized store returns NoBlocksFound.

    let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
    let target = make_block_ref(5, B256::repeat_byte(0x05), B256::ZERO);
    let result = provider.prune_earliest_state(target);
    assert!(
        matches!(result, Err(OpProofsStorageError::NoBlocksFound)),
        "expected NoBlocksFound, got {result:?}"
    );
}

#[test]
fn test_prune_earliest_state_overlapping_keys() {
    let db = setup_db();

    let addr = B256::from([0xDD; 32]);
    let acc1 = Account { nonce: 1, ..Default::default() };
    let acc2 = Account { nonce: 2, ..Default::default() };

    init_state(&db, vec![(addr, Some(Account::default()))]);

    // Block 1: update
    let b1 = make_block_ref(1, B256::repeat_byte(0x01), B256::ZERO);
    store_block(&db, b1, make_nonce_diff(addr, acc1.nonce));

    // Block 2: update same key
    let b2 = make_block_ref(2, B256::repeat_byte(0x02), B256::repeat_byte(0x01));
    store_block(&db, b2, make_nonce_diff(addr, acc2.nonce));

    // Prune to block 3
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        let b3 = make_block_ref(3, B256::repeat_byte(0x03), B256::repeat_byte(0x02));
        provider.prune_earliest_state(b3).expect("prune");
        OpProofsProviderRw::commit(provider).expect("commit");
    }

    // Current state should still have nonce 2 (latest)
    let tx = db.tx().expect("ro");
    let mut cur = tx.cursor_read::<V2HashedAccounts>().expect("cursor");
    let (_, acc) = cur.seek_exact(addr).expect("seek").expect("exists");
    assert_eq!(acc.nonce, 2, "current state should still have latest value");

    // Changesets for blocks 1 and 2 should be gone
    let mut cs = tx.cursor_read::<V2HashedAccountChangeSets>().expect("cursor");
    assert!(cs.seek_exact(1u64).expect("seek").is_none());
    assert!(cs.seek_exact(2u64).expect("seek").is_none());
}

#[test]
fn test_prune_earliest_state_comprehensive() {
    let db = setup_db();

    let addr = B256::from([0xEE; 32]);
    let slot = B256::from([0xAA; 32]);
    let path = Nibbles::from_nibbles_unchecked([0x01]);
    let storage_path = Nibbles::from_nibbles_unchecked([0x03]);
    let node_old = sample_node();
    let snode_old = sample_node();

    // Initialize with all 4 data types
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.store_hashed_accounts(vec![(addr, Some(Account::default()))]).expect("init");
        provider.store_hashed_storages(addr, vec![(slot, U256::from(100u64))]).expect("init");
        provider.store_account_branches(vec![(path, Some(node_old))]).expect("init");
        provider.store_storage_branches(addr, vec![(storage_path, Some(snode_old))]).expect("init");
        provider.set_initial_state_anchor(BlockNumHash::new(0, B256::ZERO)).expect("anchor");
        provider.commit_initial_state().expect("commit init");
        OpProofsInitProvider::commit(provider).expect("commit");
    }

    // Block 1: update account and trie
    let b1 = make_block_ref(1, B256::repeat_byte(0x01), B256::ZERO);
    {
        let mut trie_updates = TrieUpdates::default();
        trie_updates.account_nodes.insert(path, BranchNodeCompact::default());
        let mut post_state = HashedPostState::default();
        post_state.accounts.insert(addr, Some(Account { nonce: 1, ..Default::default() }));
        let mut storage = HashedStorage::default();
        storage.storage.insert(slot, U256::from(200u64));
        post_state.storages.insert(addr, storage);
        let diff = BlockStateDiff {
            sorted_trie_updates: trie_updates.into_sorted(),
            sorted_post_state: post_state.into_sorted(),
        };
        store_block(&db, b1, diff);
    }

    // Block 2: update account again
    let b2 = make_block_ref(2, B256::repeat_byte(0x02), B256::repeat_byte(0x01));
    store_block(&db, b2, make_nonce_diff(addr, 2));

    // Prune to block 3
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        let b3 = make_block_ref(3, B256::repeat_byte(0x03), B256::repeat_byte(0x02));
        provider.prune_earliest_state(b3).expect("prune");
        OpProofsProviderRw::commit(provider).expect("commit");
    }

    // Current state should still have latest values
    let tx = db.tx().expect("ro");
    let mut acc_cur = tx.cursor_read::<V2HashedAccounts>().expect("cursor");
    let (_, acc) = acc_cur.seek_exact(addr).expect("seek").expect("exists");
    assert_eq!(acc.nonce, 2);

    // Changesets should be gone for blocks 1 and 2
    let mut cs = tx.cursor_read::<V2HashedAccountChangeSets>().expect("cursor");
    assert!(cs.seek_exact(1u64).expect("seek").is_none());
    assert!(cs.seek_exact(2u64).expect("seek").is_none());

    let mut tcs = tx.cursor_read::<V2AccountTrieChangeSets>().expect("cursor");
    assert!(tcs.seek_exact(1u64).expect("seek").is_none());
}

#[test]
fn test_prune_earliest_state_returns_correct_counts() {
    let db = setup_db();
    let addr = B256::from([0xFF; 32]);
    init_state(&db, vec![(addr, Some(Account::default()))]);

    // Block 1: update
    let b1 = make_block_ref(1, B256::repeat_byte(0x01), B256::ZERO);
    store_block(&db, b1, make_nonce_diff(addr, 1));

    // Block 2: update
    let b2 = make_block_ref(2, B256::repeat_byte(0x02), B256::repeat_byte(0x01));
    store_block(&db, b2, make_nonce_diff(addr, 2));

    // Prune to block 2 — should remove changeset for block 1
    let counts = {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        let prune_ref = make_block_ref(2, B256::repeat_byte(0x02), B256::repeat_byte(0x01));
        let c = provider.prune_earliest_state(prune_ref).expect("prune");
        OpProofsProviderRw::commit(provider).expect("commit");
        c
    };

    // Range is (earliest+1)..=target = 1..=2, pruning changesets for both blocks
    assert_eq!(counts.hashed_accounts_written_total, 2);
}

// ========================== Replace tests ==========================

#[test]
fn replace_updates_prunes_and_adds_new_chain() {
    let db = setup_db();
    let addr = B256::from([0xAB; 32]);
    init_state(&db, vec![(addr, Some(Account::default()))]);

    // Build initial chain: 1 -> 2 -> 3
    let b1 = make_block_ref(1, B256::repeat_byte(0x01), B256::ZERO);
    let b2 = make_block_ref(2, B256::repeat_byte(0x02), B256::repeat_byte(0x01));
    let b3 = make_block_ref(3, B256::repeat_byte(0x03), B256::repeat_byte(0x02));

    store_block(&db, b1, make_nonce_diff(addr, 10));
    store_block(&db, b2, make_nonce_diff(addr, 20));
    store_block(&db, b3, make_nonce_diff(addr, 30));

    // Sanity: current state has nonce 30
    {
        let tx = db.tx().expect("ro");
        let mut cur = tx.cursor_read::<V2HashedAccounts>().expect("cursor");
        let (_, acc) = cur.seek_exact(addr).expect("seek").expect("exists");
        assert_eq!(acc.nonce, 30);
    }

    // Reorg at LCA = 2: prune >2, add 3' and 4' (same block numbers as
    // the old chain). This works because replace_updates now cleans
    // history bitmaps before re-inserting.
    let b3p = make_block_ref(3, B256::repeat_byte(0xA3), B256::repeat_byte(0x02));
    let b4p = make_block_ref(4, B256::repeat_byte(0xA4), B256::repeat_byte(0xA3));

    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider
            .replace_updates(
                BlockNumHash::new(2, B256::repeat_byte(0x02)),
                vec![(b3p, make_nonce_diff(addr, 300)), (b4p, make_nonce_diff(addr, 400))],
            )
            .expect("replace");
        OpProofsProviderRw::commit(provider).expect("commit");
    }

    // Verify: current state has nonce 400
    {
        let tx = db.tx().expect("ro");
        let mut cur = tx.cursor_read::<V2HashedAccounts>().expect("cursor");
        let (_, acc) = cur.seek_exact(addr).expect("seek").expect("exists");
        assert_eq!(acc.nonce, 400);
    }

    // Verify: changesets exist for blocks 1, 2, 3', 4'
    {
        let tx = db.tx().expect("ro");
        let mut cs = tx.cursor_read::<V2HashedAccountChangeSets>().expect("cursor");
        assert!(cs.seek_exact(1u64).expect("seek").is_some(), "block 1 changeset");
        assert!(cs.seek_exact(2u64).expect("seek").is_some(), "block 2 changeset");
        assert!(cs.seek_exact(3u64).expect("seek").is_some(), "block 3' changeset");
        assert!(cs.seek_exact(4u64).expect("seek").is_some(), "block 4' changeset");
    }
}

#[test]
fn test_replace_updates_beyond_earliest_returns_error() {
    let db = setup_db();
    let addr = B256::from([0xCC; 32]);
    init_state(&db, vec![(addr, Some(Account::default()))]);

    // Build chain: 1 -> 2 -> 3, then prune to earliest = 2.
    let b1 = make_block_ref(1, B256::repeat_byte(0x01), B256::ZERO);
    let b2 = make_block_ref(2, B256::repeat_byte(0x02), B256::repeat_byte(0x01));
    let b3 = make_block_ref(3, B256::repeat_byte(0x03), B256::repeat_byte(0x02));

    store_block(&db, b1, make_nonce_diff(addr, 10));
    store_block(&db, b2, make_nonce_diff(addr, 20));
    store_block(&db, b3, make_nonce_diff(addr, 30));

    // Move earliest forward to block 2.
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider
            .prune_earliest_state(make_block_ref(
                2,
                B256::repeat_byte(0x02),
                B256::repeat_byte(0x01),
            ))
            .expect("prune");
        OpProofsProviderRw::commit(provider).expect("commit");
    }

    // Attempt to replace_updates with a common block before earliest (block 1).
    let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
    let res = provider.replace_updates(BlockNumHash::new(1, B256::repeat_byte(0x01)), vec![]);
    assert!(
        matches!(res, Err(OpProofsStorageError::ReorgBaseOutOfWindow { .. })),
        "expected ReorgBaseOutOfWindow, got {res:?}"
    );
}

#[test]
fn test_replace_updates_at_earliest_returns_error() {
    let db = setup_db();
    let addr = B256::from([0xCC; 32]);
    init_state(&db, vec![(addr, Some(Account::default()))]);

    // Build chain: 1 -> 2 -> 3, then prune to earliest = 2.
    let b1 = make_block_ref(1, B256::repeat_byte(0x01), B256::ZERO);
    let b2 = make_block_ref(2, B256::repeat_byte(0x02), B256::repeat_byte(0x01));
    let b3 = make_block_ref(3, B256::repeat_byte(0x03), B256::repeat_byte(0x02));

    store_block(&db, b1, make_nonce_diff(addr, 10));
    store_block(&db, b2, make_nonce_diff(addr, 20));
    store_block(&db, b3, make_nonce_diff(addr, 30));

    // Move earliest forward to block 2.
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider
            .prune_earliest_state(make_block_ref(
                2,
                B256::repeat_byte(0x02),
                B256::repeat_byte(0x01),
            ))
            .expect("prune");
        OpProofsProviderRw::commit(provider).expect("commit");
    }

    // Attempt to replace_updates with common block at earliest (block 2) — no valid anchor.
    let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
    let res = provider.replace_updates(BlockNumHash::new(2, B256::repeat_byte(0x02)), vec![]);
    assert!(
        matches!(res, Err(OpProofsStorageError::ReorgBaseOutOfWindow { .. })),
        "expected ReorgBaseOutOfWindow, got {res:?}"
    );
}

#[test]
fn test_replace_updates_ahead_of_latest_returns_error() {
    let db = setup_db();
    let addr = B256::from([0xDD; 32]);
    init_state(&db, vec![(addr, Some(Account::default()))]);

    // Build chain: 1 -> 2. Latest = 2.
    let b1 = make_block_ref(1, B256::repeat_byte(0x01), B256::ZERO);
    let b2 = make_block_ref(2, B256::repeat_byte(0x02), B256::repeat_byte(0x01));

    store_block(&db, b1, make_nonce_diff(addr, 10));
    store_block(&db, b2, make_nonce_diff(addr, 20));

    // Attempt to replace_updates with a common block beyond latest (block 5).
    let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
    let res = provider.replace_updates(BlockNumHash::new(5, B256::repeat_byte(0x05)), vec![]);
    assert!(
        matches!(res, Err(OpProofsStorageError::ReorgBaseOutOfWindow { .. })),
        "expected ReorgBaseOutOfWindow, got {res:?}"
    );
}

// ========================== Unwind tests ==========================

#[test]
fn test_unwind_history_to_earliest() {
    let db = setup_db();
    let addr = B256::from([0xBB; 32]);

    // Initialize and set earliest at block 1
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.store_hashed_accounts(vec![(addr, Some(Account::default()))]).expect("init");
        provider
            .set_initial_state_anchor(BlockNumHash::new(1, B256::repeat_byte(0x01)))
            .expect("anchor");
        provider.commit_initial_state().expect("commit init");
        OpProofsInitProvider::commit(provider).expect("commit");
    }

    // Store blocks 2, 3
    let b2 = make_block_ref(2, B256::repeat_byte(0x02), B256::repeat_byte(0x01));
    let b3 = make_block_ref(3, B256::repeat_byte(0x03), B256::repeat_byte(0x02));
    store_block(&db, b2, make_nonce_diff(addr, 2));
    store_block(&db, b3, make_nonce_diff(addr, 3));

    // Try to unwind to block 1 (= earliest) — should error
    let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
    let unwind_to =
        BlockWithParent::new(B256::repeat_byte(0x01), NumHash::new(1, B256::repeat_byte(0x01)));
    let res = provider.unwind_history(unwind_to);
    assert!(
        matches!(res, Err(OpProofsStorageError::UnwindBeyondEarliest { .. })),
        "should error when unwinding to earliest"
    );
}

#[test]
fn test_unwind_history_with_storage() {
    let db = setup_db();

    let addr = B256::from([0xCC; 32]);
    let slot = B256::from([0xDD; 32]);

    // Initialize with account and storage
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.store_hashed_accounts(vec![(addr, Some(Account::default()))]).expect("init");
        provider.store_hashed_storages(addr, vec![(slot, U256::from(100u64))]).expect("init");
        provider.set_initial_state_anchor(BlockNumHash::new(0, B256::ZERO)).expect("anchor");
        provider.commit_initial_state().expect("commit init");
        OpProofsInitProvider::commit(provider).expect("commit");
    }

    // Block 1: update both account and storage
    {
        let mut post_state = HashedPostState::default();
        post_state.accounts.insert(addr, Some(Account { nonce: 1, ..Default::default() }));
        let mut storage = HashedStorage::default();
        storage.storage.insert(slot, U256::from(200u64));
        post_state.storages.insert(addr, storage);
        let diff = BlockStateDiff {
            sorted_trie_updates: TrieUpdates::default().into_sorted(),
            sorted_post_state: post_state.into_sorted(),
        };
        let b1 = make_block_ref(1, B256::repeat_byte(0x01), B256::ZERO);
        store_block(&db, b1, diff);
    }

    // Block 2: update storage again
    {
        let mut post_state = HashedPostState::default();
        post_state.accounts.insert(addr, Some(Account { nonce: 2, ..Default::default() }));
        let mut storage = HashedStorage::default();
        storage.storage.insert(slot, U256::from(300u64));
        post_state.storages.insert(addr, storage);
        let diff = BlockStateDiff {
            sorted_trie_updates: TrieUpdates::default().into_sorted(),
            sorted_post_state: post_state.into_sorted(),
        };
        let b2 = make_block_ref(2, B256::repeat_byte(0x02), B256::repeat_byte(0x01));
        store_block(&db, b2, diff);
    }

    // Unwind to block 2 (removes blocks 2+)
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        let unwind_to =
            BlockWithParent::new(B256::repeat_byte(0x01), NumHash::new(2, B256::repeat_byte(0x02)));
        provider.unwind_history(unwind_to).expect("unwind");
        OpProofsProviderRw::commit(provider).expect("commit");
    }

    // Verify: account restored to nonce 1, storage restored to 200
    let tx = db.tx().expect("ro");
    let mut acc_cur = tx.cursor_read::<V2HashedAccounts>().expect("cursor");
    let (_, acc) = acc_cur.seek_exact(addr).expect("seek").expect("exists");
    assert_eq!(acc.nonce, 1, "account should be restored to block 1 state");

    let mut stor_cur = tx.cursor_dup_read::<V2HashedStorages>().expect("cursor");
    let entry = stor_cur.seek_by_key_subkey(addr, slot).expect("seek").expect("exists");
    assert_eq!(entry.value, U256::from(200u64), "storage should be restored to block 1");
}

#[test]
fn test_unwind_history_with_trie_nodes() {
    let db = setup_db();

    let path1 = Nibbles::from_nibbles_unchecked([0x01]);
    let path2 = Nibbles::from_nibbles_unchecked([0x02]);
    let node1 = BranchNodeCompact::new(0b1, 0, 0, vec![], Some(B256::repeat_byte(0x11)));
    let node2 = BranchNodeCompact::new(0b10, 0, 0, vec![], Some(B256::repeat_byte(0x22)));

    // Initialize with node1 at path1
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.store_account_branches(vec![(path1, Some(node1.clone()))]).expect("init");
        provider.set_initial_state_anchor(BlockNumHash::new(0, B256::ZERO)).expect("anchor");
        provider.commit_initial_state().expect("commit init");
        OpProofsInitProvider::commit(provider).expect("commit");
    }

    // Block 1: add node2 at path2
    {
        let mut trie_updates = TrieUpdates::default();
        trie_updates.account_nodes.insert(path2, node2.clone());
        let diff = BlockStateDiff {
            sorted_trie_updates: trie_updates.into_sorted(),
            ..Default::default()
        };
        let b1 = make_block_ref(1, B256::repeat_byte(0x01), B256::ZERO);
        store_block(&db, b1, diff);
    }

    // Block 2: overwrite path1
    {
        let mut trie_updates = TrieUpdates::default();
        trie_updates.account_nodes.insert(path1, node2.clone());
        let diff = BlockStateDiff {
            sorted_trie_updates: trie_updates.into_sorted(),
            ..Default::default()
        };
        let b2 = make_block_ref(2, B256::repeat_byte(0x02), B256::repeat_byte(0x01));
        store_block(&db, b2, diff);
    }

    // Unwind to block 2 (removes blocks 2+)
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        let unwind_to =
            BlockWithParent::new(B256::repeat_byte(0x01), NumHash::new(2, B256::repeat_byte(0x02)));
        provider.unwind_history(unwind_to).expect("unwind");
        OpProofsProviderRw::commit(provider).expect("commit");
    }

    // Verify: path1 should be restored to node1, path2 should still have node2 (from block 1)
    let tx = db.tx().expect("ro");
    let mut cur = tx.cursor_read::<V2AccountsTrie>().expect("cursor");
    let (_, n) = cur.seek_exact(StoredNibbles(path1)).expect("seek").expect("exists");
    assert_eq!(n, node1, "path1 should be restored to original node");
    let (_, n2) = cur.seek_exact(StoredNibbles(path2)).expect("seek").expect("exists");
    assert_eq!(n2, node2, "path2 should still have block 1 node");
}

#[test]
fn test_unwind_history_comprehensive() {
    let db = setup_db();

    let addr1 = B256::from([0x11; 32]);
    let addr2 = B256::from([0x22; 32]);
    let slot1 = B256::from([0xA1; 32]);
    let path1 = Nibbles::from_nibbles_unchecked([0x01]);
    let path2 = Nibbles::from_nibbles_unchecked([0x02]);
    let storage_path1 = Nibbles::from_nibbles_unchecked([0x03]);

    let acc1 = Account { nonce: 1, ..Default::default() };
    let node1 = sample_node();
    let snode1 = sample_node();

    // Initialize
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.store_hashed_accounts(vec![(addr1, Some(acc1))]).expect("init");
        provider.store_hashed_storages(addr1, vec![(slot1, U256::from(1111u64))]).expect("init");
        provider.store_account_branches(vec![(path1, Some(node1))]).expect("init");
        provider.store_storage_branches(addr1, vec![(storage_path1, Some(snode1))]).expect("init");
        provider.set_initial_state_anchor(BlockNumHash::new(0, B256::ZERO)).expect("anchor");
        provider.commit_initial_state().expect("commit init");
        OpProofsInitProvider::commit(provider).expect("commit");
    }

    // Block 1: update all domains
    {
        let mut trie_updates = TrieUpdates::default();
        trie_updates.account_nodes.insert(path1, BranchNodeCompact::default());
        let mut stu = StorageTrieUpdates::default();
        stu.storage_nodes.insert(storage_path1, BranchNodeCompact::default());
        trie_updates.storage_tries.insert(addr1, stu);

        let mut post_state = HashedPostState::default();
        post_state.accounts.insert(addr1, Some(Account { nonce: 10, ..Default::default() }));
        let mut storage = HashedStorage::default();
        storage.storage.insert(slot1, U256::from(2222u64));
        post_state.storages.insert(addr1, storage);

        let diff = BlockStateDiff {
            sorted_trie_updates: trie_updates.into_sorted(),
            sorted_post_state: post_state.into_sorted(),
        };
        let b1 = make_block_ref(1, B256::repeat_byte(0x01), B256::ZERO);
        store_block(&db, b1, diff);
    }

    // Block 2: more updates
    {
        let mut trie_updates = TrieUpdates::default();
        trie_updates.account_nodes.insert(path2, BranchNodeCompact::default());

        let mut post_state = HashedPostState::default();
        post_state.accounts.insert(addr2, Some(Account { nonce: 20, ..Default::default() }));

        let diff = BlockStateDiff {
            sorted_trie_updates: trie_updates.into_sorted(),
            sorted_post_state: post_state.into_sorted(),
        };
        let b2 = make_block_ref(2, B256::repeat_byte(0x02), B256::repeat_byte(0x01));
        store_block(&db, b2, diff);
    }

    // Unwind to block 2 (removes blocks 2+)
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        let unwind_to =
            BlockWithParent::new(B256::repeat_byte(0x01), NumHash::new(2, B256::repeat_byte(0x02)));
        provider.unwind_history(unwind_to).expect("unwind");
        OpProofsProviderRw::commit(provider).expect("commit");
    }

    let tx = db.tx().expect("ro");

    // Verify account: addr1 should have block 1 state (nonce 10)
    let mut acc_cur = tx.cursor_read::<V2HashedAccounts>().expect("cursor");
    let (_, acc) = acc_cur.seek_exact(addr1).expect("seek").expect("exists");
    assert_eq!(acc.nonce, 10, "addr1 should have block 1 state");
    // addr2 should not exist (was added in block 2, unwound)
    assert!(acc_cur.seek_exact(addr2).expect("seek").is_none(), "addr2 should be removed");

    // Verify trie: path1 should have block 1 value
    let mut trie_cur = tx.cursor_read::<V2AccountsTrie>().expect("cursor");
    assert!(trie_cur.seek_exact(StoredNibbles(path1)).expect("seek").is_some());
    // path2 should not exist (added in block 2, unwound)
    assert!(
        trie_cur.seek_exact(StoredNibbles(path2)).expect("seek").is_none(),
        "path2 should be removed"
    );

    // Verify storage: should have block 1 value
    let mut stor_cur = tx.cursor_dup_read::<V2HashedStorages>().expect("cursor");
    let entry = stor_cur.seek_by_key_subkey(addr1, slot1).expect("seek").expect("exists");
    assert_eq!(entry.value, U256::from(2222u64), "storage should have block 1 value");

    // Verify changesets for blocks 2+ are gone
    let mut cs = tx.cursor_read::<V2HashedAccountChangeSets>().expect("cursor");
    assert!(cs.seek_exact(1u64).expect("seek").is_some(), "block 1 changeset should remain");
    assert!(cs.seek_exact(2u64).expect("seek").is_none(), "block 2 changeset should be removed");

    // Verify V2ProofWindow latest
    let mut pw_cur = tx.cursor_read::<V2ProofWindow>().expect("cursor");
    let (_, val) = pw_cur.seek_exact(ProofWindowKey::LatestBlock).expect("seek").expect("exists");
    assert_eq!(val.number(), 1);
}

#[test]
fn test_unwind_history_empty_chain() {
    let db = setup_db();

    // No blocks stored — uninitialized proof window returns NoBlocksFound.
    let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
    let unwind_to = BlockWithParent::new(B256::ZERO, NumHash::new(0, B256::ZERO));
    let result = provider.unwind_history(unwind_to);
    assert!(
        matches!(result, Err(OpProofsStorageError::NoBlocksFound)),
        "expected NoBlocksFound, got {result:?}"
    );
}

#[test]
fn test_unwind_history_idempotent() {
    let db = setup_db();
    let addr = B256::from([0xDD; 32]);
    init_state(&db, vec![(addr, Some(Account::default()))]);

    // Store blocks 1, 2, 3
    let b1 = make_block_ref(1, B256::repeat_byte(0x01), B256::ZERO);
    let b2 = make_block_ref(2, B256::repeat_byte(0x02), B256::repeat_byte(0x01));
    let b3 = make_block_ref(3, B256::repeat_byte(0x03), B256::repeat_byte(0x02));
    store_block(&db, b1, make_nonce_diff(addr, 10));
    store_block(&db, b2, make_nonce_diff(addr, 20));
    store_block(&db, b3, make_nonce_diff(addr, 30));

    // Unwind to block 2
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.unwind_history(b2).expect("first unwind");
        OpProofsProviderRw::commit(provider).expect("commit");
    }

    // Unwind again (should be idempotent)
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.unwind_history(b2).expect("second unwind");
        OpProofsProviderRw::commit(provider).expect("commit");
    }

    // Verify state
    let tx = db.tx().expect("ro");
    let mut cur = tx.cursor_read::<V2HashedAccounts>().expect("cursor");
    let (_, acc) = cur.seek_exact(addr).expect("seek").expect("exists");
    assert_eq!(acc.nonce, 10, "should have block 1 state after unwind to block 2");
}

#[test]
fn test_unwind_history_beyond_latest() {
    let db = setup_db();
    let addr = B256::from([0xEE; 32]);
    init_state(&db, vec![(addr, Some(Account::default()))]);

    // Store blocks 1, 2, 3
    let b1 = make_block_ref(1, B256::repeat_byte(0x01), B256::ZERO);
    let b2 = make_block_ref(2, B256::repeat_byte(0x02), B256::repeat_byte(0x01));
    let b3 = make_block_ref(3, B256::repeat_byte(0x03), B256::repeat_byte(0x02));
    store_block(&db, b1, make_nonce_diff(addr, 10));
    store_block(&db, b2, make_nonce_diff(addr, 20));
    store_block(&db, b3, make_nonce_diff(addr, 30));

    // Unwind to block 5 (beyond latest) — should be no-op
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        let b5 = make_block_ref(5, B256::repeat_byte(0x05), B256::repeat_byte(0x04));
        provider.unwind_history(b5).expect("unwind");
        OpProofsProviderRw::commit(provider).expect("commit");
    }

    // All blocks should remain
    let tx = db.tx().expect("ro");
    let mut cur = tx.cursor_read::<V2HashedAccounts>().expect("cursor");
    let (_, acc) = cur.seek_exact(addr).expect("seek").expect("exists");
    assert_eq!(acc.nonce, 30, "state should be unchanged");

    let mut pw_cur = tx.cursor_read::<V2ProofWindow>().expect("cursor");
    let (_, val) = pw_cur.seek_exact(ProofWindowKey::LatestBlock).expect("seek").expect("exists");
    assert_eq!(val.number(), 3, "latest should be unchanged");
}

/// Helper: count the total number of duplicate entries for a given primary key
/// in the `V2HashedStorages` `DupSort` table.
fn count_hashed_storage_entries(db: &DatabaseEnv, addr: B256) -> usize {
    let tx = db.tx().expect("ro");
    let mut cur = tx.cursor_dup_read::<V2HashedStorages>().expect("cursor");
    let mut count = 0;
    if cur.seek_by_key_subkey(addr, B256::ZERO).expect("seek").is_some() {
        count += 1;
        while cur.next_dup_val().expect("next").is_some() {
            count += 1;
        }
    }
    count
}

/// Helper: collect all (slot, value) pairs for an address from `V2HashedStorages`.
fn collect_hashed_storage_slots(db: &DatabaseEnv, addr: B256) -> Vec<(B256, U256)> {
    let tx = db.tx().expect("ro");
    let mut cur = tx.cursor_dup_read::<V2HashedStorages>().expect("cursor");
    let mut entries = Vec::new();
    if let Some(entry) = cur.seek_by_key_subkey(addr, B256::ZERO).expect("seek") {
        entries.push((entry.key, entry.value));
        while let Some(entry) = cur.next_dup_val().expect("next") {
            entries.push((entry.key, entry.value));
        }
    }
    entries
}

/// Regression: updating the same slot across multiple blocks must NOT create
/// duplicate entries. Each (address, slot) pair should appear exactly once in
/// the current-state table.
#[test]
fn hashed_storages_no_duplicate_entries_after_multi_block_update() {
    let db = setup_db();

    let addr = B256::from([0xDE; 32]);
    let slot = B256::from([0xAB; 32]);

    // Initialize with one storage slot
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.store_hashed_storages(addr, vec![(slot, U256::from(100u64))]).expect("seed");
        provider.set_initial_state_anchor(BlockNumHash::new(0, B256::ZERO)).expect("anchor");
        provider.commit_initial_state().expect("commit init");
        OpProofsInitProvider::commit(provider).expect("commit");
    }

    // Verify initial state: exactly 1 entry
    assert_eq!(count_hashed_storage_entries(&db, addr), 1, "initial: exactly 1 entry");

    // Store 5 blocks, each updating the same slot to a new value
    let mut parent = B256::ZERO;
    for block_num in 1u64..=5 {
        let hash = B256::repeat_byte(block_num as u8);
        let mut post_state = HashedPostState::default();
        let mut storage = HashedStorage::default();
        storage.storage.insert(slot, U256::from(block_num * 1000));
        post_state.storages.insert(addr, storage);

        let diff = BlockStateDiff {
            sorted_trie_updates: TrieUpdates::default().into_sorted(),
            sorted_post_state: post_state.into_sorted(),
        };
        store_block(&db, make_block_ref(block_num, hash, parent), diff);
        parent = hash;

        // After each block: still exactly 1 entry
        assert_eq!(
            count_hashed_storage_entries(&db, addr),
            1,
            "block {block_num}: must still be exactly 1 entry, no duplicates"
        );
    }

    // Verify final value is correct
    let slots = collect_hashed_storage_slots(&db, addr);
    assert_eq!(slots.len(), 1);
    assert_eq!(slots[0], (slot, U256::from(5000u64)));
}

/// Regression: multiple slots for the same address should each appear exactly
/// once after updates across blocks.
#[test]
fn hashed_storages_no_duplicates_multiple_slots() {
    let db = setup_db();

    let addr = B256::from([0xCC; 32]);
    let slot_a = B256::from([0x01; 32]);
    let slot_b = B256::from([0x02; 32]);
    let slot_c = B256::from([0x03; 32]);

    // Initialize with 2 slots
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider
            .store_hashed_storages(
                addr,
                vec![(slot_a, U256::from(10u64)), (slot_b, U256::from(20u64))],
            )
            .expect("seed");
        provider.set_initial_state_anchor(BlockNumHash::new(0, B256::ZERO)).expect("anchor");
        provider.commit_initial_state().expect("commit init");
        OpProofsInitProvider::commit(provider).expect("commit");
    }

    assert_eq!(count_hashed_storage_entries(&db, addr), 2, "initial: 2 entries");

    // Block 1: update slot_a, add slot_c
    {
        let mut post_state = HashedPostState::default();
        let mut storage = HashedStorage::default();
        storage.storage.insert(slot_a, U256::from(11u64));
        storage.storage.insert(slot_c, U256::from(30u64));
        post_state.storages.insert(addr, storage);

        let diff = BlockStateDiff {
            sorted_trie_updates: TrieUpdates::default().into_sorted(),
            sorted_post_state: post_state.into_sorted(),
        };
        store_block(&db, make_block_ref(1, B256::repeat_byte(0x01), B256::ZERO), diff);
    }

    // Should be 3 entries: slot_a (updated), slot_b (untouched), slot_c (new)
    assert_eq!(count_hashed_storage_entries(&db, addr), 3, "block 1: exactly 3 entries");

    // Block 2: update all 3 slots
    {
        let mut post_state = HashedPostState::default();
        let mut storage = HashedStorage::default();
        storage.storage.insert(slot_a, U256::from(12u64));
        storage.storage.insert(slot_b, U256::from(22u64));
        storage.storage.insert(slot_c, U256::from(32u64));
        post_state.storages.insert(addr, storage);

        let diff = BlockStateDiff {
            sorted_trie_updates: TrieUpdates::default().into_sorted(),
            sorted_post_state: post_state.into_sorted(),
        };
        store_block(&db, make_block_ref(2, B256::repeat_byte(0x02), B256::repeat_byte(0x01)), diff);
    }

    // Still 3 entries — no duplicates
    assert_eq!(count_hashed_storage_entries(&db, addr), 3, "block 2: exactly 3, no dupes");

    // Block 3: delete slot_b (set to zero)
    {
        let mut post_state = HashedPostState::default();
        let mut storage = HashedStorage::default();
        storage.storage.insert(slot_b, U256::ZERO);
        post_state.storages.insert(addr, storage);

        let diff = BlockStateDiff {
            sorted_trie_updates: TrieUpdates::default().into_sorted(),
            sorted_post_state: post_state.into_sorted(),
        };
        store_block(&db, make_block_ref(3, B256::repeat_byte(0x03), B256::repeat_byte(0x02)), diff);
    }

    // 2 entries: slot_a and slot_c remain, slot_b deleted
    let slots = collect_hashed_storage_slots(&db, addr);
    assert_eq!(slots.len(), 2, "block 3: slot_b deleted, 2 remain");
    assert!(slots.iter().all(|(k, _)| *k != slot_b), "slot_b should be gone");
}

/// Regression: wipe followed by re-add in same block must leave exactly the
/// new slots, no ghosts from pre-wipe state.
#[test]
fn hashed_storages_wipe_then_readd_no_duplicates() {
    let db = setup_db();

    let addr = B256::from([0xEE; 32]);
    let old_slot = B256::from([0x01; 32]);
    let new_slot = B256::from([0x02; 32]);

    // Initialize with old_slot
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.store_hashed_storages(addr, vec![(old_slot, U256::from(999u64))]).expect("seed");
        provider.set_initial_state_anchor(BlockNumHash::new(0, B256::ZERO)).expect("anchor");
        provider.commit_initial_state().expect("commit init");
        OpProofsInitProvider::commit(provider).expect("commit");
    }

    assert_eq!(count_hashed_storage_entries(&db, addr), 1);

    // Block 1: wipe + write new_slot
    {
        let mut post_state = HashedPostState::default();
        let mut storage = HashedStorage::new(true); // wiped = true
        storage.storage.insert(new_slot, U256::from(42u64));
        post_state.storages.insert(addr, storage);

        let diff = BlockStateDiff {
            sorted_trie_updates: TrieUpdates::default().into_sorted(),
            sorted_post_state: post_state.into_sorted(),
        };
        store_block(&db, make_block_ref(1, B256::repeat_byte(0x01), B256::ZERO), diff);
    }

    // Exactly 1 entry: only new_slot, old_slot wiped
    let slots = collect_hashed_storage_slots(&db, addr);
    assert_eq!(slots.len(), 1, "after wipe+add: exactly 1 entry");
    assert_eq!(slots[0], (new_slot, U256::from(42u64)));

    // Block 2: wipe + re-add the same new_slot with different value
    {
        let mut post_state = HashedPostState::default();
        let mut storage = HashedStorage::new(true);
        storage.storage.insert(new_slot, U256::from(84u64));
        post_state.storages.insert(addr, storage);

        let diff = BlockStateDiff {
            sorted_trie_updates: TrieUpdates::default().into_sorted(),
            sorted_post_state: post_state.into_sorted(),
        };
        store_block(&db, make_block_ref(2, B256::repeat_byte(0x02), B256::repeat_byte(0x01)), diff);
    }

    // Still exactly 1 entry
    let slots = collect_hashed_storage_slots(&db, addr);
    assert_eq!(slots.len(), 1, "after second wipe+add: exactly 1 entry");
    assert_eq!(slots[0], (new_slot, U256::from(84u64)));
}

/// Regression: batch store (`store_trie_updates_batch`) updating the same slot
/// across multiple blocks in one batch must not leak duplicates.
#[test]
fn hashed_storages_batch_no_duplicates() {
    let db = setup_db();

    let addr = B256::from([0xBB; 32]);
    let slot = B256::from([0xAA; 32]);

    // Initialize
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.store_hashed_storages(addr, vec![(slot, U256::from(1u64))]).expect("seed");
        provider.set_initial_state_anchor(BlockNumHash::new(0, B256::ZERO)).expect("anchor");
        provider.commit_initial_state().expect("commit init");
        OpProofsInitProvider::commit(provider).expect("commit");
    }

    // Build 3 blocks in a batch, each updating the same slot
    let blocks: Vec<(BlockWithParent, BlockStateDiff)> = (1u64..=3)
        .map(|n| {
            let mut post_state = HashedPostState::default();
            let mut storage = HashedStorage::default();
            storage.storage.insert(slot, U256::from(n * 100));
            post_state.storages.insert(addr, storage);

            let diff = BlockStateDiff {
                sorted_trie_updates: TrieUpdates::default().into_sorted(),
                sorted_post_state: post_state.into_sorted(),
            };
            let parent = if n == 1 { B256::ZERO } else { B256::repeat_byte((n - 1) as u8) };
            (make_block_ref(n, B256::repeat_byte(n as u8), parent), diff)
        })
        .collect();

    // Store as batch
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.store_trie_updates_batch(blocks).expect("batch store");
        OpProofsProviderRw::commit(provider).expect("commit");
    }

    // Exactly 1 entry with final value
    let slots = collect_hashed_storage_slots(&db, addr);
    assert_eq!(slots.len(), 1, "batch: exactly 1 entry, no duplicates");
    assert_eq!(slots[0], (slot, U256::from(300u64)));
}

// ========================== Sharded-key sort order regression ==========================
//
// End-to-end check that round-tripping `AccountTrieShardedKey` / `StorageTrieShardedKey`
// through a real MDBX env preserves `Nibbles`' lex-by-nibble order. Under the old
// length-prefixed encoding, the length byte at position 0 dominated MDBX's byte-wise sort
// and `[0x05]` walked before `[0x01, 0x05]` even though logically `[0x01, ...]` must
// come first.

#[test]
fn account_trie_history_cursor_walk_returns_lex_by_nibble_order() {
    let db = setup_db();

    let short = StoredNibbles::from(Nibbles::from_nibbles_unchecked([0x05]));
    let long = StoredNibbles::from(Nibbles::from_nibbles_unchecked([0x01, 0x05]));

    // Insert in the order that *would* have been wrong under the old encoding.
    {
        let wtx = db.tx_mut().expect("rw");
        let mut cur = wtx.cursor_write::<V2AccountsTrieHistory>().expect("cursor");
        cur.upsert(
            AccountTrieShardedKey::new(short.clone(), u64::MAX),
            &BlockNumberList::new_pre_sorted([0]),
        )
        .expect("upsert short");
        cur.upsert(
            AccountTrieShardedKey::new(long.clone(), u64::MAX),
            &BlockNumberList::new_pre_sorted([0]),
        )
        .expect("upsert long");
        wtx.commit().expect("commit");
    }

    // Walk the table; MDBX returns entries in encoded-byte order.
    let tx = db.tx().expect("ro");
    let mut cur = tx.cursor_read::<V2AccountsTrieHistory>().expect("cursor");
    let mut walked = Vec::new();
    let mut entry = cur.first().expect("first");
    while let Some((k, _)) = entry {
        walked.push(k.key);
        entry = cur.next().expect("next");
    }

    assert_eq!(
        walked,
        vec![long, short],
        "[0x01, 0x05] must precede [0x05]: nibble 0x01 < 0x05 dominates lex order",
    );
}

#[test]
fn storage_trie_history_cursor_walk_returns_lex_by_nibble_order() {
    let db = setup_db();

    let addr = B256::repeat_byte(0x11);
    let short = StoredNibbles::from(Nibbles::from_nibbles_unchecked([0x05]));
    let long = StoredNibbles::from(Nibbles::from_nibbles_unchecked([0x01, 0x05]));

    {
        let wtx = db.tx_mut().expect("rw");
        let mut cur = wtx.cursor_write::<V2StoragesTrieHistory>().expect("cursor");
        cur.upsert(
            StorageTrieShardedKey::new(addr, short.clone(), u64::MAX),
            &BlockNumberList::new_pre_sorted([0]),
        )
        .expect("upsert short");
        cur.upsert(
            StorageTrieShardedKey::new(addr, long.clone(), u64::MAX),
            &BlockNumberList::new_pre_sorted([0]),
        )
        .expect("upsert long");
        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro");
    let mut cur = tx.cursor_read::<V2StoragesTrieHistory>().expect("cursor");
    let mut walked = Vec::new();
    let mut entry = cur.first().expect("first");
    while let Some((k, _)) = entry {
        walked.push(k.key);
        entry = cur.next().expect("next");
    }

    assert_eq!(walked, vec![long, short], "within an address, [0x01, 0x05] must precede [0x05]",);
}

// ========================== Backfill (prepend_block) tests ==========================

#[test]
fn prepend_block_basic_advances_earliest_and_writes_changeset() {
    let db = setup_db();

    let addr = B256::from([0xDD; 32]);
    let block5_hash = B256::repeat_byte(0x05);
    let block4_hash = B256::repeat_byte(0x04);
    let before_account = Account { nonce: 10, ..Default::default() };

    // Init the proof window at (5, block5_hash) — earliest == latest == 5.
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.set_initial_state_anchor(BlockNumHash::new(5, block5_hash)).expect("anchor");
        provider.commit_initial_state().expect("commit init");
        OpProofsInitProvider::commit(provider).expect("commit");
    }

    // Prepend block 5 with addr's before-account = state at end of block 4.
    let counts = {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        let mut post_state = HashedPostState::default();
        post_state.accounts.insert(addr, Some(before_account));
        let diff = BlockStateDiff {
            sorted_trie_updates: TrieUpdates::default().into_sorted(),
            sorted_post_state: post_state.into_sorted(),
        };
        let block_ref = make_block_ref(5, block5_hash, block4_hash);
        let counts = provider.prepend_block(block_ref, diff).expect("prepend");
        OpProofsBackfillProvider::commit(provider).expect("commit");
        counts
    };
    assert_eq!(counts.hashed_accounts_written_total, 1);
    assert_eq!(counts.hashed_storages_written_total, 0);
    assert_eq!(counts.account_trie_updates_written_total, 0);
    assert_eq!(counts.storage_trie_updates_written_total, 0);

    // earliest advanced to (4, block4_hash).
    {
        let provider = MdbxProofsProviderV2::new(db.tx().expect("ro"));
        assert_eq!(provider.get_earliest_block().expect("get"), NumHash::new(4, block4_hash));
    }

    // Changeset entry for block 5 carries the supplied before-value.
    {
        let tx = db.tx().expect("ro");
        let mut cur = tx.cursor_dup_read::<V2HashedAccountChangeSets>().expect("cursor");
        let entry = cur.seek_by_key_subkey(5u64, addr).expect("seek").expect("exists");
        assert_eq!(entry.hashed_address, addr);
        assert_eq!(entry.info.unwrap().nonce, 10);
    }

    // History bitmap created with sentinel shard holding [5].
    {
        let tx = db.tx().expect("ro");
        let mut cur = tx.cursor_read::<V2HashedAccountsHistory>().expect("cursor");
        let shard_key = HashedAccountShardedKey::new(addr, u64::MAX);
        let (_, bitmap) = cur.seek_exact(shard_key).expect("seek").expect("exists");
        assert_eq!(bitmap.iter().collect::<Vec<_>>(), vec![5]);
    }
}

#[test]
fn prepend_block_hash_mismatch_rejects() {
    let db = setup_db();

    let block5_hash = B256::repeat_byte(0x05);
    let wrong_hash = B256::repeat_byte(0xFF);

    // Init the proof window at (5, block5_hash).
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.set_initial_state_anchor(BlockNumHash::new(5, block5_hash)).expect("anchor");
        provider.commit_initial_state().expect("commit init");
        OpProofsInitProvider::commit(provider).expect("commit");
    }

    // Attempt to prepend with a non-matching hash → PrependOutOfOrder.
    let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
    let block_ref = make_block_ref(5, wrong_hash, B256::repeat_byte(0x04));
    let res = provider.prepend_block(block_ref, BlockStateDiff::default());
    assert!(
        matches!(res, Err(OpProofsStorageError::PrependOutOfOrder { .. })),
        "expected PrependOutOfOrder, got {res:?}"
    );
}

#[test]
fn prepend_block_idempotent_when_changeset_exists() {
    let db = setup_db();

    let addr = B256::from([0xEE; 32]);
    let block1_hash = B256::repeat_byte(0x01);

    // Init the proof window at (1, block1_hash) — earliest == latest == 1.
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.store_hashed_accounts(vec![(addr, Some(Account::default()))]).expect("seed");
        provider.set_initial_state_anchor(BlockNumHash::new(1, block1_hash)).expect("anchor");
        provider.commit_initial_state().expect("commit init");
        OpProofsInitProvider::commit(provider).expect("commit");
    }

    // Plant a block-1 changeset entry directly so the prepend idempotency
    // guard fires (would normally have been written by a prior forward
    // `store_trie_updates(block 1)`, but doing that would require init at
    // block 0 plus an earliest-rewind helper).
    {
        let tx = db.tx_mut().expect("rw");
        let mut cur = tx.cursor_dup_write::<V2HashedAccountChangeSets>().expect("cursor");
        cur.upsert(1u64, &HashedAccountBeforeTx::new(addr, Some(Account::default())))
            .expect("plant changeset");
        tx.commit().expect("commit");
    }

    // Attempt to prepend block 1 with a would-be value (nonce 42). The guard
    // sees the planted changeset and short-circuits.
    let counts = {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        let block_ref = make_block_ref(1, block1_hash, B256::ZERO);
        let counts = provider.prepend_block(block_ref, make_nonce_diff(addr, 42)).expect("prepend");
        OpProofsBackfillProvider::commit(provider).expect("commit");
        counts
    };

    // Idempotency: zero write counts AND earliest unchanged.
    assert_eq!(counts, WriteCounts::default());
    {
        let provider = MdbxProofsProviderV2::new(db.tx().expect("ro"));
        assert_eq!(provider.get_earliest_block().expect("get"), NumHash::new(1, block1_hash));
    }

    // Changeset retains the planted "before" value (nonce: 0), not the
    // would-be-prepended value (nonce: 42).
    let tx = db.tx().expect("ro");
    let mut cur = tx.cursor_dup_read::<V2HashedAccountChangeSets>().expect("cursor");
    let entry = cur.seek_by_key_subkey(1u64, addr).expect("seek").expect("exists");
    assert_eq!(entry.info.unwrap_or_default().nonce, 0);
}

#[test]
fn prepend_block_descending_chain_accumulates_history() {
    let db = setup_db();

    let addr = B256::from([0xCD; 32]);
    let hashes = [
        B256::repeat_byte(0x00),
        B256::repeat_byte(0x01),
        B256::repeat_byte(0x02),
        B256::repeat_byte(0x03),
    ];

    // Init the proof window at (3, hash_3). Backfill blocks 3, 2, 1 descending.
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.set_initial_state_anchor(BlockNumHash::new(3, hashes[3])).expect("anchor");
        provider.commit_initial_state().expect("commit init");
        OpProofsInitProvider::commit(provider).expect("commit");
    }

    for block_num in (1u64..=3).rev() {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        let block_ref =
            make_block_ref(block_num, hashes[block_num as usize], hashes[(block_num - 1) as usize]);
        provider.prepend_block(block_ref, make_nonce_diff(addr, block_num)).expect("prepend");
        OpProofsBackfillProvider::commit(provider).expect("commit");
    }

    // earliest now at (0, hash_0).
    {
        let provider = MdbxProofsProviderV2::new(db.tx().expect("ro"));
        assert_eq!(provider.get_earliest_block().expect("get"), NumHash::new(0, hashes[0]));
    }

    // History bitmap accumulated all three prepended blocks in ascending order.
    let tx = db.tx().expect("ro");
    let mut cur = tx.cursor_read::<V2HashedAccountsHistory>().expect("cursor");
    let shard_key = HashedAccountShardedKey::new(addr, u64::MAX);
    let (_, bitmap) = cur.seek_exact(shard_key).expect("seek").expect("exists");
    assert_eq!(bitmap.iter().collect::<Vec<_>>(), vec![1, 2, 3]);
}

/// End-to-end backfill of `NUM_OF_INDICES_IN_SHARD + 1` blocks through the
/// real `prepend_block` machinery, verifying the history-shard overflow path.
///
/// The first 2000 prepends fill the sentinel shard exactly to capacity; the
/// 2001st must spawn a fresh singleton at the new block's key and leave the
/// full sentinel untouched. Each block's diff touches both an account and a
/// storage slot so both history bitmaps grow together.
#[test]
fn prepend_block_overflows_history_shard_after_filling_sentinel() {
    use super::NUM_OF_INDICES_IN_SHARD;

    let db = setup_db();
    let addr = B256::from([0xCD; 32]);
    let slot = B256::from([0x55; 32]);

    // Deterministic block hashes: encode block number as a 32-byte big-endian value.
    let hash = |n: u64| B256::from(U256::from(n));

    const ANCHOR_BLOCK: u64 = 4000;
    // 2000 prepends move earliest from 4000 down to 2000 (prepending 4000..=2001).
    let last_filled_block = ANCHOR_BLOCK + 1 - NUM_OF_INDICES_IN_SHARD as u64; // 2001
    let overflow_block = last_filled_block - 1; // 2000

    // Init proof window at (4000, hash(4000)). `prepend_block` doesn't read
    // current-state tables, so seeding `addr`/`slot` there isn't necessary.
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider
            .set_initial_state_anchor(BlockNumHash::new(ANCHOR_BLOCK, hash(ANCHOR_BLOCK)))
            .expect("anchor");
        provider.commit_initial_state().expect("commit init");
        OpProofsInitProvider::commit(provider).expect("commit");
    }

    // Backfill descending: each block writes a diff touching `addr` and
    // `(addr, slot)`, growing both history bitmaps in lock-step.
    for block_num in (last_filled_block..=ANCHOR_BLOCK).rev() {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        let block_ref = make_block_ref(block_num, hash(block_num), hash(block_num - 1));
        let diff = make_account_and_storage_diff(addr, slot, block_num);
        provider.prepend_block(block_ref, diff).expect("prepend");
        OpProofsBackfillProvider::commit(provider).expect("commit");
    }

    // After NUM_OF_INDICES_IN_SHARD prepends both history sentinels hold the
    // full range in a single row each — no overflow yet.
    let expected_filled: Vec<u64> = (last_filled_block..=ANCHOR_BLOCK).collect();
    {
        let tx = db.tx().expect("ro");
        let mut acc_cur = tx.cursor_read::<V2HashedAccountsHistory>().expect("acc cur");
        let (key, list) = acc_cur
            .seek(HashedAccountShardedKey::new(addr, 0))
            .expect("seek acc")
            .expect("acc sentinel exists");
        assert_eq!(key, HashedAccountShardedKey::new(addr, u64::MAX));
        assert_eq!(list.iter().collect::<Vec<_>>(), expected_filled);
        assert!(acc_cur.next().expect("next").is_none(), "single acc shard before overflow");

        let mut stor_cur = tx.cursor_read::<V2HashedStoragesHistory>().expect("stor cur");
        let stor_seek =
            HashedStorageShardedKey { hashed_address: addr, sharded_key: ShardedKey::new(slot, 0) };
        let (key, list) = stor_cur.seek(stor_seek).expect("seek stor").expect("stor sentinel");
        assert_eq!(
            key,
            HashedStorageShardedKey {
                hashed_address: addr,
                sharded_key: ShardedKey::new(slot, u64::MAX),
            }
        );
        assert_eq!(list.iter().collect::<Vec<_>>(), expected_filled);
        assert!(stor_cur.next().expect("next").is_none(), "single stor shard before overflow");
    }

    // One more prepend pushes past `NUM_OF_INDICES_IN_SHARD` — the sentinel is
    // full, so this must spawn a fresh singleton rather than rewrite it.
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        let block_ref =
            make_block_ref(overflow_block, hash(overflow_block), hash(overflow_block - 1));
        let diff = make_account_and_storage_diff(addr, slot, overflow_block);
        provider.prepend_block(block_ref, diff).expect("prepend overflow");
        OpProofsBackfillProvider::commit(provider).expect("commit");
    }

    // Two shards in each history table now: singleton at the new block, plus
    // the untouched sentinel still holding `NUM_OF_INDICES_IN_SHARD` entries.
    {
        let tx = db.tx().expect("ro");
        let mut acc_cur = tx.cursor_read::<V2HashedAccountsHistory>().expect("acc cur");
        let (singleton_key, singleton) = acc_cur
            .seek(HashedAccountShardedKey::new(addr, 0))
            .expect("seek acc lo")
            .expect("acc singleton exists");
        assert_eq!(singleton_key, HashedAccountShardedKey::new(addr, overflow_block));
        assert_eq!(singleton.iter().collect::<Vec<_>>(), vec![overflow_block]);

        let (sentinel_key, sentinel) =
            acc_cur.next().expect("acc next").expect("acc sentinel exists");
        assert_eq!(sentinel_key, HashedAccountShardedKey::new(addr, u64::MAX));
        assert_eq!(sentinel.iter().collect::<Vec<_>>(), expected_filled);

        assert!(acc_cur.next().expect("acc next2").is_none(), "exactly two acc shards");

        let mut stor_cur = tx.cursor_read::<V2HashedStoragesHistory>().expect("stor cur");
        let stor_seek =
            HashedStorageShardedKey { hashed_address: addr, sharded_key: ShardedKey::new(slot, 0) };
        let (singleton_key, singleton) =
            stor_cur.seek(stor_seek).expect("seek stor lo").expect("stor singleton exists");
        assert_eq!(
            singleton_key,
            HashedStorageShardedKey {
                hashed_address: addr,
                sharded_key: ShardedKey::new(slot, overflow_block),
            }
        );
        assert_eq!(singleton.iter().collect::<Vec<_>>(), vec![overflow_block]);

        let (sentinel_key, sentinel) =
            stor_cur.next().expect("stor next").expect("stor sentinel exists");
        assert_eq!(
            sentinel_key,
            HashedStorageShardedKey {
                hashed_address: addr,
                sharded_key: ShardedKey::new(slot, u64::MAX),
            }
        );
        assert_eq!(sentinel.iter().collect::<Vec<_>>(), expected_filled);

        assert!(stor_cur.next().expect("stor next2").is_none(), "exactly two stor shards");
    }
}
