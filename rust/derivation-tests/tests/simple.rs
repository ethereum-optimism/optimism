//! Simple derivation test scenarios exercising the full framework.
//!
//! See the crate README for instructions on updating golden super root values.

use alloy_primitives::b256;
use derivation_tests::{
    config::DeterministicConfig,
    harness::{BatchConfig, DerivationTest},
};

/// Expected super root for 3 empty L2 blocks (deposit-only) derived from empty L1 blocks.
const EXPECTED_EMPTY_BLOCKS_ROOT: alloy_primitives::B256 =
    b256!("07aa5a2fbd3d1aa90feb559c2f2728fdb4ede52343f69e7cc2f21b955e0724c0");

/// Expected super root for 1 empty L2 block submitted as a singular batch.
const EXPECTED_SINGLE_BATCH_ROOT: alloy_primitives::B256 =
    b256!("b15f96b2f5acd76c6d4ee122beec68dbd192a847dca78238fc95eeaed4743ec4");

/// Expected super root for 1 L2 block containing a 1 ETH transfer.
const EXPECTED_SINGLE_TRANSFER_ROOT: alloy_primitives::B256 =
    b256!("b050e9d4999932e0ea832e9459ebbb86b284e614bb1503bc8165231dd8796e85");

/// Expected super root for 6 empty L2 blocks (one full epoch).
const EXPECTED_MULTI_BLOCK_ROOT: alloy_primitives::B256 =
    b256!("2c389bb74e8cadd6ef68de97f6f3252fc37d791ffbba47f94a6d94c578d61249");

/// Build a test with N empty L2 blocks derived from N empty L1 blocks.
fn build_empty_blocks_test(l2_count: usize) -> DerivationTest {
    let mut test = DerivationTest::new();
    test.advance_l1(l2_count);
    test.derive_empty_l2_blocks(l2_count);
    test
}

#[test]
fn test_empty_blocks_deterministic() {
    let test1 = build_empty_blocks_test(3);
    let test2 = build_empty_blocks_test(3);

    let root1 = test1.expected_super_root();
    let root2 = test2.expected_super_root();

    assert_eq!(root1, root2, "same inputs should produce same super root");
    assert_eq!(
        root1, EXPECTED_EMPTY_BLOCKS_ROOT,
        "super root mismatch — update EXPECTED_EMPTY_BLOCKS_ROOT if this is an intentional change"
    );
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
    test.advance_l1(2);
    test.derive_empty_l2_block();
    test.submit_batch_with(BatchConfig::singular_calldata());

    let root1 = test.expected_super_root();
    let root2 = test.expected_super_root();
    assert_eq!(root1, root2);
    assert_eq!(
        root1, EXPECTED_SINGLE_BATCH_ROOT,
        "super root mismatch — update EXPECTED_SINGLE_BATCH_ROOT if this is an intentional change"
    );
}

#[test]
fn test_multiple_l2_blocks() {
    let mut test = DerivationTest::new();
    test.advance_l1(5);
    test.derive_empty_l2_blocks(6);
    test.submit_batch_with(BatchConfig::singular_calldata());

    // 7 blocks total: genesis + 6 built
    assert_eq!(test.l2.blocks().len(), 7);

    let root = test.expected_super_root();
    assert_eq!(
        root, EXPECTED_MULTI_BLOCK_ROOT,
        "super root mismatch — update EXPECTED_MULTI_BLOCK_ROOT if this is an intentional change"
    );
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
    use derivation_tests::batch::{ChannelOut, CompressionAlgo, L1Origin, build_span_batch};

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

    let l1_origin =
        L1Origin { number: l1_block.header.inner().number, hash: l1_block.header.hash() };
    let rollup_config = test.config.rollup_config();

    // Build span batch
    let span_batch = build_span_batch(&blocks, l1_origin, rollup_config);
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

#[test]
fn test_single_transfer() {
    use alloy_primitives::{Address, U256};

    let mut test = DerivationTest::new();
    test.advance_l1(2);
    test.derive_l2_block()
        .with_funded_transfer(
            Address::with_last_byte(0x99),
            U256::from(1_000_000_000_000_000_000u64),
        )
        .build();
    test.submit_batch_with(BatchConfig::singular_calldata());

    let root1 = test.expected_super_root();
    let root2 = test.expected_super_root();
    assert_eq!(root1, root2, "super root should be deterministic with a transfer");
    assert_eq!(
        root1, EXPECTED_SINGLE_TRANSFER_ROOT,
        "super root mismatch — update EXPECTED_SINGLE_TRANSFER_ROOT if this is an intentional change"
    );

    // Super root differs from an empty-blocks-only chain
    let empty_test = build_empty_blocks_test(1);
    let empty_root = empty_test.expected_super_root();
    assert_ne!(
        root1, empty_root,
        "transfer should produce a different super root than empty blocks"
    );
}

#[test]
fn test_blob_batch() {
    let mut test = DerivationTest::new();
    test.advance_l1(2);
    test.derive_empty_l2_blocks(3);
    test.submit_batch();

    let root1 = test.expected_super_root();
    let root2 = test.expected_super_root();
    assert_eq!(root1, root2, "super root should be deterministic after blob batch");
}

#[tokio::test]
async fn test_blob_batch_beacon_endpoint() {
    let mut test = DerivationTest::new();
    test.advance_l1(2);
    test.derive_empty_l2_block();
    test.submit_batch();

    let servers = test.serve().await.unwrap();
    let client = reqwest::Client::new();

    // The blob was submitted on the block after the two empty blocks (block 3).
    // Slot = (timestamp - genesis_timestamp) / seconds_per_slot
    let blob_block = test.l1.block_at(3).unwrap();
    let slot = test.l1.timestamp_to_slot(blob_block.header.inner().timestamp);

    let resp: serde_json::Value = client
        .get(format!("{}/eth/v1/beacon/blobs/{}", servers.beacon_url(), slot))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();

    let data = resp["data"].as_array().expect("beacon blobs should have data array");
    assert!(!data.is_empty(), "blob slot should have blob data");

    servers.stop();
}

#[test]
fn test_system_config_update() {
    use derivation_tests::l1::SystemConfigUpdate;
    use kona_genesis::{CONFIG_UPDATE_EVENT_VERSION_0, CONFIG_UPDATE_TOPIC};

    let mut test = DerivationTest::new();

    let new_batcher = alloy_primitives::address!("0xDEAD000000000000000000000000000000000001");
    test.l1.emit_block_with_system_config_update(SystemConfigUpdate::BatcherAddress(new_batcher));

    let block = test.l1.block_at(1).unwrap();

    // Block should have a receipt with a log
    assert_eq!(block.receipts.len(), 1, "should have one receipt");

    // Extract the log from the receipt
    let receipt = &block.receipts[0];
    let logs: Vec<_> = match receipt {
        alloy_consensus::ReceiptEnvelope::Eip1559(rwb) => rwb.receipt.logs.clone(),
        _ => panic!("expected EIP-1559 receipt"),
    };
    assert_eq!(logs.len(), 1, "should have one log");

    let log = &logs[0];
    assert_eq!(log.address, test.config.system_config, "log should be from system config address");
    assert_eq!(log.data.topics().len(), 3, "should have 3 topics");
    assert_eq!(log.data.topics()[0], CONFIG_UPDATE_TOPIC, "topic[0] should be ConfigUpdate");
    assert_eq!(log.data.topics()[1], CONFIG_UPDATE_EVENT_VERSION_0, "topic[1] should be version 0");
}
