//! `op-reth proofs unwind`: removing the blocks above a target from the proofs storage.

use crate::fixture::SeededDatadir;
use rstest::rstest;

const CHAIN_BLOCKS: u64 = 8;
const EARLIEST: u64 = 3;

/// A datadir of 8 blocks whose proofs storage holds `[3, 8]`.
fn seeded_storage() -> SeededDatadir {
    let seeded = SeededDatadir::with_blocks(CHAIN_BLOCKS);
    seeded.run(&["init", "--proofs-history.window=5"]).expect("init must succeed");
    assert_eq!(seeded.proof_window(), Some(seeded.window_of(EARLIEST, CHAIN_BLOCKS)));
    seeded
}

#[rstest]
#[case::earliest(3, 3)]
#[case::earliest_plus_one(4, 4)]
#[case::latest_minus_one(7, 7)]
#[case::latest(8, 8)]
#[case::latest_plus_one(9, 8)]
fn unwind_removes_every_block_above_the_target(#[case] target: u64, #[case] expected_latest: u64) {
    let seeded = seeded_storage();

    seeded.run(&["unwind", &format!("--target={target}")]).expect("unwind must succeed");

    assert_eq!(seeded.proof_window(), Some(seeded.window_of(EARLIEST, expected_latest)));
}

#[test]
fn unwind_refuses_a_target_below_the_earliest_block() {
    let seeded = seeded_storage();

    let err = seeded.run(&["unwind", "--target=2"]).expect_err("the target must be refused");

    assert!(err.to_string().contains("below the earliest block 3"), "{err}");
    assert_eq!(seeded.proof_window(), Some(seeded.window_of(EARLIEST, CHAIN_BLOCKS)));
}

#[test]
fn unwind_refuses_a_target_whose_body_reth_pruned() {
    let seeded = SeededDatadir::with_blocks_and_pruned_body(CHAIN_BLOCKS, 6);
    seeded.run(&["init", "--proofs-history.window=5"]).expect("init must succeed");

    let err = seeded.run(&["unwind", "--target=6"]).expect_err("the target must be refused");

    assert!(err.to_string().contains("body has been pruned"), "{err}");
    assert_eq!(seeded.proof_window(), Some(seeded.window_of(EARLIEST, CHAIN_BLOCKS)));
}

#[test]
fn unwind_to_the_reth_tip_recovers_a_storage_that_is_ahead_of_reth() {
    let seeded = seeded_storage();
    let store_only_block = seeded.append_store_only_block();
    assert_eq!(seeded.proof_window().map(|window| window.latest), Some(store_only_block));

    seeded.run(&["unwind", "--target=8"]).expect("unwind must succeed");

    assert_eq!(seeded.proof_window(), Some(seeded.window_of(EARLIEST, CHAIN_BLOCKS)));
}

#[test]
fn unwind_refuses_a_target_reth_does_not_hold() {
    let seeded = seeded_storage();
    seeded.append_store_only_block();
    let store_tip = seeded.append_store_only_block();

    let err = seeded.run(&["unwind", "--target=9"]).expect_err("the target must be refused");

    assert!(err.to_string().contains("Target block 9 not found"), "{err}");
    assert_eq!(seeded.proof_window().map(|window| window.latest), Some(store_tip));
}

#[test]
fn unwind_of_an_empty_storage_does_nothing() {
    let seeded = SeededDatadir::with_blocks(CHAIN_BLOCKS);

    seeded.run(&["unwind", "--target=3"]).expect("unwind must succeed");

    assert_eq!(seeded.proof_window(), None);
}
