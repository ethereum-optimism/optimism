use super::*;
use crate::db::{
    models,
    models::{HashedAccountBeforeTx, TrieChangeSetsEntry},
};
use alloy_primitives::{B256, U256};
use reth_db::{
    BlockNumberList, Database, DatabaseEnv,
    cursor::{DbCursorRW, DbDupCursorRW},
    mdbx::{DatabaseArguments, init_db_for},
    transaction::{DbTx, DbTxMut},
};
use reth_primitives_traits::{Account, StorageEntry};
use reth_trie::{
    BranchNodeCompact, Nibbles, StoredNibbles, StoredNibblesSubKey,
    hashed_cursor::{HashedCursor, HashedStorageCursor},
    trie_cursor::{TrieCursor, TrieStorageCursor},
};
use reth_trie_common::StorageTrieEntry;
use tempfile::TempDir;

use crate::db::models::{
    AccountTrieShardedKey, BlockNumberHashedAddress, HashedAccountShardedKey,
    HashedStorageShardedKey, StorageTrieShardedKey, V2AccountTrieChangeSets, V2AccountsTrie,
    V2AccountsTrieHistory, V2HashedAccountChangeSets, V2HashedAccounts, V2HashedAccountsHistory,
    V2HashedStorageChangeSets, V2HashedStorages, V2HashedStoragesHistory, V2StorageTrieChangeSets,
    V2StoragesTrie, V2StoragesTrieHistory,
};
use reth_db::models::sharded_key::ShardedKey;

fn setup_db() -> DatabaseEnv {
    let tmp = TempDir::new().expect("create tmpdir");
    init_db_for::<_, models::Tables>(tmp, DatabaseArguments::default()).expect("init db")
}

fn node() -> BranchNodeCompact {
    BranchNodeCompact::new(0b11, 0, 0, vec![], Some(B256::repeat_byte(0xAB)))
}

fn node2() -> BranchNodeCompact {
    BranchNodeCompact::new(0b101, 0, 0, vec![], Some(B256::repeat_byte(0xCD)))
}

fn sample_account(nonce: u64) -> Account {
    Account { nonce, ..Default::default() }
}

// ====================== find_source unit tests ======================

#[test]
fn find_source_returns_current_state_when_no_history() {
    let db = setup_db();
    let addr = B256::from([0xAA; 32]);

    let tx = db.tx().expect("ro tx");
    let mut cursor = tx.cursor_read::<V2HashedAccountsHistory>().expect("c");

    let result = find_source::<V2HashedAccountsHistory, _>(
        &mut cursor,
        10,
        |bn| HashedAccountShardedKey::new(addr, bn),
        |k| k.0.key == addr,
    )
    .expect("ok");

    assert_eq!(result, ResolvedSource::FromCurrentState);
}

#[test]
fn find_source_returns_changeset_when_modification_after_target() {
    let db = setup_db();
    let addr = B256::from([0xBB; 32]);

    {
        let wtx = db.tx_mut().expect("rw tx");
        wtx.cursor_write::<V2HashedAccountsHistory>()
            .expect("c")
            .upsert(
                HashedAccountShardedKey::new(addr, u64::MAX),
                &BlockNumberList::new_pre_sorted([5, 10, 15]),
            )
            .expect("upsert");
        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");
    let mut cursor = tx.cursor_read::<V2HashedAccountsHistory>().expect("c");

    // Target block 7 → first block > 7 in [5, 10, 15] is 10
    let result = find_source::<V2HashedAccountsHistory, _>(
        &mut cursor,
        7,
        |bn| HashedAccountShardedKey::new(addr, bn),
        |k| k.0.key == addr,
    )
    .expect("ok");

    assert_eq!(result, ResolvedSource::FromChangeset(10));
}

#[test]
fn find_source_returns_current_state_when_no_modification_after_target() {
    let db = setup_db();
    let addr = B256::from([0xCC; 32]);

    {
        let wtx = db.tx_mut().expect("rw tx");
        wtx.cursor_write::<V2HashedAccountsHistory>()
            .expect("c")
            .upsert(
                HashedAccountShardedKey::new(addr, u64::MAX),
                &BlockNumberList::new_pre_sorted([3, 7]),
            )
            .expect("upsert");
        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");
    let mut cursor = tx.cursor_read::<V2HashedAccountsHistory>().expect("c");

    // Target block 10 → no block > 10 in [3, 7]
    let result = find_source::<V2HashedAccountsHistory, _>(
        &mut cursor,
        10,
        |bn| HashedAccountShardedKey::new(addr, bn),
        |k| k.0.key == addr,
    )
    .expect("ok");

    assert_eq!(result, ResolvedSource::FromCurrentState);
}

#[test]
fn find_source_handles_exact_match_block() {
    let db = setup_db();
    let addr = B256::from([0xDD; 32]);

    {
        let wtx = db.tx_mut().expect("rw tx");
        wtx.cursor_write::<V2HashedAccountsHistory>()
            .expect("c")
            .upsert(
                HashedAccountShardedKey::new(addr, u64::MAX),
                &BlockNumberList::new_pre_sorted([5, 10, 15]),
            )
            .expect("upsert");
        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");
    let mut cursor = tx.cursor_read::<V2HashedAccountsHistory>().expect("c");

    // Target block 10 (exactly in the bitmap) → first block > 10 is 15
    let result = find_source::<V2HashedAccountsHistory, _>(
        &mut cursor,
        10,
        |bn| HashedAccountShardedKey::new(addr, bn),
        |k| k.0.key == addr,
    )
    .expect("ok");

    assert_eq!(result, ResolvedSource::FromChangeset(15));
}

// ====================== find_source with AccountTrieShardedKey tests ======================

#[test]
fn find_source_resolves_root_path_despite_child_history() {
    // Regression test: the root trie path [] has history at blocks [10, 15].
    // A child path [0] also has history. With the old `ShardedKey<StoredNibbles>`
    // encoding (no length prefix), `cursor.seek` would land on the wrong path.
    // With `AccountTrieShardedKey`'s length-prefixed encoding, `find_source` works
    // correctly: all shards of [] sort before all shards of [0].
    let db = setup_db();
    let root_path = StoredNibbles(Nibbles::default());
    let child_path = StoredNibbles(Nibbles::from_nibbles([0]));

    {
        let wtx = db.tx_mut().expect("rw tx");
        let mut cursor = wtx.cursor_write::<V2AccountsTrieHistory>().expect("c");
        // Root path history: modified at blocks 10, 15
        cursor
            .upsert(
                AccountTrieShardedKey::new(root_path.clone(), u64::MAX),
                &BlockNumberList::new_pre_sorted([10, 15]),
            )
            .expect("upsert root");
        // Child path [0] history: modified at blocks 10, 15
        cursor
            .upsert(
                AccountTrieShardedKey::new(child_path, u64::MAX),
                &BlockNumberList::new_pre_sorted([10, 15]),
            )
            .expect("upsert child");
        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");
    let mut cursor = tx.cursor_read::<V2AccountsTrieHistory>().expect("c");

    // Query at block 12 — should find changeset at block 15
    // (the first modification after block 12).
    let result = find_source::<V2AccountsTrieHistory, _>(
        &mut cursor,
        12,
        |bn| AccountTrieShardedKey::new(root_path.clone(), bn),
        |k| k.key == root_path,
    )
    .expect("ok");

    assert_eq!(result, ResolvedSource::FromChangeset(15));
}

#[test]
fn find_source_trie_returns_current_state_when_no_history() {
    let db = setup_db();
    let root_path = StoredNibbles(Nibbles::default());

    let tx = db.tx().expect("ro tx");
    let mut cursor = tx.cursor_read::<V2AccountsTrieHistory>().expect("c");

    let result = find_source::<V2AccountsTrieHistory, _>(
        &mut cursor,
        10,
        |bn| AccountTrieShardedKey::new(root_path.clone(), bn),
        |k| k.key == root_path,
    )
    .expect("ok");

    assert_eq!(result, ResolvedSource::FromCurrentState);
}

#[test]
fn find_source_trie_returns_current_state_when_all_modifications_before_target() {
    let db = setup_db();
    let root_path = StoredNibbles(Nibbles::default());

    {
        let wtx = db.tx_mut().expect("rw tx");
        wtx.cursor_write::<V2AccountsTrieHistory>()
            .expect("c")
            .upsert(
                AccountTrieShardedKey::new(root_path.clone(), u64::MAX),
                &BlockNumberList::new_pre_sorted([5, 8]),
            )
            .expect("upsert");
        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");
    let mut cursor = tx.cursor_read::<V2AccountsTrieHistory>().expect("c");

    // Target block 10 — all modifications (5, 8) are ≤ 10
    let result = find_source::<V2AccountsTrieHistory, _>(
        &mut cursor,
        10,
        |bn| AccountTrieShardedKey::new(root_path.clone(), bn),
        |k| k.key == root_path,
    )
    .expect("ok");

    assert_eq!(result, ResolvedSource::FromCurrentState);
}

#[test]
fn find_source_handles_root_path_with_child_history() {
    // Verifies the encoding fix: `find_source` with `AccountTrieShardedKey`
    // correctly resolves the root path even when child path [0] has history.
    // Before the length-prefix fix, this would return `FromCurrentState`
    // due to encoding ambiguity. Now it correctly returns `FromChangeset(15)`.
    let db = setup_db();
    let root_path = StoredNibbles(Nibbles::default());
    let child_path = StoredNibbles(Nibbles::from_nibbles([0]));

    {
        let wtx = db.tx_mut().expect("rw tx");
        let mut cursor = wtx.cursor_write::<V2AccountsTrieHistory>().expect("c");
        cursor
            .upsert(
                AccountTrieShardedKey::new(root_path.clone(), u64::MAX),
                &BlockNumberList::new_pre_sorted([10, 15]),
            )
            .expect("upsert root");
        cursor
            .upsert(
                AccountTrieShardedKey::new(child_path, u64::MAX),
                &BlockNumberList::new_pre_sorted([10, 15]),
            )
            .expect("upsert child");
        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");
    let mut cursor = tx.cursor_read::<V2AccountsTrieHistory>().expect("c");

    // find_source with AccountTrieShardedKey correctly returns FromChangeset(15)
    let result = find_source::<V2AccountsTrieHistory, _>(
        &mut cursor,
        12,
        |bn| AccountTrieShardedKey::new(root_path.clone(), bn),
        |k| k.key == root_path,
    )
    .expect("ok");

    assert_eq!(
        result,
        ResolvedSource::FromChangeset(15),
        "find_source with AccountTrieShardedKey should correctly resolve root path"
    );
}

// ====================== Account Cursor tests ======================

#[test]
fn account_cursor_reads_current_state_when_no_history() {
    let db = setup_db();
    let addr = B256::from([0xAA; 32]);
    let acc = sample_account(42);

    {
        let wtx = db.tx_mut().expect("rw tx");
        wtx.cursor_write::<V2HashedAccounts>().expect("c").upsert(addr, &acc).expect("upsert");
        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");
    let mut cur = V2AccountCursor::new(
        tx.cursor_read::<V2HashedAccounts>().expect("c"),
        tx.cursor_read::<V2HashedAccountsHistory>().expect("c"),
        tx.cursor_read::<V2HashedAccountsHistory>().expect("c"),
        tx.cursor_dup_read::<V2HashedAccountChangeSets>().expect("c"),
        u64::MAX,
        true,
    );

    let result = cur.seek(addr).expect("ok").expect("should find");
    assert_eq!(result.0, addr);
    assert_eq!(result.1.nonce, 42);
}

#[test]
fn account_cursor_resolves_from_changeset_when_modified_after_target() {
    let db = setup_db();
    let addr = B256::from([0xBB; 32]);

    {
        let wtx = db.tx_mut().expect("rw tx");

        // Current state: nonce=10 (applied at block 5)
        wtx.cursor_write::<V2HashedAccounts>()
            .expect("c")
            .upsert(addr, &sample_account(10))
            .expect("upsert");

        // History bitmap: block 5 modified this account
        wtx.cursor_write::<V2HashedAccountsHistory>()
            .expect("c")
            .upsert(
                HashedAccountShardedKey::new(addr, u64::MAX),
                &BlockNumberList::new_pre_sorted([5]),
            )
            .expect("upsert");

        // Changeset: before block 5, account had nonce=3
        wtx.cursor_dup_write::<V2HashedAccountChangeSets>()
            .expect("c")
            .append_dup(5u64, HashedAccountBeforeTx::new(addr, Some(sample_account(3))))
            .expect("append");

        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");

    // Query at block 4 (before the modification at block 5)
    let mut cur = V2AccountCursor::new(
        tx.cursor_read::<V2HashedAccounts>().expect("c"),
        tx.cursor_read::<V2HashedAccountsHistory>().expect("c"),
        tx.cursor_read::<V2HashedAccountsHistory>().expect("c"),
        tx.cursor_dup_read::<V2HashedAccountChangeSets>().expect("c"),
        4,
        false,
    );

    let result = cur.seek(addr).expect("ok").expect("should find");
    assert_eq!(result.0, addr);
    assert_eq!(result.1.nonce, 3, "should get changeset value (before block 5)");
}

#[test]
fn account_cursor_returns_current_state_when_at_or_after_last_modification() {
    let db = setup_db();
    let addr = B256::from([0xCC; 32]);

    {
        let wtx = db.tx_mut().expect("rw tx");

        // Current state: nonce=20
        wtx.cursor_write::<V2HashedAccounts>()
            .expect("c")
            .upsert(addr, &sample_account(20))
            .expect("upsert");

        // History bitmap: [3, 7]
        wtx.cursor_write::<V2HashedAccountsHistory>()
            .expect("c")
            .upsert(
                HashedAccountShardedKey::new(addr, u64::MAX),
                &BlockNumberList::new_pre_sorted([3, 7]),
            )
            .expect("upsert");

        // Changeset at 3
        wtx.cursor_dup_write::<V2HashedAccountChangeSets>()
            .expect("c")
            .append_dup(3u64, HashedAccountBeforeTx::new(addr, Some(sample_account(1))))
            .expect("append");

        // Changeset at 7
        wtx.cursor_dup_write::<V2HashedAccountChangeSets>()
            .expect("c")
            .append_dup(7u64, HashedAccountBeforeTx::new(addr, Some(sample_account(5))))
            .expect("append");

        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");

    // Query at block 10 (after last modification at block 7)
    let mut cur = V2AccountCursor::new(
        tx.cursor_read::<V2HashedAccounts>().expect("c"),
        tx.cursor_read::<V2HashedAccountsHistory>().expect("c"),
        tx.cursor_read::<V2HashedAccountsHistory>().expect("c"),
        tx.cursor_dup_read::<V2HashedAccountChangeSets>().expect("c"),
        10,
        true,
    );

    let result = cur.seek(addr).expect("ok").expect("should find");
    assert_eq!(result.0, addr);
    assert_eq!(result.1.nonce, 20, "current state (no modification after block 10)");
}

#[test]
fn account_cursor_returns_none_when_not_yet_created() {
    let db = setup_db();
    let addr = B256::from([0xDD; 32]);

    {
        let wtx = db.tx_mut().expect("rw tx");

        // Current state: account exists (created at block 5)
        wtx.cursor_write::<V2HashedAccounts>()
            .expect("c")
            .upsert(addr, &sample_account(1))
            .expect("upsert");

        // History: first write at block 5
        wtx.cursor_write::<V2HashedAccountsHistory>()
            .expect("c")
            .upsert(
                HashedAccountShardedKey::new(addr, u64::MAX),
                &BlockNumberList::new_pre_sorted([5]),
            )
            .expect("upsert");

        // Changeset at 5: didn't exist before (info = None)
        wtx.cursor_dup_write::<V2HashedAccountChangeSets>()
            .expect("c")
            .append_dup(5u64, HashedAccountBeforeTx::new(addr, None))
            .expect("append");

        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");

    // Query at block 4 (before first write at 5)
    // The changeset at block 5 says info=None → the cursor's resolve returns None
    // → next_live_from skips this entry → seek returns None
    let mut cur = V2AccountCursor::new(
        tx.cursor_read::<V2HashedAccounts>().expect("c"),
        tx.cursor_read::<V2HashedAccountsHistory>().expect("c"),
        tx.cursor_read::<V2HashedAccountsHistory>().expect("c"),
        tx.cursor_dup_read::<V2HashedAccountChangeSets>().expect("c"),
        4,
        false,
    );

    let result = cur.seek(addr).expect("ok");
    assert!(result.is_none(), "account should not exist at block 4");
}

#[test]
fn account_cursor_seek_and_next_skip_dead_entries() {
    let db = setup_db();
    let k1 = B256::from([0x01; 32]);
    let k2 = B256::from([0x02; 32]);
    let k3 = B256::from([0x03; 32]);

    {
        let wtx = db.tx_mut().expect("rw tx");
        let mut c = wtx.cursor_write::<V2HashedAccounts>().expect("c");
        c.upsert(k1, &sample_account(1)).expect("upsert");
        c.upsert(k2, &sample_account(2)).expect("upsert");
        c.upsert(k3, &sample_account(3)).expect("upsert");

        // k2 was created at block 10
        wtx.cursor_write::<V2HashedAccountsHistory>()
            .expect("c")
            .upsert(
                HashedAccountShardedKey::new(k2, u64::MAX),
                &BlockNumberList::new_pre_sorted([10]),
            )
            .expect("upsert");

        // Changeset at 10: k2 didn't exist before
        wtx.cursor_dup_write::<V2HashedAccountChangeSets>()
            .expect("c")
            .append_dup(10u64, HashedAccountBeforeTx::new(k2, None))
            .expect("append");

        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");

    // Query at block 5 (before k2 was created)
    let mut cur = V2AccountCursor::new(
        tx.cursor_read::<V2HashedAccounts>().expect("c"),
        tx.cursor_read::<V2HashedAccountsHistory>().expect("c"),
        tx.cursor_read::<V2HashedAccountsHistory>().expect("c"),
        tx.cursor_dup_read::<V2HashedAccountChangeSets>().expect("c"),
        5,
        false,
    );

    // Seek k1 → should find k1 (no history = current state)
    let result = cur.seek(k1).expect("ok").expect("should find k1");
    assert_eq!(result.0, k1);
    assert_eq!(result.1.nonce, 1);

    // Next → should skip k2 (doesn't exist at block 5) and find k3
    let result = cur.next().expect("ok").expect("should skip k2, find k3");
    assert_eq!(result.0, k3);
    assert_eq!(result.1.nonce, 3);
}

/// Account was deleted (SELFDESTRUCT) after the target block, so it's not
/// in the current-state table. The history walk must discover it.
#[test]
fn account_cursor_discovers_key_deleted_after_target_block() {
    let db = setup_db();
    let k1 = B256::from([0x01; 32]);
    let k2 = B256::from([0x02; 32]); // deleted after target
    let k3 = B256::from([0x03; 32]);

    {
        let wtx = db.tx_mut().expect("rw tx");
        let mut c = wtx.cursor_write::<V2HashedAccounts>().expect("c");
        // k1 and k3 exist in current state; k2 was deleted at block 10
        c.upsert(k1, &sample_account(1)).expect("upsert");
        c.upsert(k3, &sample_account(3)).expect("upsert");

        // k2 history: modified at blocks [5, 10]
        wtx.cursor_write::<V2HashedAccountsHistory>()
            .expect("c")
            .upsert(
                HashedAccountShardedKey::new(k2, u64::MAX),
                &BlockNumberList::new_pre_sorted([5, 10]),
            )
            .expect("upsert");

        // Changeset at block 10: value before block 10 = nonce 7
        wtx.cursor_dup_write::<V2HashedAccountChangeSets>()
            .expect("c")
            .append_dup(10u64, HashedAccountBeforeTx::new(k2, Some(sample_account(7))))
            .expect("append");

        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");
    // Query at block 9: k2 existed with nonce=7
    let mut cur = V2AccountCursor::new(
        tx.cursor_read::<V2HashedAccounts>().expect("c"),
        tx.cursor_read::<V2HashedAccountsHistory>().expect("c"),
        tx.cursor_read::<V2HashedAccountsHistory>().expect("c"),
        tx.cursor_dup_read::<V2HashedAccountChangeSets>().expect("c"),
        9,
        false,
    );

    let r1 = cur.seek(B256::ZERO).expect("ok").expect("k1");
    assert_eq!(r1.0, k1);
    assert_eq!(r1.1.nonce, 1);

    let r2 = cur.next().expect("ok").expect("k2 from history");
    assert_eq!(r2.0, k2);
    assert_eq!(r2.1.nonce, 7);

    let r3 = cur.next().expect("ok").expect("k3");
    assert_eq!(r3.0, k3);
    assert_eq!(r3.1.nonce, 3);

    assert!(cur.next().expect("ok").is_none());
}

/// All accounts are deleted after the target block — only the history
/// walk can find them.
#[test]
fn account_cursor_all_keys_from_history() {
    let db = setup_db();
    let k1 = B256::from([0x10; 32]);
    let k2 = B256::from([0x20; 32]);

    {
        let wtx = db.tx_mut().expect("rw tx");
        // Nothing in current state.

        // k1 modified at block 5
        wtx.cursor_write::<V2HashedAccountsHistory>()
            .expect("c")
            .upsert(
                HashedAccountShardedKey::new(k1, u64::MAX),
                &BlockNumberList::new_pre_sorted([5]),
            )
            .expect("upsert");
        wtx.cursor_dup_write::<V2HashedAccountChangeSets>()
            .expect("c")
            .append_dup(5u64, HashedAccountBeforeTx::new(k1, Some(sample_account(11))))
            .expect("append");

        // k2 modified at block 8
        wtx.cursor_write::<V2HashedAccountsHistory>()
            .expect("c")
            .upsert(
                HashedAccountShardedKey::new(k2, u64::MAX),
                &BlockNumberList::new_pre_sorted([8]),
            )
            .expect("upsert");
        wtx.cursor_dup_write::<V2HashedAccountChangeSets>()
            .expect("c")
            .append_dup(8u64, HashedAccountBeforeTx::new(k2, Some(sample_account(22))))
            .expect("append");

        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");
    let mut cur = V2AccountCursor::new(
        tx.cursor_read::<V2HashedAccounts>().expect("c"),
        tx.cursor_read::<V2HashedAccountsHistory>().expect("c"),
        tx.cursor_read::<V2HashedAccountsHistory>().expect("c"),
        tx.cursor_dup_read::<V2HashedAccountChangeSets>().expect("c"),
        4,
        false,
    );

    let r1 = cur.seek(B256::ZERO).expect("ok").expect("k1");
    assert_eq!(r1.0, k1);
    assert_eq!(r1.1.nonce, 11);

    let r2 = cur.next().expect("ok").expect("k2");
    assert_eq!(r2.0, k2);
    assert_eq!(r2.1.nonce, 22);

    assert!(cur.next().expect("ok").is_none());
}

/// Duplicate key in both current state and history — the merge should
/// yield it exactly once.
#[test]
fn account_cursor_deduplicates_key_in_both_cursors() {
    let db = setup_db();
    let k = B256::from([0x55; 32]);

    {
        let wtx = db.tx_mut().expect("rw tx");
        wtx.cursor_write::<V2HashedAccounts>()
            .expect("c")
            .upsert(k, &sample_account(99))
            .expect("upsert");

        wtx.cursor_write::<V2HashedAccountsHistory>()
            .expect("c")
            .upsert(
                HashedAccountShardedKey::new(k, u64::MAX),
                &BlockNumberList::new_pre_sorted([5]),
            )
            .expect("upsert");
        wtx.cursor_dup_write::<V2HashedAccountChangeSets>()
            .expect("c")
            .append_dup(5u64, HashedAccountBeforeTx::new(k, Some(sample_account(50))))
            .expect("append");

        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");
    // At block 4 → changeset at 5 gives nonce=50
    let mut cur = V2AccountCursor::new(
        tx.cursor_read::<V2HashedAccounts>().expect("c"),
        tx.cursor_read::<V2HashedAccountsHistory>().expect("c"),
        tx.cursor_read::<V2HashedAccountsHistory>().expect("c"),
        tx.cursor_dup_read::<V2HashedAccountChangeSets>().expect("c"),
        4,
        false,
    );

    let r = cur.seek(B256::ZERO).expect("ok").expect("one result");
    assert_eq!(r.0, k);
    assert_eq!(r.1.nonce, 50);
    assert!(cur.next().expect("ok").is_none(), "no duplicates");
}

// ====================== Account Trie Cursor tests ======================

#[test]
fn account_trie_cursor_reads_current_state_when_no_history() {
    let db = setup_db();
    let path = Nibbles::from_nibbles([0x0A]);
    let n = node();

    {
        let wtx = db.tx_mut().expect("rw tx");
        wtx.cursor_write::<V2AccountsTrie>()
            .expect("c")
            .upsert(StoredNibbles(path), &n)
            .expect("upsert");
        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");
    let mut cur = V2AccountTrieCursor::new(
        tx.cursor_read::<V2AccountsTrie>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_dup_read::<V2AccountTrieChangeSets>().expect("c"),
        u64::MAX,
        true,
    );

    let out = TrieCursor::seek_exact(&mut cur, path).expect("ok").expect("some");
    assert_eq!(out.0, path);
    assert_eq!(out.1, n);
}

#[test]
fn account_trie_cursor_resolves_old_node_from_changeset() {
    let db = setup_db();
    let path = Nibbles::from_nibbles([0x0B]);
    let old_node = node();
    let new_node = node2();

    {
        let wtx = db.tx_mut().expect("rw tx");

        // Current state has new_node (applied at block 10)
        wtx.cursor_write::<V2AccountsTrie>()
            .expect("c")
            .upsert(StoredNibbles(path), &new_node)
            .expect("upsert");

        // History: modified at block 10
        wtx.cursor_write::<V2AccountsTrieHistory>()
            .expect("c")
            .upsert(
                AccountTrieShardedKey::new(StoredNibbles(path), u64::MAX),
                &BlockNumberList::new_pre_sorted([10]),
            )
            .expect("upsert");

        // Changeset at block 10: old_node was the value before
        let cs_entry = TrieChangeSetsEntry {
            nibbles: StoredNibblesSubKey(path),
            node: Some(old_node.clone()),
        };
        wtx.cursor_dup_write::<V2AccountTrieChangeSets>()
            .expect("c")
            .append_dup(10u64, cs_entry)
            .expect("append");

        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");

    // Query at block 9 (before modification at 10)
    let mut cur = V2AccountTrieCursor::new(
        tx.cursor_read::<V2AccountsTrie>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_dup_read::<V2AccountTrieChangeSets>().expect("c"),
        9,
        false,
    );

    let out = TrieCursor::seek_exact(&mut cur, path).expect("ok").expect("some");
    assert_eq!(out.0, path);
    assert_eq!(out.1, old_node, "should get old node from changeset");
}

#[test]
fn account_trie_cursor_seek_and_next_skip_dead_nodes() {
    let db = setup_db();
    let p1 = Nibbles::from_nibbles([0x01]);
    let p2 = Nibbles::from_nibbles([0x02]);
    let p3 = Nibbles::from_nibbles([0x03]);

    {
        let wtx = db.tx_mut().expect("rw tx");
        let mut c = wtx.cursor_write::<V2AccountsTrie>().expect("c");
        c.upsert(StoredNibbles(p1), &node()).expect("upsert");
        c.upsert(StoredNibbles(p2), &node()).expect("upsert");
        c.upsert(StoredNibbles(p3), &node()).expect("upsert");

        // p2 was created at block 5, didn't exist before
        wtx.cursor_write::<V2AccountsTrieHistory>()
            .expect("c")
            .upsert(
                AccountTrieShardedKey::new(StoredNibbles(p2), u64::MAX),
                &BlockNumberList::new_pre_sorted([5]),
            )
            .expect("upsert");

        let cs_entry = TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(p2), node: None };
        wtx.cursor_dup_write::<V2AccountTrieChangeSets>()
            .expect("c")
            .append_dup(5u64, cs_entry)
            .expect("append");

        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");

    // Query at block 3 (before p2 was created)
    let mut cur = V2AccountTrieCursor::new(
        tx.cursor_read::<V2AccountsTrie>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_dup_read::<V2AccountTrieChangeSets>().expect("c"),
        3,
        false,
    );

    // Seek p1 → should find p1
    let out = TrieCursor::seek(&mut cur, p1).expect("ok").expect("some");
    assert_eq!(out.0, p1);

    // Next → should skip p2 (didn't exist at block 3) and find p3
    let out = TrieCursor::next(&mut cur).expect("ok").expect("some");
    assert_eq!(out.0, p3, "should skip p2 which didn't exist at block 3");
}

#[test]
fn account_trie_cursor_seek_returns_gte() {
    let db = setup_db();
    let p_a = Nibbles::from_nibbles([0x0A]);
    let p_c = Nibbles::from_nibbles([0x0C]);
    let p_b = Nibbles::from_nibbles([0x0B]);

    {
        let wtx = db.tx_mut().expect("rw tx");
        let mut c = wtx.cursor_write::<V2AccountsTrie>().expect("c");
        c.upsert(StoredNibbles(p_a), &node()).expect("upsert");
        c.upsert(StoredNibbles(p_c), &node()).expect("upsert");
        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");
    let mut cur = V2AccountTrieCursor::new(
        tx.cursor_read::<V2AccountsTrie>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_dup_read::<V2AccountTrieChangeSets>().expect("c"),
        u64::MAX,
        true,
    );

    // Seek to 0x0B → should land on 0x0C (first ≥ 0x0B)
    let out = TrieCursor::seek(&mut cur, p_b).expect("ok").expect("some");
    assert_eq!(out.0, p_c);
}

#[test]
fn account_trie_cursor_empty_returns_none() {
    let db = setup_db();
    let tx = db.tx().expect("ro tx");
    let mut cur = V2AccountTrieCursor::new(
        tx.cursor_read::<V2AccountsTrie>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_dup_read::<V2AccountTrieChangeSets>().expect("c"),
        u64::MAX,
        true,
    );

    assert!(TrieCursor::seek(&mut cur, Nibbles::default()).expect("ok").is_none());
    assert!(TrieCursor::next(&mut cur).expect("ok").is_none());
}

#[test]
fn account_trie_cursor_discovers_key_deleted_after_target_block() {
    // Scenario from the design discussion:
    //   block 2 adds [a1, b1, c1]
    //   block 3 adds [d1, z1] and deletes a1
    // Query at block 2 → should see a1, b1, c1 (not d1 or z1)
    let db = setup_db();
    let a1 = Nibbles::from_nibbles([0x0A, 0x01]);
    let b1 = Nibbles::from_nibbles([0x0B, 0x01]);
    let c1 = Nibbles::from_nibbles([0x0C, 0x01]);
    let d1 = Nibbles::from_nibbles([0x0D, 0x01]);
    let z1 = Nibbles::from_nibbles([0x0F, 0x01]);
    let n = node();

    {
        let wtx = db.tx_mut().expect("rw tx");
        let mut c = wtx.cursor_write::<V2AccountsTrie>().expect("c");

        // Current state after block 3: {b1, c1, d1, z1} (a1 deleted)
        c.upsert(StoredNibbles(b1), &n).expect("upsert");
        c.upsert(StoredNibbles(c1), &n).expect("upsert");
        c.upsert(StoredNibbles(d1), &n).expect("upsert");
        c.upsert(StoredNibbles(z1), &n).expect("upsert");

        // History bitmaps
        let mut hc = wtx.cursor_write::<V2AccountsTrieHistory>().expect("c");
        // a1 modified at blocks 2 and 3
        hc.upsert(
            AccountTrieShardedKey::new(StoredNibbles(a1), u64::MAX),
            &BlockNumberList::new_pre_sorted([2, 3]),
        )
        .expect("upsert");
        // b1 modified at block 2
        hc.upsert(
            AccountTrieShardedKey::new(StoredNibbles(b1), u64::MAX),
            &BlockNumberList::new_pre_sorted([2]),
        )
        .expect("upsert");
        // c1 modified at block 2
        hc.upsert(
            AccountTrieShardedKey::new(StoredNibbles(c1), u64::MAX),
            &BlockNumberList::new_pre_sorted([2]),
        )
        .expect("upsert");
        // d1 modified at block 3
        hc.upsert(
            AccountTrieShardedKey::new(StoredNibbles(d1), u64::MAX),
            &BlockNumberList::new_pre_sorted([3]),
        )
        .expect("upsert");
        // z1 modified at block 3
        hc.upsert(
            AccountTrieShardedKey::new(StoredNibbles(z1), u64::MAX),
            &BlockNumberList::new_pre_sorted([3]),
        )
        .expect("upsert");

        // Changesets
        let mut csc = wtx.cursor_dup_write::<V2AccountTrieChangeSets>().expect("c");

        // Block 2 changesets: a1, b1, c1 didn't exist before
        csc.append_dup(2u64, TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(a1), node: None })
            .expect("append");
        csc.append_dup(2u64, TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(b1), node: None })
            .expect("append");
        csc.append_dup(2u64, TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(c1), node: None })
            .expect("append");

        // Block 3 changesets: a1 existed (deleted), d1 and z1 didn't exist
        csc.append_dup(
            3u64,
            TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(a1), node: Some(n.clone()) },
        )
        .expect("append");
        csc.append_dup(3u64, TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(d1), node: None })
            .expect("append");
        csc.append_dup(3u64, TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(z1), node: None })
            .expect("append");

        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");

    // Query at block 2: a1 should be visible even though it's deleted
    // from current state (deleted at block 3).
    let mut cur = V2AccountTrieCursor::new(
        tx.cursor_read::<V2AccountsTrie>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_dup_read::<V2AccountTrieChangeSets>().expect("c"),
        2,
        false,
    );

    // seek(default) → a1 (discovered via history walk, resolved from changeset)
    let out = TrieCursor::seek(&mut cur, Nibbles::default()).expect("ok").expect("should find a1");
    assert_eq!(out.0, a1, "a1 must be visible at block 2");
    assert_eq!(out.1, n);

    // next → b1
    let out = TrieCursor::next(&mut cur).expect("ok").expect("should find b1");
    assert_eq!(out.0, b1);

    // next → c1
    let out = TrieCursor::next(&mut cur).expect("ok").expect("should find c1");
    assert_eq!(out.0, c1);

    // next → None (d1 and z1 didn't exist at block 2)
    let out = TrieCursor::next(&mut cur).expect("ok");
    assert!(out.is_none(), "d1 and z1 must NOT be visible at block 2");
}

#[test]
fn account_trie_cursor_deleted_key_only_in_history() {
    // Key exists ONLY in history (not in current state), no other keys at all.
    // Ensures the history-walk alone can produce results when current state is empty.
    let db = setup_db();
    let p = Nibbles::from_nibbles([0x05]);
    let n = node();

    {
        let wtx = db.tx_mut().expect("rw tx");
        // Current state: empty (p was deleted at block 4)

        // History: [2, 4]
        wtx.cursor_write::<V2AccountsTrieHistory>()
            .expect("c")
            .upsert(
                AccountTrieShardedKey::new(StoredNibbles(p), u64::MAX),
                &BlockNumberList::new_pre_sorted([2, 4]),
            )
            .expect("upsert");

        // Changeset block 2: p didn't exist before
        let mut csc = wtx.cursor_dup_write::<V2AccountTrieChangeSets>().expect("c");
        csc.append_dup(2u64, TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(p), node: None })
            .expect("append");
        // Changeset block 4: p had value n before deletion
        csc.append_dup(
            4u64,
            TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(p), node: Some(n.clone()) },
        )
        .expect("append");

        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");

    // Query at block 3: p should be visible (created at 2, deleted at 4)
    let mut cur = V2AccountTrieCursor::new(
        tx.cursor_read::<V2AccountsTrie>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_dup_read::<V2AccountTrieChangeSets>().expect("c"),
        3,
        false,
    );

    let out = TrieCursor::seek(&mut cur, Nibbles::default())
        .expect("ok")
        .expect("should find p at block 3");
    assert_eq!(out.0, p);
    assert_eq!(out.1, n, "should resolve from changeset at block 4");

    // next → None
    assert!(TrieCursor::next(&mut cur).expect("ok").is_none());

    // Also: query at block 1 → p didn't exist yet
    let mut cur2 = V2AccountTrieCursor::new(
        tx.cursor_read::<V2AccountsTrie>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_dup_read::<V2AccountTrieChangeSets>().expect("c"),
        1,
        false,
    );
    assert!(
        TrieCursor::seek(&mut cur2, Nibbles::default()).expect("ok").is_none(),
        "p should not exist at block 1"
    );
}

#[test]
fn account_trie_cursor_seek_exact_on_deleted_key() {
    // seek_exact on a key that is deleted from current state but alive at
    // the target block.
    let db = setup_db();
    let p = Nibbles::from_nibbles([0x0A]);
    let n = node();

    {
        let wtx = db.tx_mut().expect("rw tx");
        // Current state: empty (p deleted at block 10)

        wtx.cursor_write::<V2AccountsTrieHistory>()
            .expect("c")
            .upsert(
                AccountTrieShardedKey::new(StoredNibbles(p), u64::MAX),
                &BlockNumberList::new_pre_sorted([5, 10]),
            )
            .expect("upsert");

        let mut csc = wtx.cursor_dup_write::<V2AccountTrieChangeSets>().expect("c");
        // Block 5: created (old = None)
        csc.append_dup(5u64, TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(p), node: None })
            .expect("append");
        // Block 10: deleted (old = n)
        csc.append_dup(
            10u64,
            TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(p), node: Some(n.clone()) },
        )
        .expect("append");

        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");

    // seek_exact at block 8 → should find p
    let mut cur = V2AccountTrieCursor::new(
        tx.cursor_read::<V2AccountsTrie>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_dup_read::<V2AccountTrieChangeSets>().expect("c"),
        8,
        false,
    );
    let out = TrieCursor::seek_exact(&mut cur, p).expect("ok").expect("should find");
    assert_eq!(out.0, p);
    assert_eq!(out.1, n);

    // seek_exact at block 3 → should NOT find p (created at 5)
    let mut cur2 = V2AccountTrieCursor::new(
        tx.cursor_read::<V2AccountsTrie>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_dup_read::<V2AccountTrieChangeSets>().expect("c"),
        3,
        false,
    );
    assert!(TrieCursor::seek_exact(&mut cur2, p).expect("ok").is_none());
}

#[test]
fn account_trie_cursor_current_tracks_last_yielded() {
    // current() should return the last key yielded by seek/next.
    let db = setup_db();
    let p1 = Nibbles::from_nibbles([0x01]);
    let p2 = Nibbles::from_nibbles([0x02]);
    let n = node();

    {
        let wtx = db.tx_mut().expect("rw tx");
        let mut c = wtx.cursor_write::<V2AccountsTrie>().expect("c");
        c.upsert(StoredNibbles(p1), &n).expect("upsert");
        c.upsert(StoredNibbles(p2), &n).expect("upsert");
        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");
    let mut cur = V2AccountTrieCursor::new(
        tx.cursor_read::<V2AccountsTrie>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_dup_read::<V2AccountTrieChangeSets>().expect("c"),
        u64::MAX,
        true,
    );

    // Before any seek, current is None
    assert!(TrieCursor::current(&mut cur).expect("ok").is_none());

    TrieCursor::seek(&mut cur, p1).expect("ok");
    assert_eq!(TrieCursor::current(&mut cur).expect("ok"), Some(p1));

    TrieCursor::next(&mut cur).expect("ok");
    assert_eq!(TrieCursor::current(&mut cur).expect("ok"), Some(p2));
}

#[test]
fn account_trie_cursor_seek_exact_then_next() {
    // After seek_exact, next() should return the key after the sought key.
    let db = setup_db();
    let p1 = Nibbles::from_nibbles([0x01]);
    let p2 = Nibbles::from_nibbles([0x02]);
    let p3 = Nibbles::from_nibbles([0x03]);
    let n = node();

    {
        let wtx = db.tx_mut().expect("rw tx");
        let mut c = wtx.cursor_write::<V2AccountsTrie>().expect("c");
        c.upsert(StoredNibbles(p1), &n).expect("upsert");
        c.upsert(StoredNibbles(p2), &n).expect("upsert");
        c.upsert(StoredNibbles(p3), &n).expect("upsert");
        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");
    let mut cur = V2AccountTrieCursor::new(
        tx.cursor_read::<V2AccountsTrie>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_dup_read::<V2AccountTrieChangeSets>().expect("c"),
        u64::MAX,
        true,
    );

    // seek_exact p2
    let out = TrieCursor::seek_exact(&mut cur, p2).expect("ok").expect("some");
    assert_eq!(out.0, p2);

    // next → p3
    let out = TrieCursor::next(&mut cur).expect("ok").expect("some");
    assert_eq!(out.0, p3);

    // next → None
    assert!(TrieCursor::next(&mut cur).expect("ok").is_none());
}

#[test]
fn account_trie_cursor_seek_gte_skips_dead_landing() {
    // seek() lands on a dead key (in current state but not alive at target
    // block) and must skip forward to the next live key.
    let db = setup_db();
    let p_a = Nibbles::from_nibbles([0x0A]);
    let p_b = Nibbles::from_nibbles([0x0B]);
    let p_c = Nibbles::from_nibbles([0x0C]);

    {
        let wtx = db.tx_mut().expect("rw tx");
        let mut c = wtx.cursor_write::<V2AccountsTrie>().expect("c");
        c.upsert(StoredNibbles(p_a), &node()).expect("upsert");
        c.upsert(StoredNibbles(p_b), &node()).expect("upsert");
        c.upsert(StoredNibbles(p_c), &node()).expect("upsert");

        // p_b was created at block 10
        wtx.cursor_write::<V2AccountsTrieHistory>()
            .expect("c")
            .upsert(
                AccountTrieShardedKey::new(StoredNibbles(p_b), u64::MAX),
                &BlockNumberList::new_pre_sorted([10]),
            )
            .expect("upsert");

        wtx.cursor_dup_write::<V2AccountTrieChangeSets>()
            .expect("c")
            .append_dup(
                10u64,
                TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(p_b), node: None },
            )
            .expect("append");

        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");

    // At block 5, seek(p_b) → p_b is dead → should skip to p_c
    let mut cur = V2AccountTrieCursor::new(
        tx.cursor_read::<V2AccountsTrie>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_dup_read::<V2AccountTrieChangeSets>().expect("c"),
        5,
        false,
    );

    let out = TrieCursor::seek(&mut cur, p_b).expect("ok").expect("some");
    assert_eq!(out.0, p_c, "should skip dead p_b and land on p_c");
}

#[test]
fn account_trie_cursor_all_keys_dead() {
    // Every key in current state is dead at the target block, and no
    // history-only keys exist → None.
    let db = setup_db();
    let p1 = Nibbles::from_nibbles([0x01]);
    let p2 = Nibbles::from_nibbles([0x02]);

    {
        let wtx = db.tx_mut().expect("rw tx");
        let mut c = wtx.cursor_write::<V2AccountsTrie>().expect("c");
        c.upsert(StoredNibbles(p1), &node()).expect("upsert");
        c.upsert(StoredNibbles(p2), &node()).expect("upsert");

        let mut hc = wtx.cursor_write::<V2AccountsTrieHistory>().expect("c");
        // Both created at block 5
        hc.upsert(
            AccountTrieShardedKey::new(StoredNibbles(p1), u64::MAX),
            &BlockNumberList::new_pre_sorted([5]),
        )
        .expect("upsert");
        hc.upsert(
            AccountTrieShardedKey::new(StoredNibbles(p2), u64::MAX),
            &BlockNumberList::new_pre_sorted([5]),
        )
        .expect("upsert");

        let mut csc = wtx.cursor_dup_write::<V2AccountTrieChangeSets>().expect("c");
        csc.append_dup(5u64, TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(p1), node: None })
            .expect("append");
        csc.append_dup(5u64, TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(p2), node: None })
            .expect("append");

        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");

    // At block 3 → both dead
    let mut cur = V2AccountTrieCursor::new(
        tx.cursor_read::<V2AccountsTrie>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_dup_read::<V2AccountTrieChangeSets>().expect("c"),
        3,
        false,
    );

    assert!(TrieCursor::seek(&mut cur, Nibbles::default()).expect("ok").is_none());
}

#[test]
fn account_trie_cursor_interleaved_current_and_history_keys() {
    // Current state has {b, d}. History has {a, c, e} (all deleted after
    // target block). The merge should yield a, b, c, d, e in order.
    let db = setup_db();
    let a = Nibbles::from_nibbles([0x01]);
    let b = Nibbles::from_nibbles([0x02]);
    let c = Nibbles::from_nibbles([0x03]);
    let d = Nibbles::from_nibbles([0x04]);
    let e = Nibbles::from_nibbles([0x05]);
    let n = node();

    {
        let wtx = db.tx_mut().expect("rw tx");
        let mut cs = wtx.cursor_write::<V2AccountsTrie>().expect("c");
        // Current state: {b, d}
        cs.upsert(StoredNibbles(b), &n).expect("upsert");
        cs.upsert(StoredNibbles(d), &n).expect("upsert");

        let mut hc = wtx.cursor_write::<V2AccountsTrieHistory>().expect("c");
        // a: created block 2, deleted block 10
        hc.upsert(
            AccountTrieShardedKey::new(StoredNibbles(a), u64::MAX),
            &BlockNumberList::new_pre_sorted([2, 10]),
        )
        .expect("upsert");
        // b: created block 2 (stays in current state)
        hc.upsert(
            AccountTrieShardedKey::new(StoredNibbles(b), u64::MAX),
            &BlockNumberList::new_pre_sorted([2]),
        )
        .expect("upsert");
        // c: created block 2, deleted block 10
        hc.upsert(
            AccountTrieShardedKey::new(StoredNibbles(c), u64::MAX),
            &BlockNumberList::new_pre_sorted([2, 10]),
        )
        .expect("upsert");
        // d: created block 2 (stays)
        hc.upsert(
            AccountTrieShardedKey::new(StoredNibbles(d), u64::MAX),
            &BlockNumberList::new_pre_sorted([2]),
        )
        .expect("upsert");
        // e: created block 2, deleted block 10
        hc.upsert(
            AccountTrieShardedKey::new(StoredNibbles(e), u64::MAX),
            &BlockNumberList::new_pre_sorted([2, 10]),
        )
        .expect("upsert");

        let mut csc = wtx.cursor_dup_write::<V2AccountTrieChangeSets>().expect("c");
        // Block 2: all created (old = None)
        csc.append_dup(2u64, TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(a), node: None })
            .expect("append");
        csc.append_dup(2u64, TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(b), node: None })
            .expect("append");
        csc.append_dup(2u64, TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(c), node: None })
            .expect("append");
        csc.append_dup(2u64, TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(d), node: None })
            .expect("append");
        csc.append_dup(2u64, TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(e), node: None })
            .expect("append");
        // Block 10: a, c, e deleted (old = n)
        csc.append_dup(
            10u64,
            TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(a), node: Some(n.clone()) },
        )
        .expect("append");
        csc.append_dup(
            10u64,
            TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(c), node: Some(n.clone()) },
        )
        .expect("append");
        csc.append_dup(
            10u64,
            TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(e), node: Some(n.clone()) },
        )
        .expect("append");

        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");

    // At block 5: a, b, c, d, e all alive (a, c, e via changeset at 10)
    let mut cur = V2AccountTrieCursor::new(
        tx.cursor_read::<V2AccountsTrie>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_dup_read::<V2AccountTrieChangeSets>().expect("c"),
        5,
        false,
    );

    let out = TrieCursor::seek(&mut cur, Nibbles::default()).expect("ok").expect("a");
    assert_eq!(out.0, a, "first key should be a (from history)");

    let out = TrieCursor::next(&mut cur).expect("ok").expect("b");
    assert_eq!(out.0, b, "second key should be b (from current state)");

    let out = TrieCursor::next(&mut cur).expect("ok").expect("c");
    assert_eq!(out.0, c, "third key should be c (from history)");

    let out = TrieCursor::next(&mut cur).expect("ok").expect("d");
    assert_eq!(out.0, d, "fourth key should be d (from current state)");

    let out = TrieCursor::next(&mut cur).expect("ok").expect("e");
    assert_eq!(out.0, e, "fifth key should be e (from history)");

    assert!(TrieCursor::next(&mut cur).expect("ok").is_none());
}

#[test]
fn account_trie_cursor_duplicate_key_in_both_cursors() {
    // Key exists in BOTH current state and history. The merge should NOT
    // yield it twice.
    let db = setup_db();
    let p = Nibbles::from_nibbles([0x0A]);
    let n = node();
    let n2 = node2();

    {
        let wtx = db.tx_mut().expect("rw tx");

        // Current state: p -> n2 (updated at block 10)
        wtx.cursor_write::<V2AccountsTrie>()
            .expect("c")
            .upsert(StoredNibbles(p), &n2)
            .expect("upsert");

        // History: modified at block 10
        wtx.cursor_write::<V2AccountsTrieHistory>()
            .expect("c")
            .upsert(
                AccountTrieShardedKey::new(StoredNibbles(p), u64::MAX),
                &BlockNumberList::new_pre_sorted([10]),
            )
            .expect("upsert");

        wtx.cursor_dup_write::<V2AccountTrieChangeSets>()
            .expect("c")
            .append_dup(
                10u64,
                TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(p), node: Some(n.clone()) },
            )
            .expect("append");

        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");

    // At block 8: resolve from changeset at 10 → old value n
    let mut cur = V2AccountTrieCursor::new(
        tx.cursor_read::<V2AccountsTrie>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_dup_read::<V2AccountTrieChangeSets>().expect("c"),
        8,
        false,
    );

    let out = TrieCursor::seek(&mut cur, p).expect("ok").expect("should find");
    assert_eq!(out.0, p);
    assert_eq!(out.1, n, "should get old value from changeset");

    // next → None (should NOT yield p again)
    assert!(TrieCursor::next(&mut cur).expect("ok").is_none());
}

#[test]
fn account_trie_cursor_query_at_latest_block() {
    // When max_block_number == u64::MAX, everything reads from current
    // state — even keys with history. This exercises the
    // FromCurrentState path in find_source.
    let db = setup_db();
    let p1 = Nibbles::from_nibbles([0x01]);
    let p2 = Nibbles::from_nibbles([0x02]);
    let n2 = node2();

    {
        let wtx = db.tx_mut().expect("rw tx");
        let mut c = wtx.cursor_write::<V2AccountsTrie>().expect("c");
        c.upsert(StoredNibbles(p1), &node()).expect("upsert");
        c.upsert(StoredNibbles(p2), &n2).expect("upsert");

        // p2 has history at block 5
        wtx.cursor_write::<V2AccountsTrieHistory>()
            .expect("c")
            .upsert(
                AccountTrieShardedKey::new(StoredNibbles(p2), u64::MAX),
                &BlockNumberList::new_pre_sorted([5]),
            )
            .expect("upsert");

        wtx.cursor_dup_write::<V2AccountTrieChangeSets>()
            .expect("c")
            .append_dup(5u64, TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(p2), node: None })
            .expect("append");

        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");
    let mut cur = V2AccountTrieCursor::new(
        tx.cursor_read::<V2AccountsTrie>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_dup_read::<V2AccountTrieChangeSets>().expect("c"),
        u64::MAX,
        true,
    );

    let out = TrieCursor::seek(&mut cur, Nibbles::default()).expect("ok").expect("p1");
    assert_eq!(out.0, p1);

    let out = TrieCursor::next(&mut cur).expect("ok").expect("p2");
    assert_eq!(out.0, p2);
    assert_eq!(out.1, n2, "should read current state value");

    assert!(TrieCursor::next(&mut cur).expect("ok").is_none());
}

// ——— Storage Trie Cursor: dual-cursor merge tests ———

#[test]
fn storage_trie_cursor_discovers_deleted_key() {
    // Same scenario as account trie deleted-key test, but for storage trie.
    let db = setup_db();
    let addr = B256::from([0xAA; 32]);
    let a1 = Nibbles::from_nibbles([0x0A, 0x01]);
    let b1 = Nibbles::from_nibbles([0x0B, 0x01]);
    let c1 = Nibbles::from_nibbles([0x0C, 0x01]);
    let n = node();

    {
        let wtx = db.tx_mut().expect("rw tx");

        // Current state: {b1, c1} (a1 deleted at block 5)
        let mut sc = wtx.cursor_dup_write::<V2StoragesTrie>().expect("c");
        sc.upsert(addr, &StorageTrieEntry { nibbles: StoredNibblesSubKey(b1), node: n.clone() })
            .expect("upsert");
        sc.upsert(addr, &StorageTrieEntry { nibbles: StoredNibblesSubKey(c1), node: n.clone() })
            .expect("upsert");

        let mut hc = wtx.cursor_write::<V2StoragesTrieHistory>().expect("c");
        // a1: created at block 2, deleted at block 5
        hc.upsert(
            StorageTrieShardedKey::new(addr, StoredNibbles(a1), u64::MAX),
            &BlockNumberList::new_pre_sorted([2, 5]),
        )
        .expect("upsert");
        // b1: created at block 2
        hc.upsert(
            StorageTrieShardedKey::new(addr, StoredNibbles(b1), u64::MAX),
            &BlockNumberList::new_pre_sorted([2]),
        )
        .expect("upsert");
        // c1: created at block 2
        hc.upsert(
            StorageTrieShardedKey::new(addr, StoredNibbles(c1), u64::MAX),
            &BlockNumberList::new_pre_sorted([2]),
        )
        .expect("upsert");

        let mut csc = wtx.cursor_dup_write::<V2StorageTrieChangeSets>().expect("c");
        // Block 2: all created
        let cs_key2 = BlockNumberHashedAddress((2u64, addr));
        csc.append_dup(
            cs_key2,
            TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(a1), node: None },
        )
        .expect("append");
        csc.append_dup(
            cs_key2,
            TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(b1), node: None },
        )
        .expect("append");
        csc.append_dup(
            cs_key2,
            TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(c1), node: None },
        )
        .expect("append");
        // Block 5: a1 deleted (old = n)
        let cs_key5 = BlockNumberHashedAddress((5u64, addr));
        csc.append_dup(
            cs_key5,
            TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(a1), node: Some(n) },
        )
        .expect("append");

        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");

    // Query at block 3: a1 should be visible
    let mut cur = V2StorageTrieCursor::new(
        tx.cursor_dup_read::<V2StoragesTrie>().expect("c"),
        tx.cursor_read::<V2StoragesTrieHistory>().expect("c"),
        tx.cursor_read::<V2StoragesTrieHistory>().expect("c"),
        tx.cursor_dup_read::<V2StorageTrieChangeSets>().expect("c"),
        addr,
        3,
        false,
    );

    let out = TrieCursor::seek(&mut cur, Nibbles::default()).expect("ok").expect("should find a1");
    assert_eq!(out.0, a1, "a1 must be visible at block 3");

    let out = TrieCursor::next(&mut cur).expect("ok").expect("b1");
    assert_eq!(out.0, b1);

    let out = TrieCursor::next(&mut cur).expect("ok").expect("c1");
    assert_eq!(out.0, c1);

    assert!(TrieCursor::next(&mut cur).expect("ok").is_none());
}

#[test]
fn storage_trie_cursor_deleted_key_does_not_cross_address() {
    // Deleted history key from addr_b must NOT appear when walking addr_a.
    let db = setup_db();
    let addr_a = B256::from([0x11; 32]);
    let addr_b = B256::from([0x22; 32]);
    let p1 = Nibbles::from_nibbles([0x01]);
    let p2 = Nibbles::from_nibbles([0x02]);
    let n = node();

    {
        let wtx = db.tx_mut().expect("rw tx");

        // Current state: addr_a has {p1}, addr_b is empty (p2 deleted)
        let mut sc = wtx.cursor_dup_write::<V2StoragesTrie>().expect("c");
        sc.upsert(addr_a, &StorageTrieEntry { nibbles: StoredNibblesSubKey(p1), node: n.clone() })
            .expect("upsert");

        // addr_b: p2 history (created block 2, deleted block 5)
        wtx.cursor_write::<V2StoragesTrieHistory>()
            .expect("c")
            .upsert(
                StorageTrieShardedKey::new(addr_b, StoredNibbles(p2), u64::MAX),
                &BlockNumberList::new_pre_sorted([2, 5]),
            )
            .expect("upsert");

        let mut csc = wtx.cursor_dup_write::<V2StorageTrieChangeSets>().expect("c");
        csc.append_dup(
            BlockNumberHashedAddress((2u64, addr_b)),
            TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(p2), node: None },
        )
        .expect("append");
        csc.append_dup(
            BlockNumberHashedAddress((5u64, addr_b)),
            TrieChangeSetsEntry { nibbles: StoredNibblesSubKey(p2), node: Some(n) },
        )
        .expect("append");

        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");

    // Walk addr_a at block 3 → only p1
    let mut cur = V2StorageTrieCursor::new(
        tx.cursor_dup_read::<V2StoragesTrie>().expect("c"),
        tx.cursor_read::<V2StoragesTrieHistory>().expect("c"),
        tx.cursor_read::<V2StoragesTrieHistory>().expect("c"),
        tx.cursor_dup_read::<V2StorageTrieChangeSets>().expect("c"),
        addr_a,
        3,
        true,
    );

    let out = TrieCursor::seek(&mut cur, Nibbles::default()).expect("ok").expect("p1");
    assert_eq!(out.0, p1);
    assert!(
        TrieCursor::next(&mut cur).expect("ok").is_none(),
        "must not leak addr_b's history into addr_a"
    );
}

#[test]
fn storage_trie_cursor_set_hashed_address_resets_merge_state() {
    // After set_hashed_address, the merge state must be reset so seek/next
    // operate correctly on the new address.
    let db = setup_db();
    let addr_a = B256::from([0x55; 32]);
    let addr_b = B256::from([0x66; 32]);
    let p1 = Nibbles::from_nibbles([0x01]);
    let p2 = Nibbles::from_nibbles([0x02]);
    let n = node();

    {
        let wtx = db.tx_mut().expect("rw tx");
        let mut c = wtx.cursor_dup_write::<V2StoragesTrie>().expect("c");
        c.upsert(addr_a, &StorageTrieEntry { nibbles: StoredNibblesSubKey(p1), node: n.clone() })
            .expect("upsert");
        c.upsert(addr_b, &StorageTrieEntry { nibbles: StoredNibblesSubKey(p2), node: n })
            .expect("upsert");
        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");
    let mut cur = V2StorageTrieCursor::new(
        tx.cursor_dup_read::<V2StoragesTrie>().expect("c"),
        tx.cursor_read::<V2StoragesTrieHistory>().expect("c"),
        tx.cursor_read::<V2StoragesTrieHistory>().expect("c"),
        tx.cursor_dup_read::<V2StorageTrieChangeSets>().expect("c"),
        addr_a,
        u64::MAX,
        true,
    );

    // Seek on addr_a
    let out = TrieCursor::seek(&mut cur, p1).expect("ok").expect("p1");
    assert_eq!(out.0, p1);

    // Switch to addr_b
    cur.set_hashed_address(addr_b);
    let out = TrieCursor::seek(&mut cur, p2).expect("ok").expect("p2");
    assert_eq!(out.0, p2);

    assert!(TrieCursor::next(&mut cur).expect("ok").is_none());
}

// ====================== Storage Cursor tests ======================

#[test]
fn storage_cursor_reads_current_state_when_no_history() {
    let db = setup_db();
    let addr = B256::from([0xAA; 32]);
    let slot = B256::from([0x01; 32]);

    {
        let wtx = db.tx_mut().expect("rw tx");
        wtx.cursor_dup_write::<V2HashedStorages>()
            .expect("c")
            .upsert(addr, &StorageEntry { key: slot, value: U256::from(42u64) })
            .expect("upsert");
        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");
    let mut cur = V2StorageCursor::new(
        tx.cursor_dup_read::<V2HashedStorages>().expect("c"),
        tx.cursor_read::<V2HashedStoragesHistory>().expect("c"),
        tx.cursor_read::<V2HashedStoragesHistory>().expect("c"),
        tx.cursor_dup_read::<V2HashedStorageChangeSets>().expect("c"),
        addr,
        u64::MAX,
        true,
    );

    let result = cur.seek(slot).expect("ok").expect("should find");
    assert_eq!(result, (slot, U256::from(42u64)));
}

#[test]
fn storage_cursor_resolves_from_changeset() {
    let db = setup_db();
    let addr = B256::from([0xAA; 32]);
    let slot = B256::from([0x01; 32]);

    {
        let wtx = db.tx_mut().expect("rw tx");

        // Current state: value=1000
        wtx.cursor_dup_write::<V2HashedStorages>()
            .expect("c")
            .upsert(addr, &StorageEntry { key: slot, value: U256::from(1000u64) })
            .expect("upsert");

        // History: modified at block 8
        wtx.cursor_write::<V2HashedStoragesHistory>()
            .expect("c")
            .upsert(
                HashedStorageShardedKey {
                    hashed_address: addr,
                    sharded_key: ShardedKey::new(slot, u64::MAX),
                },
                &BlockNumberList::new_pre_sorted([8]),
            )
            .expect("upsert");

        // Changeset at block 8: old value was 500
        let cs_key = BlockNumberHashedAddress((8u64, addr));
        wtx.cursor_dup_write::<V2HashedStorageChangeSets>()
            .expect("c")
            .append_dup(cs_key, StorageEntry { key: slot, value: U256::from(500u64) })
            .expect("append");

        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");

    // Query at block 7 (before modification at 8)
    let mut cur = V2StorageCursor::new(
        tx.cursor_dup_read::<V2HashedStorages>().expect("c"),
        tx.cursor_read::<V2HashedStoragesHistory>().expect("c"),
        tx.cursor_read::<V2HashedStoragesHistory>().expect("c"),
        tx.cursor_dup_read::<V2HashedStorageChangeSets>().expect("c"),
        addr,
        7,
        false,
    );

    let result = cur.seek(slot).expect("ok").expect("should find");
    assert_eq!(result.0, slot);
    assert_eq!(result.1, U256::from(500u64), "should get changeset value");
}

#[test]
fn storage_cursor_is_storage_empty() {
    let db = setup_db();
    let addr_with = B256::from([0xBB; 32]);
    let addr_without = B256::from([0xCC; 32]);

    {
        let wtx = db.tx_mut().expect("rw tx");
        wtx.cursor_dup_write::<V2HashedStorages>()
            .expect("c")
            .upsert(
                addr_with,
                &StorageEntry { key: B256::from([0x01; 32]), value: U256::from(1u64) },
            )
            .expect("upsert");
        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");

    let mut cur_with = V2StorageCursor::new(
        tx.cursor_dup_read::<V2HashedStorages>().expect("c"),
        tx.cursor_read::<V2HashedStoragesHistory>().expect("c"),
        tx.cursor_read::<V2HashedStoragesHistory>().expect("c"),
        tx.cursor_dup_read::<V2HashedStorageChangeSets>().expect("c"),
        addr_with,
        u64::MAX,
        true,
    );
    assert!(!cur_with.is_storage_empty().expect("ok"));

    let mut cur_without = V2StorageCursor::new(
        tx.cursor_dup_read::<V2HashedStorages>().expect("c"),
        tx.cursor_read::<V2HashedStoragesHistory>().expect("c"),
        tx.cursor_read::<V2HashedStoragesHistory>().expect("c"),
        tx.cursor_dup_read::<V2HashedStorageChangeSets>().expect("c"),
        addr_without,
        u64::MAX,
        true,
    );
    assert!(cur_without.is_storage_empty().expect("ok"));
}

/// Storage slot was zeroed (deleted from `V2HashedStorages`) after the target
/// block. The history walk must discover it.
#[test]
fn storage_cursor_discovers_slot_deleted_after_target_block() {
    let db = setup_db();
    let addr = B256::from([0xAA; 32]);
    let s1 = B256::from([0x01; 32]);
    let s2 = B256::from([0x02; 32]); // deleted after target
    let s3 = B256::from([0x03; 32]);

    {
        let wtx = db.tx_mut().expect("rw tx");
        let mut c = wtx.cursor_dup_write::<V2HashedStorages>().expect("c");
        // s1 and s3 exist; s2 was zeroed at block 10
        c.upsert(addr, &StorageEntry { key: s1, value: U256::from(100u64) }).expect("upsert");
        c.upsert(addr, &StorageEntry { key: s3, value: U256::from(300u64) }).expect("upsert");

        // s2 history: modified at [5, 10]
        wtx.cursor_write::<V2HashedStoragesHistory>()
            .expect("c")
            .upsert(
                HashedStorageShardedKey {
                    hashed_address: addr,
                    sharded_key: ShardedKey::new(s2, u64::MAX),
                },
                &BlockNumberList::new_pre_sorted([5, 10]),
            )
            .expect("upsert");

        // Changeset at block 10: s2 = 200 before block 10
        let cs_key = BlockNumberHashedAddress((10u64, addr));
        wtx.cursor_dup_write::<V2HashedStorageChangeSets>()
            .expect("c")
            .append_dup(cs_key, StorageEntry { key: s2, value: U256::from(200u64) })
            .expect("append");

        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");
    let mut cur = V2StorageCursor::new(
        tx.cursor_dup_read::<V2HashedStorages>().expect("c"),
        tx.cursor_read::<V2HashedStoragesHistory>().expect("c"),
        tx.cursor_read::<V2HashedStoragesHistory>().expect("c"),
        tx.cursor_dup_read::<V2HashedStorageChangeSets>().expect("c"),
        addr,
        9,
        false,
    );

    let r1 = cur.seek(B256::ZERO).expect("ok").expect("s1");
    assert_eq!(r1.0, s1);
    assert_eq!(r1.1, U256::from(100u64));

    let r2 = cur.next().expect("ok").expect("s2 from history");
    assert_eq!(r2.0, s2);
    assert_eq!(r2.1, U256::from(200u64));

    let r3 = cur.next().expect("ok").expect("s3");
    assert_eq!(r3.0, s3);
    assert_eq!(r3.1, U256::from(300u64));

    assert!(cur.next().expect("ok").is_none());
}

/// `is_storage_empty` must return false when storage existed at the target
/// block but has since been wiped from current state.
#[test]
fn storage_cursor_is_storage_empty_false_for_historical_only_slots() {
    let db = setup_db();
    let addr = B256::from([0xDD; 32]);
    let slot = B256::from([0x01; 32]);

    {
        let wtx = db.tx_mut().expect("rw tx");
        // No current state for addr — all storage was wiped at block 10.

        // History: slot modified at [5, 10]
        wtx.cursor_write::<V2HashedStoragesHistory>()
            .expect("c")
            .upsert(
                HashedStorageShardedKey {
                    hashed_address: addr,
                    sharded_key: ShardedKey::new(slot, u64::MAX),
                },
                &BlockNumberList::new_pre_sorted([5, 10]),
            )
            .expect("upsert");

        // Changeset at 10: value=42 before block 10
        let cs_key = BlockNumberHashedAddress((10u64, addr));
        wtx.cursor_dup_write::<V2HashedStorageChangeSets>()
            .expect("c")
            .append_dup(cs_key, StorageEntry { key: slot, value: U256::from(42u64) })
            .expect("append");

        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");
    let mut cur = V2StorageCursor::new(
        tx.cursor_dup_read::<V2HashedStorages>().expect("c"),
        tx.cursor_read::<V2HashedStoragesHistory>().expect("c"),
        tx.cursor_read::<V2HashedStoragesHistory>().expect("c"),
        tx.cursor_dup_read::<V2HashedStorageChangeSets>().expect("c"),
        addr,
        9,
        false,
    );

    assert!(!cur.is_storage_empty().expect("ok"), "should find historical slot");
}

/// History-only storage key must NOT cross into a different address.
#[test]
fn storage_cursor_deleted_slot_does_not_cross_address() {
    let db = setup_db();
    let addr1 = B256::from([0x01; 32]);
    let addr2 = B256::from([0x02; 32]);
    let slot = B256::from([0x0A; 32]);

    {
        let wtx = db.tx_mut().expect("rw tx");
        // No current storage for addr1.
        // addr2 has a history entry for the same slot.
        wtx.cursor_write::<V2HashedStoragesHistory>()
            .expect("c")
            .upsert(
                HashedStorageShardedKey {
                    hashed_address: addr2,
                    sharded_key: ShardedKey::new(slot, u64::MAX),
                },
                &BlockNumberList::new_pre_sorted([5]),
            )
            .expect("upsert");

        let cs_key = BlockNumberHashedAddress((5u64, addr2));
        wtx.cursor_dup_write::<V2HashedStorageChangeSets>()
            .expect("c")
            .append_dup(cs_key, StorageEntry { key: slot, value: U256::from(99u64) })
            .expect("append");

        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");
    let mut cur = V2StorageCursor::new(
        tx.cursor_dup_read::<V2HashedStorages>().expect("c"),
        tx.cursor_read::<V2HashedStoragesHistory>().expect("c"),
        tx.cursor_read::<V2HashedStoragesHistory>().expect("c"),
        tx.cursor_dup_read::<V2HashedStorageChangeSets>().expect("c"),
        addr1,
        4,
        true,
    );

    // addr1 has no storage — history walk finds addr2's entry but must filter it out.
    assert!(cur.seek(B256::ZERO).expect("ok").is_none());
}

/// `set_hashed_address` resets merge state so the cursor works correctly
/// for the new address.
#[test]
fn storage_cursor_set_hashed_address_resets_merge_state() {
    let db = setup_db();
    let addr1 = B256::from([0x01; 32]);
    let addr2 = B256::from([0x02; 32]);
    let slot = B256::from([0x0A; 32]);

    {
        let wtx = db.tx_mut().expect("rw tx");
        let mut c = wtx.cursor_dup_write::<V2HashedStorages>().expect("c");
        c.upsert(addr1, &StorageEntry { key: slot, value: U256::from(11u64) }).expect("upsert");
        c.upsert(addr2, &StorageEntry { key: slot, value: U256::from(22u64) }).expect("upsert");
        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");
    let mut cur = V2StorageCursor::new(
        tx.cursor_dup_read::<V2HashedStorages>().expect("c"),
        tx.cursor_read::<V2HashedStoragesHistory>().expect("c"),
        tx.cursor_read::<V2HashedStoragesHistory>().expect("c"),
        tx.cursor_dup_read::<V2HashedStorageChangeSets>().expect("c"),
        addr1,
        u64::MAX,
        true,
    );

    let r1 = cur.seek(B256::ZERO).expect("ok").expect("addr1 slot");
    assert_eq!(r1.1, U256::from(11u64));

    cur.set_hashed_address(addr2);
    let r2 = cur.seek(B256::ZERO).expect("ok").expect("addr2 slot");
    assert_eq!(r2.1, U256::from(22u64));
}

// Regression: root trie path [] with child path [0] history.
// Before the length-prefixed encoding fix, the V2AccountTrieCursor would return
// the current-state root node instead of the historical one.
#[test]
fn account_trie_cursor_root_path_resolves_historical_with_child_paths() {
    let db = setup_db();
    let root_path = Nibbles::default();
    let child_path = Nibbles::from_nibbles([0]);

    let root_node_at_block5 =
        BranchNodeCompact::new(0b11, 0, 0, vec![], Some(B256::repeat_byte(0x55)));
    let root_node_at_block10 =
        BranchNodeCompact::new(0b111, 0, 0, vec![], Some(B256::repeat_byte(0xAA)));
    let child_node_at_block10 =
        BranchNodeCompact::new(0b101, 0, 0, vec![], Some(B256::repeat_byte(0xBB)));

    {
        let wtx = db.tx_mut().expect("rw tx");

        // Current state: root has block 10's node, child has block 10's node
        wtx.cursor_write::<V2AccountsTrie>()
            .expect("c")
            .upsert(StoredNibbles(root_path), &root_node_at_block10)
            .expect("upsert root");
        wtx.cursor_write::<V2AccountsTrie>()
            .expect("c")
            .upsert(StoredNibbles(child_path), &child_node_at_block10)
            .expect("upsert child");

        // Changeset at block 10: root had block5's node before block 10
        wtx.cursor_dup_write::<V2AccountTrieChangeSets>()
            .expect("c")
            .append_dup(
                10,
                TrieChangeSetsEntry {
                    nibbles: StoredNibblesSubKey(root_path),
                    node: Some(root_node_at_block5.clone()),
                },
            )
            .expect("append root cs");

        // History: root modified at block 10
        wtx.cursor_write::<V2AccountsTrieHistory>()
            .expect("c")
            .upsert(
                AccountTrieShardedKey::new(StoredNibbles(root_path), u64::MAX),
                &BlockNumberList::new_pre_sorted([10]),
            )
            .expect("upsert root history");

        // History: child [0] modified at block 10
        wtx.cursor_write::<V2AccountsTrieHistory>()
            .expect("c")
            .upsert(
                AccountTrieShardedKey::new(StoredNibbles(child_path), u64::MAX),
                &BlockNumberList::new_pre_sorted([10]),
            )
            .expect("upsert child history");

        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");
    let mut cur = V2AccountTrieCursor::new(
        tx.cursor_read::<V2AccountsTrie>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_read::<V2AccountsTrieHistory>().expect("c"),
        tx.cursor_dup_read::<V2AccountTrieChangeSets>().expect("c"),
        8,     // max_block_number: query at block 8 (before block 10's change)
        false, // is_latest = false (historical query)
    );

    // seek_exact on root path should return the historical value (block 5's node)
    let out = TrieCursor::seek_exact(&mut cur, root_path)
        .expect("ok")
        .expect("root should exist at block 8");
    assert_eq!(
        out.1, root_node_at_block5,
        "Root path should return the historical node from changeset, not current state"
    );
}

// ====================== Storage Trie Cursor tests ======================

#[test]
fn storage_trie_cursor_reads_current_state_when_no_history() {
    let db = setup_db();
    let addr = B256::from([0x55; 32]);
    let path = Nibbles::from_nibbles([0x0D]);

    {
        let wtx = db.tx_mut().expect("rw tx");
        wtx.cursor_dup_write::<V2StoragesTrie>()
            .expect("c")
            .upsert(addr, &StorageTrieEntry { nibbles: StoredNibblesSubKey(path), node: node() })
            .expect("upsert");
        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");
    let mut cur = V2StorageTrieCursor::new(
        tx.cursor_dup_read::<V2StoragesTrie>().expect("c"),
        tx.cursor_read::<V2StoragesTrieHistory>().expect("c"),
        tx.cursor_read::<V2StoragesTrieHistory>().expect("c"),
        tx.cursor_dup_read::<V2StorageTrieChangeSets>().expect("c"),
        addr,
        u64::MAX,
        true,
    );

    let out = TrieCursor::seek_exact(&mut cur, path).expect("ok").expect("some");
    assert_eq!(out.0, path);
}

#[test]
fn storage_trie_cursor_resolves_from_changeset() {
    let db = setup_db();
    let addr = B256::from([0x55; 32]);
    let path = Nibbles::from_nibbles([0x0D]);
    let old_node = node();
    let new_node = node2();

    {
        let wtx = db.tx_mut().expect("rw tx");

        // Current state
        wtx.cursor_dup_write::<V2StoragesTrie>()
            .expect("c")
            .upsert(addr, &StorageTrieEntry { nibbles: StoredNibblesSubKey(path), node: new_node })
            .expect("upsert");

        // History: modified at block 6
        wtx.cursor_write::<V2StoragesTrieHistory>()
            .expect("c")
            .upsert(
                StorageTrieShardedKey::new(addr, StoredNibbles(path), u64::MAX),
                &BlockNumberList::new_pre_sorted([6]),
            )
            .expect("upsert");

        // Changeset at block 6: old node
        let cs_key = BlockNumberHashedAddress((6u64, addr));
        let cs_entry = TrieChangeSetsEntry {
            nibbles: StoredNibblesSubKey(path),
            node: Some(old_node.clone()),
        };
        wtx.cursor_dup_write::<V2StorageTrieChangeSets>()
            .expect("c")
            .append_dup(cs_key, cs_entry)
            .expect("append");

        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");

    // Query at block 5 (before modification at 6)
    let mut cur = V2StorageTrieCursor::new(
        tx.cursor_dup_read::<V2StoragesTrie>().expect("c"),
        tx.cursor_read::<V2StoragesTrieHistory>().expect("c"),
        tx.cursor_read::<V2StoragesTrieHistory>().expect("c"),
        tx.cursor_dup_read::<V2StorageTrieChangeSets>().expect("c"),
        addr,
        5,
        false,
    );

    let out = TrieCursor::seek_exact(&mut cur, path).expect("ok").expect("some");
    assert_eq!(out.0, path);
    assert_eq!(out.1, old_node, "should get old node from changeset");
}

#[test]
fn storage_trie_cursor_respects_address_boundary() {
    let db = setup_db();
    let addr_a = B256::from([0x33; 32]);
    let addr_b = B256::from([0x44; 32]);
    let p1 = Nibbles::from_nibbles([0x05]);
    let p2 = Nibbles::from_nibbles([0x06]);

    {
        let wtx = db.tx_mut().expect("rw tx");
        let mut c = wtx.cursor_dup_write::<V2StoragesTrie>().expect("c");
        c.upsert(addr_a, &StorageTrieEntry { nibbles: StoredNibblesSubKey(p1), node: node() })
            .expect("upsert");
        c.upsert(addr_b, &StorageTrieEntry { nibbles: StoredNibblesSubKey(p2), node: node() })
            .expect("upsert");
        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");
    let mut cur = V2StorageTrieCursor::new(
        tx.cursor_dup_read::<V2StoragesTrie>().expect("c"),
        tx.cursor_read::<V2StoragesTrieHistory>().expect("c"),
        tx.cursor_read::<V2StoragesTrieHistory>().expect("c"),
        tx.cursor_dup_read::<V2StorageTrieChangeSets>().expect("c"),
        addr_a,
        u64::MAX,
        true,
    );

    let out = TrieCursor::seek(&mut cur, p1).expect("ok").expect("some");
    assert_eq!(out.0, p1);

    // next() should return None — crossed address boundary
    let out = TrieCursor::next(&mut cur).expect("ok");
    assert!(out.is_none(), "must not cross address boundary (DupSort)");
}

#[test]
fn storage_trie_cursor_set_hashed_address() {
    let db = setup_db();
    let addr_a = B256::from([0x55; 32]);
    let addr_b = B256::from([0x66; 32]);
    let path = Nibbles::from_nibbles([0x01]);

    {
        let wtx = db.tx_mut().expect("rw tx");
        let mut c = wtx.cursor_dup_write::<V2StoragesTrie>().expect("c");
        c.upsert(addr_a, &StorageTrieEntry { nibbles: StoredNibblesSubKey(path), node: node() })
            .expect("upsert");
        c.upsert(addr_b, &StorageTrieEntry { nibbles: StoredNibblesSubKey(path), node: node() })
            .expect("upsert");
        wtx.commit().expect("commit");
    }

    let tx = db.tx().expect("ro tx");
    let mut cur = V2StorageTrieCursor::new(
        tx.cursor_dup_read::<V2StoragesTrie>().expect("c"),
        tx.cursor_read::<V2StoragesTrieHistory>().expect("c"),
        tx.cursor_read::<V2StoragesTrieHistory>().expect("c"),
        tx.cursor_dup_read::<V2StorageTrieChangeSets>().expect("c"),
        addr_a,
        u64::MAX,
        true,
    );

    assert!(TrieCursor::seek_exact(&mut cur, path).expect("ok").is_some());
    cur.set_hashed_address(addr_b);
    assert!(TrieCursor::seek_exact(&mut cur, path).expect("ok").is_some());
}
