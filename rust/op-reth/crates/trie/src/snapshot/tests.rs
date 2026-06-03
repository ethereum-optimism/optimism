//! End-to-end tests for [`SnapshotInitJob`].
//!
//! Reuses chain-construction helpers from [`crate::backfill::tests`] to
//! produce a real reth-side chain + initialized v2 proofs storage, then drives
//! the snapshot init job and asserts the resulting state.
//!
//! The job's own [`SnapshotInitJob::validate_state_root`] is the strongest
//! correctness check: a successful run means the computed root from the
//! snapshot tables + live hashed leaves matches reth's header at `target`.
//! These tests therefore focus on lifecycle behavior (outcome shape, refusal
//! to redo work, target-window validation) rather than table inspection.
//!
//! [`SnapshotInitJob::validate_state_root`]: super::job

use super::{SnapshotError, SnapshotInitJob};
use crate::{
    MdbxProofsStorageV2, OpProofsBackfillProvider, OpProofsBackfillStore, OpProofsProviderRO,
    OpProofsSnapshotInitProvider, OpProofsSnapshotProviderRO, OpProofsStore, SnapshotInitStatus,
    test_utils::{
        build_chain_and_initialize_storage, build_chain_with_storage_writes_and_initialize_storage,
    },
};
use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use reth_provider::DatabaseProviderFactory;
use reth_trie::{Nibbles, trie_cursor::TrieCursor};
use serial_test::serial;
use std::sync::Arc;

/// Count rows the history-aware `account_trie_cursor` would yield at
/// `target_block` — i.e., the number of entries the snapshot job's
/// `drain_account_trie` sees as input.
fn count_source_account_trie(storage: &Arc<MdbxProofsStorageV2>, target_block: u64) -> usize {
    let provider = storage.provider_ro().expect("ro");
    let mut cursor = provider.account_trie_cursor(target_block).expect("cursor");
    let mut n = 0usize;
    let mut entry = cursor.seek(Nibbles::default()).expect("seek");
    while entry.is_some() {
        n += 1;
        entry = cursor.next().expect("next");
    }
    n
}

/// Count rows in the destination `V2AccountsTrieSnapshot` table via the
/// snapshot reader cursor.
fn count_snapshot_account_trie(storage: &Arc<MdbxProofsStorageV2>) -> usize {
    let sp = storage.snapshot_provider_ro().expect("ro");
    let mut cursor = sp.snapshot_account_trie_cursor().expect("cursor");
    let mut n = 0usize;
    let mut entry = cursor.seek(Nibbles::default()).expect("seek");
    while entry.is_some() {
        n += 1;
        entry = cursor.next().expect("next");
    }
    n
}

#[test]
#[serial]
fn snapshot_init_at_latest_completes_and_anchor_matches() {
    // 3-block chain; storage initialized at block 3 (earliest = latest = 3).
    let (provider_factory, storage, latest_num, latest_hash) =
        build_chain_and_initialize_storage(3);
    let target = BlockNumHash::new(latest_num, latest_hash);

    // Drive the snapshot init job at `target`.
    let reth_provider = provider_factory.database_provider_ro().unwrap();
    let outcome =
        SnapshotInitJob::new(reth_provider, storage.clone()).run(latest_num).expect("snapshot");

    assert_eq!(outcome.block, target);
    assert_eq!(outcome.status, SnapshotInitStatus::Completed);

    // Invariant: the snapshot drained every account-trie row visible to the
    // history-aware cursor at `target`, and the destination table now mirrors
    // that count exactly. Robust to chain size: if the source trie has zero
    // branches (legitimate for a small genesis with random-prefix accounts),
    // all three counts are zero; otherwise they're all equal and non-zero.
    let source_count = count_source_account_trie(&storage, latest_num);
    let dest_count = count_snapshot_account_trie(&storage);
    assert_eq!(
        outcome.account_nodes_copied as usize, source_count,
        "outcome count mismatch (source has {source_count} account-trie rows)"
    );
    assert_eq!(dest_count, source_count, "snapshot table doesn't match source");

    // After completion the snapshot is Ready at `target`.
    let sp = storage.snapshot_provider_ro().unwrap();
    let anchor = sp.snapshot_anchor().expect("ready");
    assert_eq!(anchor, target);
}

#[test]
#[serial]
fn snapshot_init_target_outside_window_errors() {
    let (provider_factory, storage, _latest_num, _latest_hash) =
        build_chain_and_initialize_storage(3);

    // earliest = latest = 3; target = 4 is past `latest`.
    let reth_provider = provider_factory.database_provider_ro().unwrap();
    let err = SnapshotInitJob::new(reth_provider, storage).run(4).unwrap_err();
    assert!(
        matches!(
            err,
            SnapshotError::SnapshotInitTargetOutsideWindow {
                target_block: 4,
                earliest: 3,
                latest: 3,
            }
        ),
        "got {err:?}"
    );
}

#[test]
#[serial]
fn snapshot_init_refuses_second_run_when_completed() {
    let (provider_factory, storage, latest_num, _latest_hash) =
        build_chain_and_initialize_storage(3);

    // First run: succeeds.
    let reth_provider = provider_factory.database_provider_ro().unwrap();
    SnapshotInitJob::new(reth_provider, storage.clone()).run(latest_num).expect("first run");

    // Second run on the same target: snapshot is already Completed, so the
    // job must refuse rather than redo the work.
    let reth_provider = provider_factory.database_provider_ro().unwrap();
    let err = SnapshotInitJob::new(reth_provider, storage).run(latest_num).unwrap_err();
    match err {
        SnapshotError::SnapshotAlreadyExists { existing_block, existing_status } => {
            assert_eq!(existing_block, latest_num);
            assert_eq!(existing_status, SnapshotInitStatus::Completed);
        }
        other => panic!("expected SnapshotAlreadyExists, got {other:?}"),
    }
}

#[test]
#[serial]
fn snapshot_init_drift_detection_aborts_run() {
    // Build a chain and plant a `Building` meta at a *different* anchor than
    // the one the job will compute for `latest`. The classify step must
    // notice the mismatch and bail with SnapshotResumeDriftDetected.
    let (provider_factory, storage, latest_num, _latest_hash) =
        build_chain_and_initialize_storage(3);

    // Plant Building meta at a fabricated anchor (different block number, so
    // it can't possibly match the target the job derives for `latest_num`).
    let planted_anchor = BlockNumHash::new(99, B256::repeat_byte(0xFE));
    {
        let sp = storage.snapshot_initialization_provider().expect("init");
        sp.set_snapshot_init_anchor(planted_anchor).expect("plant");
        OpProofsSnapshotInitProvider::commit(sp).expect("commit");
    }

    let reth_provider = provider_factory.database_provider_ro().unwrap();
    let err = SnapshotInitJob::new(reth_provider, storage).run(latest_num).unwrap_err();
    match err {
        SnapshotError::SnapshotResumeDriftDetected { anchor_block, .. } => {
            assert_eq!(anchor_block, planted_anchor.number);
        }
        other => panic!("expected SnapshotResumeDriftDetected, got {other:?}"),
    }
}

#[test]
#[serial]
fn snapshot_init_succeeds_on_chain_with_storage_writes() {
    // Drive the job over a chain whose every block touches a storage slot.
    // This exercises the storage-trie phase (`drain_storage_trie` +
    // `collect_storage_chunk`) end-to-end, including its interaction with
    // `account_hashed_cursor` and per-address `storage_trie_cursor`.
    //
    // We don't assert `storage_nodes_copied > 0`: each block writes the same
    // single slot of one contract, so the storage trie is a single leaf with
    // no branch nodes — the snapshot can legitimately be empty. The job's
    // internal state-root validation is the real correctness check.
    let (provider_factory, storage, latest_num, _latest_hash) =
        build_chain_with_storage_writes_and_initialize_storage(3);

    let reth_provider = provider_factory.database_provider_ro().unwrap();
    let outcome = SnapshotInitJob::new(reth_provider, storage).run(latest_num).expect("snapshot");
    assert_eq!(outcome.status, SnapshotInitStatus::Completed);
}

#[test]
#[serial]
fn snapshot_init_with_small_chunk_size_drives_multi_chunk_drain() {
    let (provider_factory, storage, latest_num, latest_hash) =
        build_chain_with_storage_writes_and_initialize_storage(5);
    let target = BlockNumHash::new(latest_num, latest_hash);

    let reth_provider = provider_factory.database_provider_ro().unwrap();
    let outcome = SnapshotInitJob::new(reth_provider, storage.clone())
        .with_chunk_size(1)
        .run(latest_num)
        .expect("snapshot");

    assert_eq!(outcome.block, target);
    assert_eq!(outcome.status, SnapshotInitStatus::Completed);

    // Destination must match source row-for-row even across many tiny commits.
    let source_count = count_source_account_trie(&storage, latest_num);
    let dest_count = count_snapshot_account_trie(&storage);
    assert_eq!(
        outcome.account_nodes_copied as usize, source_count,
        "outcome count mismatch (source has {source_count} account-trie rows)"
    );
    assert_eq!(dest_count, source_count, "snapshot table doesn't match source");
}

#[test]
#[serial]
fn snapshot_init_clear_then_rebuild_succeeds() {
    let (provider_factory, storage, latest_num, latest_hash) =
        build_chain_and_initialize_storage(3);
    let target = BlockNumHash::new(latest_num, latest_hash);

    // First run: lands a Completed snapshot at `target`.
    {
        let reth_provider = provider_factory.database_provider_ro().unwrap();
        SnapshotInitJob::new(reth_provider, storage.clone()).run(latest_num).expect("first run");
    }

    // Drop the snapshot — status reverts to NotStarted as far as the init
    // anchor is concerned.
    {
        let sp = storage.backfill_provider().expect("rw");
        sp.clear_snapshot().expect("clear");
        OpProofsBackfillProvider::commit(sp).expect("commit");
    }

    // Second run must succeed (no SnapshotAlreadyExists) and produce a fresh
    // Completed snapshot at the same anchor.
    let reth_provider = provider_factory.database_provider_ro().unwrap();
    let outcome = SnapshotInitJob::new(reth_provider, storage).run(latest_num).expect("rebuild");
    assert_eq!(outcome.block, target);
    assert_eq!(outcome.status, SnapshotInitStatus::Completed);
}
