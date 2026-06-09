//! Tests for the plain snapshot cursors over `V2*TrieSnapshot` tables.

use super::{V2AccountTrieSnapshotCursor, V2StorageTrieSnapshotCursor};
use crate::db::{
    models,
    models::{V2AccountsTrieSnapshot, V2StoragesTrieSnapshot},
};
use alloy_primitives::B256;
use reth_db::{
    Database, DatabaseEnv,
    cursor::DbCursorRW,
    mdbx::{DatabaseArguments, init_db_for},
    transaction::{DbTx, DbTxMut},
};
use reth_trie::{
    BranchNodeCompact, Nibbles, StorageTrieEntry, StoredNibbles, StoredNibblesSubKey,
    trie_cursor::{TrieCursor, TrieStorageCursor},
};
use tempfile::TempDir;

// ---------- helpers ----------

fn setup_db() -> DatabaseEnv {
    let tmp = TempDir::new().expect("create tmpdir");
    init_db_for::<_, models::Tables>(tmp, DatabaseArguments::default()).expect("init db")
}

fn sample_node(tag: u8) -> BranchNodeCompact {
    BranchNodeCompact::new(0b1, 0, 0, vec![], Some(B256::repeat_byte(tag)))
}

/// Seed `V2AccountsTrieSnapshot` with `(path, node)` pairs.
fn seed_account_snapshot(db: &DatabaseEnv, rows: &[(Nibbles, BranchNodeCompact)]) {
    let wtx = db.tx_mut().expect("rw");
    let mut c = wtx.cursor_write::<V2AccountsTrieSnapshot>().expect("cursor");
    for (path, node) in rows {
        c.upsert(StoredNibbles(*path), node).expect("upsert");
    }
    wtx.commit().expect("commit");
}

/// Seed `V2StoragesTrieSnapshot` with per-address `(path, node)` pairs.
fn seed_storage_snapshot(db: &DatabaseEnv, rows: &[(B256, Nibbles, BranchNodeCompact)]) {
    let wtx = db.tx_mut().expect("rw");
    let mut c = wtx.cursor_dup_write::<V2StoragesTrieSnapshot>().expect("cursor");
    for (addr, path, node) in rows {
        c.upsert(
            *addr,
            &StorageTrieEntry { nibbles: StoredNibblesSubKey(*path), node: node.clone() },
        )
        .expect("upsert");
    }
    wtx.commit().expect("commit");
}

// ====================== V2AccountTrieSnapshotCursor ======================

#[test]
fn account_trie_snapshot_cursor_seek_exact_hit_and_miss() {
    let db = setup_db();
    let p1 = Nibbles::from_nibbles_unchecked([0x01]);
    let p2 = Nibbles::from_nibbles_unchecked([0x02]);
    seed_account_snapshot(&db, &[(p1, sample_node(0xAB))]);

    let tx = db.tx().expect("ro tx");
    let mut cur =
        V2AccountTrieSnapshotCursor::new(tx.cursor_read::<V2AccountsTrieSnapshot>().expect("c"));

    // Hit: exact match, returns the row and updates `current`.
    let (k, v) = TrieCursor::seek_exact(&mut cur, p1).expect("ok").expect("exists");
    assert_eq!(k, p1);
    assert_eq!(v, sample_node(0xAB));
    assert_eq!(TrieCursor::current(&mut cur).expect("current"), Some(p1));

    // Miss: no exact match → None, and `current` must NOT have advanced.
    assert!(TrieCursor::seek_exact(&mut cur, p2).expect("ok").is_none());
    assert_eq!(TrieCursor::current(&mut cur).expect("current"), Some(p1));
}

#[test]
fn account_trie_snapshot_cursor_seek_returns_gte() {
    let db = setup_db();
    let p1 = Nibbles::from_nibbles_unchecked([0x01]);
    let p3 = Nibbles::from_nibbles_unchecked([0x03]);
    let between = Nibbles::from_nibbles_unchecked([0x02]);
    seed_account_snapshot(&db, &[(p1, sample_node(0xAB)), (p3, sample_node(0xCD))]);

    let tx = db.tx().expect("ro tx");
    let mut cur =
        V2AccountTrieSnapshotCursor::new(tx.cursor_read::<V2AccountsTrieSnapshot>().expect("c"));

    // Seeking a key between rows lands on the next one.
    let (k, v) = TrieCursor::seek(&mut cur, between).expect("ok").expect("exists");
    assert_eq!(k, p3);
    assert_eq!(v, sample_node(0xCD));
    assert_eq!(TrieCursor::current(&mut cur).expect("current"), Some(p3));

    // Seeking past the last key returns None.
    let beyond = Nibbles::from_nibbles_unchecked([0x04]);
    assert!(TrieCursor::seek(&mut cur, beyond).expect("ok").is_none());
}

#[test]
fn account_trie_snapshot_cursor_next_walks_in_order() {
    let db = setup_db();
    let p1 = Nibbles::from_nibbles_unchecked([0x01]);
    let p2 = Nibbles::from_nibbles_unchecked([0x02]);
    let p3 = Nibbles::from_nibbles_unchecked([0x03]);
    seed_account_snapshot(
        &db,
        &[(p1, sample_node(0x11)), (p2, sample_node(0x22)), (p3, sample_node(0x33))],
    );

    let tx = db.tx().expect("ro tx");
    let mut cur =
        V2AccountTrieSnapshotCursor::new(tx.cursor_read::<V2AccountsTrieSnapshot>().expect("c"));

    let (k, _) = TrieCursor::seek(&mut cur, p1).expect("ok").expect("first");
    assert_eq!(k, p1);
    let (k, _) = TrieCursor::next(&mut cur).expect("ok").expect("second");
    assert_eq!(k, p2);
    let (k, _) = TrieCursor::next(&mut cur).expect("ok").expect("third");
    assert_eq!(k, p3);
    assert!(TrieCursor::next(&mut cur).expect("ok").is_none());
}

#[test]
fn account_trie_snapshot_cursor_reset_clears_current() {
    let db = setup_db();
    let p1 = Nibbles::from_nibbles_unchecked([0x01]);
    seed_account_snapshot(&db, &[(p1, sample_node(0xAB))]);

    let tx = db.tx().expect("ro tx");
    let mut cur =
        V2AccountTrieSnapshotCursor::new(tx.cursor_read::<V2AccountsTrieSnapshot>().expect("c"));
    TrieCursor::seek(&mut cur, p1).expect("ok");
    assert_eq!(TrieCursor::current(&mut cur).expect("current"), Some(p1));

    cur.reset();
    assert_eq!(TrieCursor::current(&mut cur).expect("current"), None);
}

// ====================== V2StorageTrieSnapshotCursor ======================

#[test]
fn storage_trie_snapshot_cursor_seek_exact_scoped_to_address() {
    let db = setup_db();
    let addr_a = B256::repeat_byte(0xAA);
    let addr_b = B256::repeat_byte(0xBB);
    let path = Nibbles::from_nibbles_unchecked([0x05]);
    // Only addr_a has `path`; addr_b has a different one. seek_exact under
    // addr_a's scope must find it and not leak addr_b's rows.
    let other = Nibbles::from_nibbles_unchecked([0x06]);
    seed_storage_snapshot(
        &db,
        &[(addr_a, path, sample_node(0xAB)), (addr_b, other, sample_node(0xCD))],
    );

    let tx = db.tx().expect("ro tx");
    let mut cur = V2StorageTrieSnapshotCursor::new(
        tx.cursor_dup_read::<V2StoragesTrieSnapshot>().expect("c"),
        addr_a,
    );

    let (k, v) = TrieCursor::seek_exact(&mut cur, path).expect("ok").expect("exists");
    assert_eq!(k, path);
    assert_eq!(v, sample_node(0xAB));

    // Path that only exists under addr_b must not be visible under addr_a's scope.
    assert!(TrieCursor::seek_exact(&mut cur, other).expect("ok").is_none());
}

#[test]
fn storage_trie_snapshot_cursor_seek_returns_gte_within_address() {
    let db = setup_db();
    let addr = B256::repeat_byte(0xCC);
    let p1 = Nibbles::from_nibbles_unchecked([0x01]);
    let p3 = Nibbles::from_nibbles_unchecked([0x03]);
    let between = Nibbles::from_nibbles_unchecked([0x02]);
    seed_storage_snapshot(&db, &[(addr, p1, sample_node(0xAB)), (addr, p3, sample_node(0xCD))]);

    let tx = db.tx().expect("ro tx");
    let mut cur = V2StorageTrieSnapshotCursor::new(
        tx.cursor_dup_read::<V2StoragesTrieSnapshot>().expect("c"),
        addr,
    );

    let (k, v) = TrieCursor::seek(&mut cur, between).expect("ok").expect("found");
    assert_eq!(k, p3);
    assert_eq!(v, sample_node(0xCD));
    assert_eq!(TrieCursor::current(&mut cur).expect("current"), Some(p3));
}

#[test]
fn storage_trie_snapshot_cursor_next_walks_dup_entries() {
    let db = setup_db();
    let addr = B256::repeat_byte(0xDD);
    let p1 = Nibbles::from_nibbles_unchecked([0x01]);
    let p2 = Nibbles::from_nibbles_unchecked([0x02]);
    let p3 = Nibbles::from_nibbles_unchecked([0x03]);
    seed_storage_snapshot(
        &db,
        &[
            (addr, p1, sample_node(0x11)),
            (addr, p2, sample_node(0x22)),
            (addr, p3, sample_node(0x33)),
        ],
    );

    let tx = db.tx().expect("ro tx");
    let mut cur = V2StorageTrieSnapshotCursor::new(
        tx.cursor_dup_read::<V2StoragesTrieSnapshot>().expect("c"),
        addr,
    );

    let (k, _) = TrieCursor::seek(&mut cur, p1).expect("ok").expect("first");
    assert_eq!(k, p1);
    let (k, _) = TrieCursor::next(&mut cur).expect("ok").expect("second");
    assert_eq!(k, p2);
    let (k, _) = TrieCursor::next(&mut cur).expect("ok").expect("third");
    assert_eq!(k, p3);
    assert!(TrieCursor::next(&mut cur).expect("ok").is_none());
}

#[test]
fn storage_trie_snapshot_cursor_set_hashed_address_rebinds_and_resets() {
    let db = setup_db();
    let addr_a = B256::repeat_byte(0xAA);
    let addr_b = B256::repeat_byte(0xBB);
    let path_a = Nibbles::from_nibbles_unchecked([0x01]);
    let path_b = Nibbles::from_nibbles_unchecked([0x02]);
    seed_storage_snapshot(
        &db,
        &[(addr_a, path_a, sample_node(0xAB)), (addr_b, path_b, sample_node(0xCD))],
    );

    let tx = db.tx().expect("ro tx");
    let mut cur = V2StorageTrieSnapshotCursor::new(
        tx.cursor_dup_read::<V2StoragesTrieSnapshot>().expect("c"),
        addr_a,
    );

    TrieCursor::seek(&mut cur, path_a).expect("ok");
    assert_eq!(TrieCursor::current(&mut cur).expect("current"), Some(path_a));

    cur.set_hashed_address(addr_b);
    // Rebind clears `current` and rescopes lookups.
    assert_eq!(TrieCursor::current(&mut cur).expect("current"), None);
    let (k, v) = TrieCursor::seek_exact(&mut cur, path_b).expect("ok").expect("exists");
    assert_eq!(k, path_b);
    assert_eq!(v, sample_node(0xCD));
    // path_a now belongs to a different address scope, so seek_exact returns None.
    assert!(TrieCursor::seek_exact(&mut cur, path_a).expect("ok").is_none());
}
