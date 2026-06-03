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
    BackfillJob, InitializationJob, MdbxProofsStorageV2, OpProofsBackfillProvider,
    OpProofsBackfillStore, OpProofsProviderRO, OpProofsSnapshotInitProvider,
    OpProofsSnapshotProviderRO, OpProofsStore, RethTrieStorageLayout, SnapshotInitStatus,
    test_utils::{
        build_chain_and_initialize_storage, build_chain_with_storage_writes_and_initialize_storage,
        build_transfer_block, chain_spec_with_address, commit_block_to_database, create_storage,
        deterministic_keypair, execute_block, public_key_to_address,
    },
};
use alloy_eips::BlockNumHash;
use alloy_primitives::{Address, B256};
use reth_db::Database;
use reth_db_common::init::init_genesis;
use reth_provider::{
    DatabaseProviderFactory, StorageSettingsCache,
    test_utils::create_test_provider_factory_with_chain_spec,
};
use reth_trie::{Nibbles, StoredNibbles, trie_cursor::TrieCursor};
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

/// Negative test for the snapshot's validation safety net.
///
/// Without this, a bug in `validate_state_root` (wrong cursor factory, wrong
/// target block, inverted compare, …) would let every existing test pass
/// while silently marking a corrupt snapshot `Ready`.
#[test]
fn snapshot_init_aborts_with_state_root_mismatch_when_header_corrupted() {
    // Custom chain build — need to perturb the latest block's header before
    // commit, so we can't reuse `build_chain_and_initialize_storage`.
    let key_pair = deterministic_keypair();
    let sender = public_key_to_address(key_pair.public_key());
    let chain_spec = chain_spec_with_address(sender);
    let provider_factory = create_test_provider_factory_with_chain_spec(chain_spec.clone());
    init_genesis(&provider_factory).unwrap();

    let recipient = Address::repeat_byte(0x42);
    const NUM_BLOCKS: u64 = 3;
    // Snapshot validates state_root at `target_block` itself (unlike backfill,
    // which validates at block_number - 1), so we corrupt the target.
    const CORRUPTED_BLOCK: u64 = NUM_BLOCKS;
    const BOGUS_ROOT: B256 = B256::repeat_byte(0xAB);

    let mut last_hash = chain_spec.genesis_hash();
    for n in 1..=NUM_BLOCKS {
        let mut block = build_transfer_block(n, last_hash, &chain_spec, key_pair, n - 1, recipient);
        let exec = execute_block(&mut block, &provider_factory, &chain_spec);
        if n == CORRUPTED_BLOCK {
            block.set_state_root(BOGUS_ROOT);
        }
        commit_block_to_database(&block, &exec, &provider_factory);
        last_hash = block.hash();
    }

    let storage = create_storage();
    {
        let trie_layout = if provider_factory.cached_storage_settings().is_v2() {
            RethTrieStorageLayout::Packed
        } else {
            RethTrieStorageLayout::Legacy
        };
        let tx = provider_factory.db_ref().tx().unwrap();
        InitializationJob::new(storage.clone(), tx, trie_layout)
            .run(NUM_BLOCKS, last_hash)
            .unwrap();
    }

    let reth_provider = provider_factory.database_provider_ro().unwrap();
    let err = SnapshotInitJob::new(reth_provider, storage.clone()).run(NUM_BLOCKS).unwrap_err();
    match err {
        SnapshotError::StateRootMismatch { block_number, expected, .. } => {
            assert_eq!(
                block_number, NUM_BLOCKS,
                "validation fires at target_block (not target_block - 1)"
            );
            assert_eq!(expected, BOGUS_ROOT, "expected root must come from the tampered header");
        }
        other => panic!("expected StateRootMismatch, got {other:?}"),
    }

    let init_anchor = storage
        .snapshot_initialization_provider()
        .expect("init")
        .snapshot_init_anchor()
        .expect("anchor");
    assert_eq!(
        init_anchor.status,
        SnapshotInitStatus::InProgress,
        "meta must stay Building on validation failure",
    );
}

/// Resume happy path for [`SnapshotInitJob::start_or_resume`] — the
/// `InProgress` arm at the *matching* anchor
#[test]
fn snapshot_init_resumes_from_partial_building_at_matching_anchor() {
    let (provider_factory, storage, latest_num, latest_hash) =
        build_chain_with_storage_writes_and_initialize_storage(5);
    let target = BlockNumHash::new(latest_num, latest_hash);

    let source_count = count_source_account_trie(&storage, latest_num);
    assert!(
        source_count >= 2,
        "resume test needs ≥2 source rows; got {source_count}. \
         Enrich the chain helper (more genesis allocs, multi-slot storage) \
         to restore meaningful resume coverage.",
    );

    // Read the first source row — this is what we'll seed as a "previously
    // committed" chunk. Must match a real source row so the snapshot table
    // stays consistent with the source for state-root validation.
    let first_row = {
        let ro = storage.provider_ro().expect("ro");
        let mut cursor = ro.account_trie_cursor(latest_num).expect("cursor");
        let (path, node) = cursor.seek(Nibbles::default()).expect("seek").expect("first row");
        (StoredNibbles(path), node)
    };

    // Plant a partial Building snapshot at `target`: meta + one row.
    {
        let sp = storage.snapshot_initialization_provider().expect("init");
        sp.set_snapshot_init_anchor(target).expect("plant");
        sp.store_account_trie_snapshot_branches(vec![first_row.clone()]).expect("seed");
        OpProofsSnapshotInitProvider::commit(sp).expect("commit");
    }

    // Sanity: the planted state is what `start_or_resume` will read.
    let pre = storage
        .snapshot_initialization_provider()
        .expect("init")
        .snapshot_init_anchor()
        .expect("anchor");
    assert_eq!(pre.status, SnapshotInitStatus::InProgress);
    assert_eq!(pre.block, Some(target));
    assert_eq!(pre.last_account_trie_key, Some(first_row.0));

    // Resume. `chunk_size = 1` forces every remaining row through the
    // `Some(resume_after)` branch, not just the first iteration.
    let reth_provider = provider_factory.database_provider_ro().unwrap();
    let outcome = SnapshotInitJob::new(reth_provider, storage.clone())
        .with_chunk_size(1)
        .run(latest_num)
        .expect("resume");

    assert_eq!(outcome.block, target);
    assert_eq!(outcome.status, SnapshotInitStatus::Completed);
    assert_eq!(
        outcome.account_nodes_copied as usize,
        source_count - 1,
        "resumed run must not re-count the seeded row",
    );

    // Destination mirrors source — drain skipped the seeded row and
    // appended the rest with no duplicate-key error, no missing rows.
    assert_eq!(count_snapshot_account_trie(&storage), source_count);

    // `finalize_ready` ran — snapshot_anchor() returns `target` only when
    // meta is Ready.
    let post = storage.snapshot_provider_ro().expect("ro").snapshot_anchor().expect("ready");
    assert_eq!(post, target);
}

fn widen_window<F>(provider_factory: &F, storage: Arc<MdbxProofsStorageV2>, target_earliest: u64)
where
    F: DatabaseProviderFactory<
        Provider: reth_provider::DBProvider
                      + reth_provider::StageCheckpointReader
                      + reth_provider::ChangeSetReader
                      + reth_provider::StorageChangeSetReader
                      + reth_provider::BlockNumReader
                      + reth_provider::BlockHashReader
                      + reth_provider::HeaderProvider
                      + reth_provider::StorageSettingsCache
                      + Send,
    >,
{
    let provider = provider_factory.database_provider_ro().unwrap();
    BackfillJob::new(provider, storage).run(target_earliest).expect("backfill widens window");
}

/// Interior-target snapshot — the history-aware cursors at `target` actually
/// have to reconstruct trie state at a block below the tip.
#[test]
fn snapshot_init_at_interior_target() {
    // Build chain [1..=5] then backfill earliest to 2 → window [2, 5].
    let (provider_factory, storage, latest_num, _latest_hash) =
        build_chain_with_storage_writes_and_initialize_storage(5);
    assert_eq!(latest_num, 5);
    widen_window(&provider_factory, storage.clone(), 2);

    // Snapshot at interior block 3.
    const INTERIOR: u64 = 3;
    let reth_provider = provider_factory.database_provider_ro().unwrap();
    let outcome = SnapshotInitJob::new(reth_provider, storage.clone())
        .run(INTERIOR)
        .expect("interior snapshot");

    // `outcome.block.hash` resolves to reth's `block_hash(INTERIOR)`; the job
    // already validated the snapshot against `header[INTERIOR].state_root`
    // (not header[latest]), so reaching `Completed` means the historical
    // reconstruction agrees with reth's stored interior root.
    assert_eq!(outcome.block.number, INTERIOR);
    assert_eq!(outcome.status, SnapshotInitStatus::Completed);

    let sp = storage.snapshot_provider_ro().unwrap();
    let anchor = sp.snapshot_anchor().expect("ready");
    assert_eq!(anchor.number, INTERIOR);
}

/// Lower-bound rejection: `target_block < window.earliest` must fire
/// `SnapshotInitTargetOutsideWindow`. Existing
/// `snapshot_init_target_outside_window_errors` only hits the `> latest`
/// half; this test pins the other half of the same `if`.
#[test]
fn snapshot_init_target_below_earliest_errors() {
    // Window [2, 5] — running with target=1 falls below earliest.
    let (provider_factory, storage, latest_num, _latest_hash) =
        build_chain_with_storage_writes_and_initialize_storage(5);
    assert_eq!(latest_num, 5);
    widen_window(&provider_factory, storage.clone(), 2);

    let reth_provider = provider_factory.database_provider_ro().unwrap();
    let err = SnapshotInitJob::new(reth_provider, storage).run(1).unwrap_err();
    assert!(
        matches!(
            err,
            SnapshotError::SnapshotInitTargetOutsideWindow {
                target_block: 1,
                earliest: 2,
                latest: 5,
            }
        ),
        "got {err:?}",
    );
}

/// Inclusive window boundaries: both `target == earliest` and
/// `target == latest` must accept.
#[test]
fn snapshot_init_at_earliest_and_latest_boundaries_succeed() {
    // Earliest boundary (target == earliest = 2 in a [2, 5] window).
    {
        let (provider_factory, storage, latest_num, _latest_hash) =
            build_chain_with_storage_writes_and_initialize_storage(5);
        assert_eq!(latest_num, 5);
        widen_window(&provider_factory, storage.clone(), 2);

        let reth_provider = provider_factory.database_provider_ro().unwrap();
        let outcome =
            SnapshotInitJob::new(reth_provider, storage).run(2).expect("earliest boundary");
        assert_eq!(outcome.block.number, 2);
        assert_eq!(outcome.status, SnapshotInitStatus::Completed);
    }

    // Latest boundary (target == latest = 5 in a [2, 5] window).
    {
        let (provider_factory, storage, latest_num, _latest_hash) =
            build_chain_with_storage_writes_and_initialize_storage(5);
        assert_eq!(latest_num, 5);
        widen_window(&provider_factory, storage.clone(), 2);

        let reth_provider = provider_factory.database_provider_ro().unwrap();
        let outcome = SnapshotInitJob::new(reth_provider, storage).run(5).expect("latest boundary");
        assert_eq!(outcome.block.number, 5);
        assert_eq!(outcome.status, SnapshotInitStatus::Completed);
    }
}
