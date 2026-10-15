//! `op-reth proofs backfill`: extending the proofs storage's earliest block backward.

use crate::fixture::SeededDatadir;
use rstest::rstest;

const CHAIN_BLOCKS: u64 = 8;

/// A datadir of 8 blocks whose proofs storage holds `[8, 8]`.
fn anchored_storage() -> SeededDatadir {
    let seeded = SeededDatadir::with_blocks(CHAIN_BLOCKS);
    seeded.run(&["init", "--proofs-history.skip-backfill"]).expect("init must succeed");
    assert_eq!(seeded.proof_window(), Some(seeded.window_of(CHAIN_BLOCKS, CHAIN_BLOCKS)));
    seeded
}

#[rstest]
#[case::snapshot_within_the_chain(&[], 3, 5)]
#[case::plain_within_the_chain(&["--proofs-history.use-snapshot=false"], 3, 5)]
#[case::snapshot_reaching_below_genesis(&[], 20, 0)]
#[case::plain_reaching_below_genesis(&["--proofs-history.use-snapshot=false"], 20, 0)]
#[case::batch_size_one(&["--proofs-history.backfill-batch-size=1"], 3, 5)]
fn backfill_extends_the_earliest_block_to_a_window_behind_the_latest_block(
    #[case] extra_args: &[&str],
    #[case] window: u64,
    #[case] expected_earliest: u64,
) {
    let seeded = anchored_storage();

    let window_arg = format!("--proofs-history.window={window}");
    let mut args = vec!["backfill", window_arg.as_str()];
    args.extend_from_slice(extra_args);
    seeded.run(&args).expect("backfill must succeed");

    assert_eq!(seeded.proof_window(), Some(seeded.window_of(expected_earliest, CHAIN_BLOCKS)));
}

#[test]
fn backfill_does_nothing_when_the_storage_already_covers_the_window() {
    let seeded = anchored_storage();
    seeded.run(&["backfill", "--proofs-history.window=3"]).expect("backfill must succeed");
    assert_eq!(seeded.proof_window(), Some(seeded.window_of(5, CHAIN_BLOCKS)));

    seeded.run(&["backfill", "--proofs-history.window=2"]).expect("backfill must succeed");

    assert_eq!(seeded.proof_window(), Some(seeded.window_of(5, CHAIN_BLOCKS)));
}

#[test]
fn backfill_extends_an_existing_window_further_back() {
    let seeded = anchored_storage();
    seeded.run(&["backfill", "--proofs-history.window=3"]).expect("backfill must succeed");
    assert_eq!(seeded.proof_window(), Some(seeded.window_of(5, CHAIN_BLOCKS)));

    seeded.run(&["backfill", "--proofs-history.window=6"]).expect("backfill must succeed");

    assert_eq!(seeded.proof_window(), Some(seeded.window_of(2, CHAIN_BLOCKS)));
}
