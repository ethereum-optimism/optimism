//! Simple derivation test scenarios exercising the full framework.

use derivation_tests::{config::DeterministicConfig, harness::DerivationTest};

/// Build a test with N empty L2 blocks derived from N empty L1 blocks.
fn build_empty_blocks_test(l2_count: usize) -> DerivationTest {
    let mut test = DerivationTest::new();

    // Emit L1 blocks to serve as epochs
    for _ in 0..l2_count {
        test.l1.emit_empty_block();
    }

    // Set epoch to the first L1 block and build L2 blocks
    let l1_block = test.l1.block_at(1).unwrap().clone();
    test.l2.set_epoch(&l1_block);

    for _ in 0..l2_count {
        test.l2.build_empty_block().unwrap();
    }

    test
}

#[test]
fn test_empty_blocks_deterministic() {
    let test1 = build_empty_blocks_test(3);
    let test2 = build_empty_blocks_test(3);

    let root1 = test1.expected_super_root();
    let root2 = test2.expected_super_root();

    assert_eq!(root1, root2, "same inputs should produce same super root");
}

#[test]
fn test_empty_blocks_structure() {
    let test = build_empty_blocks_test(3);

    // 4 blocks total: genesis + 3 built
    assert_eq!(test.l2.blocks().len(), 4);

    // Check parent chain
    let blocks = test.l2.blocks();
    for i in 1..blocks.len() {
        assert_eq!(
            blocks[i].header.inner().parent_hash,
            blocks[i - 1].header.hash(),
            "block {} parent hash mismatch",
            i
        );
    }

    // Check timestamps increment by L2 block time
    let config = DeterministicConfig::default();
    for i in 1..blocks.len() {
        assert_eq!(
            blocks[i].header.inner().timestamp,
            blocks[i - 1].header.inner().timestamp + config.l2_block_time,
            "block {} timestamp mismatch",
            i
        );
    }
}

#[test]
fn test_single_batch_submission() {
    let mut test = DerivationTest::new();

    // Build L1 blocks
    test.l1.emit_empty_block(); // block 1
    test.l1.emit_empty_block(); // block 2

    // Build L2 blocks
    let l1_block = test.l1.block_at(1).unwrap().clone();
    test.l2.set_epoch(&l1_block);
    let block_ref = test.l2.build_empty_block().unwrap();

    // Encode as a singular batch
    let l1_origin = test.l1.block_at(1).unwrap().clone();
    let batch = test.singular_batch_calldata(&[block_ref], &l1_origin);

    // Submit batch on L1
    test.l1.emit_block_with_batches(vec![batch]);

    // Verify the super root is deterministic
    let root1 = test.expected_super_root();
    let root2 = test.expected_super_root();
    assert_eq!(root1, root2);
}

#[test]
fn test_multiple_l2_blocks() {
    let mut test = DerivationTest::new();

    // Build L1 blocks
    for _ in 0..5 {
        test.l1.emit_empty_block();
    }

    // Build 6 L2 blocks per L1 block (2s L2 / 12s L1 = 6 blocks per epoch)
    let l1_block = test.l1.block_at(1).unwrap().clone();
    test.l2.set_epoch(&l1_block);

    let mut block_refs = Vec::new();
    for _ in 0..6 {
        block_refs.push(test.l2.build_empty_block().unwrap());
    }

    // Encode all blocks as batches and submit
    let l1_origin = test.l1.block_at(1).unwrap().clone();
    let batch = test.singular_batch_calldata(&block_refs, &l1_origin);
    test.l1.emit_block_with_batches(vec![batch]);

    // 7 blocks total: genesis + 6 built
    assert_eq!(test.l2.blocks().len(), 7);

    // Super root is computable
    let _root = test.expected_super_root();
}

#[test]
fn test_l1_chain_structure() {
    let config = DeterministicConfig::default();
    let mut test = DerivationTest::new();

    // Emit 5 empty L1 blocks
    for _ in 0..5 {
        test.l1.emit_empty_block();
    }

    let blocks = test.l1.blocks();
    assert_eq!(blocks.len(), 6); // genesis + 5

    // Check parent chain
    for i in 1..blocks.len() {
        assert_eq!(blocks[i].header.inner().parent_hash, blocks[i - 1].header.hash(),);
    }

    // Check timestamps
    for i in 1..blocks.len() {
        assert_eq!(
            blocks[i].header.inner().timestamp,
            blocks[i - 1].header.inner().timestamp + config.l1_block_time,
        );
    }
}

#[test]
fn test_span_batch_submission() {
    use derivation_tests::batch::{ChannelOut, CompressionAlgo, build_span_batch, L1Origin};

    let mut test = DerivationTest::new();

    // Build L1 blocks
    test.l1.emit_empty_block(); // block 1
    test.l1.emit_empty_block(); // block 2

    // Build multiple L2 blocks in the same epoch
    let l1_block = test.l1.block_at(1).unwrap().clone();
    test.l2.set_epoch(&l1_block);

    let mut block_refs = Vec::new();
    for _ in 0..3 {
        block_refs.push(test.l2.build_empty_block().unwrap());
    }

    // Collect block references for span batch construction
    let blocks: Vec<&_> = block_refs.iter().map(|r| test.l2.block(*r)).collect();

    let l1_origin = L1Origin {
        number: l1_block.header.inner().number,
        hash: l1_block.header.hash(),
    };
    let rollup_config = test.config.rollup_config();

    // Build span batch
    let span_batch = build_span_batch(&blocks, l1_origin, &rollup_config);
    assert_eq!(span_batch.batches.len(), 3);

    // Encode into channel and produce calldata
    let channel_id = [0xFFu8; 16];
    let mut channel = ChannelOut::new(channel_id, CompressionAlgo::Zlib);
    channel.add_span_batch(&span_batch).unwrap();
    channel.close().unwrap();

    let calldata = channel.to_calldata(100_000);
    assert!(!calldata.is_empty(), "expected at least one frame of calldata");

    // First byte is DerivationVersion0
    assert_eq!(calldata[0][0], 0x00);

    // Submit as batch on L1
    let batch_submission =
        derivation_tests::l1::BatchSubmission::Calldata(calldata.into_iter().next().unwrap());
    test.l1.emit_block_with_batches(vec![batch_submission]);

    // Verify super root is still computable and deterministic
    let root1 = test.expected_super_root();
    let root2 = test.expected_super_root();
    assert_eq!(root1, root2, "super root should be deterministic after span batch submission");
}

/// Integration test that requires op-program binary.
/// Run with: OP_PROGRAM_PATH=/path/to/op-program cargo test --ignored
#[test]
#[ignore]
fn test_op_program_empty_blocks() {
    // This test would start servers and run op-program.
    // Placeholder until we have the binary available.
    let _test = build_empty_blocks_test(3);
    eprintln!("op-program integration test not yet implemented");
}
