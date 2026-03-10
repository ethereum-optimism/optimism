//! Fine-grained debugging assertions for derivation tests.

use crate::harness::test::DerivationTest;

/// Assert the total number of L2 blocks (including genesis).
pub fn assert_l2_block_count(test: &DerivationTest, expected: usize) {
    let actual = test.l2.blocks().len();
    assert_eq!(
        actual, expected,
        "expected {expected} L2 blocks (including genesis), got {actual}"
    );
}

/// Assert that a block has only deposit transactions (no user txs).
pub fn assert_l2_block_has_deposit_only(test: &DerivationTest, block_num: u64) {
    let blocks = test.l2.blocks();
    let block = &blocks[block_num as usize];
    let user_tx_count = block
        .transactions
        .iter()
        .filter(|tx| !matches!(tx, op_alloy_consensus::OpTxEnvelope::Deposit(_)))
        .count();
    assert_eq!(
        user_tx_count, 0,
        "block {block_num} should be deposit-only but has {user_tx_count} user txs"
    );
}
