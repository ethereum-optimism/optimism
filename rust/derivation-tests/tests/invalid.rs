//! Tests for invalid/adversarial batch scenarios.
//!
//! These tests verify the test framework can construct chains with invalid batches
//! and that `expected_super_root()` remains computable (doesn't panic).

use alloy_primitives::Bytes;
use derivation_tests::{
    batch::{ChannelOut, CompressionAlgo, L1Origin},
    harness::DerivationTest,
};

#[test]
fn test_wrong_batcher_address() {
    let mut test = DerivationTest::new();
    test.advance_l1(2);
    test.derive_empty_l2_block();

    // Drop to low-level for adversarial part
    let fake_batch_data = Bytes::from(vec![0x00, 0xDE, 0xAD, 0xBE, 0xEF]);
    test.l1.emit_block_with_raw_txs(vec![fake_batch_data]);

    let root1 = test.expected_super_root();
    let root2 = test.expected_super_root();
    assert_eq!(root1, root2, "super root should be deterministic even with invalid batches on L1");
}

#[test]
fn test_truncated_frame() {
    let mut test = DerivationTest::new();
    test.advance_l1(2);
    test.derive_empty_l2_block();

    // Drop to low-level for adversarial part
    let truncated_frame = Bytes::from(vec![0x00, 0x01, 0x02]);
    test.l1.emit_block_with_raw_txs(vec![truncated_frame]);

    let random_garbage = Bytes::from(vec![0xFF, 0xFE, 0xFD, 0xFC, 0xFB, 0xFA]);
    test.l1.emit_block_with_raw_txs(vec![random_garbage]);

    let root = test.expected_super_root();
    assert_ne!(
        root,
        alloy_primitives::B256::ZERO,
        "super root should be non-zero even with garbage on L1"
    );
}

#[test]
fn test_future_timestamp_batch() {
    use kona_protocol::SingleBatch;

    let mut test = DerivationTest::new();
    test.advance_l1(2);
    test.derive_empty_l2_block();

    // Drop to low-level — manually construct an invalid batch
    let l1_origin = test.l1.block_at(0).unwrap();
    let origin =
        L1Origin { number: l1_origin.header.inner().number, hash: l1_origin.header.hash() };

    let future_batch = SingleBatch {
        parent_hash: test.l2.head().header.hash(),
        epoch_num: origin.number,
        epoch_hash: origin.hash,
        timestamp: 9_999_999_999, // far in the future
        transactions: vec![],
    };

    // Encode and submit to L1
    let channel_id = [0xFFu8; 16];
    let mut channel = ChannelOut::new(channel_id, CompressionAlgo::Zlib);
    channel.add_singular_batch(&future_batch).unwrap();
    channel.close().unwrap();

    let calldata = channel.to_calldata(100_000);
    let batch =
        derivation_tests::l1::BatchSubmission::Calldata(calldata.into_iter().next().unwrap());
    test.l1.emit_block_with_batches(vec![batch]);

    // The framework should handle this without panicking
    let root1 = test.expected_super_root();
    let root2 = test.expected_super_root();
    assert_eq!(root1, root2, "super root should be deterministic even with future-timestamp batch");
}
