//! `op-reth proofs prune`: moving the proofs storage's earliest block up to a window behind its
//! latest block.

use crate::fixture::SeededDatadir;
use reth_optimism_trie::PrunerError;
use rstest::rstest;

const CHAIN_BLOCKS: u64 = 8;
const EARLIEST: u64 = 2;

/// A datadir of `CHAIN_BLOCKS` blocks whose proofs storage holds `[EARLIEST, CHAIN_BLOCKS]`.
fn seeded_storage() -> SeededDatadir {
    let seeded = SeededDatadir::with_blocks(CHAIN_BLOCKS);
    seeded.run_with_window("init", CHAIN_BLOCKS - EARLIEST, &[]).expect("init must succeed");
    assert_eq!(seeded.proof_window(), Some(seeded.window_of(EARLIEST, CHAIN_BLOCKS)));
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

    seeded.run_with_window("prune", window, extra_args).expect("prune must succeed");

    assert_eq!(seeded.proof_window(), Some(seeded.window_of(expected_earliest, CHAIN_BLOCKS)));
    seeded.verify_state_root_at(expected_earliest);
}

#[test]
fn prune_rejects_a_batch_size_of_zero() {
    let seeded = seeded_storage();

    let err = seeded
        .run(&["prune", "--proofs-history.window=3", "--proofs-history.prune-batch-size=0"])
        .expect_err("a batch size of zero must be rejected");

    let parse_error = err.downcast_ref::<clap::Error>().expect("rejected while parsing");
    assert_eq!(parse_error.kind(), clap::error::ErrorKind::ValueValidation, "{err}");
    assert!(err.to_string().contains("--proofs-history.prune-batch-size"), "{err}");
    assert_eq!(seeded.proof_window(), Some(seeded.window_of(EARLIEST, CHAIN_BLOCKS)));
}

#[test]
fn prune_of_an_empty_storage_does_nothing() {
    let seeded = SeededDatadir::with_blocks(CHAIN_BLOCKS);

    seeded.run(&["prune", "--proofs-history.window=3"]).expect("prune must succeed");

    assert_eq!(seeded.proof_window(), None);
}

/// A storage two blocks ahead of reth, the way an unclean shutdown leaves it: reth holds
/// `[0, CHAIN_BLOCKS]`, the storage `[EARLIEST, CHAIN_BLOCKS + 2]`.
fn seeded_storage_ahead_of_reth() -> SeededDatadir {
    let seeded = seeded_storage();
    seeded.append_store_only_block();
    seeded.append_store_only_block();
    seeded
}

/// A window of one block puts the new earliest block at `CHAIN_BLOCKS + 1`, which has no hash in
/// reth, so the prune fails and leaves the storage as it was.
#[test]
fn prune_fails_when_the_new_earliest_block_is_not_in_reth() {
    let seeded = seeded_storage_ahead_of_reth();
    let window_before = seeded.proof_window();

    let err = seeded
        .run(&["prune", "--proofs-history.window=1"])
        .expect_err("pruning up to a block reth does not hold must fail");

    assert!(
        matches!(
            err.downcast_ref::<PrunerError>(),
            Some(PrunerError::BlockNotFound(number)) if *number == CHAIN_BLOCKS + 1
        ),
        "{err}"
    );
    assert_eq!(seeded.proof_window(), window_before);
}

/// A window of one block puts the new earliest block at `CHAIN_BLOCKS + 1`, past reth's last
/// block; a pruner that never moves the earliest block past that one prunes up to it and
/// succeeds, with the storage's state there still matching reth's header.
#[test]
#[ignore = "holds once the pruner caps the new earliest block at reth's last block (#23358 on the reth 2.6 pin); until then the prune fails, see prune_fails_when_the_new_earliest_block_is_not_in_reth"]
fn prune_stops_at_reths_last_block_when_the_storage_is_ahead_of_reth() {
    let seeded = seeded_storage_ahead_of_reth();
    let store_tip = seeded.proof_window().expect("the storage holds blocks").latest;

    seeded.run(&["prune", "--proofs-history.window=1"]).expect("prune must succeed");

    let window = seeded.proof_window().expect("the storage keeps its window");
    assert_eq!(window.earliest, seeded.block_num_hash(CHAIN_BLOCKS));
    assert_eq!(window.latest, store_tip);
    seeded.verify_state_root_at(CHAIN_BLOCKS);
}
