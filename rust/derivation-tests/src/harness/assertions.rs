//! Fine-grained debugging assertions for derivation tests.

use alloy_primitives::B256;

use crate::harness::test::DerivationTest;

/// Assert the total number of L2 blocks (including genesis).
pub fn assert_l2_block_count(test: &DerivationTest, expected: usize) {
    let actual = test.l2.blocks().len();
    assert_eq!(actual, expected, "expected {expected} L2 blocks (including genesis), got {actual}");
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

/// Assert that an L2 block contains a transaction with the given hash.
pub fn assert_l2_block_contains_tx(test: &DerivationTest, block_num: u64, tx_hash: B256) {
    let blocks = test.l2.blocks();
    let block = &blocks[block_num as usize];
    let found = block.transactions.iter().any(|tx| tx.tx_hash() == tx_hash);
    assert!(found, "block {block_num} does not contain tx {tx_hash:?}");
}
