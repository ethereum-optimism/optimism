//! `op-reth proofs prune`: moving the proofs storage's earliest block up to a window behind its
//! latest block.

use crate::fixture::SeededDatadir;
use rstest::rstest;

const CHAIN_BLOCKS: u64 = 8;

/// A datadir of 8 blocks whose proofs storage holds `[2, 8]`.
fn seeded_storage() -> SeededDatadir {
    let seeded = SeededDatadir::with_blocks(CHAIN_BLOCKS);
    seeded.run(&["init", "--proofs-history.window=6"]).expect("init must succeed");
    assert_eq!(seeded.proof_window(), Some(seeded.window_of(2, CHAIN_BLOCKS)));
    seeded
}

#[rstest]
#[case::default_batch_size(&[], 3, 5)]
#[case::batch_size_one(&["--proofs-history.prune-batch-size=1"], 3, 5)]
#[case::batch_size_two(&["--proofs-history.prune-batch-size=2"], 3, 5)]
#[case::window_covering_the_storage(&[], 6, 2)]
#[case::window_wider_than_the_storage(&[], 10, 2)]
fn prune_keeps_a_window_of_blocks_behind_the_latest_block(
    #[case] extra_args: &[&str],
    #[case] window: u64,
    #[case] expected_earliest: u64,
) {
    let seeded = seeded_storage();

    let window_arg = format!("--proofs-history.window={window}");
    let mut args = vec!["prune", window_arg.as_str()];
    args.extend_from_slice(extra_args);
    seeded.run(&args).expect("prune must succeed");

    assert_eq!(seeded.proof_window(), Some(seeded.window_of(expected_earliest, CHAIN_BLOCKS)));
}

#[test]
fn prune_rejects_a_batch_size_of_zero() {
    let seeded = seeded_storage();

    let err = seeded
        .run(&["prune", "--proofs-history.window=3", "--proofs-history.prune-batch-size=0"])
        .expect_err("a batch size of zero must be rejected");

    assert!(err.to_string().contains("--proofs-history.prune-batch-size"), "{err}");
    assert_eq!(seeded.proof_window(), Some(seeded.window_of(2, CHAIN_BLOCKS)));
}

#[test]
fn prune_fails_when_the_new_earliest_block_is_not_in_reth() {
    let seeded = seeded_storage();
    seeded.append_store_only_block();
    let store_tip = seeded.append_store_only_block();

    let err = seeded
        .run(&["prune", "--proofs-history.window=1"])
        .expect_err("pruning up to a block reth does not hold must fail");

    assert!(err.to_string().contains('9'), "the error must name the missing block: {err}");
    let window = seeded.proof_window().expect("the storage keeps its window");
    assert_eq!(window.earliest, seeded.window_of(2, CHAIN_BLOCKS).earliest);
    assert_eq!(window.latest, store_tip);
}
