//! `op-reth proofs snapshot init|drop`: the trie-state snapshot the backfill reads through.

use crate::fixture::SeededDatadir;
use alloy_eips::BlockNumHash;
use reth_optimism_trie::{OpProofsStorageError, SnapshotInitStatus};
use rstest::rstest;

const CHAIN_BLOCKS: u64 = 8;
const EARLIEST: u64 = 4;

/// A datadir of 8 blocks whose proofs storage holds `[4, 8]`, backfilled without a snapshot.
fn storage_without_snapshot() -> SeededDatadir {
    let seeded = SeededDatadir::with_blocks(CHAIN_BLOCKS);
    seeded
        .run(&["init", "--proofs-history.window=4", "--proofs-history.use-snapshot=false"])
        .expect("init must succeed");
    assert_eq!(seeded.proof_window(), Some(seeded.window_of(EARLIEST, CHAIN_BLOCKS)));
    assert_no_snapshot(&seeded);
    seeded
}

fn anchor_at(seeded: &SeededDatadir, number: u64) -> BlockNumHash {
    BlockNumHash::new(number, seeded.block_hash(number))
}

fn assert_no_snapshot(seeded: &SeededDatadir) {
    let err = seeded.snapshot_anchor().expect_err("the storage must hold no snapshot");
    assert!(
        matches!(
            err,
            OpProofsStorageError::SnapshotNotReady { status: SnapshotInitStatus::NotStarted }
        ),
        "{err:?}"
    );
}

#[test]
fn snapshot_init_builds_a_ready_snapshot_at_the_earliest_block_by_default() {
    let seeded = storage_without_snapshot();

    seeded.run(&["snapshot", "init"]).expect("snapshot init must succeed");

    assert_eq!(seeded.snapshot_anchor().expect("ready snapshot"), anchor_at(&seeded, EARLIEST));
}

#[test]
fn snapshot_init_builds_a_ready_snapshot_at_the_target_block() {
    let seeded = storage_without_snapshot();

    seeded
        .run(&["snapshot", "init", "--proofs-history.snapshot-target-block=6"])
        .expect("snapshot init must succeed");

    assert_eq!(seeded.snapshot_anchor().expect("ready snapshot"), anchor_at(&seeded, 6));
}

#[rstest]
#[case::below_the_earliest_block(3)]
#[case::above_the_latest_block(9)]
fn snapshot_init_refuses_a_target_outside_the_window(#[case] target: u64) {
    let seeded = storage_without_snapshot();

    let err = seeded
        .run(&["snapshot", "init", &format!("--proofs-history.snapshot-target-block={target}")])
        .expect_err("the target must be refused");

    assert!(err.to_string().contains("outside proof window [4, 8]"), "{err}");
    assert_no_snapshot(&seeded);
}

#[rstest]
#[case::same_anchor(4)]
#[case::another_anchor(6)]
fn snapshot_init_refuses_to_rebuild_while_a_snapshot_is_ready(#[case] target: u64) {
    let seeded = storage_without_snapshot();
    seeded.run(&["snapshot", "init"]).expect("snapshot init must succeed");

    let err = seeded
        .run(&["snapshot", "init", &format!("--proofs-history.snapshot-target-block={target}")])
        .expect_err("a rebuild must be refused until the ready snapshot is dropped");

    assert!(err.to_string().contains("drop it before rebuilding"), "{err}");
    assert_eq!(seeded.snapshot_anchor().expect("ready snapshot"), anchor_at(&seeded, EARLIEST));
}

#[test]
fn snapshot_drop_removes_the_snapshot() {
    let seeded = storage_without_snapshot();
    seeded.run(&["snapshot", "init"]).expect("snapshot init must succeed");

    seeded.run(&["snapshot", "drop"]).expect("snapshot drop must succeed");

    assert_no_snapshot(&seeded);
    assert_eq!(seeded.proof_window(), Some(seeded.window_of(EARLIEST, CHAIN_BLOCKS)));
    seeded.run(&["snapshot", "drop"]).expect("dropping no snapshot must succeed");
}

#[test]
fn snapshot_drop_lets_a_snapshot_backfill_continue_after_a_plain_backfill_moved_the_window() {
    let seeded = SeededDatadir::with_blocks(CHAIN_BLOCKS);
    seeded.run(&["init", "--proofs-history.skip-backfill"]).expect("init must succeed");
    seeded.run(&["backfill", "--proofs-history.window=2"]).expect("snapshot backfill must succeed");
    assert_eq!(seeded.proof_window(), Some(seeded.window_of(6, CHAIN_BLOCKS)));
    assert_eq!(seeded.snapshot_anchor().expect("ready snapshot"), anchor_at(&seeded, 6));

    seeded
        .run(&["backfill", "--proofs-history.window=4", "--proofs-history.use-snapshot=false"])
        .expect("plain backfill must succeed");
    assert_eq!(seeded.proof_window(), Some(seeded.window_of(4, CHAIN_BLOCKS)));
    assert_eq!(
        seeded.snapshot_anchor().expect("ready snapshot"),
        anchor_at(&seeded, 6),
        "a plain backfill leaves the snapshot where it was"
    );

    let err = seeded
        .run(&["backfill", "--proofs-history.window=6"])
        .expect_err("a snapshot backfill must refuse a snapshot anchored off the earliest block");
    assert!(err.to_string().to_lowercase().contains("snapshot"), "{err}");
    assert_eq!(seeded.proof_window(), Some(seeded.window_of(4, CHAIN_BLOCKS)));

    seeded.run(&["snapshot", "drop"]).expect("snapshot drop must succeed");
    seeded.run(&["backfill", "--proofs-history.window=6"]).expect("snapshot backfill must succeed");

    assert_eq!(seeded.proof_window(), Some(seeded.window_of(2, CHAIN_BLOCKS)));
    assert_eq!(seeded.snapshot_anchor().expect("ready snapshot"), anchor_at(&seeded, 2));
}
