//! `op-reth proofs init`: seeding a proofs storage from a datadir.

use crate::fixture::SeededDatadir;
use rstest::rstest;

const CHAIN_BLOCKS: u64 = 6;

#[rstest]
#[case::snapshot_backfill(&[], 4, 2)]
#[case::plain_backfill(&["--proofs-history.use-snapshot=false"], 4, 2)]
#[case::window_reaching_below_genesis(&[], 20, 0)]
fn init_anchors_the_storage_at_the_datadir_tip_and_backfills_a_window(
    #[case] extra_args: &[&str],
    #[case] window: u64,
    #[case] expected_earliest: u64,
) {
    let seeded = SeededDatadir::with_blocks(CHAIN_BLOCKS);
    assert!(!seeded.storage_path().exists(), "the storage must not exist before the init");

    seeded.run_with_window("init", window, extra_args).expect("init must succeed");

    assert_eq!(seeded.proof_window(), Some(seeded.window_of(expected_earliest, CHAIN_BLOCKS)));
}

#[test]
fn init_with_skip_backfill_leaves_the_storage_at_the_datadir_tip_only() {
    let seeded = SeededDatadir::with_blocks(CHAIN_BLOCKS);

    seeded
        .run(&["init", "--proofs-history.window=4", "--proofs-history.skip-backfill"])
        .expect("init must succeed");

    assert_eq!(seeded.proof_window(), Some(seeded.window_of(CHAIN_BLOCKS, CHAIN_BLOCKS)));
}

#[test]
fn init_is_idempotent() {
    let seeded = SeededDatadir::with_blocks(CHAIN_BLOCKS);
    let args = ["init", "--proofs-history.window=4"];
    seeded.run(&args).expect("first init must succeed");
    let seeded_window = seeded.proof_window();

    seeded.run(&args).expect("a second init must succeed");

    assert_eq!(seeded.proof_window(), seeded_window);
    assert_eq!(seeded_window, Some(seeded.window_of(2, CHAIN_BLOCKS)));
}

#[test]
fn init_on_an_initialized_storage_only_extends_the_backfill() {
    let seeded = SeededDatadir::with_blocks(CHAIN_BLOCKS);
    seeded.run(&["init", "--proofs-history.window=2"]).expect("init must succeed");
    assert_eq!(seeded.proof_window(), Some(seeded.window_of(4, CHAIN_BLOCKS)));

    seeded.run(&["init", "--proofs-history.window=4"]).expect("a wider init must succeed");
    assert_eq!(seeded.proof_window(), Some(seeded.window_of(2, CHAIN_BLOCKS)));

    seeded.run(&["init", "--proofs-history.window=1"]).expect("a narrower init must succeed");
    assert_eq!(
        seeded.proof_window(),
        Some(seeded.window_of(2, CHAIN_BLOCKS)),
        "init never prunes the storage"
    );
}
