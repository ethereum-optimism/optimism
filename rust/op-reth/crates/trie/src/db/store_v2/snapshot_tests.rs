//! Tests for the V2 snapshot read/write/init providers.

use super::MdbxProofsProviderV2;
use crate::{
    BlockStateDiff, OpProofsStorageError,
    api::{
        OpProofsBackfillProvider, OpProofsSnapshotInitProvider, OpProofsSnapshotProviderRO,
        SnapshotInitStatus,
    },
    db::{
        SnapshotMeta, SnapshotMetaKey, SnapshotStatus, StorageTrieKey,
        models::{
            self, V2AccountsTrieSnapshot, V2HashedAccountsSnapshot, V2HashedStoragesSnapshot,
            V2SnapshotMeta, V2StoragesTrieSnapshot,
        },
    },
};
use alloy_eips::BlockNumHash;
use alloy_primitives::{B256, U256, map::B256Map};
use reth_db::{
    Database, DatabaseEnv,
    cursor::{DbCursorRO, DbDupCursorRO},
    mdbx::{DatabaseArguments, init_db_for},
    transaction::DbTx,
};
use reth_primitives_traits::{Account, StorageEntry};
use reth_trie::{
    BranchNodeCompact, HashedPostStateSorted, HashedStorageSorted, Nibbles, StoredNibbles,
    StoredNibblesSubKey,
    updates::{StorageTrieUpdates, TrieUpdates},
};
use tempfile::TempDir;

// ---------- helpers ----------

fn setup_db() -> DatabaseEnv {
    let tmp = TempDir::new().expect("create tmpdir");
    init_db_for::<_, models::Tables>(tmp, DatabaseArguments::default()).expect("init db")
}

fn anchor(n: u64, byte: u8) -> BlockNumHash {
    BlockNumHash::new(n, B256::repeat_byte(byte))
}

fn sample_node(tag: u8) -> BranchNodeCompact {
    BranchNodeCompact::new(0b1, 0, 0, vec![], Some(B256::repeat_byte(tag)))
}

/// Build a Ready snapshot at `anchor` with no tables populated.
fn prepare_ready_snapshot(db: &DatabaseEnv, anchor: BlockNumHash) {
    let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
    provider.set_snapshot_init_anchor(anchor).expect("set anchor");
    provider.commit_snapshot().expect("flip to Ready");
    OpProofsSnapshotInitProvider::commit(provider).expect("commit tx");
}

// ========================== snapshot_init.rs tests ==========================

#[test]
fn init_set_snapshot_init_anchor_writes_building_meta() {
    let db = setup_db();
    let target = anchor(7, 0x07);

    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.set_snapshot_init_anchor(target).expect("set anchor");
        OpProofsSnapshotInitProvider::commit(provider).expect("commit");
    }

    let tx = db.tx().expect("ro");
    let mut cur = tx.cursor_read::<V2SnapshotMeta>().expect("cursor");
    let (_, meta) = cur.seek_exact(SnapshotMetaKey::Singleton).expect("seek").expect("exists");
    assert_eq!(meta, SnapshotMeta::new(target, SnapshotStatus::Building));
}

#[test]
fn init_set_snapshot_init_anchor_errors_when_meta_already_exists() {
    let db = setup_db();
    let target = anchor(7, 0x07);
    let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
    provider.set_snapshot_init_anchor(target).expect("first set");

    let err = provider.set_snapshot_init_anchor(target).unwrap_err();
    // mdbx's `insert` raises a key-already-exists DB error.
    assert!(matches!(err, OpProofsStorageError::DatabaseError(_)), "got {err:?}");
}

#[test]
fn init_snapshot_init_anchor_maps_lifecycle_statuses() {
    let db = setup_db();

    // 1. No meta row → NotStarted, block = None.
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        let a = provider.snapshot_init_anchor().expect("anchor");
        assert_eq!(a.status, SnapshotInitStatus::NotStarted);
        assert!(a.block.is_none());
    }

    // 2. Building → InProgress.
    let target = anchor(5, 0x05);
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.set_snapshot_init_anchor(target).expect("set anchor");
        OpProofsSnapshotInitProvider::commit(provider).expect("commit");
    }
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        let a = provider.snapshot_init_anchor().expect("anchor");
        assert_eq!(a.status, SnapshotInitStatus::InProgress);
        assert_eq!(a.block, Some(target));
    }

    // 3. After commit_snapshot → Completed.
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.commit_snapshot().expect("flip ready");
        OpProofsSnapshotInitProvider::commit(provider).expect("commit");
    }
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        let a = provider.snapshot_init_anchor().expect("anchor");
        assert_eq!(a.status, SnapshotInitStatus::Completed);
        assert_eq!(a.block, Some(target));
    }
}

#[test]
fn init_snapshot_init_anchor_returns_resume_keys() {
    let db = setup_db();
    let target = anchor(3, 0x03);
    let addr = B256::repeat_byte(0xCC);
    let acc_path = Nibbles::from_nibbles_unchecked([0x01, 0x02]);
    let stor_path = Nibbles::from_nibbles_unchecked([0x0A, 0x0B]);

    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.set_snapshot_init_anchor(target).expect("set anchor");
        provider
            .store_account_trie_snapshot_branches(vec![(
                StoredNibbles(acc_path),
                sample_node(0x01),
            )])
            .expect("write account");
        provider
            .store_storage_trie_snapshot_branches(addr, vec![(stor_path, Some(sample_node(0x02)))])
            .expect("write storage");
        OpProofsSnapshotInitProvider::commit(provider).expect("commit");
    }

    let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
    let a = provider.snapshot_init_anchor().expect("anchor");
    assert_eq!(a.last_account_trie_key, Some(StoredNibbles(acc_path)));
    assert_eq!(a.last_storage_trie_key, Some(StorageTrieKey::new(addr, StoredNibbles(stor_path))));
}

#[test]
fn init_commit_snapshot_errors_when_no_meta() {
    let db = setup_db();
    let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
    let err = provider.commit_snapshot().unwrap_err();
    assert!(matches!(err, OpProofsStorageError::SnapshotNotInitialized), "got {err:?}");
}

#[test]
fn init_commit_snapshot_errors_when_not_building() {
    let db = setup_db();
    let target = anchor(9, 0x09);
    prepare_ready_snapshot(&db, target);

    let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
    let err = provider.commit_snapshot().unwrap_err();
    match err {
        OpProofsStorageError::SnapshotCommitInvalidStatus { status } => {
            assert_eq!(status, SnapshotStatus::Ready);
        }
        other => panic!("unexpected err {other:?}"),
    }
}

#[test]
fn init_store_storage_trie_snapshot_branches_skips_none() {
    let db = setup_db();
    let addr = B256::repeat_byte(0x77);
    let kept = Nibbles::from_nibbles_unchecked([0x01]);
    let dropped = Nibbles::from_nibbles_unchecked([0x02]);

    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider
            .store_storage_trie_snapshot_branches(
                addr,
                vec![(kept, Some(sample_node(0x11))), (dropped, None)],
            )
            .expect("write");
        OpProofsSnapshotInitProvider::commit(provider).expect("commit");
    }

    let tx = db.tx().expect("ro");
    let mut cur = tx.cursor_dup_read::<V2StoragesTrieSnapshot>().expect("cursor");
    let kept_subkey = StoredNibblesSubKey(kept);
    let dropped_subkey = StoredNibblesSubKey(dropped);

    let entry =
        cur.seek_by_key_subkey(addr, kept_subkey.clone()).expect("seek kept").expect("kept exists");
    assert_eq!(entry.nibbles, kept_subkey);

    // The None entry must not have been inserted under any subkey.
    let absent = cur.seek_by_key_subkey(addr, dropped_subkey.clone()).expect("seek dropped");
    assert!(
        absent.is_none_or(|e| e.nibbles != dropped_subkey),
        "None payload should not have produced a row"
    );
}

// ========================== snapshot_read.rs tests ==========================

#[test]
fn read_snapshot_anchor_returns_block_when_ready() {
    let db = setup_db();
    let target = anchor(11, 0x11);
    prepare_ready_snapshot(&db, target);

    let provider = MdbxProofsProviderV2::new(db.tx().expect("ro"));
    let got = provider.snapshot_anchor().expect("anchor");
    assert_eq!(got, target);
}

#[test]
fn read_snapshot_anchor_errors_when_building() {
    let db = setup_db();
    let target = anchor(11, 0x11);
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.set_snapshot_init_anchor(target).expect("set anchor");
        OpProofsSnapshotInitProvider::commit(provider).expect("commit");
    }

    let provider = MdbxProofsProviderV2::new(db.tx().expect("ro"));
    let err = provider.snapshot_anchor().unwrap_err();
    match err {
        OpProofsStorageError::SnapshotNotReady { status } => {
            assert_eq!(status, SnapshotInitStatus::InProgress);
        }
        other => panic!("unexpected err {other:?}"),
    }
}

#[test]
fn read_snapshot_anchor_errors_when_not_started() {
    let db = setup_db();
    let provider = MdbxProofsProviderV2::new(db.tx().expect("ro"));
    let err = provider.snapshot_anchor().unwrap_err();
    match err {
        OpProofsStorageError::SnapshotNotReady { status } => {
            assert_eq!(status, SnapshotInitStatus::NotStarted);
        }
        other => panic!("unexpected err {other:?}"),
    }
}

// ===================== snapshot RW (in backfill.rs) tests =====================

#[test]
fn write_clear_snapshot_wipes_tables() {
    let db = setup_db();
    let target = anchor(2, 0x02);
    let addr = B256::repeat_byte(0xDD);
    let acc_path = Nibbles::from_nibbles_unchecked([0x01, 0x02]);
    let stor_path = Nibbles::from_nibbles_unchecked([0x0A, 0x0B]);

    // Seed a Building snapshot with rows in both tables.
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.set_snapshot_init_anchor(target).expect("set anchor");
        provider
            .store_account_trie_snapshot_branches(vec![(
                StoredNibbles(acc_path),
                sample_node(0x01),
            )])
            .expect("acc");
        provider
            .store_storage_trie_snapshot_branches(addr, vec![(stor_path, Some(sample_node(0x02)))])
            .expect("stor");
        OpProofsSnapshotInitProvider::commit(provider).expect("commit");
    }

    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.clear_snapshot().expect("clear");
        OpProofsBackfillProvider::commit(provider).expect("commit");
    }

    let tx = db.tx().expect("ro");
    let mut meta_cur = tx.cursor_read::<V2SnapshotMeta>().expect("meta cur");
    assert!(meta_cur.first().expect("first").is_none());
    let mut acc_cur = tx.cursor_read::<V2AccountsTrieSnapshot>().expect("acc cur");
    assert!(acc_cur.first().expect("first").is_none());
    let mut stor_cur = tx.cursor_dup_read::<V2StoragesTrieSnapshot>().expect("stor cur");
    assert!(stor_cur.first().expect("first").is_none());
}

#[test]
fn write_update_snapshot_applies_diff_and_advances_anchor() {
    let db = setup_db();
    let old_anchor = anchor(10, 0x10);
    let new_anchor = anchor(9, 0x09);
    let addr = B256::repeat_byte(0xEE);
    let acc_path = Nibbles::from_nibbles_unchecked([0x01]);
    let stor_path = Nibbles::from_nibbles_unchecked([0x0F]);

    prepare_ready_snapshot(&db, old_anchor);

    let acc_node = sample_node(0x33);
    let stor_node = sample_node(0x44);

    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        let mut updates = TrieUpdates::default();
        updates.account_nodes.insert(acc_path, acc_node.clone());
        let mut st = StorageTrieUpdates::default();
        st.storage_nodes.insert(stor_path, stor_node.clone());
        updates.storage_tries.insert(addr, st);
        let diff = BlockStateDiff {
            sorted_trie_updates: updates.into_sorted(),
            sorted_post_state: HashedPostStateSorted::default(),
        };

        let counts = provider.update_snapshot(new_anchor, &diff).expect("update");
        assert_eq!(counts.account_trie_updates_written_total, 1);
        assert_eq!(counts.storage_trie_updates_written_total, 1);
        assert_eq!(counts.hashed_accounts_written_total, 0);
        assert_eq!(counts.hashed_storages_written_total, 0);
        OpProofsBackfillProvider::commit(provider).expect("commit");
    }

    let tx = db.tx().expect("ro");
    // Anchor advanced; status stayed Ready.
    let mut meta_cur = tx.cursor_read::<V2SnapshotMeta>().expect("meta cur");
    let (_, meta) = meta_cur.seek_exact(SnapshotMetaKey::Singleton).expect("seek").expect("exists");
    assert_eq!(meta, SnapshotMeta::new(new_anchor, SnapshotStatus::Ready));

    // Diff applied: account row + storage row exist.
    let mut acc_cur = tx.cursor_read::<V2AccountsTrieSnapshot>().expect("acc cur");
    let (_, acc_val) =
        acc_cur.seek_exact(StoredNibbles(acc_path)).expect("acc seek").expect("acc exists");
    assert_eq!(acc_val, acc_node);

    let mut stor_cur = tx.cursor_dup_read::<V2StoragesTrieSnapshot>().expect("stor cur");
    let entry = stor_cur
        .seek_by_key_subkey(addr, StoredNibblesSubKey(stor_path))
        .expect("stor seek")
        .expect("stor exists");
    assert_eq!(entry.node, stor_node);
}

#[test]
fn write_update_snapshot_handles_removals() {
    let db = setup_db();
    let old_anchor = anchor(10, 0x10);
    let new_anchor = anchor(9, 0x09);
    let acc_path = Nibbles::from_nibbles_unchecked([0x01]);

    prepare_ready_snapshot(&db, old_anchor);

    // Seed an account row to be removed by the diff.
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider
            .store_account_trie_snapshot_branches(vec![(
                StoredNibbles(acc_path),
                sample_node(0x77),
            )])
            .expect("seed acc");
        OpProofsSnapshotInitProvider::commit(provider).expect("commit seed");
    }

    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        let mut updates = TrieUpdates::default();
        updates.removed_nodes.insert(acc_path);
        let diff = BlockStateDiff {
            sorted_trie_updates: updates.into_sorted(),
            sorted_post_state: HashedPostStateSorted::default(),
        };
        provider.update_snapshot(new_anchor, &diff).expect("update");
        OpProofsBackfillProvider::commit(provider).expect("commit");
    }

    let tx = db.tx().expect("ro");
    let mut acc_cur = tx.cursor_read::<V2AccountsTrieSnapshot>().expect("acc cur");
    assert!(
        acc_cur.seek_exact(StoredNibbles(acc_path)).expect("seek").is_none(),
        "removal should have deleted the row"
    );
}

#[test]
fn write_update_snapshot_removes_storage_node_and_keeps_sibling() {
    let db = setup_db();
    let old_anchor = anchor(10, 0x10);
    let new_anchor = anchor(9, 0x09);
    let addr = B256::repeat_byte(0xEE);
    let dropped_path = Nibbles::from_nibbles_unchecked([0x01]);
    let kept_path = Nibbles::from_nibbles_unchecked([0x0F]);

    prepare_ready_snapshot(&db, old_anchor);

    let dropped_node = sample_node(0x11);
    let kept_node = sample_node(0x22);

    // Seed two storage-trie nodes under the same address, both via the
    // init-path writer (mimics what `SnapshotInitJob` would have produced).
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider
            .store_storage_trie_snapshot_branches(
                addr,
                vec![(dropped_path, Some(dropped_node)), (kept_path, Some(kept_node.clone()))],
            )
            .expect("seed");
        OpProofsSnapshotInitProvider::commit(provider).expect("commit seed");
    }

    // Apply a diff that removes only `dropped_path` from `addr`'s storage trie.
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        let mut updates = TrieUpdates::default();
        let mut st = StorageTrieUpdates::default();
        st.removed_nodes.insert(dropped_path);
        updates.storage_tries.insert(addr, st);
        let diff = BlockStateDiff {
            sorted_trie_updates: updates.into_sorted(),
            sorted_post_state: HashedPostStateSorted::default(),
        };
        provider.update_snapshot(new_anchor, &diff).expect("update");
        OpProofsBackfillProvider::commit(provider).expect("commit");
    }

    let tx = db.tx().expect("ro");
    let mut stor_cur = tx.cursor_dup_read::<V2StoragesTrieSnapshot>().expect("stor cur");

    // Dropped row is gone.
    let dropped =
        stor_cur.seek_by_key_subkey(addr, StoredNibblesSubKey(dropped_path)).expect("seek dropped");
    assert!(
        dropped.is_none_or(|e| e.nibbles != StoredNibblesSubKey(dropped_path)),
        "dropped storage node should have been deleted",
    );

    let kept = stor_cur
        .seek_by_key_subkey(addr, StoredNibblesSubKey(kept_path))
        .expect("seek kept")
        .expect("kept row still exists");
    assert_eq!(kept.nibbles, StoredNibblesSubKey(kept_path));
    assert_eq!(kept.node, kept_node);
}

#[test]
fn write_update_snapshot_errors_when_building() {
    let db = setup_db();
    let target = anchor(10, 0x10);
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.set_snapshot_init_anchor(target).expect("set anchor");
        OpProofsSnapshotInitProvider::commit(provider).expect("commit");
    }

    let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
    let diff = BlockStateDiff {
        sorted_trie_updates: TrieUpdates::default().into_sorted(),
        sorted_post_state: HashedPostStateSorted::default(),
    };
    let err = provider.update_snapshot(anchor(9, 0x09), &diff).unwrap_err();
    match err {
        OpProofsStorageError::SnapshotUpdateNotReady { status } => {
            assert_eq!(status, SnapshotStatus::Building);
        }
        other => panic!("unexpected err {other:?}"),
    }
}

#[test]
fn write_update_snapshot_upserts_existing_account_leaf() {
    let db = setup_db();
    let old_anchor = anchor(10, 0x10);
    let new_anchor = anchor(9, 0x09);
    let addr = B256::repeat_byte(0xA1);
    let old_value = Account { nonce: 1, balance: U256::from(1u64), ..Default::default() };
    let new_value = Account { nonce: 2, balance: U256::from(2u64), ..Default::default() };

    prepare_ready_snapshot(&db, old_anchor);
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider.store_hashed_accounts_snapshot(vec![(addr, old_value)]).expect("seed");
        OpProofsSnapshotInitProvider::commit(provider).expect("commit seed");
    }

    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        let sorted_post_state =
            HashedPostStateSorted::new(vec![(addr, Some(new_value))], B256Map::default());
        let diff = BlockStateDiff {
            sorted_trie_updates: TrieUpdates::default().into_sorted(),
            sorted_post_state,
        };
        let counts = provider.update_snapshot(new_anchor, &diff).expect("update");
        assert_eq!(counts.hashed_accounts_written_total, 1);
        OpProofsBackfillProvider::commit(provider).expect("commit");
    }

    let tx = db.tx().expect("ro");
    let mut cur = tx.cursor_read::<V2HashedAccountsSnapshot>().expect("cur");
    let (_, got) = cur.seek_exact(addr).expect("seek").expect("row");
    assert_eq!(got, new_value, "upsert must overwrite the prior value");
}

#[test]
fn write_update_snapshot_deletes_destroyed_account_leaf() {
    let db = setup_db();
    let old_anchor = anchor(10, 0x10);
    let new_anchor = anchor(9, 0x09);
    let addr = B256::repeat_byte(0xA2);

    prepare_ready_snapshot(&db, old_anchor);
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider
            .store_hashed_accounts_snapshot(vec![(
                addr,
                Account { nonce: 0, balance: U256::from(5u64), ..Default::default() },
            )])
            .expect("seed");
        OpProofsSnapshotInitProvider::commit(provider).expect("commit seed");
    }

    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        let sorted_post_state = HashedPostStateSorted::new(vec![(addr, None)], B256Map::default());
        let diff = BlockStateDiff {
            sorted_trie_updates: TrieUpdates::default().into_sorted(),
            sorted_post_state,
        };
        let counts = provider.update_snapshot(new_anchor, &diff).expect("update");
        assert_eq!(counts.hashed_accounts_written_total, 1);
        OpProofsBackfillProvider::commit(provider).expect("commit");
    }

    let tx = db.tx().expect("ro");
    let mut cur = tx.cursor_read::<V2HashedAccountsSnapshot>().expect("cur");
    assert!(cur.seek_exact(addr).expect("seek").is_none(), "destroyed leaf must be gone");
}

#[test]
fn write_update_snapshot_wipes_all_storage_slots() {
    let db = setup_db();
    let old_anchor = anchor(10, 0x10);
    let new_anchor = anchor(9, 0x09);
    let addr = B256::repeat_byte(0xA3);
    let slot_1 = B256::repeat_byte(0x01);
    let slot_2 = B256::repeat_byte(0x02);

    prepare_ready_snapshot(&db, old_anchor);
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider
            .store_hashed_storages_snapshot(
                addr,
                vec![(slot_1, U256::from(11u64)), (slot_2, U256::from(22u64))],
            )
            .expect("seed");
        OpProofsSnapshotInitProvider::commit(provider).expect("commit seed");
    }

    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        let mut storages: B256Map<HashedStorageSorted> = B256Map::default();
        storages.insert(addr, HashedStorageSorted { storage_slots: vec![], wiped: true });
        let sorted_post_state = HashedPostStateSorted::new(Vec::new(), storages);
        let diff = BlockStateDiff {
            sorted_trie_updates: TrieUpdates::default().into_sorted(),
            sorted_post_state,
        };
        let counts = provider.update_snapshot(new_anchor, &diff).expect("update");
        assert_eq!(counts.hashed_storages_written_total, 1, "wipe counts once per address");
        OpProofsBackfillProvider::commit(provider).expect("commit");
    }

    let tx = db.tx().expect("ro");
    let mut cur = tx.cursor_dup_read::<V2HashedStoragesSnapshot>().expect("cur");
    assert!(
        cur.seek_by_key_subkey(addr, slot_1).expect("seek").is_none_or(|e| e.key != slot_1),
        "slot 1 must be wiped",
    );
    assert!(
        cur.seek_by_key_subkey(addr, slot_2).expect("seek").is_none_or(|e| e.key != slot_2),
        "slot 2 must be wiped (delete_current_duplicates drops every dup, not just the first)",
    );
}

#[test]
fn write_update_snapshot_wipes_then_adds_slots_in_same_block() {
    let db = setup_db();
    let old_anchor = anchor(10, 0x10);
    let new_anchor = anchor(9, 0x09);
    let addr = B256::repeat_byte(0xA6);
    let slot_old_1 = B256::repeat_byte(0x01);
    let slot_old_2 = B256::repeat_byte(0x02);
    // Sorted slot order is required by the per-slot loop's cursor seeks
    // (later iterations rely on monotonic positioning on the dup cursor).
    let slot_new_a = B256::repeat_byte(0x10);
    let slot_new_b = B256::repeat_byte(0x20);
    let new_value_a = U256::from(111u64);
    let new_value_b = U256::from(222u64);

    prepare_ready_snapshot(&db, old_anchor);
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider
            .store_hashed_storages_snapshot(
                addr,
                vec![(slot_old_1, U256::from(11u64)), (slot_old_2, U256::from(22u64))],
            )
            .expect("seed");
        OpProofsSnapshotInitProvider::commit(provider).expect("commit seed");
    }

    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        let mut storages: B256Map<HashedStorageSorted> = B256Map::default();
        storages.insert(
            addr,
            HashedStorageSorted {
                storage_slots: vec![(slot_new_a, new_value_a), (slot_new_b, new_value_b)],
                wiped: true,
            },
        );
        let sorted_post_state = HashedPostStateSorted::new(Vec::new(), storages);
        let diff = BlockStateDiff {
            sorted_trie_updates: TrieUpdates::default().into_sorted(),
            sorted_post_state,
        };
        let counts = provider.update_snapshot(new_anchor, &diff).expect("update");
        // The wipe counts once + one per new slot.
        assert_eq!(counts.hashed_storages_written_total, 3);
        OpProofsBackfillProvider::commit(provider).expect("commit");
    }

    let tx = db.tx().expect("ro");
    let mut cur = tx.cursor_dup_read::<V2HashedStoragesSnapshot>().expect("cur");

    // Old slots are gone (wipe phase).
    assert!(
        cur.seek_by_key_subkey(addr, slot_old_1).expect("seek").is_none_or(|e| e.key != slot_old_1),
        "old slot 1 must be wiped",
    );
    assert!(
        cur.seek_by_key_subkey(addr, slot_old_2).expect("seek").is_none_or(|e| e.key != slot_old_2),
        "old slot 2 must be wiped",
    );

    // New slots are present with the values from the diff (upsert phase).
    let got_a = cur
        .seek_by_key_subkey(addr, slot_new_a)
        .expect("seek")
        .expect("new slot A row exists after wipe");
    assert_eq!(got_a.key, slot_new_a);
    assert_eq!(got_a.value, new_value_a);

    let got_b = cur
        .seek_by_key_subkey(addr, slot_new_b)
        .expect("seek")
        .expect("new slot B row exists after wipe");
    assert_eq!(got_b.key, slot_new_b);
    assert_eq!(got_b.value, new_value_b);
}

#[test]
fn write_update_snapshot_deletes_zero_value_storage_slot() {
    let db = setup_db();
    let old_anchor = anchor(10, 0x10);
    let new_anchor = anchor(9, 0x09);
    let addr = B256::repeat_byte(0xA4);
    let slot = B256::repeat_byte(0x03);

    prepare_ready_snapshot(&db, old_anchor);
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider
            .store_hashed_storages_snapshot(addr, vec![(slot, U256::from(33u64))])
            .expect("seed");
        OpProofsSnapshotInitProvider::commit(provider).expect("commit seed");
    }

    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        let mut storages: B256Map<HashedStorageSorted> = B256Map::default();
        storages.insert(
            addr,
            HashedStorageSorted { storage_slots: vec![(slot, U256::ZERO)], wiped: false },
        );
        let sorted_post_state = HashedPostStateSorted::new(Vec::new(), storages);
        let diff = BlockStateDiff {
            sorted_trie_updates: TrieUpdates::default().into_sorted(),
            sorted_post_state,
        };
        let counts = provider.update_snapshot(new_anchor, &diff).expect("update");
        assert_eq!(counts.hashed_storages_written_total, 1);
        OpProofsBackfillProvider::commit(provider).expect("commit");
    }

    let tx = db.tx().expect("ro");
    let mut cur = tx.cursor_dup_read::<V2HashedStoragesSnapshot>().expect("cur");
    assert!(
        cur.seek_by_key_subkey(addr, slot)
            .expect("seek")
            .is_none_or(|e: StorageEntry| e.key != slot),
        "zero-value slot must be deleted and not re-inserted",
    );
}

#[test]
fn write_update_snapshot_deletes_storage_trie_when_is_deleted() {
    let db = setup_db();
    let old_anchor = anchor(10, 0x10);
    let new_anchor = anchor(9, 0x09);
    let addr = B256::repeat_byte(0xA5);
    let path = Nibbles::from_nibbles_unchecked([0x0F]);

    prepare_ready_snapshot(&db, old_anchor);
    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        provider
            .store_storage_trie_snapshot_branches(addr, vec![(path, Some(sample_node(0xEE)))])
            .expect("seed");
        OpProofsSnapshotInitProvider::commit(provider).expect("commit seed");
    }

    {
        let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
        let mut updates = TrieUpdates::default();
        let st = StorageTrieUpdates { is_deleted: true, ..Default::default() };
        updates.storage_tries.insert(addr, st);
        let diff = BlockStateDiff {
            sorted_trie_updates: updates.into_sorted(),
            sorted_post_state: HashedPostStateSorted::default(),
        };
        let counts = provider.update_snapshot(new_anchor, &diff).expect("update");
        assert_eq!(counts.storage_trie_updates_written_total, 1, "is_deleted counts once");
        OpProofsBackfillProvider::commit(provider).expect("commit");
    }

    let tx = db.tx().expect("ro");
    let mut cur = tx.cursor_dup_read::<V2StoragesTrieSnapshot>().expect("cur");
    assert!(
        cur.seek_by_key_subkey(addr, StoredNibblesSubKey(path))
            .expect("seek")
            .is_none_or(|e| e.nibbles != StoredNibblesSubKey(path)),
        "is_deleted must drop every storage-trie row under the address",
    );
}

#[test]
fn write_update_snapshot_errors_when_missing() {
    let db = setup_db();
    let provider = MdbxProofsProviderV2::new(db.tx_mut().expect("rw"));
    let diff = BlockStateDiff {
        sorted_trie_updates: TrieUpdates::default().into_sorted(),
        sorted_post_state: HashedPostStateSorted::default(),
    };
    let err = provider.update_snapshot(anchor(9, 0x09), &diff).unwrap_err();
    assert!(matches!(err, OpProofsStorageError::SnapshotNotInitialized), "got {err:?}");
}
